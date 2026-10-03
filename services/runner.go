package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"workspace-manager/db"
	"workspace-manager/native"
	"workspace-manager/types"
)

type childProc struct {
	pty *native.Pty
}

type session struct {
	id              string
	appID           int64
	appName         string
	kind            string
	mode            string
	processes       []types.ProcessState
	children        map[int64]*childProc
	running         bool
	abortSequential bool
	restored        bool
	// userStopped marks a manual stop so no "finished" notification is sent.
	userStopped bool
	// hadError is set on spawn/template failure or a non-zero exit.
	hadError      bool
	readyNotified bool
	idleNotified  bool
	// idleDone stops the idle watcher; lastLogAt is refreshed on every chunk of
	// output so the countdown only runs once the terminal goes quiet.
	idleDone  chan struct{}
	idleOnce  sync.Once
	lastLogAt atomic.Int64
	// outputs holds the recent terminal output of each command, for a UI that
	// opens after the command started.
	outMu   sync.Mutex
	outputs map[int64]*termBuffer
}

// subject names what the session executes, for notifications.
func (s *session) subject() string {
	if s.kind == types.KindBuild {
		return s.appName + " build"
	}
	return s.appName
}

type Runner struct {
	db        *db.DB
	broadcast func(appID int64, event any)
	notify    func(title, body string)

	mu       sync.Mutex
	sessions map[int64]*session
	// cols and rows are the size the UI last reported; new commands start there.
	cols, rows int
}

func NewRunner(d *db.DB, broadcast func(appID int64, event any), notify func(title, body string)) *Runner {
	return &Runner{
		db: d, broadcast: broadcast, notify: notify, sessions: map[int64]*session{},
		cols: native.DefaultPtyCols, rows: native.DefaultPtyRows,
	}
}

func (r *Runner) emit(s *session, event any) {
	if r.broadcast != nil {
		r.broadcast(s.appID, event)
	}
}

// emitOutput records a chunk of a command's terminal output and sends it to the UI.
func (r *Runner) emitOutput(s *session, commandID int64, p []byte) {
	if len(p) == 0 {
		return
	}
	s.outMu.Lock()
	buf := s.outputs[commandID]
	if buf == nil {
		buf = &termBuffer{}
		s.outputs[commandID] = buf
	}
	offset := buf.add(p)
	s.outMu.Unlock()
	r.emit(s, types.LogEvent{
		Type: "log", AppID: s.appID, CommandID: commandID, Offset: offset,
		Data: base64.StdEncoding.EncodeToString(p), Ts: time.Now().UnixMilli(),
	})
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
		Type: "status", SessionID: s.id, AppID: s.appID, Kind: s.kind,
		Running: s.running, Processes: procs, Ts: time.Now().UnixMilli(),
	}
	if errMsg != "" {
		ev.Error = errMsg
	}
	return ev
}

// systemLog writes one of the app's own messages (dimmed) into a command's
// terminal, on a fresh line.
func (r *Runner) systemLog(s *session, commandID int64, text string) {
	line := dimLine(text)
	s.outMu.Lock()
	if buf := s.outputs[commandID]; buf != nil && buf.total > 0 && buf.lastByte != '\n' {
		line = append([]byte("\r\n"), line...)
	}
	s.outMu.Unlock()
	r.emitOutput(s, commandID, line)
}

func (r *Runner) noteReadyURL(s *session, commandID int64, line string) {
	if !s.running || s.kind == types.KindBuild {
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
		if !s.readyNotified {
			s.readyNotified = true
			r.stopIdleWatcher(s)
			r.maybeNotify(s, NotifyReady, s.appName+" is running", match.URL)
		}
		r.emit(s, r.statusEvent(s, ""))
		return
	}
}

func (r *Runner) clearReadyURLs(s *session) {
	for i := range s.processes {
		s.processes[i].URLs = []string{}
	}
}

func (r *Runner) stopIdleWatcher(s *session) {
	if s.idleDone != nil {
		s.idleOnce.Do(func() { close(s.idleDone) })
	}
}

