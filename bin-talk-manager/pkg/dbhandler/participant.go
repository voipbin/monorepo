package dbhandler

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/commondatabasehandler"
	"monorepo/bin-talk-manager/models/participant"
)

const tableParticipants = "talk_participants"

func (h *dbHandler) ParticipantCreate(ctx context.Context, p *participant.Participant) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	p.TMJoined = now

	// Use UPSERT to handle re-joins (participant leaves and joins again)
	// ON DUPLICATE KEY UPDATE prevents unique constraint violations
	query := `
		INSERT INTO talk_participants
		(id, customer_id, chat_id, owner_type, owner_id, tm_joined)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		tm_joined = VALUES(tm_joined)
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
		log.Errorf("Failed to create/update participant: %v", err)
		return err
	}

	return nil
}

func (h *dbHandler) ParticipantGet(ctx context.Context, id uuid.UUID) (*participant.Participant, error) {
	query := sq.Select(participant.GetDBFields()...).
		From(tableParticipants).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var p participant.Participant
	row := h.db.QueryRowContext(ctx, sqlQuery, args...)
	err = commondb.ScanRow(row, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (h *dbHandler) ParticipantList(ctx context.Context, filters map[participant.Field]any) ([]*participant.Participant, error) {
	query := sq.Select(participant.GetDBFields()...).
		From(tableParticipants).
		OrderBy("tm_joined DESC").
		PlaceholderFormat(sq.Question)

	// Apply filters
	query = commondb.ApplyFields(query, filters, nil)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := h.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
