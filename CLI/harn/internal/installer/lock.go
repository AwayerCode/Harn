package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrOperationInProgress = errors.New("another harn operation is in progress")
	errLockBusy            = errors.New("lock busy")
)

type projectLock struct {
	file *os.File
}

func acquireProjectLock(projectRoot string) (*projectLock, error) {
	harnessRoot := filepath.Join(projectRoot, ".harness")
	if err := os.MkdirAll(harnessRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create .harness directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(harnessRoot, ".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open project lock: %w", err)
	}
	if err := lockFile(file); err != nil {
		file.Close()
		if errors.Is(err, errLockBusy) {
			return nil, ErrOperationInProgress
		}
		return nil, fmt.Errorf("lock .harness: %w", err)
	}
	return &projectLock{file: file}, nil
}

func (lock *projectLock) Close() error {
	unlockErr := unlockFile(lock.file)
	closeErr := lock.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
