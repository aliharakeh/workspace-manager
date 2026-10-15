//go:build !windows

package db

import (
	"os"
	"syscall"
)

// lockFile waits for an exclusive lock, so a second process starting at the
// same moment waits for the first to finish migrating instead of failing.
func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
