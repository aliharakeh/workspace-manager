//go:build windows

package db

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFile waits for an exclusive lock, so a second process starting at the
// same moment waits for the first to finish migrating instead of failing.
func lockFile(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, new(windows.Overlapped))
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
