package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Коды из SIS 3 (см. db/sql/expr_smallint.env).
var channelCode = map[string]int16{"push": 1, "email": 2, "sms": 3, "in-app": 4}
var statusName = map[int16]string{1: "accepted", 2: "in_queue", 3: "delivered", 4: "failed"}

// Postgres — запись на primary, чтение статуса на реплике (SIS 3, §6–§8).
// Оба адреса идут через PgBouncer (transaction pooling): w -> :6432/notif, r -> :6432/notif_ro.
type Postgres struct {
	w, r *pgxpool.Pool
	// Окно read-your-writes: статус свежего уведомления читаем с primary, потому что
	// реплика отстаёт (в замерах до 124 мс под нагрузкой). Окно с запасом над этим лагом.
	ryw time.Duration
}

// NewPostgres: readURL может быть пустым — тогда читаем с того же пула, что и пишем.
func NewPostgres(ctx context.Context, writeURL, readURL string, ryw time.Duration) (*Postgres, error) {
	w, err := newPool(ctx, writeURL)
	if err != nil {
		return nil, fmt.Errorf("write pool: %w", err)
	}
	r := w
	if readURL != "" {
		if r, err = newPool(ctx, readURL); err != nil {
			w.Close()
			return nil, fmt.Errorf("read pool: %w", err)
		}
	}
	return &Postgres{w: w, r: r, ryw: ryw}, nil
}

func newPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	// PgBouncer в transaction-режиме: следующий запрос может уйти на другое серверное соединение,
	// поэтому не используем именованные prepared statements (pgx по умолчанию их делает).
	// CacheDescribe кэширует только описание запроса и работает с любой версией PgBouncer.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	// Пул приложения больше, чем у PgBouncer (8 серверных соединений): очередь ждёт в PgBouncer.
	cfg.MaxConns = 32
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// notification_id для клиента = "<id>-<unix секунды создания>".
// Секунда создания нужна для partition pruning в GetStatus (таблица партицирована по created_at).
func formatID(id int64, created time.Time) string {
	return fmt.Sprintf("%d-%d", id, created.Unix())
}

func parseID(s string) (id, unix int64, err error) {
	a, b, ok := strings.Cut(s, "-")
	if !ok {
		return 0, 0, ErrInvalid
	}
	if id, err = strconv.ParseInt(a, 10, 64); err != nil {
		return 0, 0, ErrInvalid
	}
	if unix, err = strconv.ParseInt(b, 10, 64); err != nil {
		return 0, 0, ErrInvalid
	}
	return id, unix, nil
}

func (p *Postgres) Create(ctx context.Context, n NewNotification) (string, bool, error) {
	uid, err := strconv.ParseInt(n.UserID, 10, 64)
	if err != nil {
		return "", false, fmt.Errorf("%w: user_id must be a number", ErrInvalid)
	}
	ch, ok := channelCode[n.Channel]
	if !ok {
		return "", false, fmt.Errorf("%w: unknown channel %q", ErrInvalid, n.Channel)
	}
	title := n.TemplateID
	if title == "" {
		title = "notification"
	}

	// Одна короткая транзакция: ключ идемпотентности + строка inbox.
	tx, err := p.w.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var (
		id      int64
		created time.Time
	)
	if n.IdempotencyKey != "" {
		// UNIQUE по ключу: параллельный дубль ждёт коммита первого, затем получает ErrNoRows.
		err = tx.QueryRow(ctx,
			`INSERT INTO idempotency_keys (key, notification_id, created_at)
			 VALUES ($1, nextval('notif_id_seq'), now())
			 ON CONFLICT (key) DO NOTHING
			 RETURNING notification_id, created_at`, n.IdempotencyKey).Scan(&id, &created)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx,
				`SELECT notification_id, created_at FROM idempotency_keys WHERE key = $1`,
				n.IdempotencyKey).Scan(&id, &created)
			if err != nil {
				return "", false, err
			}
			return formatID(id, created), false, nil
		}
		if err != nil {
			return "", false, err
		}
	} else if err = tx.QueryRow(ctx, `SELECT nextval('notif_id_seq'), now()`).Scan(&id, &created); err != nil {
		return "", false, err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO notifications (id, user_id, created_at, channel, status, is_read, title, body)
		 VALUES ($1, $2, $3, $4, 1, false, $5, $6)`,
		id, uid, created, ch, title, n.Payload)
	if err != nil {
		return "", false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return formatID(id, created), true, nil
}

func (p *Postgres) Status(ctx context.Context, id string) (string, error) {
	nid, unix, err := parseID(id)
	if err != nil {
		return "", err
	}
	pool := p.r
	// Свежее уведомление читаем с primary (read-your-writes). unix обрезает доли секунды,
	// поэтому реальный возраст на <1 с меньше посчитанного: окно ryw надо брать >= 2–3 с.
	if time.Since(time.Unix(unix, 0)) < p.ryw {
		pool = p.w
	}
	var st int16
	// Диапазон по created_at (секунда создания) => partition pruning: читается одна партиция.
	err = pool.QueryRow(ctx,
		`SELECT status FROM notifications
		 WHERE id = $1
		   AND created_at >= to_timestamp($2::bigint::double precision)
		   AND created_at <  to_timestamp($2::bigint::double precision + 1)`,
		nid, unix).Scan(&st)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return statusName[st], nil
}

func (p *Postgres) Close() {
	p.w.Close()
	if p.r != p.w {
		p.r.Close()
	}
}
