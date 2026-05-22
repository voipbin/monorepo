package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/aiprompthistory"
)

const (
	promptHistoryTable = "ai_ai_prompt_histories"
)

// AIPromptHistoryCreate inserts a new AIPromptHistory row.
func (h *handler) AIPromptHistoryCreate(ctx context.Context, p *aiprompthistory.AIPromptHistory) error {
	p.TMCreate = h.utilHandler.TimeNow()

	fields, err := commondatabasehandler.PrepareFields(p)
	if err != nil {
		return fmt.Errorf("AIPromptHistoryCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(promptHistoryTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AIPromptHistoryCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIPromptHistoryCreate: could not execute query. err: %v", err)
	}

	return nil
}

// aiPromptHistoryGetFromDB fetches a single row by primary key.
func (h *handler) aiPromptHistoryGetFromDB(id uuid.UUID) (*aiprompthistory.AIPromptHistory, error) {
	cols := commondatabasehandler.GetDBFields(aiprompthistory.AIPromptHistory{})

	query, args, err := sq.Select(cols...).
		From(promptHistoryTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("aiPromptHistoryGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("aiPromptHistoryGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &aiprompthistory.AIPromptHistory{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("aiPromptHistoryGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// AIPromptHistoryGet returns a single entry by ID.
// No cache — infrequent access; prompt history is append-only.
func (h *handler) AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error) {
	return h.aiPromptHistoryGetFromDB(id)
}

// AIPromptHistoryGetsByAIID returns entries for the given AI, newest first.
// Token is a tm_create timestamp cursor (WHERE tm_create < token ORDER BY tm_create DESC).
// When token == "", falls back to h.utilHandler.TimeGetCurTime().
func (h *handler) AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(aiprompthistory.AIPromptHistory{})

	query, args, err := sq.Select(cols...).
		From(promptHistoryTable).
		Where(sq.Eq{"ai_id": aiID.Bytes()}).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("AIPromptHistoryGetsByAIID: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AIPromptHistoryGetsByAIID: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*aiprompthistory.AIPromptHistory{}
	for rows.Next() {
		u := &aiprompthistory.AIPromptHistory{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("AIPromptHistoryGetsByAIID: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}
