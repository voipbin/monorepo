package dbhandler

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/mention"
	"monorepo/bin-talk-manager/models/message"
)

const tableMentions = "talk_mentions"

// MessageCreateWithMentions stores the message and its mention rows in one transaction. If any
// insert fails, nothing is stored, so a message without the mentions it carries never exists.
func (h *dbHandler) MessageCreateWithMentions(ctx context.Context, m *message.Message, mentions []*mention.Mention) error {
	sqlQuery, args, err := h.messageInsertQuery(m)
	if err != nil {
		return err
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "could not begin the transaction")
	}
	defer func() {
		// no-op after a successful commit
		if errRollback := tx.Rollback(); errRollback != nil && !errors.Is(errRollback, sql.ErrTxDone) {
			logrus.Errorf("Failed to roll back: %v", errRollback)
		}
	}()

	if _, err := tx.ExecContext(ctx, sqlQuery, args...); err != nil {
		logrus.Errorf("Failed to create message: %v", err)
		return errors.Wrap(err, "could not insert the message")
	}

	for _, mn := range mentions {
		// the mention row follows the message it belongs to, including its creation time
		mn.CustomerID = m.CustomerID
		mn.ChatID = m.ChatID
		mn.MessageID = m.ID
		mn.TMCreate = m.TMCreate

		fields, err := commondb.PrepareFields(mn)
		if err != nil {
			return errors.Wrap(err, "could not prepare the mention fields")
		}

		mentionQuery, mentionArgs, err := sq.Insert(tableMentions).
			SetMap(fields).
			PlaceholderFormat(sq.Question).
			ToSql()
		if err != nil {
			return errors.Wrap(err, "could not build the mention query")
		}

		if _, err := tx.ExecContext(ctx, mentionQuery, mentionArgs...); err != nil {
			logrus.Errorf("Failed to create mention: %v", err)
			return errors.Wrap(err, "could not insert the mention")
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "could not commit")
	}

	return nil
}

// MentionList returns the messages that mention the agent, newest first.
//
// The access rule R is applied in the query: the chat and the message are not deleted, the chat
// belongs to the customer, and the chat is a talk (public) chat or the agent is a participant.
func (h *dbHandler) MentionList(ctx context.Context, customerID, agentID uuid.UUID, token string, size uint64) ([]*message.Message, error) {
	fields := commondb.GetDBFields(&message.Message{})
	columns := make([]string, 0, len(fields))
	for _, f := range fields {
		columns = append(columns, "m."+f)
	}

	query := sq.Select(columns...).
		From(tableMentions+" t").
		Join(tableMessages+" m ON m.id = t.message_id").
		Join(tableChats+" c ON c.id = m.chat_id").
		Where(sq.Eq{"t.customer_id": customerID.Bytes(), "t.agent_id": agentID.Bytes(), "c.customer_id": customerID.Bytes()}).
		Where("m.tm_delete IS NULL").
		Where("c.tm_delete IS NULL").
		Where("(c.type = ? OR EXISTS (SELECT 1 FROM "+tableParticipants+" p WHERE p.chat_id = c.id AND p.owner_type = ? AND p.owner_id = ?))",
			string(chat.TypeTalk), string(commonidentity.OwnerTypeAgent), agentID.Bytes()).
		OrderBy("t.tm_create DESC").
		Limit(size).
		PlaceholderFormat(sq.Question)

	if token != "" {
		query = query.Where(sq.Lt{"t.tm_create": token})
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

	res := []*message.Message{}
	for rows.Next() {
		var m message.Message
		if err := commondb.ScanRow(rows, &m); err != nil {
			return nil, err
		}
		res = append(res, &m)
	}
	if errRows := rows.Err(); errRows != nil {
		return nil, errRows
	}

	return res, nil
}
