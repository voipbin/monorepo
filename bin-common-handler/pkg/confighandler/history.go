package confighandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

const (
	defaultHistoryPageSize = 20
	maxHistoryPageSize     = 100
)

func (h *handler) SecretWritable() bool {
	return h.crypto.HasKey()
}

func (h *handler) Histories(ctx context.Context, key, pageToken string, pageSize int) ([]*History, string, error) {
	k, err := h.lookupKey(key)
	if err != nil {
		return nil, "", err
	}

	switch {
	case pageSize < 0:
		return nil, "", fmt.Errorf("%w: the page size is negative", ErrInvalid)
	case pageSize == 0:
		pageSize = defaultHistoryPageSize
	case pageSize > maxHistoryPageSize:
		pageSize = maxHistoryPageSize
	}

	cursor, err := decodeHistoryToken(k.Name, pageToken)
	if err != nil {
		return nil, "", err
	}

	// one more than the page tells whether a next page exists
	rows, err := h.store.ListHistories(ctx, k.Name, cursor, pageSize+1)
	if err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > pageSize {
		rows = rows[:pageSize]
		next = encodeHistoryToken(k.Name, rows[pageSize-1])
	}
	if rows == nil {
		rows = []*History{}
	}

	return rows, next, nil
}

// The page token is opaque to the caller: base64url of "<key>|<time>|<id>" of the last row of the
// page. The time is the one the store returned, in UTC.
func encodeHistoryToken(key string, last *History) string {
	raw := fmt.Sprintf("%s|%s|%s", key, last.TMCreate.UTC().Format(time.RFC3339Nano), last.ID.String())

	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeHistoryToken(key, token string) (HistoryCursor, error) {
	if token == "" {
		return HistoryCursor{}, nil
	}

	invalid := fmt.Errorf("%w: the page token is not valid", ErrInvalid)

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return HistoryCursor{}, invalid
	}

	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || parts[0] != key {
		return HistoryCursor{}, invalid
	}

	tm, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return HistoryCursor{}, invalid
	}
	id, err := uuid.FromString(parts[2])
	if err != nil || id == uuid.Nil {
		return HistoryCursor{}, invalid
	}

	return HistoryCursor{TMCreate: tm.UTC(), ID: id}, nil
}
