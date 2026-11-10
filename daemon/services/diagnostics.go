package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

const (
	maxDiagnostics       = 512
	maxDiagnosticContext = 8192
	maxDiagnosticMessage = 1024
	diagnosticDetection  = "Pattern-based detection; no detected errors does not establish application health. Problems remain historical until a new run; resolution is not inferred."
)

type diagnosticSnapshot struct {
	AppID          int64              `json:"appId"`
	SessionID      string             `json:"sessionId"`
	ConfigSetID    int64              `json:"configSetId"`
	Kind           string             `json:"kind"`
	Revision       int64              `json:"revision"`
	Occurrences    int64              `json:"occurrences"`
	Dropped        int64              `json:"dropped"`
	InputTruncated bool               `json:"inputTruncated"`
	Items          []types.Diagnostic `json:"items"`
}

type diagnosticTap struct {
	lines          lineTap
	before         []string
	active         int64
	trace          string
	traceTruncated bool
}

// runDiagnostics is independent of the terminal buffer. Memory and the persisted
// snapshot are bounded; SQLite retains only the latest run of each app.
type runDiagnostics struct {
	mu               sync.Mutex
	persistMu        sync.Mutex
	db               *db.DB
	data             diagnosticSnapshot
	taps             map[int64]*diagnosticTap
	keys             map[string]int
	dirty            bool
	persistenceError string
}

func newRunDiagnostics(d *db.DB, s *session) (*runDiagnostics, error) {
	c := &runDiagnostics{db: d, taps: map[int64]*diagnosticTap{}, keys: map[string]int{},
		data: diagnosticSnapshot{AppID: s.appID, SessionID: s.id, ConfigSetID: s.setID, Kind: s.kind, Items: []types.Diagnostic{}}}
	raw, err := json.Marshal(c.data)
	if err != nil {
		return nil, err
	}
	err = d.BeginRunnerDiagnostics(context.Background(), db.BeginRunnerDiagnosticsParams{AppID: s.appID, SessionID: s.id, Snapshot: string(raw)})
	return c, err
}

func (c *runDiagnostics) persistLoop(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c.persist()
		}
	}
}

func (c *runDiagnostics) persist() {
	c.persistMu.Lock()
	defer c.persistMu.Unlock()
	c.mu.Lock()
	if !c.dirty {
		c.mu.Unlock()
		return
	}
	revision := c.data.Revision
	raw, err := json.Marshal(c.data)
	c.mu.Unlock()
	if err == nil {
		err = c.db.SaveRunnerDiagnostics(context.Background(), db.SaveRunnerDiagnosticsParams{AppID: c.data.AppID, SessionID: c.data.SessionID, Snapshot: string(raw)})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.persistenceError = err.Error()
		return
	}
	c.persistenceError = ""
	if c.data.Revision == revision {
		c.dirty = false
	}
}

// boundedText preserves valid UTF-8 when limiting excerpts by bytes.
func boundedText(s string, limit int) (string, bool) {
	s = strings.ToValidUTF8(s, "\uFFFD")
	if len(s) <= limit {
		return s, false
	}
	s = s[:limit]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s, true
}

var (
	diagnosticSeverity     = regexp.MustCompile(`(?i)(?:\b(?:fatal|error|panic|exception)\b|\b\w+(?:Error|Exception)\b|\bERR_[A-Z_]+\b|\b(?:EADDRINUSE|ECONNREFUSED|ENOENT|EACCES|EPERM)\b)`)
	diagnosticWarning      = regexp.MustCompile(`(?i)\bwarn(?:ing)?\b`)
	diagnosticNegative     = regexp.MustCompile(`(?i)\b(?:no|zero|0)\s+(?:errors?|warnings?)\b|\b(?:errors?|warnings?)\s*[:=]\s*0\b`)
	diagnosticLocation     = regexp.MustCompile(`^(.+\.[a-zA-Z0-9]+):(\d+)(?::(\d+))?:\s*(.+)$`)
	diagnosticTSLocation   = regexp.MustCompile(`^(.+\.[a-zA-Z0-9]+)\((\d+),(\d+)\):\s*(.+)$`)
	diagnosticContinuation = regexp.MustCompile(`^\s|^Caused by:|^goroutine \d+|^created by |^[\w./()*]+\(.*\)$`)
)

