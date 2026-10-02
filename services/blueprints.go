package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
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

type blueprintRun struct {
	cancelled atomic.Bool
	pid       atomic.Int64
}

// BlueprintRunner creates apps from blueprints: it runs the blueprint's shell
// commands, runs git init, then registers the app.
type BlueprintRunner struct {
	d    *db.DB
	emit func(types.BlueprintLogEvent)
	mu   sync.Mutex
	runs map[string]*blueprintRun
}

func NewBlueprintRunner(d *db.DB, emit func(types.BlueprintLogEvent)) *BlueprintRunner {
	return &BlueprintRunner{d: d, emit: emit, runs: map[string]*blueprintRun{}}
}

func (b *BlueprintRunner) log(runID, stream, text string) {
	if b.emit != nil {
		b.emit(types.BlueprintLogEvent{RunID: runID, Stream: stream, Text: text, Ts: time.Now().UnixMilli()})
	}
}

func (b *BlueprintRunner) pipe(runID string, r io.Reader, stream string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		b.log(runID, stream, stripANSI(sc.Text()))
	}
}

// Cancel stops the running command of a run; the run then fails as cancelled.
func (b *BlueprintRunner) Cancel(runID string) {
	b.mu.Lock()
	run := b.runs[runID]
	b.mu.Unlock()
	if run == nil {
		return
	}
	run.cancelled.Store(true)
	if pid := run.pid.Load(); pid > 0 {
		_ = native.KillPid(int(pid))
	}
}

func (b *BlueprintRunner) shell(runID string, run *blueprintRun, command, cwd string) (int, error) {
	if run.cancelled.Load() {
		return 1, errBlueprintCancelled
	}
	child, err := native.SpawnShell(command, cwd, native.MergeSpawnEnv(nil))
	if err != nil {
		return 1, err
	}
	stdout, err := child.StdoutPipe()
	if err != nil {
		return 1, err
	}
	stderr, err := child.StderrPipe()
	if err != nil {
		return 1, err
	}
	if err := child.Start(); err != nil {
		return 1, err
	}
	run.pid.Store(int64(child.Process.Pid))
	if run.cancelled.Load() {
		_ = native.KillPid(child.Process.Pid)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); b.pipe(runID, stdout, "stdout") }()
	go func() { defer wg.Done(); b.pipe(runID, stderr, "stderr") }()
	waitErr := child.Wait()
	wg.Wait()
	run.pid.Store(0)
	if run.cancelled.Load() {
		return 1, errBlueprintCancelled
	}
	if waitErr != nil {
		if ee, ok := waitErr.(interface{ ExitCode() int }); ok {
			return ee.ExitCode(), nil
		}
		return 1, waitErr
	}
	return 0, nil
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

	run := &blueprintRun{}
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
		if err != nil {
			return types.BlueprintRunResult{}, err
		}
		if code != 0 {
			return types.BlueprintRunResult{}, fmt.Errorf("Command failed with exit code %d: %s", code, command)
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
