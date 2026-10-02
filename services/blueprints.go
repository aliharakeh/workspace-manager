package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"workspace-manager/db"
	"workspace-manager/lib"
	"workspace-manager/native"
	"workspace-manager/types"
)

var errBlueprintCancelled = errors.New("Cancelled")

var blueprintVarRe = regexp.MustCompile(`\{\{\s*(app_name|folder_name|app_dir)\s*\}\}`)

// renderBlueprintCommand replaces {{app_name}}, {{folder_name}} and {{app_dir}}.
// Values are inserted verbatim (no shell quoting); other {{tokens}} are kept.
func renderBlueprintCommand(command string, vars map[string]string) string {
	return blueprintVarRe.ReplaceAllStringFunc(command, func(m string) string {
		return vars[blueprintVarRe.FindStringSubmatch(m)[1]]
	})
}

func validateFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("Folder name is required")
	}
	if name == "." || name == ".." || strings.HasSuffix(name, ".") || strings.ContainsAny(name, "<>:\"/\\|?*") {
		return "", fmt.Errorf("Folder name is not valid: %s", name)
	}
	for _, r := range name {
		if r < 32 {
			return "", fmt.Errorf("Folder name is not valid: %s", name)
		}
	}
	return name, nil
}

// Terminal size a blueprint command starts with, until the UI reports its own.
const (
	blueprintCols = 100
	blueprintRows = 24
)

type blueprintRun struct {
	cancelled atomic.Bool
	pid       atomic.Int64
	// decision carries the UI's answer ("skip" or "abort") after a command fails.
	decision chan string
	// lastByte is the last byte shown in the terminal, to start app messages
	// on a fresh line.
	lastByte atomic.Uint32

	// pty is the running command's terminal; nil between commands. cols and rows
	// are the size the UI last reported.
	ptyMu      sync.Mutex
	pty        *native.Pty
	cols, rows int
}

func (r *blueprintRun) setPty(p *native.Pty) {
	r.ptyMu.Lock()
	r.pty = p
	r.ptyMu.Unlock()
}

func (r *blueprintRun) size() (int, int) {
	r.ptyMu.Lock()
	defer r.ptyMu.Unlock()
	return r.cols, r.rows
}

func (r *blueprintRun) write(data []byte) error {
	r.ptyMu.Lock()
	defer r.ptyMu.Unlock()
	if r.pty == nil {
		return errors.New("No command is running")
	}
	_, err := r.pty.Write(data)
	return err
}

func (r *blueprintRun) resize(cols, rows int) {
	r.ptyMu.Lock()
	defer r.ptyMu.Unlock()
	r.cols, r.rows = cols, rows
	if r.pty != nil {
		_ = r.pty.Resize(cols, rows)
	}
}

// BlueprintRunner creates apps from blueprints: it runs the blueprint's shell
// commands in a terminal, runs git init, then registers the app.
type BlueprintRunner struct {
	d    *db.DB
	emit func(types.BlueprintLogEvent)
	mu   sync.Mutex
	runs map[string]*blueprintRun
}

func NewBlueprintRunner(d *db.DB, emit func(types.BlueprintLogEvent)) *BlueprintRunner {
	return &BlueprintRunner{d: d, emit: emit, runs: map[string]*blueprintRun{}}
}

func (b *BlueprintRunner) get(runID string) *blueprintRun {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.runs[runID]
}

// output sends terminal bytes to the UI.
func (b *BlueprintRunner) output(runID string, p []byte) {
	if len(p) == 0 {
		return
	}
	if run := b.get(runID); run != nil {
		run.lastByte.Store(uint32(p[len(p)-1]))
	}
	if b.emit != nil {
		b.emit(types.BlueprintLogEvent{
			RunID: runID, Stream: "data", Data: base64.StdEncoding.EncodeToString(p), Ts: time.Now().UnixMilli(),
		})
	}
}

// log shows one of the app's own messages in the run's terminal. Stream
// "failed" also tells the UI that a command failed and waits for an answer.
func (b *BlueprintRunner) log(runID, stream, text string) {
	line := dimLine(text)
	if stream == "failed" {
		line = redLine(text)
	}
	if run := b.get(runID); run != nil && run.lastByte.Load() != 0 && run.lastByte.Load() != '\n' {
		line = append([]byte("\r\n"), line...)
	}
	b.output(runID, line)
	if stream == "failed" && b.emit != nil {
		b.emit(types.BlueprintLogEvent{RunID: runID, Stream: "failed", Text: text, Ts: time.Now().UnixMilli()})
	}
}

// Write sends keystrokes to the command a run is currently executing.
func (b *BlueprintRunner) Write(runID, data string) error {
	run := b.get(runID)
	if run == nil {
		return errors.New("Run is not active")
	}
	return run.write([]byte(data))
}

// Resize sets the size of the run's terminal.
func (b *BlueprintRunner) Resize(runID string, cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	if run := b.get(runID); run != nil {
		run.resize(cols, rows)
	}
}

// Cancel stops the running command of a run; the run then fails as cancelled.
func (b *BlueprintRunner) Cancel(runID string) {
	run := b.get(runID)
	if run == nil {
		return
	}
	run.cancelled.Store(true)
	run.answer("abort")
	if pid := run.pid.Load(); pid > 0 {
		_ = native.KillPid(int(pid))
	}
}

// Resolve answers a failed command of a run: "skip" continues with the next
// command, anything else stops the run.
func (b *BlueprintRunner) Resolve(runID, action string) {
	if run := b.get(runID); run != nil {
		run.answer(action)
	}
}

