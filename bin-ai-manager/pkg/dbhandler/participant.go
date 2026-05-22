package dbhandler

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/participant"
)

const participantTable = "ai_aicall_participants"

// ParticipantCreate inserts a new participant row linking an aicall to an ai.
// Duplicate (ai_id, aicall_id) pairs are silently ignored (return nil).
// All other errors propagate normally.
func (h *handler) ParticipantCreate(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error {
	query, args, err := sq.Insert(participantTable).
		SetMap(map[string]any{
			"ai_id":     aiID.Bytes(),
			"aicall_id": aicallID.Bytes(),
			"tm_create": h.utilHandler.TimeNow(),
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("ParticipantCreate: could not build query. err: %v", err)
	}

	_, err = h.db.ExecContext(ctx, query, args...)
	if err == nil {
		return nil
	}
	// Silently ignore duplicate primary key violations — the (ai_id, aicall_id) pair already exists.
	// MySQL returns error 1062 ("Duplicate entry"); SQLite returns "UNIQUE constraint failed".
	errStr := err.Error()
	if strings.Contains(errStr, "Duplicate entry") || strings.Contains(errStr, "UNIQUE constraint failed") {
		return nil
	}
	return fmt.Errorf("ParticipantCreate: could not execute. err: %v", err)
}

// ParticipantListByAIcallID returns all participant rows for the given aicall,
// ordered by tm_create desc, limited to size rows before the token timestamp.
func (h *handler) ParticipantListByAIcallID(ctx context.Context, aicallID uuid.UUID, size uint64, token string) ([]*participant.Participant, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(participant.Participant{})
	query, args, err := sq.Select(cols...).
		From(participantTable).
		Where(sq.Eq{"aicall_id": aicallID.Bytes()}).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ParticipantListByAIcallID: could not build query. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ParticipantListByAIcallID: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*participant.Participant{}
	for rows.Next() {
		u := &participant.Participant{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("ParticipantListByAIcallID: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// ParticipantListByAIID returns all participant rows for the given AI agent,
// ordered by tm_create desc, limited to size rows before the token timestamp.
func (h *handler) ParticipantListByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*participant.Participant, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(participant.Participant{})
	query, args, err := sq.Select(cols...).
		From(participantTable).
		Where(sq.Eq{"ai_id": aiID.Bytes()}).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ParticipantListByAIID: could not build query. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ParticipantListByAIID: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*participant.Participant{}
	for rows.Next() {
		u := &participant.Participant{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("ParticipantListByAIID: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}
