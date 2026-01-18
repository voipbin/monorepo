package dbhandler

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/participant"
)

const tableParticipants = "talk_participants"

func (h *dbHandler) ParticipantCreate(ctx context.Context, p *participant.Participant) error {
	now := h.utilHandler.TimeGetCurTime()
	p.TMJoined = now

	// Use UPSERT to handle re-joins (participant leaves and joins again)
	// SQLite: ON CONFLICT DO UPDATE prevents unique constraint violations
	query := `
		INSERT INTO talk_participants
		(id, customer_id, chat_id, owner_type, owner_id, tm_joined)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, owner_type, owner_id) DO UPDATE SET
		id = excluded.id,
		tm_joined = excluded.tm_joined
	`

	_, err := h.db.ExecContext(ctx, query,
		p.ID.Bytes(),
		p.CustomerID.Bytes(),
		p.ChatID.Bytes(),
		p.OwnerType,
		p.OwnerID.Bytes(),
		now,
	)

	if err != nil {
		logrus.Errorf("Failed to create/update participant: %v", err)
		return err
	}

	return nil
}

func (h *dbHandler) ParticipantGet(ctx context.Context, id uuid.UUID) (*participant.Participant, error) {
	fields := commondb.GetDBFields(&participant.Participant{})

	query := sq.Select(fields...).
		From(tableParticipants).
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
			logrus.Errorf("Failed to close rows: %v", closeErr)
		}
	}()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var p participant.Participant
	err = commondb.ScanRow(rows, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (h *dbHandler) ParticipantList(ctx context.Context, filters map[participant.Field]any) ([]*participant.Participant, error) {
	fields := commondb.GetDBFields(&participant.Participant{})

	query := sq.Select(fields...).
		From(tableParticipants).
		OrderBy("tm_joined DESC").
		PlaceholderFormat(sq.Question)

	// Apply filters
	query, err := commondb.ApplyFields(query, filters)
	if err != nil {
		logrus.Errorf("Failed to apply filters: %v", err)
		return nil, err
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
			logrus.Errorf("Failed to close rows: %v", closeErr)
		}
	}()

	var participants []*participant.Participant
	for rows.Next() {
		var p participant.Participant
		err = commondb.ScanRow(rows, &p)
		if err != nil {
			return nil, err
		}
		participants = append(participants, &p)
	}

	return participants, nil
}

func (h *dbHandler) ParticipantDelete(ctx context.Context, id uuid.UUID) error {
	query := sq.Delete(tableParticipants).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	return err
}

// ParticipantListByChatIDs gets all participants for multiple chat IDs
func (h *dbHandler) ParticipantListByChatIDs(ctx context.Context, chatIDs []uuid.UUID) ([]*participant.Participant, error) {
	if len(chatIDs) == 0 {
		return []*participant.Participant{}, nil
	}

	fields := commondb.GetDBFields(&participant.Participant{})

	// Convert UUIDs to bytes for query
	chatIDBytes := make([]interface{}, len(chatIDs))
	for i, id := range chatIDs {
		chatIDBytes[i] = id.Bytes()
	}

	query := sq.Select(fields...).
		From(tableParticipants).
		Where(sq.Eq{"chat_id": chatIDBytes}).
		OrderBy("tm_joined DESC").
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
			logrus.Errorf("Failed to close rows: %v", closeErr)
		}
	}()

	var participants []*participant.Participant
	for rows.Next() {
		var p participant.Participant
		err = commondb.ScanRow(rows, &p)
		if err != nil {
			return nil, err
		}
		participants = append(participants, &p)
	}

	return participants, nil
}