func parseDiagnostic(line string) (types.Diagnostic, bool) {
	d := types.Diagnostic{Message: strings.TrimSpace(line), Source: "pattern"}
	// Prefer explicit levels in JSON logs to guessing from the message text.
	var log map[string]any
	if json.Unmarshal([]byte(line), &log) == nil && log != nil {
		level, _ := log["level"].(string)
		if level == "" {
			level, _ = log["severity"].(string)
		}
		switch strings.ToLower(level) {
		case "error", "fatal", "panic":
			d.Severity = "error"
		case "warn", "warning":
			d.Severity = "warning"
		default:
			return d, false
		}
		d.Source = "structured"
		for _, key := range []string{"message", "msg"} {
			if message, ok := log[key].(string); ok {
				d.Message = message
				break
			}
		}
		return d, true
	}
	if diagnosticNegative.MatchString(line) {
		return d, false
	}
	if diagnosticSeverity.MatchString(line) {
		d.Severity = "error"
	} else if diagnosticWarning.MatchString(line) {
		d.Severity = "warning"
	}
	location := diagnosticLocation.FindStringSubmatch(strings.TrimSpace(line))
	if location == nil {
		location = diagnosticTSLocation.FindStringSubmatch(strings.TrimSpace(line))
	}
	if location != nil {
		d.File = location[1]
		d.Line, _ = strconv.Atoi(location[2])
		d.Column, _ = strconv.Atoi(location[3])
		d.Source = "compiler"
		// Go diagnostics have file:line:column without an "error" prefix.
		if d.Severity == "" {
			d.Severity = "error"
		}
	}
	if strings.HasPrefix(strings.TrimSpace(line), "Traceback (most recent call last):") {
		d.Severity, d.Source = "error", "runtime"
	}
	if strings.HasPrefix(strings.TrimSpace(line), "panic:") || strings.Contains(strings.SplitN(strings.TrimSpace(line), ":", 2)[0], "Error") {
		d.Source = "runtime"
	}
	return d, d.Severity != ""
}

func (c *runDiagnostics) changedLocked() {
	c.data.Revision++
	c.dirty = true
}

func (c *runDiagnostics) addLocked(commandID int64, d types.Diagnostic, context string) int64 {
	message, truncated := boundedText(d.Message, maxDiagnosticMessage)
	key := fmt.Sprintf("%d\x00%s\x00%s\x00%s", commandID, d.Severity, d.Source, d.Message)
	c.data.Occurrences++
	c.changedLocked()
	if index, ok := c.keys[key]; ok {
		item := &c.data.Items[index]
		item.Count++
		item.LastSeen, item.Revision = time.Now().UnixMilli(), c.data.Revision
		return item.ID
	}
	if len(c.data.Items) >= maxDiagnostics {
		c.data.Dropped++
		return 0
	}
	d.ID, d.CommandID, d.Message = int64(len(c.data.Items)+1), commandID, message
	d.Context, d.ContextTruncated = boundedText(context, maxDiagnosticContext)
	d.ContextTruncated = d.ContextTruncated || truncated
	d.Count, d.FirstSeen, d.LastSeen, d.Revision = 1, time.Now().UnixMilli(), time.Now().UnixMilli(), c.data.Revision
	c.keys[key] = len(c.data.Items)
	c.data.Items = append(c.data.Items, d)
	return d.ID
}

func (c *runDiagnostics) lineLocked(commandID int64, tap *diagnosticTap, line string) {
	line, truncated := boundedText(line, maxDiagnosticContext)
	if truncated {
		c.data.InputTruncated = true
		c.changedLocked()
	}
	d, detected := parseDiagnostic(line)
	if strings.HasPrefix(strings.TrimSpace(line), "Traceback (most recent call last):") {
		c.flushTraceLocked(commandID, tap)
		tap.active = 0
		tap.trace, tap.traceTruncated = boundedText(strings.Join(append(tap.before, line), "\n"), maxDiagnosticContext)
		return
	}
	if tap.trace != "" {
		text, cut := boundedText(tap.trace+"\n"+line, maxDiagnosticContext)
		tap.trace, tap.traceTruncated = text, tap.traceTruncated || cut || truncated
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// The final exception names the group, so different tracebacks are not merged.
			d.Severity, d.Source = "error", "runtime"
			id := c.addLocked(commandID, d, tap.trace)
			if id > 0 {
				c.data.Items[id-1].ContextTruncated = c.data.Items[id-1].ContextTruncated || tap.traceTruncated
			}
			tap.trace, tap.traceTruncated = "", false
		}
		return
	}
	continuation := diagnosticContinuation.MatchString(line) && !detected
	if tap.active > 0 && strings.TrimSpace(line) != line && d.Source == "compiler" && c.data.Items[tap.active-1].Source == "runtime" {
		continuation = true
	}
	if tap.active > 0 && continuation {
		item := &c.data.Items[tap.active-1]
		text, cut := boundedText(item.Context+"\n"+line, maxDiagnosticContext)
		item.Context, item.ContextTruncated = text, item.ContextTruncated || cut || truncated
		c.changedLocked()
		item.Revision = c.data.Revision
	} else if detected {
		tap.active = c.addLocked(commandID, d, strings.Join(append(tap.before, line), "\n"))
	} else {
		tap.active = 0
	}
	short, _ := boundedText(line, 512)
	tap.before = append(tap.before, short)
	if len(tap.before) > 3 {
		tap.before = tap.before[1:]
	}
}

