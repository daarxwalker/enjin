package bus

import (
	"fmt"
	"sync"
)

var binding struct {
	driver   Driver
	mu       sync.Mutex
	resolved bool
}

func Use(driver Driver) error {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	if binding.resolved {
		return fmt.Errorf("use bus driver failed: %w", ErrInUse)
	}
	binding.driver = driver
	return nil
}

func claimDriver() Driver {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.resolved = true
	return binding.driver
}
