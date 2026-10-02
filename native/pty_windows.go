//go:build windows

package native

import (
	"fmt"
	"io"
	"os"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// winPty is a ConPTY pseudo-console (Windows 10 1809+).
type winPty struct {
	hpc     windows.Handle
	process windows.Handle
	pid     int
	in      windows.Handle // we write keystrokes here
	out     windows.Handle // we read terminal output here
	once    sync.Once
}

func startPty(command, cwd string, env []string, cols, rows int) (ptyImpl, error) {
	var inRead, inWrite, outRead, outWrite windows.Handle
	if err := windows.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("CreatePipe: %w", err)
	}
	if err := windows.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		return nil, fmt.Errorf("CreatePipe: %w", err)
	}
	var hpc windows.Handle
	err := windows.CreatePseudoConsole(windows.Coord{X: int16(cols), Y: int16(rows)}, inRead, outWrite, 0, &hpc)
	// The console owns its ends now. Keeping ours open would stop the output
	// pipe from ever reaching EOF.
	windows.CloseHandle(inRead)
	windows.CloseHandle(outWrite)
	if err != nil {
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, fmt.Errorf("ConPTY is not available: %w", err)
	}
	fail := func(err error) (ptyImpl, error) {
		windows.ClosePseudoConsole(hpc)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, err
	}

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return fail(err)
	}
	defer attrs.Delete()
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&hpc)), unsafe.Sizeof(hpc)); err != nil {
		return fail(err)
	}
	si := windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(si))
	// Null standard handles: without this a child inherits our own redirected
	// stdio instead of the pseudo-console.
	si.Flags |= windows.STARTF_USESTDHANDLES

	shell := os.Getenv("ComSpec")
	if shell == "" {
		shell = `C:\Windows\System32\cmd.exe`
	}
	// /s makes cmd strip exactly the outer quotes, so the command keeps its own.
	cmdline, err := windows.UTF16PtrFromString(`"` + shell + `" /d /s /c "` + command + `"`)
	if err != nil {
		return fail(err)
	}
	var dir *uint16
	if cwd != "" {
		if dir, err = windows.UTF16PtrFromString(cwd); err != nil {
			return fail(err)
		}
	}
	var envBlock *uint16
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT)
	if env != nil {
		flags |= windows.CREATE_UNICODE_ENVIRONMENT
		envBlock = windowsEnvBlock(env)
	}
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(nil, cmdline, nil, nil, false, flags, envBlock, dir, &si.StartupInfo, &pi); err != nil {
		return fail(fmt.Errorf("Failed to start %q: %w", command, err))
	}
	windows.CloseHandle(pi.Thread)
	return &winPty{hpc: hpc, process: pi.Process, pid: int(pi.ProcessId), in: inWrite, out: outRead}, nil
}

// windowsEnvBlock builds a double-NUL terminated UTF-16 environment block.
func windowsEnvBlock(env []string) *uint16 {
	var block []uint16
	for _, kv := range env {
		block = append(block, utf16.Encode([]rune(kv))...)
		block = append(block, 0)
	}
	if len(block) == 0 {
		block = append(block, 0)
	}
	block = append(block, 0)
	return &block[0]
}

func (p *winPty) Pid() int { return p.pid }

func (p *winPty) Read(b []byte) (int, error) {
	var n uint32
	err := windows.ReadFile(p.out, b, &n, nil)
	if err != nil {
		if n > 0 {
			return int(n), nil
		}
		return 0, io.EOF
	}
	return int(n), nil
}

func (p *winPty) Write(b []byte) (int, error) {
	var n uint32
	err := windows.WriteFile(p.in, b, &n, nil)
	return int(n), err
}

func (p *winPty) Resize(cols, rows int) error {
	return windows.ResizePseudoConsole(p.hpc, windows.Coord{X: int16(cols), Y: int16(rows)})
}

func (p *winPty) wait() (int, error) {
	if _, err := windows.WaitForSingleObject(p.process, windows.INFINITE); err != nil {
		return 1, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(p.process, &code); err != nil {
		return 1, err
	}
	return int(code), nil
}

func (p *winPty) closeConsole() { windows.ClosePseudoConsole(p.hpc) }

func (p *winPty) close() {
	p.once.Do(func() {
		// Unblock a Read that is still waiting before its handle goes away.
		_ = windows.CancelIoEx(p.out, nil)
		windows.CloseHandle(p.out)
		windows.CloseHandle(p.in)
		windows.CloseHandle(p.process)
	})
}
