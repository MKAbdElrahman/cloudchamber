package messagedb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// Hub turns a Message DB NOTIFY into a wake-up for long-poll handlers.
// The payload is only a signal. A missed notification is repaired by reading from the cursor.
type Hub struct {
	log     *logger.Logger
	db      *sqlx.DB
	mu      sync.Mutex
	waiters map[chan struct{}]struct{}
}

// NewHub constructs a listener for the message_store channel.
func NewHub(log *logger.Logger, db *sqlx.DB) *Hub {
	return &Hub{
		log:     log,
		db:      db,
		waiters: map[chan struct{}]struct{}{},
	}
}

// Start listens until ctx is canceled.
func (h *Hub) Start(ctx context.Context) {
	go h.listen(ctx)
}

func (h *Hub) listen(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		err := h.listenOnce(ctx)
		if err == nil || ctx.Err() != nil {
			return
		}
		h.log.Error(ctx, "messagedb.listen", "err", err)
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (h *Hub) listenOnce(ctx context.Context) error {
	conn, err := h.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("listen conn: %w", err)
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		std, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("listen conn: expected pgx")
		}
		pg := std.Conn()
		if _, err := pg.Exec(ctx, `LISTEN message_store`); err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		for {
			if _, err := pg.WaitForNotification(ctx); err != nil {
				return err
			}
			h.wake()
		}
	})
}

func (h *Hub) wake() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Wait returns when a message is notified, the duration elapses, or ctx is canceled.
func (h *Hub) Wait(ctx context.Context, d time.Duration) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.waiters[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.waiters, ch)
		h.mu.Unlock()
	}()

	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-ch:
	case <-timer.C:
	}
}
