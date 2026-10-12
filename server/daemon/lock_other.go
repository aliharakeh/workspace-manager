//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// tryLock takes an exclusive lock on the file, failing when another process
// holds it.
func tryLock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func unlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
