package dbhandler

import (
	"context"
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/participant"
)

const tableParticipants = "talk_participants"

func (h *dbHandler) ParticipantCreate(ctx context.Context, p *participant.Participant) error {
	now := h.utilHandler.TimeNow()
	p.TMJoined = now

	// Try INSERT first
	fields, err := commondb.PrepareFields(p)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Insert(tableParticipants).
		SetMap(fields).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		logrus.Errorf("Failed to build query: %v", err)
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		// If unique constraint violation, update existing record
		// Error 1062 (MySQL) or "UNIQUE constraint failed" (SQLite)
		if sqErr, ok := err.(interface{ Error() string }); ok {
			errMsg := sqErr.Error()
			if contains(errMsg, "UNIQUE") || contains(errMsg, "1062") || contains(errMsg, "Duplicate") {
				// Update existing participant
				updateFields := map[participant.Field]any{
					participant.FieldID:       p.ID,
					participant.FieldTMJoined: now,
				}

				preparedUpdateFields, err := commondb.PrepareFields(updateFields)
				if err != nil {
					logrus.Errorf("Failed to prepare update fields: %v", err)
					return err
				}

				updateQuery := sq.Update(tableParticipants).
					SetMap(preparedUpdateFields).
					Where(sq.And{
						sq.Eq{"chat_id": p.ChatID.Bytes()},
						sq.Eq{"owner_type": p.OwnerType},
						sq.Eq{"owner_id": p.OwnerID.Bytes()},
					}).
					PlaceholderFormat(sq.Question)

				updateSQL, updateArgs, err := updateQuery.ToSql()
				if err != nil {
					logrus.Errorf("Failed to build update query: %v", err)
					return err
				}

				_, err = h.db.ExecContext(ctx, updateSQL, updateArgs...)
				if err != nil {
					logrus.Errorf("Failed to update participant: %v", err)
					return err
				}

				return nil
			}
		}

		logrus.Errorf("Failed to create participant: %v", err)
		return err
	}

	return nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
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

	participants := []*participant.Participant{}
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

	participants := []*participant.Participant{}
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
