package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/message"
)

const tableMessages = "talk_messages"

func (h *dbHandler) MessageCreate(ctx context.Context, m *message.Message) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	m.TMCreate = now
	m.TMUpdate = now

	// Initialize empty metadata if not set
	if m.Metadata == "" {
		m.Metadata = `{"reactions":[]}`
	}

	fields, err := commondb.PrepareFields(m)
	if err != nil {
		log.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	// Handle ParentID manually (nullable UUID)
	if m.ParentID != nil {
		fields["parent_id"] = m.ParentID.Bytes()
	} else {
		fields["parent_id"] = nil
	}

	query := sq.Insert(tableMessages).
		SetMap(fields).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		log.Errorf("Failed to build query: %v", err)
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		log.Errorf("Failed to create message: %v", err)
		return err
	}

	return nil
}

func (h *dbHandler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	fields := commondb.GetDBFields(&message.Message{})

	query := sq.Select(fields...).
		From(tableMessages).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := h.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Errorf("Failed to close rows: %v", closeErr)
		}
	}()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var m message.Message
	err = commondb.ScanRow(rows, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (h *dbHandler) MessageList(ctx context.Context, filters map[message.Field]any, token string, size uint64) ([]*message.Message, error) {
	fields := commondb.GetDBFields(&message.Message{})

	query := sq.Select(fields...).
		From(tableMessages).
		OrderBy("tm_create DESC").
		Limit(size).
		PlaceholderFormat(sq.Question)

	// Apply filters
	query, err := commondb.ApplyFields(query, filters)
	if err != nil {
		log.Errorf("Failed to apply filters: %v", err)
		return nil, err
	}

	// Apply pagination token
	if token != "" {
		query = query.Where(sq.Lt{"tm_create": token})
	}

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := h.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Errorf("Failed to close rows: %v", closeErr)
		}
	}()

	var messages []*message.Message
	for rows.Next() {
		var m message.Message
		err = commondb.ScanRow(rows, &m)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}

	return messages, nil
}

func (h *dbHandler) MessageUpdate(ctx context.Context, id uuid.UUID, fields map[message.Field]any) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	fields[message.FieldTMUpdate] = now

	preparedFields, err := commondb.PrepareFields(fields)
	if err != nil {
		log.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Update(tableMessages).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	return err
}

func (h *dbHandler) MessageDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")

	query := sq.Update(tableMessages).
		Set("tm_delete", now).
		Set("tm_update", now).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	return err
}

// MessageAddReactionAtomic atomically adds a reaction using JSON functions
// This prevents race conditions when multiple users add reactions simultaneously
func (h *dbHandler) MessageAddReactionAtomic(ctx context.Context, messageID uuid.UUID, reactionJSON string) error {
	// SQLite-compatible JSON manipulation
	// Get current metadata, parse it, add reaction, and update
	var metadataJSON string
	err := h.db.QueryRowContext(ctx,
		"SELECT metadata FROM talk_messages WHERE id = ?",
		messageID.Bytes(),
	).Scan(&metadataJSON)
	if err != nil {
		return err
	}

	// Parse current metadata
	var metadata message.Metadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		log.Errorf("Failed to unmarshal metadata: %v", err)
		return err
	}

	// Parse and append new reaction
	var newReaction message.Reaction
	if err := json.Unmarshal([]byte(reactionJSON), &newReaction); err != nil {
		log.Errorf("Failed to unmarshal reaction: %v", err)
		return err
	}

	metadata.Reactions = append(metadata.Reactions, newReaction)

	// Serialize back to JSON
	updatedJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Errorf("Failed to marshal metadata: %v", err)
		return err
	}

	// Update atomically
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	query := `UPDATE talk_messages SET metadata = ?, tm_update = ? WHERE id = ?`

	_, err = h.db.ExecContext(ctx, query, string(updatedJSON), now, messageID.Bytes())
	if err != nil {
		log.Errorf("Failed to add reaction atomically: %v", err)
		return err
	}

	return nil
}

// MessageRemoveReactionAtomic atomically removes a reaction by filtering the JSON array
func (h *dbHandler) MessageRemoveReactionAtomic(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) error {
	// Get current metadata
	var metadataJSON string
	err := h.db.QueryRowContext(ctx,
		"SELECT metadata FROM talk_messages WHERE id = ?",
		messageID.Bytes(),
	).Scan(&metadataJSON)
	if err != nil {
		return err
	}

	// Parse and filter reactions
	var metadata message.Metadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		log.Errorf("Failed to unmarshal metadata: %v", err)
		return err
	}

	var filtered []message.Reaction
	for _, r := range metadata.Reactions {
		if r.Emoji != emoji || r.OwnerType != ownerType || r.OwnerID != ownerID {
			filtered = append(filtered, r)
		}
	}

	// Update with filtered reactions
	metadata.Reactions = filtered
	updatedJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Errorf("Failed to marshal metadata: %v", err)
		return err
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	query := `UPDATE talk_messages SET metadata = ?, tm_update = ? WHERE id = ?`

	_, err = h.db.ExecContext(ctx, query, string(updatedJSON), now, messageID.Bytes())
	if err != nil {
		log.Errorf("Failed to remove reaction atomically: %v", err)
		return err
	}

	return nil
}
