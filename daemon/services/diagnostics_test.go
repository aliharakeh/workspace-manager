package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

func diagnosticFixture(t *testing.T) (*Runner, *session) {
	t.Helper()
	t.Setenv("WM_DATA_DIR", t.TempDir())
	d := newBlueprintTestDB(t)
	ctx := context.Background()
	ws, err := d.CreateWorkspaceT(ctx, "diagnostics", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	app, err := d.CreateAppT(ctx, ws.ID, "app", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	set, err := d.ResolveActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	s := &session{appID: app.ID, id: "first", setID: set.ID, kind: types.KindDev, outputs: map[int64]*termBuffer{}}
	s.diagnostics, err = newRunDiagnostics(d, s)
	if err != nil {
		t.Fatal(err)
	}
	r := NewRunner(d, nil, nil)
	r.sessions[app.ID] = s
	return r, s
}

func TestDiagnosticMatchers(t *testing.T) {
	for _, tt := range []struct {
		line, severity, file string
		row, col             int
	}{
		{`main.go:12:3: undefined: missing`, "error", "main.go", 12, 3},
		{`C:\src\app.ts(4,9): error TS2322: wrong type`, "error", `C:\src\app.ts`, 4, 9},
		{`src/app.ts:8:2: warning: unused variable`, "warning", "src/app.ts", 8, 2},
		{`TypeError: cannot read properties`, "error", "", 0, 0},
		{`panic: broken`, "error", "", 0, 0},
		{`listen EADDRINUSE: address already in use`, "error", "", 0, 0},
		{`{"level":"ERROR","msg":"connection refused"}`, "error", "", 0, 0},
		{`{"level":"info","msg":"error count 3"}`, "", "", 0, 0},
		{`Found 0 errors. Watching for file changes.`, "", "", 0, 0},
		{`errors: 0`, "", "", 0, 0},
		{`server listening at http://localhost:5174`, "", "", 0, 0},
		{`environment loaded`, "", "", 0, 0},
	} {
		t.Run(tt.line, func(t *testing.T) {
			d, ok := parseDiagnostic(tt.line)
			if ok != (tt.severity != "") || d.Severity != tt.severity || d.File != tt.file || d.Line != tt.row || d.Column != tt.col {
				t.Fatalf("got %+v, detected=%v", d, ok)
			}
		})
	}
}

func TestDiagnosticsShutdownFlushAndPersistenceFailure(t *testing.T) {
	r, s := diagnosticFixture(t)
	s.diagnostics.failure(1, "captured before shutdown")
	r.StopAll(context.Background())
	restarted := NewRunner(r.db, nil, nil)
	retained, err := restarted.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || len(retained.Items) != 1 {
		t.Fatalf("shutdown snapshot: %+v %v", retained, err)
	}
	if _, err := r.db.SQL.Exec(`CREATE TRIGGER fail_diagnostic_save BEFORE UPDATE ON runner_diagnostics BEGIN SELECT RAISE(FAIL, 'simulated write failure'); END`); err != nil {
		t.Fatal(err)
	}
	s.diagnostics.failure(1, "new error during disk failure")
	s.diagnostics.persist()
	result, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || result.PersistenceError == "" || len(result.Items) != 2 {
		t.Fatalf("persistence failure hidden: %+v %v", result, err)
	}
	if _, err := r.db.SQL.Exec(`DROP TRIGGER fail_diagnostic_save`); err != nil {
		t.Fatal(err)
	}
	s.diagnostics.persist()
	result, err = r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || result.PersistenceError != "" {
		t.Fatalf("persistence did not recover: %+v %v", result, err)
	}
}

func TestDiagnosticsOutliveScrollbackAndPersist(t *testing.T) {
	r, s := diagnosticFixture(t)
	chunks := []string{"starting\r\n\x1b[3", "1mTypeEr", "ror: broken\x1b[0m\r\n", "    at start (app.js:2:1)\r\n", "ready\r\nTypeError: broken\r\n"}
	for _, chunk := range chunks {
		r.emitOutput(s, 11, []byte(chunk))
		s.diagnostics.feed(11, []byte(chunk), false)
	}
	r.emitOutput(s, 11, []byte(strings.Repeat("ordinary output\n", 200000)))
	output, _ := base64.StdEncoding.DecodeString(r.GetOutput(s.appID, 11).Data)
	if strings.Contains(string(output), "TypeError") {
		t.Fatal("test did not evict error from terminal")
	}
	page, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || len(page.Items) != 1 || page.Items[0].Count != 2 || page.Items[0].Context != "" {
		t.Fatalf("summary: %+v %v", page, err)
	}
	detail, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, SessionID: s.id, ID: page.Items[0].ID})
	if err != nil || !strings.Contains(detail.Items[0].Context, "at start") {
		t.Fatalf("detail: %+v %v", detail, err)
	}
	s.diagnostics.persist()
	restarted := NewRunner(r.db, nil, nil)
	retained, err := restarted.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || len(retained.Items) != 1 || retained.Items[0].Count != 2 {
		t.Fatalf("retained: %+v %v", retained, err)
	}
	// An old run flushing late cannot overwrite a newer run's snapshot.
	newSession := &session{appID: s.appID, id: "second", setID: s.setID}
	newSession.diagnostics, err = newRunDiagnostics(r.db, newSession)
	if err != nil {
		t.Fatal(err)
	}
	s.diagnostics.failure(11, "late old error")
	s.diagnostics.persist()
	retained, err = restarted.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || retained.SessionID != "second" || len(retained.Items) != 0 {
		t.Fatalf("session overwritten: %+v %v", retained, err)
	}
	if _, err := r.db.DeleteApp(context.Background(), s.appID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.db.GetRunnerDiagnostics(context.Background(), s.appID); err == nil {
		t.Fatal("app deletion retained diagnostics")
	}
}

func TestDiagnosticsIncrementalQueriesAndLimits(t *testing.T) {
	r, s := diagnosticFixture(t)
	c := s.diagnostics
	c.feed(1, []byte("error: first\nwarning: second\n"), false)
	first, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, Limit: 1})
	if err != nil || !first.HasMore || len(first.Items) != 1 {
		t.Fatalf("first page %+v %v", first, err)
	}
	second, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, SessionID: s.id, Cursor: first.NextCursor})
	if err != nil || second.HasMore || len(second.Items) != 1 {
		t.Fatalf("second page %+v %v", second, err)
	}
	c.feed(1, []byte("error: first\n"), false)
	updates, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, SessionID: s.id, Cursor: second.NextCursor})
	if err != nil || len(updates.Items) != 1 || updates.Items[0].Count != 2 {
		t.Fatalf("update %+v %v", updates, err)
	}
	filtered, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, Severity: "warning", CommandID: 1})
	if err != nil || len(filtered.Items) != 1 || filtered.ErrorGroups != 1 || filtered.WarningGroups != 1 {
		t.Fatalf("filtered %+v %v", filtered, err)
	}
	for _, q := range []types.DiagnosticsQuery{
		{Cursor: 1}, {ID: 1}, {Limit: 21}, {Cursor: -1}, {CommandID: -1}, {Severity: "info"}, {SessionID: "old", Cursor: 1}, {SessionID: s.id, Cursor: 999},
	} {
		q.AppID = s.appID
		if _, err := r.Diagnostics(q); err == nil {
			t.Fatalf("accepted %+v", q)
		}
	}
	if _, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, SessionID: "old"}); api.KindOf(err) != api.KindConflict {
		t.Fatalf("stale session: %v", err)
	}
	for i := 0; i < maxDiagnostics+5; i++ {
		c.failure(1, fmt.Sprintf("failure %d", i))
	}
	page, _ := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if page.Dropped != 7 || page.Occurrences != int64(maxDiagnostics+8) {
		t.Fatalf("retention %+v", page)
	}
}

