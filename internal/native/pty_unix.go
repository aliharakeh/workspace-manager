//go:build !windows

package native

import (
	"errors"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type unixPty struct {
	cmd  *exec.Cmd
	ptmx *os.File
}

func startPty(command, cwd string, env []string, cols, rows int) (ptyImpl, error) {
	c := exec.Command("sh", "-c", command)
	c.Dir = cwd
	c.Env = env
	ptmx, err := pty.StartWithSize(c, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	return &unixPty{cmd: c, ptmx: ptmx}, nil
}

func (p *unixPty) Pid() int { return p.cmd.Process.Pid }

func (p *unixPty) Read(b []byte) (int, error) { return p.ptmx.Read(b) }

func (p *unixPty) Write(b []byte) (int, error) { return p.ptmx.Write(b) }

func (p *unixPty) Resize(cols, rows int) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (p *unixPty) wait() (int, error) {
	err := p.cmd.Wait()
	if err == nil {
		return 0, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return 1, err
}

// The master side reports EIO once the process and its children are gone.
func (p *unixPty) closeConsole() {}

func (p *unixPty) close() { _ = p.ptmx.Close() }
