package rundb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// channel is where the runs table announces changes. See the runs_announce trigger.
const channel = "run_changed"

// leaderLock is the advisory lock key the leading controller holds.
// It is the first eight bytes of sha256("cloudchamber.run-controller").
const leaderLock int64 = 0x2f1c8e5a9d3b7046

// reconnectWait is how long Watch and Lead wait before trying the database again.
const reconnectWait = 2 * time.Second

// Change says which run changed. Spec is true when a person asked for
// something, and false when a worker reported a step.
type Change struct {
	RunID uuid.UUID
	Spec  bool
}

// Watch calls onChange for every run change the database announces, until ctx
// is done. It holds one connection. When that connection is lost it connects
// again and calls onGap, because changes announced in between were not heard.
// onGap is also called once at the start.
//
// A change is a hint that a run should be looked at. It carries no state.
func Watch(ctx context.Context, log *logger.Logger, db *sqlx.DB, onChange func(Change), onGap func()) {
	for ctx.Err() == nil {
		err := watchOnce(ctx, db, onChange, onGap)
		if ctx.Err() != nil {
			return
		}
		log.Error(ctx, "rundb.watch", "err", err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectWait):
		}
	}
}

func watchOnce(ctx context.Context, db *sqlx.DB, onChange func(Change), onGap func()) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("conn: %w", err)
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		std, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("listen needs a pgx connection, got %T", driverConn)
		}
		pg := std.Conn()

		if _, err := pg.Exec(ctx, "LISTEN "+channel); err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		onGap()

		for {
			note, err := pg.WaitForNotification(ctx)
			if err != nil {
				return fmt.Errorf("wait: %w", err)
			}
			if change, ok := parseChange(note.Payload); ok {
				onChange(change)
			}
		}
	})
}

// parseChange reads "<run id> spec" or "<run id> step".
func parseChange(payload string) (Change, bool) {
	id, kind, ok := strings.Cut(payload, " ")
	if !ok {
		return Change{}, false
	}
	runID, err := uuid.Parse(id)
	if err != nil {
		return Change{}, false
	}
	return Change{RunID: runID, Spec: kind == "spec"}, true
}

// Lead runs lead while this process holds the controller's lock, and waits
// for the lock while another process holds it. The lock is tied to one
// database connection, so it is released when this process stops or loses that
// connection, and another process takes over. The context passed to lead is
// canceled when the lock is lost.
//
// Leading keeps several API replicas from doing the same work. It is not what
// makes that work safe: a connection can be lost a moment before this process
// notices. What makes it safe is that every write to a run is guarded by the
// run's generation and every action on the engine can be repeated.
func Lead(ctx context.Context, log *logger.Logger, db *sqlx.DB, lead func(ctx context.Context)) {
	for ctx.Err() == nil {
		if err := leadOnce(ctx, log, db, lead); err != nil && ctx.Err() == nil {
			log.Error(ctx, "rundb.lead", "err", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectWait):
		}
	}
}

func leadOnce(ctx context.Context, log *logger.Logger, db *sqlx.DB, lead func(ctx context.Context)) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("conn: %w", err)
	}
	defer conn.Close()

	var held bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", leaderLock).Scan(&held); err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	if !held {
		return nil
	}
	// Unlocking on a fresh context matters when ctx is already done: without
	// it the pool could hand this connection, still holding the lock, to another caller.
	defer func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", leaderLock)
	}()

	log.Info(ctx, "rundb.lead", "status", "leading")
	defer log.Info(ctx, "rundb.lead", "status", "stopped leading")

	leadCtx, stop := context.WithCancel(ctx)
	defer stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		lead(leadCtx)
	}()

	// The lock lives as long as the connection. Check the connection, and stop leading when it is gone.
	ticker := time.NewTicker(reconnectWait)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return nil
		case <-ticker.C:
			if err := conn.PingContext(leadCtx); err != nil {
				stop()
				<-done
				return fmt.Errorf("lost the lock's connection: %w", err)
			}
		}
	}
}
