package database

// NOTE: This helper is intended ONLY for test code to allow resetting
// the singleton state between tests. It should not be used in production code.
import "sync"

// ResetForTest resets the internal singleton so tests can start with a clean state.
func ResetForTest() {
	instance = nil
	once = sync.Once{}
}
