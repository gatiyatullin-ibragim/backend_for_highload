package store

import (
	"context"
	"fmt"
	"sync"
)

// Memory — реализация из SIS 2: две мапы под одним RWMutex.
type Memory struct {
	mu            sync.RWMutex
	notifications map[string]string // id -> статус
	processedKeys map[string]string // ключ идемпотентности -> id
}

func NewMemory() *Memory {
	return &Memory{
		notifications: make(map[string]string),
		processedKeys: make(map[string]string),
	}
}

func (m *Memory) Create(_ context.Context, n NewNotification) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if n.IdempotencyKey != "" {
		if id, ok := m.processedKeys[n.IdempotencyKey]; ok {
			return id, false, nil
		}
	}
	id := fmt.Sprintf("notif-%d", len(m.notifications)+1)
	m.notifications[id] = "accepted"
	if n.IdempotencyKey != "" {
		m.processedKeys[n.IdempotencyKey] = id
	}
	return id, true, nil
}

func (m *Memory) Status(_ context.Context, id string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	st, ok := m.notifications[id]
	if !ok {
		return "", ErrNotFound
	}
	return st, nil
}

func (m *Memory) Close() {}