func (r *blueprintRun) answer(action string) {
	select {
	case r.decision <- action:
	default:
	}
}

// askSkip reports a failed command to the UI and blocks until it answers.
// It returns nil when the command should be skipped.
func (b *BlueprintRunner) askSkip(runID string, run *blueprintRun, failure string) error {
	select {
	case <-run.decision: // drop an answer left over from an earlier prompt
	default:
	}
	b.log(runID, "failed", failure)
	choice := <-run.decision
	if run.cancelled.Load() {
		return errBlueprintCancelled
	}
	if choice != "skip" {
		return errors.New(failure)
	}
	b.log(runID, "system", "Skipped failed command, continuing")
	return nil
}

// shell runs a command on a terminal and returns its exit code.
func (b *BlueprintRunner) shell(runID string, run *blueprintRun, command, cwd string) (int, error) {
	if run.cancelled.Load() {
		return 1, errBlueprintCancelled
	}
	cols, rows := run.size()
	proc, err := native.StartPty(command, cwd, native.MergeTerminalEnv(nil), cols, rows)
	if err != nil {
		return 1, err
	}
	run.setPty(proc)
	run.pid.Store(int64(proc.Pid()))
	if run.cancelled.Load() {
		_ = native.KillPid(proc.Pid())
	}
	code, err := proc.Stream(func(p []byte) { b.output(runID, p) })
	run.setPty(nil)
	run.pid.Store(0)
	if run.cancelled.Load() {
		return 1, errBlueprintCancelled
	}
	if err != nil && code == 0 {
		return 1, err
	}
	return code, nil
}

func (b *BlueprintRunner) Run(ctx context.Context, in types.BlueprintRunInput) (types.BlueprintRunResult, error) {
	if strings.TrimSpace(in.RunID) == "" {
		return types.BlueprintRunResult{}, fmt.Errorf("run_id is required")
	}
	bp, err := b.d.GetBlueprintT(ctx, in.BlueprintID)
	if err != nil {
		return types.BlueprintRunResult{}, err
	}
	if _, err := b.d.GetWorkspaceT(ctx, in.WorkspaceID); err != nil {
		return types.BlueprintRunResult{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return types.BlueprintRunResult{}, fmt.Errorf("name is required")
	}
	folder, err := validateFolderName(in.FolderName)
	if err != nil {
		return types.BlueprintRunResult{}, err
	}
	ok, parent, errMsg := lib.ValidateProjectPath(in.ParentPath)
	if !ok {
		return types.BlueprintRunResult{}, fmt.Errorf("%s", errMsg)
	}
	appDir := filepath.Join(parent, folder)
	if entries, err := os.ReadDir(appDir); err == nil {
		if len(entries) > 0 {
			return types.BlueprintRunResult{}, fmt.Errorf("Folder already exists and is not empty: %s", appDir)
		}
	} else if !os.IsNotExist(err) {
		return types.BlueprintRunResult{}, fmt.Errorf("Cannot use folder %s: %v", appDir, err)
	}

	vars := map[string]string{"app_name": name, "folder_name": folder, "app_dir": appDir}
	commands := make([]string, len(bp.Commands))
	for i, c := range bp.Commands {
		commands[i] = renderBlueprintCommand(c.Command, vars)
	}

	run := &blueprintRun{decision: make(chan string, 1), cols: blueprintCols, rows: blueprintRows}
	b.mu.Lock()
	if _, busy := b.runs[in.RunID]; busy {
		b.mu.Unlock()
		return types.BlueprintRunResult{}, fmt.Errorf("Run already in progress")
	}
	b.runs[in.RunID] = run
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.runs, in.RunID)
		b.mu.Unlock()
	}()

	cwd := parent
	if in.CreateFolder {
		cwd = appDir
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			return types.BlueprintRunResult{}, err
		}
		b.log(in.RunID, "system", "Created folder "+appDir)
	}
	for _, command := range commands {
		b.log(in.RunID, "system", "$ "+command)
		code, err := b.shell(in.RunID, run, command, cwd)
		if err == errBlueprintCancelled {
			return types.BlueprintRunResult{}, err
		}
		if err == nil && code != 0 {
			err = fmt.Errorf("Command failed with exit code %d: %s", code, command)
		}
		if err == nil {
			continue
		}
		if !in.AskOnError {
			return types.BlueprintRunResult{}, err
		}
		if err := b.askSkip(in.RunID, run, err.Error()); err != nil {
			return types.BlueprintRunResult{}, err
		}
	}

	if info, err := os.Stat(appDir); err != nil || !info.IsDir() {
		return types.BlueprintRunResult{}, fmt.Errorf("Commands finished but the app folder was not created: %s", appDir)
	}

	warning := ""
	if _, err := os.Stat(filepath.Join(appDir, ".git")); os.IsNotExist(err) {
		b.log(in.RunID, "system", "$ git init")
		code, err := b.shell(in.RunID, run, "git init", appDir)
		if err == errBlueprintCancelled {
			return types.BlueprintRunResult{}, err
		}
		if err != nil || code != 0 {
			warning = "git init failed; the app was created without a git repository"
			b.log(in.RunID, "system", warning)
		}
	} else {
		b.log(in.RunID, "system", "Git repository already present, skipping git init")
	}

	app, err := b.d.CreateAppT(ctx, in.WorkspaceID, name, appDir)
	if err != nil {
		return types.BlueprintRunResult{}, err
	}
	b.log(in.RunID, "system", "Created app "+name)
	return types.BlueprintRunResult{App: app, Warning: warning}, nil
}
