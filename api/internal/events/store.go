package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xuroi/xuroi/api/internal/ids"
)

var (
	ErrConflict       = errors.New("idempotency conflict")
	ErrStreamConflict = errors.New("stream sequence conflict")
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Append(ctx context.Context, tx pgx.Tx, in AppendInput) (Event, error) {
	payload, err := json.Marshal(in.Payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal payload: %w", err)
	}

	if in.IdempotencyKey != nil && *in.IdempotencyKey != "" {
		var existing Event
		err := tx.QueryRow(ctx, `
			SELECT id, stream_id, sequence, type, actor_id, payload, schema_version, idempotency_key, created_at
			FROM events
			WHERE stream_id = $1 AND idempotency_key = $2
		`, in.StreamID, *in.IdempotencyKey).Scan(
			&existing.ID, &existing.StreamID, &existing.Sequence, &existing.Type,
			&existing.ActorID, &existing.Payload, &existing.SchemaVersion,
			&existing.IdempotencyKey, &existing.CreatedAt,
		)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("check idempotency: %w", err)
		}
	}

	var nextSeq int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(sequence), 0) + 1 FROM events WHERE stream_id = $1
	`, in.StreamID).Scan(&nextSeq)
	if err != nil {
		return Event{}, fmt.Errorf("next sequence: %w", err)
	}

	schemaVersion := in.SchemaVersion
	if schemaVersion == 0 {
		schemaVersion = 1
	}

	evt := Event{
		ID:             ids.New("evt_"),
		StreamID:       in.StreamID,
		Sequence:       nextSeq,
		Type:           in.Type,
		ActorID:        in.ActorID,
		Payload:        payload,
		SchemaVersion:  schemaVersion,
		IdempotencyKey: in.IdempotencyKey,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO events (id, stream_id, sequence, type, actor_id, payload, schema_version, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`, evt.ID, evt.StreamID, evt.Sequence, evt.Type, evt.ActorID, evt.Payload, evt.SchemaVersion, evt.IdempotencyKey,
	).Scan(&evt.CreatedAt)
	if err != nil {
		return Event{}, fmt.Errorf("insert event: %w", err)
	}

	return evt, nil
}

func (s *Store) ListAll(ctx context.Context) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, stream_id, sequence, type, actor_id, payload, schema_version, idempotency_key, created_at
		FROM events
		ORDER BY stream_id, sequence
	`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var evt Event
		if err := rows.Scan(
			&evt.ID, &evt.StreamID, &evt.Sequence, &evt.Type,
			&evt.ActorID, &evt.Payload, &evt.SchemaVersion,
			&evt.IdempotencyKey, &evt.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		out = append(out, evt)
	}
	return out, rows.Err()
}