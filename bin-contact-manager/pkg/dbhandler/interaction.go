package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	mysql_driver "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/interaction"
)

const (
	interactionTable = "contact_interactions"
)

// AddressPair is a (type, target) pair used for multi-column IN expansion.
// Exported so contacthandler and tests can construct slices.
type AddressPair struct {
	Type   string
	Target string
}

// InteractionCreate inserts an Interaction row into contact_interactions.
// It is idempotent: a duplicate-key error (MySQL errno 1062) is silently
// ignored so the caller does not need to guard against at-least-once delivery.
func (h *handler) InteractionCreate(ctx context.Context, i *interaction.Interaction) error {
	fields, err := commondatabasehandler.PrepareFields(i)
	if err != nil {
		return fmt.Errorf("could not prepare fields. InteractionCreate. err: %v", err)
	}

	query, args, err := sq.Insert(interactionTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. InteractionCreate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		// Idempotent: ignore duplicate-key errors so at-least-once event
		// delivery (e.g. RabbitMQ requeue on crash) does not surface errors.
		// MySQL errno 1062 in production; SQLite UNIQUE constraint in tests.
		if me, ok := err.(*mysql_driver.MySQLError); ok && me.Number == 1062 {
			return nil
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil
		}
		return fmt.Errorf("could not create interaction. InteractionCreate. err: %v", err)
	}

	return nil
}

// interactionGetFromRow scans a single row into an Interaction struct.
func interactionGetFromRow(rows *sql.Rows) (*interaction.Interaction, error) {
	res := &interaction.Interaction{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. interactionGetFromRow. err: %v", err)
	}
	return res, nil
}

// InteractionGet fetches a single interaction by ID.
// Returns ErrNotFound if absent.
func (h *handler) InteractionGet(ctx context.Context, id uuid.UUID) (*interaction.Interaction, error) {
	columns := commondatabasehandler.GetDBFields(&interaction.Interaction{})

	query, args, err := sq.Select(columns...).
		From(interactionTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. InteractionGet. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. InteractionGet. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := interactionGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. InteractionGet. err: %v", err)
	}

	return res, nil
}

// InteractionList returns a page of interactions.
// Filter mode: either (peerType+peerTarget) OR addressSet (multi-column IN) OR since (unfiltered, time-scoped).
// If peerType/peerTarget/addressSet are all empty AND since is zero-value → returns nil, nil immediately
// (preserves the original "no filter" behavior for existing callers).
// If peerType/peerTarget/addressSet are all empty AND since is non-zero → scopes by customer_id + tm_create >= since only.
// Pagination: cursor is a tm_create timestamp string (WHERE tm_create < token ORDER BY tm_create DESC).
// This matches the platform-wide convention used by calls, messages, emails, etc.
func (h *handler) InteractionList(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	peerType, peerTarget string,
	addressSet []AddressPair,
	since time.Time,
) ([]*interaction.Interaction, error) {
	// 1. If all filters empty and since not set → return nil, nil (unchanged legacy behavior)
	if peerType == "" && peerTarget == "" && len(addressSet) == 0 && since.IsZero() {
		return nil, nil
	}

	columns := commondatabasehandler.GetDBFields(&interaction.Interaction{})

	builder := sq.Select(columns...).
		From(interactionTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()})

	// 3. Apply peer filter OR addressSet multi-column IN
	if peerType != "" || peerTarget != "" {
		builder = builder.Where(sq.Eq{"peer_type": peerType, "peer_target": peerTarget})
	} else if len(addressSet) > 0 {
		// Build: (peer_type=? AND peer_target=?) OR ...
		// Using OR expansion for SQLite compatibility (SQLite doesn't support multi-column IN).
		or := sq.Or{}
		for _, ap := range addressSet {
			or = append(or, sq.Eq{"peer_type": ap.Type, "peer_target": ap.Target})
		}
		builder = builder.Where(or)
	}

	// 3b. Unfiltered mode: scope by tm_create lower bound (customer_id is already applied above).
	if !since.IsZero() {
		builder = builder.Where(sq.GtOrEq{"tm_create": since.UTC()})
	}

	// 4. Apply cursor if token != "" (simple timestamp cursor, platform standard)
	if token != "" {
		cursorTime, err := time.Parse("2006-01-02T15:04:05.000000Z", token)
		if err != nil {
			// Fallback to RFC3339Nano for forward compatibility.
			cursorTime, err = time.Parse(time.RFC3339Nano, token)
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse page token. InteractionList. err: %v", err)
		}
		builder = builder.Where(sq.Lt{"tm_create": cursorTime.UTC()})
	}

	// 5. ORDER BY tm_create DESC
	// Return up to size rows. Caller passes size+1 to probe for hasMore.
	builder = builder.
		OrderBy("tm_create DESC").
		Limit(size)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. InteractionList. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. InteractionList. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*interaction.Interaction
	for rows.Next() {
		item, scanErr := interactionGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. InteractionList. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. InteractionList. err: %v", err)
	}

	// Return up to size rows. Caller passes size+1 to probe for hasMore.
	// The caller (buildPagedResult in contacthandler) owns the trim+token logic.
	return res, nil
}

