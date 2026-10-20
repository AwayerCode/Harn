//go:build windows

package installer

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

const (
	lockfileFailImmediately = 0x00000001
	lockfileExclusiveLock   = 0x00000002
	errorLockViolation      = syscall.Errno(33)
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

func lockFile(file *os.File) error {
	var overlapped syscall.Overlapped
	ok, _, err := procLockFileEx.Call(
		file.Fd(),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ok != 0 {
		return nil
	}
	if errors.Is(err, errorLockViolation) {
		return errLockBusy
	}
	if err == syscall.Errno(0) {
		return errors.New("LockFileEx failed")
	}
	return err
}

func unlockFile(file *os.File) error {
	var overlapped syscall.Overlapped
	ok, _, err := procUnlockFileEx.Call(
		file.Fd(),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ok != 0 {
		return nil
	}
	if err == syscall.Errno(0) {
		return errors.New("UnlockFileEx failed")
	}
	return err
}
