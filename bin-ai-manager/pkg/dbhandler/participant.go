package dbhandler

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
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