// InteractionListByIDs fetches interactions for a given set of IDs,
// scoped to customerID (tenant guard).
func (h *handler) InteractionListByIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) ([]*interaction.Interaction, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	columns := commondatabasehandler.GetDBFields(&interaction.Interaction{})

	idBytes := make([]interface{}, len(ids))
	for i, id := range ids {
		b := id.Bytes()
		idBytes[i] = b
	}

	query, args, err := sq.Select(columns...).
		From(interactionTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"id": idBytes}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. InteractionListByIDs. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. InteractionListByIDs. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*interaction.Interaction
	for rows.Next() {
		item, scanErr := interactionGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. InteractionListByIDs. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. InteractionListByIDs. err: %v", err)
	}

	return res, nil
}

// InteractionListUnresolved returns interactions that have zero-contact attribution:
//   - NOT auto-matched to any active contact_address for this customer
//   - NOT positively resolved to any contact
//   - peer_type != 'web_session'
//
// Supports pagination (size + token) and a since lower bound on tm_create.
func (h *handler) InteractionListUnresolved(ctx context.Context, customerID uuid.UUID, size uint64, token string, since time.Time) ([]*interaction.Interaction, error) {
	columns := commondatabasehandler.GetDBFields(&interaction.Interaction{})
	cols := make([]string, len(columns))
	for i, c := range columns {
		cols[i] = "i." + c
	}

	// Build the unresolved predicate using NOT EXISTS subqueries.
	// Outer query scans contact_interactions (i) for the customer since the given time.
	baseSQL := fmt.Sprintf(`
SELECT %s
FROM contact_interactions i
WHERE i.customer_id = ?
  AND i.peer_type != 'web_session'
  AND i.tm_create >= ?
  AND NOT EXISTS (
      SELECT 1 FROM contact_addresses a
      WHERE a.customer_id = i.customer_id
        AND a.type = i.peer_type
        AND a.target = i.peer_target
  )
  AND NOT EXISTS (
      SELECT 1 FROM contact_resolutions r
      WHERE r.customer_id = i.customer_id
        AND r.interaction_id = i.id
        AND r.resolution_type = 'positive'
        AND r.tm_delete IS NULL
  )`,
		strings.Join(cols, ", "),
	)

	args := []interface{}{customerID.Bytes(), since}

	// Cursor pagination.
	if token != "" {
		cursorTime, err := time.Parse("2006-01-02T15:04:05.000000Z", token)
		if err != nil {
			cursorTime, err = time.Parse(time.RFC3339Nano, token)
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse page token. InteractionListUnresolved. err: %v", err)
		}
		baseSQL += " AND i.tm_create < ?"
		args = append(args, cursorTime.UTC())
	}

	baseSQL += " ORDER BY i.tm_create DESC LIMIT ?"
	args = append(args, size)

	rows, err := h.db.Query(baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. InteractionListUnresolved. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*interaction.Interaction
	for rows.Next() {
		item, scanErr := interactionGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. InteractionListUnresolved. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. InteractionListUnresolved. err: %v", err)
	}

	return res, nil
}
