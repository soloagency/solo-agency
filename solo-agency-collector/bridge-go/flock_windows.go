//go:build windows

package main

// flock_windows.go — exclusive file lock via kernel32 LockFileEx (stdlib-only,
// no x/sys dependency). This is what unlocks Windows for the CRM: the Python
// storage backend refuses to run without fcntl.

import (
	"os"
	"syscall"
	"unsafe"
)

const lockfileExclusiveLock = 0x00000002

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

func flockExclusive(f *os.File) error {
	var ol syscall.Overlapped
	r, _, err := procLockFileEx.Call(f.Fd(), uintptr(lockfileExclusiveLock), 0, 1, 0,
		uintptr(unsafe.Pointer(&ol)))
	if r == 0 {
		return err
	}
	return nil
}

func flockUnlock(f *os.File) {
	var ol syscall.Overlapped
	_, _, _ = procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&ol)))
}
