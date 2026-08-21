package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"wails_backend/internal/db"
	"wails_backend/internal/native"
	"wails_backend/internal/types"
)

const maxLogBuffer = 5000

type childProc struct {
	cmd *exec.Cmd
}

type session struct {
	id              string
	appID           int64
	mode            string
	processes       []types.ProcessState
	children        map[int64]*childProc
	running         bool
	abortSequential bool
	restored        bool
	logBuffer       []types.LogEvent
}

type Runner struct {
	db        *db.DB
	broadcast func(appID int64, event any)

	mu       sync.Mutex
	sessions map[int64]*session
}

func NewRunner(d *db.DB, broadcast func(appID int64, event any)) *Runner {
	return &Runner{db: d, broadcast: broadcast, sessions: map[int64]*session{}}
}

func (r *Runner) emit(s *session, event any) {
	if log, ok := event.(types.LogEvent); ok {
		s.logBuffer = append(s.logBuffer, log)
		if len(s.logBuffer) > maxLogBuffer {
			s.logBuffer = s.logBuffer[len(s.logBuffer)-maxLogBuffer:]
		}
	}
	if r.broadcast != nil {
		r.broadcast(s.appID, event)
	}
}

func (r *Runner) statusEvent(s *session, errMsg string) types.StatusEvent {
	procs := make([]types.ProcessState, len(s.processes))
	copy(procs, s.processes)
	for i := range procs {
		if procs[i].URLs == nil {
			procs[i].URLs = []string{}
		}
	}
	ev := types.StatusEvent{
		Type: "status", SessionID: s.id, AppID: s.appID,
		Running: s.running, Processes: procs, Ts: time.Now().UnixMilli(),
	}
	if errMsg != "" {
		ev.Error = errMsg
	}
	return ev
}

func (r *Runner) systemLog(s *session, commandID int64, text string) {
	if !stringsHasNL(text) {
		text += "\n"
	}
	r.emit(s, types.LogEvent{
		Type: "log", AppID: s.appID, CommandID: commandID, Stream: "system",
		Text: text, Ts: time.Now().UnixMilli(),
	})
}

func stringsHasNL(s string) bool {
	return len(s) > 0 && s[len(s)-1] == '\n'
}

func (r *Runner) noteReadyURL(s *session, commandID int64, line string) {
	if !s.running {
		return
	}
	match := MatchReadyURL(context.Background(), r.db, line)
	if match == nil {
		return
	}
	for i := range s.processes {
		if s.processes[i].CommandID != commandID {
			continue
		}
		for _, u := range s.processes[i].URLs {
			if u == match.URL {
				return
			}
		}
		s.processes[i].URLs = append(s.processes[i].URLs, match.URL)
		r.systemLog(s, commandID, fmt.Sprintf("Detected URL (%s): %s", match.Label, match.URL))
		r.emit(s, r.statusEvent(s, ""))
		return
	}
}

