package native

import (
	"fmt"
	"os"
	"os/exec"
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

func MergeSpawnEnv(appEnv map[string]string) []string {
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
	if _, ok := appEnv["FORCE_COLOR"]; !ok {
		if os.Getenv("FORCE_COLOR") == "" {
			set("FORCE_COLOR", "0")
		}
	}
	if _, ok := appEnv["NO_COLOR"]; !ok {
		if os.Getenv("NO_COLOR") == "" {
			set("NO_COLOR", "1")
		}
	}
	return env
}

func SpawnShell(command, cwd string, env []string) (*exec.Cmd, error) {
	var c *exec.Cmd
	if isWindows() {
		c = exec.Command("cmd", "/c", command)
	} else {
		c = exec.Command("sh", "-c", command)
	}
	c.Dir = cwd
	c.Env = env
	hideWindow(c)
	return c, nil
}
