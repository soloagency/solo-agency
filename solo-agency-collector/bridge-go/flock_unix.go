//go:build !windows

package main

// flock_unix.go — exclusive advisory file lock, the fcntl/flock replacement on
// POSIX. Blocking, matching Python's fcntl.flock(fh, LOCK_EX).

import (
	"os"
	"syscall"
)

func flockExclusive(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func flockUnlock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
