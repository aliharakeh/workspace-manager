package native

import (
	"bytes"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func ptyCommand(windows, unix string) string {
	if runtime.GOOS == "windows" {
		return windows
	}
	return unix
}

func streamAll(t *testing.T, p *Pty) (string, int) {
	t.Helper()
	var mu sync.Mutex
	var out bytes.Buffer
	done := make(chan int, 1)
	go func() {
		code, err := p.Stream(func(b []byte) {
			mu.Lock()
			out.Write(b)
			mu.Unlock()
		})
		if err != nil {
			t.Error(err)
		}
		done <- code
	}()
	select {
	case code := <-done:
		mu.Lock()
		defer mu.Unlock()
		return out.String(), code
	case <-time.After(20 * time.Second):
		t.Fatal("pty did not finish")
		return "", 0
	}
}

func TestPtyOutputAndExitCode(t *testing.T) {
	p, err := StartPty(ptyCommand(`echo hello-pty & exit 7`, `echo hello-pty; exit 7`), t.TempDir(), MergeTerminalEnv(nil), 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	out, code := streamAll(t, p)
	if !strings.Contains(out, "hello-pty") {
		t.Fatalf("output missing: %q", out)
	}
	if code != 7 {
		t.Fatalf("exit code = %d, want 7", code)
	}
}

func TestPtyInteractive(t *testing.T) {
	// Prompts without a newline, reads a line, then echoes it back.
	cmd := ptyCommand(`set /p ans=Name: & call echo got-%^ans%`, `printf 'Name: '; read ans; echo got-$ans`)
	p, err := StartPty(cmd, t.TempDir(), MergeTerminalEnv(nil), 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var out bytes.Buffer
	prompted := make(chan struct{})
	var once sync.Once
	done := make(chan int, 1)
	go func() {
		code, _ := p.Stream(func(b []byte) {
			mu.Lock()
			out.Write(b)
			seen := strings.Contains(out.String(), "Name:")
			mu.Unlock()
			if seen {
				once.Do(func() { close(prompted) })
			}
		})
		done <- code
	}()
	select {
	case <-prompted:
	case <-time.After(10 * time.Second):
		t.Fatal("prompt never appeared")
	}
	if _, err := p.Write([]byte("bob\r")); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		mu.Lock()
		defer mu.Unlock()
		if code != 0 || !strings.Contains(out.String(), "got-bob") {
			t.Fatalf("code=%d output=%q", code, out.String())
		}
	case <-time.After(20 * time.Second):
		t.Fatal("command did not finish after input")
	}
}

func TestPtyKill(t *testing.T) {
	p, err := StartPty(ptyCommand(`ping -n 30 127.0.0.1`, `sleep 30`), t.TempDir(), MergeTerminalEnv(nil), 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(500*time.Millisecond, func() { _ = KillPid(p.Pid()) })
	start := time.Now()
	_, _ = streamAll(t, p)
	if time.Since(start) > 15*time.Second {
		t.Fatal("kill did not end the stream")
	}
}

func TestPtyResize(t *testing.T) {
	p, err := StartPty(ptyCommand(`ping -n 3 127.0.0.1`, `sleep 2`), t.TempDir(), MergeTerminalEnv(nil), 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Resize(100, 30); err != nil {
		t.Fatal(err)
	}
	if err := p.Resize(0, 30); err == nil {
		t.Fatal("invalid size must be rejected")
	}
	_, _ = streamAll(t, p)
	if err := p.Resize(90, 20); err != nil {
		t.Fatalf("resize after exit must be ignored: %v", err)
	}
}
