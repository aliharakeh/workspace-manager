package native

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func isProcessGone(detail string) bool {
	lower := strings.ToLower(detail)
	return strings.Contains(lower, "not found") ||
		strings.Contains(lower, "not running") ||
		strings.Contains(lower, "no such process") ||
		strings.Contains(lower, "no matching")
}

func killUnixTree(pid int) error {
	children, _ := Run([]string{"pgrep", "-P", strconv.Itoa(pid)})
	if children.Code == 0 {
		for _, line := range strings.Split(children.Stdout, "\n") {
			childPid, err := strconv.Atoi(strings.TrimSpace(line))
			if err != nil || childPid <= 0 {
				continue
			}
			_ = killUnixTree(childPid)
		}
	}
	result, err := Run([]string{"kill", "-9", strconv.Itoa(pid)})
	if err != nil {
		return err
	}
	if result.Code != 0 {
		detail := strings.TrimSpace(result.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(result.Stdout)
		}
		if !isProcessGone(detail) {
			if detail == "" {
				detail = fmt.Sprintf("kill failed for pid %d", pid)
			}
			return fmt.Errorf("%s", detail)
		}
	}
	return nil
}

func KillPid(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("Invalid pid")
	}
	if pid == os.Getpid() {
		return fmt.Errorf("Refusing to kill the Workspace Manager process")
	}
	if isWindows() {
		result, err := Run([]string{"taskkill", "/PID", strconv.Itoa(pid), "/T", "/F"})
		if err != nil {
			return err
		}
		if result.Code != 0 {
			detail := strings.TrimSpace(result.Stderr)
			if detail == "" {
				detail = strings.TrimSpace(result.Stdout)
			}
			if isProcessGone(detail) {
				return nil
			}
			if detail == "" {
				detail = fmt.Sprintf("taskkill failed for pid %d", pid)
			}
			return fmt.Errorf("%s", detail)
		}
		return nil
	}
	return killUnixTree(pid)
}

// MergeTerminalEnv returns the process environment plus appEnv, set up for a
// terminal: colors stay on and TERM tells programs what they can render.
func MergeTerminalEnv(appEnv map[string]string) []string {
	env := os.Environ()
	seen := map[string]int{}
	for i, kv := range env {
		if eq := strings.IndexByte(kv, '='); eq > 0 {
			seen[strings.ToUpper(kv[:eq])] = i
		}
	}
	set := func(key, value string) {
		entry := key + "=" + value
		k := strings.ToUpper(key)
		if i, ok := seen[k]; ok {
			env[i] = entry
			return
		}
		seen[k] = len(env)
		env = append(env, entry)
	}
	for k, v := range appEnv {
		set(k, v)
	}
	set("PYTHONUNBUFFERED", "1")
	for key, value := range map[string]string{"TERM": "xterm-256color", "COLORTERM": "truecolor"} {
		if _, ok := appEnv[key]; !ok {
			set(key, value)
		}
	}
	return env
}
