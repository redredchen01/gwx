//go:build windows

package auth

// withLock on Windows is a no-op: flock is unavailable.
// Concurrent writes are unlikely in the CLI use case; callers serialize via context.
func (fs *FileStore) withLock(fn func() error) error {
	return fn()
}
