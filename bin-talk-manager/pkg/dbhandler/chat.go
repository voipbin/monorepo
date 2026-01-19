package dbhandler

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/participant"
)

const tableChats = "talk_chats"

func (h *dbHandler) ChatCreate(ctx context.Context, t *chat.Chat) error {
	now := h.utilHandler.TimeGetCurTime()
	t.TMCreate = now
	t.TMUpdate = now
	t.TMDelete = commondb.DefaultTimeStamp

	fields, err := commondb.PrepareFields(t)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Insert(tableChats).
		SetMap(fields).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		logrus.Errorf("Failed to build query: %v", err)
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		logrus.Errorf("Failed to create chat: %v", err)
		return err
	}

	return nil
}

func (h *dbHandler) ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	fields := commondb.GetDBFields(&chat.Chat{})

	query := sq.Select(fields...).
		From(tableChats).
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

	var t chat.Chat
	err = commondb.ScanRow(rows, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (h *dbHandler) ChatList(ctx context.Context, filters map[chat.Field]any, token string, size uint64) ([]*chat.Chat, error) {
	// Prefix all chat fields with table name for JOIN queries
	fields := make([]string, 0, len(chat.GetDBFields()))
	for _, field := range chat.GetDBFields() {
		fields = append(fields, tableChats+"."+field)
	}

	query := sq.Select(fields...).
		From(tableChats).
		OrderBy(tableChats + ".tm_create DESC").
		Limit(size).
		PlaceholderFormat(sq.Question)

	// Check if owner filters are present (requires JOIN with participants)
	ownerType, hasOwnerType := filters[chat.FieldOwnerType]
	ownerID, hasOwnerID := filters[chat.FieldOwnerID]

	if hasOwnerType || hasOwnerID {
		// Add INNER JOIN with participants table to filter by owner
		query = query.
			InnerJoin(tableParticipants + " ON " + tableChats + ".id = " + tableParticipants + ".chat_id")

		// Apply owner filters on participants table
		if hasOwnerType {
			query = query.Where(sq.Eq{tableParticipants + ".owner_type": ownerType})
		}
		if hasOwnerID {
			// Convert ownerID to bytes if it's a UUID
			if ownerUUID, ok := ownerID.(uuid.UUID); ok {
				query = query.Where(sq.Eq{tableParticipants + ".owner_id": ownerUUID.Bytes()})
			}
		}

		// Remove owner filters from map before applying to chats table
		chatFilters := make(map[chat.Field]any)
		for k, v := range filters {
			if k != chat.FieldOwnerType && k != chat.FieldOwnerID {
				chatFilters[k] = v
			}
		}
		filters = chatFilters
	}

	// Handle deleted filter (special filter-only field)
	if deleted, hasDeleted := filters[chat.FieldDeleted]; hasDeleted {
		if deletedBool, ok := deleted.(bool); ok {
			if deletedBool {
				// Get deleted chats (tm_delete != default timestamp)
				query = query.Where(sq.NotEq{tableChats + ".tm_delete": commondb.DefaultTimeStamp})
			} else {
				// Get non-deleted chats (tm_delete == default timestamp)
				query = query.Where(sq.Eq{tableChats + ".tm_delete": commondb.DefaultTimeStamp})
			}
		}
		// Remove deleted filter from map before applying to chats table
		delete(filters, chat.FieldDeleted)
	}

	// Apply remaining filters to chats table
	query, err := commondb.ApplyFields(query, filters)
	if err != nil {
		logrus.Errorf("Failed to apply filters: %v", err)
		return nil, err
	}

	// Apply pagination token
	if token != "" {
		query = query.Where(sq.Lt{tableChats + ".tm_create": token})
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

	var talks []*chat.Chat
	for rows.Next() {
		var t chat.Chat
		err = commondb.ScanRow(rows, &t)
		if err != nil {
			return nil, err
		}
		talks = append(talks, &t)
	}

	// Load participants for all chats
	if len(talks) > 0 {
		// Collect chat IDs
		chatIDs := make([]uuid.UUID, len(talks))
		for i, t := range talks {
			chatIDs[i] = t.ID
		}

		// Fetch all participants for these chats
		participants, err := h.ParticipantListByChatIDs(ctx, chatIDs)
		if err != nil {
			logrus.Errorf("Failed to load participants: %v", err)
			// Continue without participants rather than failing entire request
		} else {
			// Group participants by chat_id
			participantsByChatID := make(map[uuid.UUID][]*participant.Participant)
			for _, p := range participants {
				participantsByChatID[p.ChatID] = append(participantsByChatID[p.ChatID], p)
			}

			// Populate each chat's participants
			for _, t := range talks {
				if ps, ok := participantsByChatID[t.ID]; ok {
					t.Participants = ps
				} else {
					t.Participants = []*participant.Participant{}
				}
			}
		}
	}

	return talks, nil
}

func (h *dbHandler) ChatUpdate(ctx context.Context, id uuid.UUID, fields map[chat.Field]any) error {
	now := h.utilHandler.TimeGetCurTime()
	fields[chat.FieldTMUpdate] = now

	preparedFields, err := commondb.PrepareFields(fields)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Update(tableChats).
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

func (h *dbHandler) ChatDelete(ctx context.Context, id uuid.UUID) error {
	now := h.utilHandler.TimeGetCurTime()

	query := sq.Update(tableChats).
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

// ChatMemberCountIncrement atomically increments the member_count by 1
func (h *dbHandler) ChatMemberCountIncrement(ctx context.Context, chatID uuid.UUID) error {
	now := h.utilHandler.TimeGetCurTime()

	sqlQuery := `UPDATE talk_chats SET member_count = member_count + 1, tm_update = ? WHERE id = ?`

	_, err := h.db.ExecContext(ctx, sqlQuery, now, chatID.Bytes())
	if err != nil {
		logrus.Errorf("Failed to increment member_count: %v", err)
		return err
	}

	return nil
}

// ChatMemberCountDecrement atomically decrements the member_count by 1 (minimum 0)
func (h *dbHandler) ChatMemberCountDecrement(ctx context.Context, chatID uuid.UUID) error {
	now := h.utilHandler.TimeGetCurTime()

	// Use GREATEST to ensure member_count doesn't go below 0
	sqlQuery := `UPDATE talk_chats SET member_count = GREATEST(member_count - 1, 0), tm_update = ? WHERE id = ?`

	_, err := h.db.ExecContext(ctx, sqlQuery, now, chatID.Bytes())
	if err != nil {
		logrus.Errorf("Failed to decrement member_count: %v", err)
		return err
	}

	return nil
}

// FindDirectChatByParticipants finds an existing direct chat between exactly two participants.
// Returns nil, nil if no matching chat is found.
func (h *dbHandler) FindDirectChatByParticipants(ctx context.Context, customerID uuid.UUID, ownerType1 string, ownerID1 uuid.UUID, ownerType2 string, ownerID2 uuid.UUID) (*chat.Chat, error) {
	// This query finds direct chats where:
	// 1. Chat is of type 'direct'
	// 2. Chat belongs to the customer
	// 3. Chat is not deleted
	// 4. Chat has exactly 2 participants
	// 5. Both specified participants are members

	// SQL approach:
	// Find chat_id from participants where both users are participants,
	// then filter by direct type, customer_id, and not deleted,
	// and ensure exactly 2 participants

	sqlQuery := `
		SELECT c.id, c.customer_id, c.type, c.name, c.detail, c.tm_create, c.tm_update, c.tm_delete
		FROM talk_chats c
		WHERE c.customer_id = ?
		  AND c.type = ?
		  AND c.tm_delete = ?
		  AND c.id IN (
		      SELECT p1.chat_id
		      FROM talk_participants p1
		      JOIN talk_participants p2 ON p1.chat_id = p2.chat_id
		      WHERE p1.owner_type = ? AND p1.owner_id = ?
		        AND p2.owner_type = ? AND p2.owner_id = ?
		  )
		  AND (
		      SELECT COUNT(*) FROM talk_participants p WHERE p.chat_id = c.id
		  ) = 2
		LIMIT 1
	`

	args := []interface{}{
		customerID.Bytes(),
		string(chat.TypeDirect),
		commondb.DefaultTimeStamp,
		ownerType1,
		ownerID1.Bytes(),
		ownerType2,
		ownerID2.Bytes(),
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
		// No existing direct chat found
		return nil, nil
	}

	var t chat.Chat
	err = commondb.ScanRow(rows, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}
