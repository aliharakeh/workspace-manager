package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"workspace-manager/core/native"
	"workspace-manager/core/services"
)

func nowMillis() int64 { return time.Now().UnixMilli() }

// quoteArg quotes a path for display in a shell line only when it needs it.
func quoteArg(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}

// runExec is "wm-mcp exec": it runs one command of a session an agent
// prepared, with the config set's full environment, and returns its exit
// code. The environment includes values hidden from the agent; they reach the
// child process and never pass through the MCP conversation.
func runExec(argv []string) int {
	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	appID := fs.Int64("app", 0, "app id")
	session := fs.String("session", "", "session id from wm_run prepare")
	commandID := fs.Int64("command", 0, "command id from wm_run prepare")
	if err := fs.Parse(argv); err != nil {
		return 2
	}
	if *appID == 0 || *session == "" || *commandID == 0 {
		fmt.Fprintln(os.Stderr, "usage: wm-mcp exec --app <id> --session <id> --command <id>")
		return 2
	}
	if err := openStore(); err != nil {
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		return 1
	}
	defer database.Close()

	ctx := context.Background()
	held, found, err := database.GetAppSession(ctx, *appID)
	switch {
	case err != nil:
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		return 1
	case !found || held.SessionID != *session:
		fmt.Fprintln(os.Stderr, "wm-mcp: this session is not active; call wm_run prepare again")
		return 1
	case held.Owner != services.OwnerAgent:
		fmt.Fprintln(os.Stderr, "wm-mcp: this session is not an agent session")
		return 1
	}
	cfg, err := database.GetOrCreateRunConfig(ctx, held.ConfigSetID, held.Kind)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		return 1
	}
	command := ""
	for _, c := range cfg.Commands {
		if c.ID == *commandID {
			command = c.Command
		}
	}
	if command == "" {
		fmt.Fprintln(os.Stderr, "wm-mcp: that command is not part of the session")
		return 1
	}
	env, err := database.EnvToRecord(ctx, held.ConfigSetID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		return 1
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		shell := os.Getenv("ComSpec")
		if shell == "" {
			shell = `C:\Windows\System32\cmd.exe`
		}
		cmd = exec.Command(shell, "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = held.Dir
	cmd.Env = native.MergeTerminalEnv(env)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		return 1
	}
	return 0
}