func (r *Runner) clearReadyURLs(s *session) {
	for i := range s.processes {
		s.processes[i].URLs = []string{}
	}
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07]*\x07`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func (r *Runner) emitLogLine(s *session, commandID int64, kind, text string) {
	if !stringsHasNL(text) {
		text += "\n"
	}
	r.emit(s, types.LogEvent{
		Type: "log", AppID: s.appID, CommandID: commandID, Stream: kind,
		Text: text, Ts: time.Now().UnixMilli(),
	})
	r.noteReadyURL(s, commandID, text)
}

func (r *Runner) pipeStream(s *session, commandID int64, reader io.Reader, kind string) {
	buf := bufio.NewReader(reader)
	var pending string
	for {
		chunk, err := buf.ReadString('\n')
		pending += chunk
		for {
			i := indexNL(pending)
			if i < 0 {
				break
			}
			line := stripANSI(trimCR(pending[:i]))
			pending = pending[i+1:]
			r.emitLogLine(s, commandID, kind, line)
		}
		if err != nil {
			rest := stripANSI(trimCR(pending))
			if rest != "" {
				r.emitLogLine(s, commandID, kind, rest)
			}
			return
		}
	}
}

func indexNL(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}

func trimCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}

func (r *Runner) spawnCommand(s *session, cmd types.RunCommand, cwd string, env []string) int {
	var processState *types.ProcessState
	for i := range s.processes {
		if s.processes[i].CommandID == cmd.ID {
			processState = &s.processes[i]
			break
		}
	}
	if processState == nil {
		return 1
	}
	processState.Status = "running"
	r.emit(s, r.statusEvent(s, ""))
	r.systemLog(s, cmd.ID, "$ "+cmd.Command)

	child, err := native.SpawnShell(cmd.Command, cwd, env)
	if err != nil {
		processState.Status = "error"
		code := int64(1)
		processState.ExitCode = &code
		r.systemLog(s, cmd.ID, "Failed to start: "+err.Error())
		r.emit(s, r.statusEvent(s, ""))
		return 1
	}
	stdout, err := child.StdoutPipe()
	if err != nil {
		processState.Status = "error"
		code := int64(1)
		processState.ExitCode = &code
		r.systemLog(s, cmd.ID, "Failed to start: "+err.Error())
		r.emit(s, r.statusEvent(s, ""))
		return 1
	}
	stderr, err := child.StderrPipe()
	if err != nil {
		processState.Status = "error"
		code := int64(1)
		processState.ExitCode = &code
		r.systemLog(s, cmd.ID, "Failed to start: "+err.Error())
		r.emit(s, r.statusEvent(s, ""))
		return 1
	}
	if err := child.Start(); err != nil {
		processState.Status = "error"
		code := int64(1)
		processState.ExitCode = &code
		r.systemLog(s, cmd.ID, "Failed to start: "+err.Error())
		r.emit(s, r.statusEvent(s, ""))
		return 1
	}
	pid := int64(child.Process.Pid)
	processState.PID = &pid
	r.mu.Lock()
	s.children[cmd.ID] = &childProc{cmd: child}
	r.mu.Unlock()
	r.emit(s, r.statusEvent(s, ""))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); r.pipeStream(s, cmd.ID, stdout, "stdout") }()
	go func() { defer wg.Done(); r.pipeStream(s, cmd.ID, stderr, "stderr") }()
	waitErr := child.Wait()
	wg.Wait()

	r.mu.Lock()
	delete(s.children, cmd.ID)
	r.mu.Unlock()

	if processState.Status == "killed" {
		r.emit(s, r.statusEvent(s, ""))
		return 1
	}
	exitCode := 0
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = 1
		}
	}
	code64 := int64(exitCode)
	processState.ExitCode = &code64
	if exitCode == 0 {
		processState.Status = "exited"
		r.systemLog(s, cmd.ID, fmt.Sprintf("Process exited with code %d", exitCode))
	} else {
		processState.Status = "error"
		r.systemLog(s, cmd.ID, fmt.Sprintf("Process failed with code %d", exitCode))
	}
	r.emit(s, r.statusEvent(s, ""))
	return exitCode
}

func (r *Runner) runSession(s *session) {
	ctx := context.Background()
	app, err := r.db.GetAppT(ctx, s.appID)
	if err != nil {
		s.running = false
		r.emit(s, r.statusEvent(s, "App not found"))
		return
	}
	if err := ApplyTemplates(ctx, r.db, s.appID, s.id); err != nil {
		s.running = false
		msg := err.Error()
		cmdID := int64(0)
		if len(s.processes) > 0 {
			cmdID = s.processes[0].CommandID
		}
		r.systemLog(s, cmdID, "Template apply failed: "+msg)
		r.emit(s, r.statusEvent(s, "Template apply failed: "+msg))
		return
	}
	cmdID := int64(0)
	if len(s.processes) > 0 {
		cmdID = s.processes[0].CommandID
	}
	r.systemLog(s, cmdID, "Templates applied")

	set, err := r.db.ResolveActive(ctx, s.appID)
	if err != nil {
		s.running = false
		r.emit(s, r.statusEvent(s, err.Error()))
		return
	}
	envMap, _ := r.db.EnvToRecord(ctx, set.ID)
	env := native.MergeSpawnEnv(envMap)
	config, err := r.db.GetRunConfigByConfigSetT(ctx, set.ID)
	if err != nil {
		s.running = false
		r.emit(s, r.statusEvent(s, err.Error()))
		return
	}
	var commands []types.RunCommand
	if config != nil {
		commands = config.Commands
	}

	func() {
		defer func() {
			s.running = false
			r.clearReadyURLs(s)
			if !s.restored {
				_ = RestoreTemplates(ctx, r.db, s.appID, s.id)
				s.restored = true
				r.systemLog(s, cmdID, "Original files restored")
			}
			r.emit(s, r.statusEvent(s, ""))
		}()
		if s.mode == "parallel" {
			var wg sync.WaitGroup
			for _, c := range commands {
				c := c
				wg.Add(1)
				go func() { defer wg.Done(); r.spawnCommand(s, c, app.ProjectPath, env) }()
			}
			wg.Wait()
			return
		}
		for _, c := range commands {
			if s.abortSequential {
				break
			}
			code := r.spawnCommand(s, c, app.ProjectPath, env)
			if code != 0 {
				r.systemLog(s, c.ID, "Sequential run stopped due to non-zero exit")
				break
			}
		}
	}()
}

func (r *Runner) createSession(ctx context.Context, appID int64) (*session, error) {
	set, err := r.db.ResolveActive(ctx, appID)
	if err != nil {
		return nil, err
	}
	config, err := r.db.GetOrCreateRunConfig(ctx, set.ID)
	if err != nil {
		return nil, err
	}
	if len(config.Commands) == 0 {
		return nil, fmt.Errorf("No run commands configured")
	}
	procs := make([]types.ProcessState, 0, len(config.Commands))
	for _, cmd := range config.Commands {
		label := cmd.Command
		if cmd.Label != nil && *cmd.Label != "" {
			label = *cmd.Label
		}
		procs = append(procs, types.ProcessState{
			CommandID: cmd.ID, Label: label, Command: cmd.Command,
			Status: "pending", URLs: []string{},
		})
	}
	return &session{
		id:        fmt.Sprintf("%d-%d", appID, time.Now().UnixMilli()),
		appID:     appID,
		mode:      config.Mode,
		processes: procs,
		children:  map[int64]*childProc{},
		running:   true,
	}, nil
}

func (r *Runner) GetStatus(appID int64) types.StatusEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.sessions[appID]
	if s == nil {
		return types.StatusEvent{Type: "status", AppID: appID, Processes: []types.ProcessState{}, Ts: time.Now().UnixMilli()}
	}
	return r.statusEvent(s, "")
}

func (r *Runner) GetSnapshot(appID int64) types.RunnerLogsSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.sessions[appID]
	if s == nil {
		return types.RunnerLogsSnapshot{
			Status: types.StatusEvent{Type: "status", AppID: appID, Processes: []types.ProcessState{}, Ts: time.Now().UnixMilli()},
			Logs:   []types.LogEvent{},
		}
	}
	logs := make([]types.LogEvent, len(s.logBuffer))
	copy(logs, s.logBuffer)
	return types.RunnerLogsSnapshot{Status: r.statusEvent(s, ""), Logs: logs}
}

func (r *Runner) Start(ctx context.Context, appID int64) (types.StatusEvent, error) {
	if _, err := r.db.GetAppT(ctx, appID); err != nil {
		return types.StatusEvent{}, err
	}
	r.mu.Lock()
	existing := r.sessions[appID]
	if existing != nil && existing.running {
		r.mu.Unlock()
		return types.StatusEvent{}, fmt.Errorf("App is already running")
	}
	r.mu.Unlock()

	s, err := r.createSession(ctx, appID)
	if err != nil {
		return types.StatusEvent{}, err
	}
	r.mu.Lock()
	r.sessions[appID] = s
	r.mu.Unlock()
	r.emit(s, r.statusEvent(s, ""))
	go r.runSession(s)
	return r.statusEvent(s, ""), nil
}

func (r *Runner) Stop(ctx context.Context, appID int64) (types.StatusEvent, error) {
	r.mu.Lock()
	s := r.sessions[appID]
	if s == nil {
		r.mu.Unlock()
		return types.StatusEvent{}, fmt.Errorf("No active session")
	}
	s.abortSequential = true
	s.running = false
	r.clearReadyURLs(s)
	for i := range s.processes {
		if s.processes[i].Status == "running" {
			s.processes[i].Status = "killed"
		}
	}
	children := make([]*childProc, 0, len(s.children))
	for _, c := range s.children {
		children = append(children, c)
	}
	s.children = map[int64]*childProc{}
	r.mu.Unlock()

	r.emit(s, r.statusEvent(s, ""))
	for _, c := range children {
		if c.cmd != nil && c.cmd.Process != nil {
			_ = native.KillPid(c.cmd.Process.Pid)
		}
	}
	if !s.restored {
		_ = RestoreTemplates(ctx, r.db, appID, s.id)
		s.restored = true
		cmdID := int64(0)
		if len(s.processes) > 0 {
			cmdID = s.processes[0].CommandID
		}
		r.systemLog(s, cmdID, "Stopped — original files restored")
	}
	r.emit(s, r.statusEvent(s, ""))
	return r.statusEvent(s, ""), nil
}

func (r *Runner) Reload(ctx context.Context, appID int64) (types.StatusEvent, error) {
	r.mu.Lock()
	existing := r.sessions[appID]
	running := existing != nil && existing.running
	r.mu.Unlock()
	if running {
		if _, err := r.Stop(ctx, appID); err != nil {
			return types.StatusEvent{}, err
		}
	}
	return r.Start(ctx, appID)
}

func (r *Runner) StopAll(ctx context.Context) {
	r.mu.Lock()
	ids := make([]int64, 0, len(r.sessions))
	for id, s := range r.sessions {
		if s.running {
			ids = append(ids, id)
		}
	}
	r.mu.Unlock()
	for _, id := range ids {
		_, _ = r.Stop(ctx, id)
	}
}
