package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/daarxwalker/enjin/token"
)

type Manager struct {
	driver Driver
	ttl    time.Duration
}

func NewManager(driver Driver, ttl time.Duration) *Manager {
	return &Manager{
		driver: driver,
		ttl:    ttl,
	}
}

func (m *Manager) Lock(ctx context.Context, name, id string) (Lease, error) {
	var zero Lease

	lease := Lease{key: m.createKey(name, id), token: token.New()}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		locked, setLockErr := m.driver.TryLock(ctx, lease.key, lease.token, m.ttl)
		if setLockErr != nil {
			return zero, fmt.Errorf("lock: lock %s failed: %w", lease.key, setLockErr)
		}
		if locked {
			return lease, nil
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *Manager) Unlock(ctx context.Context, lease Lease) error {
	released, err := m.driver.Unlock(ctx, lease.key, lease.token)
	if err != nil {
		return fmt.Errorf("lock: unlock %s failed: %w", lease.key, err)
	}
	if !released {
		return fmt.Errorf("lock: unlock %s: %w", lease.key, ErrNotHeld)
	}
	return nil
}

func (m *Manager) createKey(name, id string) string {
	return fmt.Sprintf("lock:%s:%s", name, id)
}
