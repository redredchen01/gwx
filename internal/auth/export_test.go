package auth

import "sync"

// resetDefaultStoreForTest resets the defaultStore singleton.
// Only for use in tests; never call in production code.
func resetDefaultStoreForTest() {
	defaultStore = nil
	defaultStoreOnce = sync.Once{}
}
