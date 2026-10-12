package native

import (
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	DefaultPtyCols = 120
	DefaultPtyRows = 30
	// ptyDrainTimeout bounds how long Stream waits for the last output after the
	// process has exited.
	ptyDrainTimeout = 3 * time.Second
)

// ptyImpl is the per-OS pseudo-terminal: ConPTY on Windows, a pty(7) elsewhere.
type ptyImpl interface {
	io.ReadWriter
	Pid() int
	Resize(cols, rows int) error
	// wait blocks until the process exits and returns its exit code.
	wait() (int, error)
	// closeConsole tells the terminal the process is gone so the output pipe
	// ends once everything has been read. It may block until that output is read.
	closeConsole()
	// close releases every handle and unblocks a pending Read.
	close()
}

// Pty is a shell command running attached to a pseudo-terminal, so it behaves
// as in a real terminal: colors, progress bars, prompts and TUIs all work.
type Pty struct {
	impl   ptyImpl
	mu     sync.Mutex
	closed bool
}

// StartPty runs command with the platform shell on a new pseudo-terminal.
func StartPty(command, cwd string, env []string, cols, rows int) (*Pty, error) {
	if cols <= 0 {
		cols = DefaultPtyCols
	}
	if rows <= 0 {
		rows = DefaultPtyRows
	}
	impl, err := startPty(command, cwd, env, cols, rows)
	if err != nil {
		return nil, err
	}
	return &Pty{impl: impl}, nil
}

func (p *Pty) Pid() int { return p.impl.Pid() }

// Write sends keystrokes to the process.
func (p *Pty) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, fmt.Errorf("Terminal is closed")
	}
	return p.impl.Write(data)
}

// Resize changes the terminal size; a closed terminal is ignored.
func (p *Pty) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return fmt.Errorf("Invalid terminal size %dx%d", cols, rows)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	return p.impl.Resize(cols, rows)
}

// Stream passes the process output to onData until the process has exited and
// its output is drained, then returns the exit code. onData runs on one
// goroutine, in order, and must not keep the slice.
func (p *Pty) Stream(onData func([]byte)) (int, error) {
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		buf := make([]byte, 32*1024)
		for {
			n, err := p.impl.Read(buf)
			if n > 0 {
				onData(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	code, err := p.impl.wait()
	// The process is gone: closing the console ends the output pipe once the
	// reader has consumed everything the process printed.
	go p.impl.closeConsole()
	select {
	case <-readerDone:
	case <-time.After(ptyDrainTimeout):
	}

	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.impl.close()
	<-readerDone
	return code, err
}
