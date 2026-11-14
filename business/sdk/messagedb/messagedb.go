// Package messagedb writes and reads Message DB streams.
package messagedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// Message is one row in the message store.
type Message struct {
	ID             string
	StreamName     string
	Type           string
	Position       int64
	GlobalPosition int64
	Data           json.RawMessage
	Metadata       json.RawMessage
	Time           time.Time
}

// Written is the identity of one appended message.
type Written struct {
	ID             uuid.UUID `db:"id"`
	Position       int64     `db:"position"`
	GlobalPosition int64     `db:"global_position"`
	Time           time.Time `db:"time"`
}

// Write appends one message. expected version is the stream version just read.
// A conflicting writer is retried against the new version.
func Write(ctx context.Context, db sqlx.ExtContext, id uuid.UUID, stream, typ string, data, metadata json.RawMessage) (Written, error) {
	return write(ctx, db, id, stream, typ, data, metadata, true)
}

// WriteStable appends one message and keeps its id across a version conflict.
// A second write of that id returns the row already stored.
func WriteStable(ctx context.Context, db sqlx.ExtContext, id uuid.UUID, stream, typ string, data, metadata json.RawMessage) (Written, error) {
	if id == uuid.Nil {
		return Written{}, fmt.Errorf("write message: id is required")
	}
	return write(ctx, db, id, stream, typ, data, metadata, false)
}

func write(ctx context.Context, db sqlx.ExtContext, id uuid.UUID, stream, typ string, data, metadata json.RawMessage, replaceID bool) (Written, error) {
	var err error
	var written Written
	for attempt := range 5 {
		// A failed statement aborts the surrounding transaction. The savepoint
		// lets the next attempt read the new stream version.
		if err = markWrite(ctx, db); err != nil {
			return Written{}, err
		}
		var version int64
		version, err = streamVersion(ctx, db, stream)
		if err != nil {
			return Written{}, errors.Join(err, restoreWrite(ctx, db))
		}
		written, err = writeOnce(ctx, db, id, stream, typ, data, metadata, version)
		if err == nil {
			if err = releaseWrite(ctx, db); err != nil {
				return Written{}, err
			}
			return written, nil
		}
		if rerr := restoreWrite(ctx, db); rerr != nil {
			return Written{}, errors.Join(err, rerr)
		}
		if !replaceID && duplicateID(err) {
			return writtenByID(ctx, db, id)
		}
		if !wrongVersion(err) || attempt == 4 {
			return Written{}, err
		}
		if replaceID {
			id = uuid.New()
		}
	}
	return Written{}, err
}

func markWrite(ctx context.Context, db sqlx.ExtContext) error {
	if _, ok := db.(*sqlx.Tx); !ok {
		return nil
	}
	if _, err := db.ExecContext(ctx, "SAVEPOINT message_write"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

func restoreWrite(ctx context.Context, db sqlx.ExtContext) error {
	if _, ok := db.(*sqlx.Tx); !ok {
		return nil
	}
	if _, err := db.ExecContext(ctx, "ROLLBACK TO SAVEPOINT message_write"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if _, err := db.ExecContext(ctx, "RELEASE SAVEPOINT message_write"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

func releaseWrite(ctx context.Context, db sqlx.ExtContext) error {
	if _, ok := db.(*sqlx.Tx); !ok {
		return nil
	}
	if _, err := db.ExecContext(ctx, "RELEASE SAVEPOINT message_write"); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

func streamVersion(ctx context.Context, db sqlx.ExtContext, stream string) (int64, error) {
	var version sql.NullInt64
	if err := sqlx.GetContext(ctx, db, &version, `SELECT message_store.stream_version($1)`, stream); err != nil {
		return 0, fmt.Errorf("stream version: %w", err)
	}
	if !version.Valid {
		return -1, nil
	}
	return version.Int64, nil
}

func writeOnce(ctx context.Context, db sqlx.ExtContext, id uuid.UUID, stream, typ string, data, metadata json.RawMessage, expected int64) (Written, error) {
	var position int64
	// The insert has to finish in its own statement. A join in the same
	// statement does not see the row write_message just added.
	err := sqlx.GetContext(ctx, db, &position, `
		SELECT message_store.write_message($1, $2, $3, $4::jsonb, $5::jsonb, $6)`,
		id.String(), stream, typ, string(data), string(metadata), expected)
	if err != nil {
		return Written{}, fmt.Errorf("write message: %w", err)
	}
	var written Written
	err = sqlx.GetContext(ctx, db, &written, `
		SELECT id, position, global_position, time
		FROM message_store.messages
		WHERE stream_name = $1 AND position = $2`, stream, position)
	if err != nil {
		return Written{}, fmt.Errorf("read written message: %w", err)
	}
	written.Time = written.Time.UTC()
	return written, nil
}

func wrongVersion(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && strings.Contains(pgErr.Message, "Wrong expected version")
}

func duplicateID(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == "messages_id" || strings.Contains(pgErr.Message, "messages_id")
}

func writtenByID(ctx context.Context, db sqlx.ExtContext, id uuid.UUID) (Written, error) {
	var written Written
	err := sqlx.GetContext(ctx, db, &written, `
		SELECT id, position, global_position, time
		FROM message_store.messages
		WHERE id = $1`, id)
	if err != nil {
		return Written{}, fmt.Errorf("read written message: %w", err)
	}
	written.Time = written.Time.UTC()
	return written, nil
}

// MessageColumns is the select list for one message_store.messages row.
const MessageColumns = `
	id::text AS id,
	stream_name,
	type,
	position,
	global_position,
	data::text AS data,
	metadata::text AS metadata,
	time`

// Select reads messages with a query the caller owns.
func Select(ctx context.Context, db sqlx.ExtContext, query string, args ...any) ([]Message, error) {
	var rows []row
	if err := sqlx.SelectContext(ctx, db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select messages: %w", err)
	}
	return toMessages(rows), nil
}

type row struct {
	ID             string    `db:"id"`
	StreamName     string    `db:"stream_name"`
	Type           string    `db:"type"`
	Position       int64     `db:"position"`
	GlobalPosition int64     `db:"global_position"`
	Data           string    `db:"data"`
	Metadata       string    `db:"metadata"`
	Time           time.Time `db:"time"`
}

func toMessages(rows []row) []Message {
	out := make([]Message, len(rows))
	for i, r := range rows {
		out[i] = Message{
			ID:             r.ID,
			StreamName:     r.StreamName,
			Type:           r.Type,
			Position:       r.Position,
			GlobalPosition: r.GlobalPosition,
			Data:           json.RawMessage(r.Data),
			Metadata:       json.RawMessage(r.Metadata),
			Time:           r.Time,
		}
	}
	return out
}
