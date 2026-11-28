// Package store — хранилище уведомлений за gRPC-сервером.
// Две реализации: Memory (как в SIS 2, для тестов) и Postgres (БД из SIS 3).
package store

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("notification not found")
	ErrInvalid  = errors.New("invalid argument")
)

// NewNotification — то, что приходит в SendNotification.
type NewNotification struct {
	IdempotencyKey string
	UserID         string
	TemplateID     string
	Channel        string // push, email, sms, in-app
	Payload        string
}

type Store interface {
	// Create сохраняет уведомление. created=false — ключ идемпотентности уже встречался,
	// возвращается id первого уведомления.
	Create(ctx context.Context, n NewNotification) (id string, created bool, err error)
	// Status возвращает статус: accepted / in_queue / delivered / failed.
	Status(ctx context.Context, id string) (string, error)
	Close()
}
