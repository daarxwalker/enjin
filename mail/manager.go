package mail

import (
	"context"
	"fmt"
	"time"
)

type Manager struct {
	composer composer
	driver   Driver
}

func NewManager(driver Driver) *Manager {
	return &Manager{
		driver: driver,
	}
}

func (m *Manager) Send(ctx context.Context, message Message) error {
	envelope, err := m.composer.createEnvelope(message)
	if err != nil {
		return fmt.Errorf("create e-mail envelope failed: %w", err)
	}
	raw, err := m.composer.compose(message, time.Now())
	if err != nil {
		return fmt.Errorf("compose e-mail failed: %w", err)
	}
	if sendErr := m.driver.Send(ctx, envelope, raw); sendErr != nil {
		return fmt.Errorf("send e-mail failed: %w", sendErr)
	}
	return nil
}
