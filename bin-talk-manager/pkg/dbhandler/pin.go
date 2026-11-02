package dbhandler

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/pin"
)

const tablePins = "talk_pins"

// ownerTypeAgent is the participant owner type that grants a pin its visibility in a non-talk chat.
const ownerTypeAgent = "agent"

// PinCreate stores a pin. It is idempotent per owner and target: if the owner already pinned the
// target, the existing row is returned and the given row is not stored.
func (h *dbHandler) PinCreate(ctx context.Context, p *pin.Pin) (*pin.Pin, error) {
	p.TMCreate = h.utilHandler.TimeNow()

	fields, err := commondb.PrepareFields(p)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return nil, err
	}

	sqlQuery, args, err := sq.Insert(tablePins).
		SetMap(fields).
		PlaceholderFormat(sq.Question).
		ToSql()
	if err != nil {
		logrus.Errorf("Failed to build query: %v", err)
		return nil, err
	}

	if _, err = h.db.ExecContext(ctx, sqlQuery, args...); err != nil {
		// A unique key violation (MySQL error 1062, SQLite "UNIQUE constraint failed") means the
		// owner pinned the target already. Read that row back; if there is none, the violation
		// was something else and the original error stands.
		if isPinDuplicateError(err) {
			existing, errGet := h.pinGetByTarget(ctx, string(p.OwnerType), p.OwnerID, p.TargetType, p.TargetID)
			if errGet == nil {
				return existing, nil
			}
		}

		logrus.Errorf("Failed to create pin: %v", err)
		return nil, err
	}

	return p, nil
}

// isPinDuplicateError reports whether err is a unique key violation: MySQL/MariaDB error 1062
// (Duplicate entry) or the SQLite "UNIQUE constraint failed" used by the unit tests.
func isPinDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE") || contains(msg, "1062") || contains(msg, "Duplicate")
}

// PinGet returns a pin by id.
func (h *dbHandler) PinGet(ctx context.Context, id uuid.UUID) (*pin.Pin, error) {
	return h.pinGetWhere(ctx, sq.Eq{"id": id.Bytes()})
}

func (h *dbHandler) pinGetByTarget(ctx context.Context, ownerType string, ownerID uuid.UUID, targetType pin.TargetType, targetID uuid.UUID) (*pin.Pin, error) {
	return h.pinGetWhere(ctx, sq.And{
		sq.Eq{"owner_type": ownerType},
		sq.Eq{"owner_id": ownerID.Bytes()},
		sq.Eq{"target_type": string(targetType)},
		sq.Eq{"target_id": targetID.Bytes()},
	})
}

func (h *dbHandler) pinGetWhere(ctx context.Context, where sq.Sqlizer) (*pin.Pin, error) {
	fields := commondb.GetDBFields(&pin.Pin{})

	sqlQuery, args, err := sq.Select(fields...).
		From(tablePins).
		Where(where).
		PlaceholderFormat(sq.Question).
		ToSql()
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
		if errRows := rows.Err(); errRows != nil {
			return nil, errRows
		}
		return nil, ErrNotFound
	}

	var p pin.Pin
	if err := commondb.ScanRow(rows, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// PinList returns the pins of the owner that the owner can see, newest first.
//
// Visibility (rule R): the chat of the pin is not deleted, belongs to the given customer, and is
// either a talk chat or a chat in which the owner is an agent participant; for a message pin the
// message is not deleted and belongs to the chat. The pin of a message carries its message.
// targetType and chatID filter the result when they are set.
func (h *dbHandler) PinList(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID, targetType pin.TargetType, chatID uuid.UUID, token string, size uint64) ([]*pin.Pin, error) {
	fields := commondb.GetDBFields(&pin.Pin{})
	prefixed := make([]string, len(fields))
	for i, f := range fields {
		prefixed[i] = "p." + f
	}

	query := sq.Select(prefixed...).
		From(tablePins+" p").
		Join("talk_chats c ON c.id = p.chat_id AND c.tm_delete IS NULL AND c.customer_id = ?", customerID.Bytes()).
		Where(sq.Eq{
			"p.customer_id": customerID.Bytes(),
			"p.owner_type":  ownerType,
			"p.owner_id":    ownerID.Bytes(),
		}).
		Where("(p.target_type = ? OR EXISTS (SELECT 1 FROM talk_messages m WHERE m.id = p.target_id AND m.chat_id = p.chat_id AND m.tm_delete IS NULL))", string(pin.TargetTypeChat)).
		Where("(c.type = ? OR EXISTS (SELECT 1 FROM talk_participants tp WHERE tp.chat_id = c.id AND tp.owner_type = ? AND tp.owner_id = p.owner_id))", string(chat.TypeTalk), ownerTypeAgent).
		OrderBy("p.tm_create DESC").
		Limit(size).
		PlaceholderFormat(sq.Question)

	if targetType != "" {
		query = query.Where(sq.Eq{"p.target_type": string(targetType)})
	}
	if chatID != uuid.Nil {
		query = query.Where(sq.Eq{"p.chat_id": chatID.Bytes()})
	}
	if token != "" {
		query = query.Where(sq.Lt{"p.tm_create": token})
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

	pins := []*pin.Pin{}
	for rows.Next() {
		var p pin.Pin
		if err := commondb.ScanRow(rows, &p); err != nil {
			return nil, err
		}
		pins = append(pins, &p)
	}
	if errRows := rows.Err(); errRows != nil {
		return nil, errRows
	}

	return h.pinAttachMessages(ctx, pins)
}

// pinAttachMessages sets Message of the message pins with one query. A message pin whose message
// is gone since the list query (deleted in between) is dropped.
func (h *dbHandler) pinAttachMessages(ctx context.Context, pins []*pin.Pin) ([]*pin.Pin, error) {
	ids := []interface{}{}
	for _, p := range pins {
		if p.TargetType == pin.TargetTypeMessage {
			ids = append(ids, p.TargetID.Bytes())
		}
	}
	if len(ids) == 0 {
		return pins, nil
	}

	fields := commondb.GetDBFields(&message.Message{})
	sqlQuery, args, err := sq.Select(fields...).
		From(tableMessages).
		Where(sq.Eq{"id": ids}).
		Where(sq.Eq{"tm_delete": nil}).
		PlaceholderFormat(sq.Question).
		ToSql()
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

	messages := map[uuid.UUID]*message.Message{}
	for rows.Next() {
		var m message.Message
		if err := commondb.ScanRow(rows, &m); err != nil {
			return nil, err
		}
		messages[m.ID] = &m
	}
	if errRows := rows.Err(); errRows != nil {
		return nil, errRows
	}

	res := make([]*pin.Pin, 0, len(pins))
	for _, p := range pins {
		if p.TargetType == pin.TargetTypeMessage {
			m, ok := messages[p.TargetID]
			if !ok {
				continue
			}
			p.Message = m
		}
		res = append(res, p)
	}

	return res, nil
}

// PinDelete deletes the pin of the owner. It returns ErrNotFound if there is no such pin of that
// owner and customer, which includes the pin of another owner.
func (h *dbHandler) PinDelete(ctx context.Context, id, customerID uuid.UUID, ownerType string, ownerID uuid.UUID) error {
	sqlQuery, args, err := sq.Delete(tablePins).
		Where(sq.Eq{
			"id":          id.Bytes(),
			"customer_id": customerID.Bytes(),
			"owner_type":  ownerType,
			"owner_id":    ownerID.Bytes(),
		}).
		PlaceholderFormat(sq.Question).
		ToSql()
	if err != nil {
		return err
	}

	res, err := h.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