func TestDiagnosticsMultilineAndTruncation(t *testing.T) {
	r, s := diagnosticFixture(t)
	c := s.diagnostics
	c.feed(1, []byte("Traceback (most recent call last):\n  File \"a.py\", line 1\n    fail()\nValueError: bad\n"), false)
	c.feed(1, []byte("Traceback (most recent call last):\n  File \"b.py\", line 2\nTypeError: other\n"), false)
	c.feed(2, []byte("panic: broken\n\ngoroutine 1 [running]:\nmain.main()\n\tC:/src/main.go:2 +0x1\n"), true)
	c.feed(3, []byte("error: "+strings.Repeat("界", 40000)), false)
	c.feed(3, nil, true)
	page, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || len(page.Items) != 4 || !page.InputTruncated {
		t.Fatalf("multiline %+v %v", page, err)
	}
	for _, item := range page.Items {
		detail, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, SessionID: s.id, ID: item.ID})
		if err != nil {
			t.Fatal(err)
		}
		context := detail.Items[0].Context
		if !utf8.ValidString(context) || len(context) > maxDiagnosticContext {
			t.Fatal("invalid context")
		}
		if item.CommandID == 2 && !strings.Contains(context, "main.main()") {
			t.Fatalf("missing Go stack: %s", context)
		}
	}
	c.feed(4, []byte("error: long stack\n"+strings.Repeat("    frame\n", 2000)), true)
	page, _ = r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, CommandID: 4})
	if len(page.Items) != 1 || !page.Items[0].ContextTruncated {
		t.Fatal("missing context truncation flag")
	}
	c.feed(5, []byte("error: unfinished"), true)
	page, _ = r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID, CommandID: 5})
	if len(page.Items) != 1 {
		t.Fatal("unfinished last line not captured")
	}
}

func TestDiagnosticsParallelCommands(t *testing.T) {
	r, s := diagnosticFixture(t)
	var wg sync.WaitGroup
	for command := int64(1); command <= 4; command++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				s.diagnostics.feed(id, []byte("error: repeated\n"), false)
			}
		}(command)
	}
	wg.Wait()
	page, err := r.Diagnostics(types.DiagnosticsQuery{AppID: s.appID})
	if err != nil || len(page.Items) != 4 || page.Occurrences != 120 {
		t.Fatalf("parallel %+v %v", page, err)
	}
}