func (c *runDiagnostics) flushTraceLocked(commandID int64, tap *diagnosticTap) {
	if tap.trace == "" {
		return
	}
	id := c.addLocked(commandID, types.Diagnostic{Message: "Incomplete Python traceback", Severity: "error", Source: "runtime"}, tap.trace)
	if id > 0 {
		c.data.Items[id-1].ContextTruncated = true
	}
	tap.trace, tap.traceTruncated = "", false
}

func (c *runDiagnostics) feed(commandID int64, chunk []byte, flush bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	tap := c.taps[commandID]
	if tap == nil {
		tap = &diagnosticTap{lines: lineTap{keepStart: true}}
		c.taps[commandID] = tap
	}
	for _, line := range tap.lines.feed(chunk) {
		c.lineLocked(commandID, tap, line)
	}
	if tap.lines.truncated {
		c.data.InputTruncated = true
		c.changedLocked()
		tap.lines.truncated = false
	}
	if flush {
		for _, line := range tap.lines.flush() {
			c.lineLocked(commandID, tap, line)
		}
		c.flushTraceLocked(commandID, tap)
	}
}

func (c *runDiagnostics) failure(commandID int64, message string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.addLocked(commandID, types.Diagnostic{Severity: "error", Source: "runner", Message: message}, message)
}

func (c *runDiagnostics) query(q types.DiagnosticsQuery) (types.DiagnosticsResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out, err := queryDiagnostics(c.data, q)
	out.PersistenceError = c.persistenceError
	return out, err
}

func queryDiagnostics(data diagnosticSnapshot, q types.DiagnosticsQuery) (types.DiagnosticsResult, error) {
	out := types.DiagnosticsResult{AppID: data.AppID, SessionID: data.SessionID, ConfigSetID: data.ConfigSetID, Kind: data.Kind,
		Items: []types.Diagnostic{}, Dropped: data.Dropped, InputTruncated: data.InputTruncated, Occurrences: data.Occurrences,
		NextCursor: data.Revision, Detection: diagnosticDetection}
	if q.SessionID != "" && q.SessionID != data.SessionID {
		return out, api.Errf(api.KindConflict, "diagnostic session changed; query the new run without a cursor")
	}
	if q.Cursor > data.Revision {
		return out, api.Errf(api.KindInvalid, "cursor exceeds the session's latest revision")
	}
	var matches []types.Diagnostic
	for _, item := range data.Items {
		if item.Severity == "error" {
			out.ErrorGroups++
		} else {
			out.WarningGroups++
		}
		if q.ID != 0 && item.ID != q.ID {
			continue
		}
		if q.CommandID != 0 && item.CommandID != q.CommandID {
			continue
		}
		if q.Severity != "" && item.Severity != q.Severity {
			continue
		}
		if q.ID == 0 && item.Revision <= q.Cursor {
			continue
		}
		if q.ID == 0 {
			item.Context = ""
		}
		matches = append(matches, item)
	}
	if q.ID != 0 && len(matches) == 0 {
		return out, api.Errf(api.KindNotFound, "diagnostic not found in this session")
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Revision < matches[j].Revision })
	limit := q.Limit
	if limit == 0 {
		limit = 10
	}
	if len(matches) > limit {
		matches = matches[:limit]
		out.HasMore = true
		out.NextCursor = matches[len(matches)-1].Revision
	}
	out.Items = append(out.Items, matches...)
	return out, nil
}

func (r *Runner) Diagnostics(q types.DiagnosticsQuery) (types.DiagnosticsResult, error) {
	if q.Limit < 0 || q.Limit > 20 || q.Cursor < 0 || q.ID < 0 || q.CommandID < 0 {
		return types.DiagnosticsResult{}, api.Errf(api.KindInvalid, "limit must be 0..20; cursor and ids must be nonnegative")
	}
	if q.Severity != "" && q.Severity != "error" && q.Severity != "warning" {
		return types.DiagnosticsResult{}, api.Errf(api.KindInvalid, "severity must be error or warning")
	}
	if (q.Cursor > 0 || q.ID > 0) && q.SessionID == "" {
		return types.DiagnosticsResult{}, api.Errf(api.KindInvalid, "sessionId is required with a cursor or diagnostic id")
	}
	if _, err := r.db.GetAppT(context.Background(), q.AppID); err != nil {
		return types.DiagnosticsResult{}, err
	}
	r.mu.Lock()
	s := r.sessions[q.AppID]
	r.mu.Unlock()
	if s != nil && s.diagnostics != nil {
		return s.diagnostics.query(q)
	}
	raw, err := r.db.GetRunnerDiagnostics(context.Background(), q.AppID)
	if errors.Is(err, sql.ErrNoRows) {
		return types.DiagnosticsResult{}, api.Errf(api.KindNotFound, "no retained diagnostic session for this app")
	}
	if err != nil {
		return types.DiagnosticsResult{}, err
	}
	var data diagnosticSnapshot
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return types.DiagnosticsResult{}, err
	}
	return queryDiagnostics(data, q)
}
