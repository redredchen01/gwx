//go:build !windows

package auth

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

const lockTimeout = 5 * time.Second

// withLock acquires an exclusive flock on the lock file, runs fn, then releases.
// Returns ErrLockTimeout if the lock cannot be acquired within lockTimeout.
func (fs *FileStore) withLock(fn func() error) error {
	lf, err := os.OpenFile(fs.lockPath(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer lf.Close()

	deadline := time.Now().Add(lockTimeout)
	for {
		err = syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return fmt.Errorf("flock: %w", err)
		}
		if time.Now().After(deadline) {
			return ErrLockTimeout
		}
		time.Sleep(50 * time.Millisecond)
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) //nolint:errcheck

	return fn()
}
