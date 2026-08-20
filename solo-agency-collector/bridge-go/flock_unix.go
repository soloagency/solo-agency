//go:build !windows

package main

// flock_unix.go — exclusive advisory file lock, the fcntl/flock replacement on
// POSIX. flockExclusive blocks, matching Python's fcntl.flock(fh, LOCK_EX);
// flockTry does not, because an HTTP handler that blocks on a lock holds the
// request open with no way to give up — the browser waits, and Go will not kill
// the goroutine. A busy lock must become an answer, not a hang.

import (
	"os"
	"syscall"
)

func flockExclusive(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

// flockTry takes the lock or reports ErrLocked immediately. Never blocks.
func flockTry(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK {
		return errLocked
	}
	return err
}

func flockUnlock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