func (r *Runner) maybeNotify(s *session, kind, title, body string) {
	if r.notify == nil {
		return
	}
	settings, err := r.db.SettingsMap(context.Background())
	if err != nil {
		return
	}
	if !notifyEnabled(settings, kind) {
		return
	}
	r.notify(title, body)
}

func firstCommandID(s *session) int64 {
	if len(s.processes) > 0 {
		return s.processes[0].CommandID
	}
	return 0
}

// startIdleWatcher is the fallback for apps whose logs never match a ready-URL
// pattern. It assumes the app is running once no new output has arrived for the
// idle delay: every log line refreshes lastLogAt, so the clock only advances
// after the output goes quiet. The watcher exits on ready, idle, or stop.
func (r *Runner) startIdleWatcher(s *session) {
	secs := DefaultIdleSeconds
	if settings, err := r.db.SettingsMap(context.Background()); err == nil {
		secs = idleSeconds(settings)
	}
	window := time.Duration(secs) * time.Second
	s.lastLogAt.Store(time.Now().UnixNano())

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.idleDone:
				return
			case <-ticker.C:
				if s.readyNotified || s.idleNotified {
					return
				}
				if time.Since(time.Unix(0, s.lastLogAt.Load())) < window {
					continue
				}
				s.idleNotified = true
				r.systemLog(s, firstCommandID(s), "No output for a while — assuming the app is running")
				r.maybeNotify(s, NotifyIdle, s.appName+" is running", "No output for a while — assuming it is running")
				return
			}
		}
	}()
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
	fail := func(err error) int {
		processState.Status = "error"
		code := int64(1)
		processState.ExitCode = &code
		s.hadError = true
		r.systemLog(s, cmd.ID, "Failed to start: "+err.Error())
		r.maybeNotify(s, NotifyError, s.subject()+" failed to start", err.Error())
		r.emit(s, r.statusEvent(s, ""))
		return 1
	}
	processState.Status = "running"
	r.emit(s, r.statusEvent(s, ""))
	r.systemLog(s, cmd.ID, "$ "+cmd.Command)

	r.mu.Lock()
	cols, rows := r.cols, r.rows
	r.mu.Unlock()
	proc, err := native.StartPty(cmd.Command, cwd, env, cols, rows)
	if err != nil {
		return fail(err)
	}
	pid := int64(proc.Pid())
	processState.PID = &pid
	r.mu.Lock()
	s.children[cmd.ID] = &childProc{pty: proc}
	r.mu.Unlock()
	r.emit(s, r.statusEvent(s, ""))

	// The commands' output is only watched, never answered: the UI shows it
	// read-only, so nothing is ever written to the terminal's input.
	tap := &lineTap{}
	exitCode, waitErr := proc.Stream(func(p []byte) {
		s.lastLogAt.Store(time.Now().UnixNano())
		r.emitOutput(s, cmd.ID, p)
		for _, line := range tap.feed(p) {
			r.noteReadyURL(s, cmd.ID, line)
		}
	})
	for _, line := range tap.flush() {
		r.noteReadyURL(s, cmd.ID, line)
	}

	r.mu.Lock()
	delete(s.children, cmd.ID)
	r.mu.Unlock()

	if processState.Status == "killed" {
		r.emit(s, r.statusEvent(s, ""))
		return 1
	}
	if waitErr != nil && exitCode == 0 {
		exitCode = 1
	}
	code64 := int64(exitCode)
	processState.ExitCode = &code64
	if exitCode == 0 {
		processState.Status = "exited"
		r.systemLog(s, cmd.ID, fmt.Sprintf("Process exited with code %d", exitCode))
	} else {
		processState.Status = "error"
		s.hadError = true
		r.systemLog(s, cmd.ID, fmt.Sprintf("Process failed with code %d", exitCode))
		r.maybeNotify(s, NotifyError, s.subject()+" failed",
			fmt.Sprintf("%s exited with code %d", processState.Label, exitCode))
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
	s.appName = app.Name
	if err := ApplyTemplates(ctx, r.db, s.appID, s.id); err != nil {
		s.running = false
		s.hadError = true
		msg := err.Error()
		cmdID := int64(0)
		if len(s.processes) > 0 {
			cmdID = s.processes[0].CommandID
		}
		r.systemLog(s, cmdID, "Template apply failed: "+msg)
		r.maybeNotify(s, NotifyError, s.subject()+" failed to start", msg)
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
	env := native.MergeTerminalEnv(envMap)
	config, err := r.db.GetRunConfigByConfigSetT(ctx, set.ID, s.kind)
	if err != nil {
		s.running = false
		r.emit(s, r.statusEvent(s, err.Error()))
		return
	}
	var commands []types.RunCommand
	if config != nil {
		commands = config.Commands
	}

	// Only a run is waited on to come up; a build just ends.
	if s.kind == types.KindRun {
		r.startIdleWatcher(s)
	}

	func() {
		defer func() {
			s.running = false
			r.stopIdleWatcher(s)
			r.clearReadyURLs(s)
			if !s.restored {
				_ = RestoreTemplates(ctx, r.db, s.appID, s.id)
				s.restored = true
				r.systemLog(s, cmdID, "Original files restored")
			}
			if !s.userStopped && !s.hadError {
				r.maybeNotify(s, NotifyFinished, s.subject()+" finished", "")
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
				r.systemLog(s, c.ID, "Sequential "+s.kind+" stopped due to non-zero exit")
				break
			}
		}
	}()
}

func (r *Runner) createSession(ctx context.Context, appID int64, kind string) (*session, error) {
	set, err := r.db.ResolveActive(ctx, appID)
	if err != nil {
		return nil, err
	}
	config, err := r.db.GetOrCreateRunConfig(ctx, set.ID, kind)
	if err != nil {
		return nil, err
	}
	if len(config.Commands) == 0 {
		return nil, fmt.Errorf("No %s commands configured", kind)
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
		kind:      kind,
		mode:      config.Mode,
		processes: procs,
		children:  map[int64]*childProc{},
		running:   true,
		idleDone:  make(chan struct{}),
		outputs:   map[int64]*termBuffer{},
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

// GetOutput returns the recent terminal output of one command of the app's
// current session.
func (r *Runner) GetOutput(appID, commandID int64) types.RunnerOutput {
	r.mu.Lock()
	s := r.sessions[appID]
	r.mu.Unlock()
	if s == nil {
		return types.RunnerOutput{}
	}
	s.outMu.Lock()
	defer s.outMu.Unlock()
	buf := s.outputs[commandID]
	if buf == nil {
		return types.RunnerOutput{SessionID: s.id}
	}
	return types.RunnerOutput{
		SessionID: s.id,
		Data:      base64.StdEncoding.EncodeToString(buf.snapshot()),
		End:       buf.total,
	}
}

// Resize sets the terminal size of a command; later commands start at that size too.
func (r *Runner) Resize(appID, commandID int64, cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	r.mu.Lock()
	r.cols, r.rows = cols, rows
	var child *childProc
	if s := r.sessions[appID]; s != nil {
		child = s.children[commandID]
	}
	r.mu.Unlock()
	if child != nil {
		_ = child.pty.Resize(cols, rows)
	}
}

// Start runs the app's run commands or build commands, by kind. An app has one
// session at a time because both apply and restore the templates, so Start
// fails while either kind is still going.
func (r *Runner) Start(ctx context.Context, appID int64, kind string) (types.StatusEvent, error) {
	if _, err := r.db.GetAppT(ctx, appID); err != nil {
		return types.StatusEvent{}, err
	}
	r.mu.Lock()
	existing := r.sessions[appID]
	if existing != nil && existing.running {
		building := existing.kind == types.KindBuild
		r.mu.Unlock()
		if building {
			return types.StatusEvent{}, fmt.Errorf("App is already building")
		}
		return types.StatusEvent{}, fmt.Errorf("App is already running")
	}
	r.mu.Unlock()

	s, err := r.createSession(ctx, appID, kind)
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
	s.userStopped = true
	r.stopIdleWatcher(s)
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
		if c.pty != nil {
			_ = native.KillPid(c.pty.Pid())
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
	return r.Start(ctx, appID, types.KindRun)
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
