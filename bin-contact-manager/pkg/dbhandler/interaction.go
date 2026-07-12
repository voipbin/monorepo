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

// OwnershipPeriodBound is a (type, target, valid_from, valid_to) window
// used by InteractionListByOwnershipPeriods to time-scope its OR-expanded
// peer match (design §6.2, round-16/17). Unlike AddressPair's pure
// equality match, each bound additionally constrains the interaction's
// effective time (COALESCE(tm_interaction, tm_create)) to fall inside
// [ValidFrom or -inf, ValidTo or +inf) for that (type, target) pair.
// ValidFrom == nil means unbounded past; ValidTo == nil means still open
// (unbounded future).
type OwnershipPeriodBound struct {
	Type      string
	Target    string
	ValidFrom *time.Time
	ValidTo   *time.Time
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

// InteractionListByOwnershipPeriods is InteractionList's sibling for
// interactionListByContact's STEP2 (design §6.2, round-18): time-scoped
// peer matching against a Contact's full ownership history instead of
// pure (type, target) equality. Kept as a separate function rather than
// widening InteractionList's signature, because InteractionList already
// ends in a variadic-incompatible `since time.Time` parameter (round-18) --
// InteractionList itself, and every existing call site
// (interaction_read.go's STEP5/interactionListByAddress,
// mock_main.go, main.go, interaction_test.go), is unchanged.
//
// Each bound in periods contributes one OR'd clause:
// (peer_type = ? AND peer_target = ?
//
//	[AND COALESCE(tm_interaction, tm_create) >= ValidFrom, if set]
//	[AND COALESCE(tm_interaction, tm_create) <  ValidTo, if set])
//
// A nil ValidFrom/ValidTo simply omits that AND-term rather than
// synthesizing a MySQL-only "+ INTERVAL 1 SECOND" tautology (the design
// prose's illustrative SQL) -- SQLite (the test harness driver) has no
// portable equivalent, and an absent AND-term is exactly "no constraint
// on that side" with no dialect-specific SQL required. The round-21
// NULL-safety fix (falling back to tm_create when tm_interaction is
// nullable) and the round-17 rule that a range condition must live
// inside one raw sq.Expr fragment, never as a value handed to a
// squirrel column-comparison builder (Eq/Lt/GtOrEq only special-case
// driver.Valuer, not Sqlizer), both still apply.
func (h *handler) InteractionListByOwnershipPeriods(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	peerType, peerTarget string,
	bounds []OwnershipPeriodBound,
	since time.Time,
) ([]*interaction.Interaction, error) {
	if peerType == "" && peerTarget == "" && len(bounds) == 0 && since.IsZero() {
		return nil, nil
	}

	columns := commondatabasehandler.GetDBFields(&interaction.Interaction{})

	builder := sq.Select(columns...).
		From(interactionTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()})

	if peerType != "" || peerTarget != "" {
		builder = builder.Where(sq.Eq{"peer_type": peerType, "peer_target": peerTarget})
	} else if len(bounds) > 0 {
		or := sq.Or{}
		for _, b := range bounds {
			// Portable across MySQL (production) and SQLite (test
			// harness): rather than the design's illustrative
			// "+ INTERVAL 1 SECOND" tautology for an unbounded upper
			// edge (MySQL-only syntax SQLite has no equivalent for),
			// each edge is simply OMITTED from the clause when nil --
			// an absent AND-term is exactly "no constraint on that
			// side," the same semantics with no dialect-specific SQL.
			clause := "(peer_type = ? AND peer_target = ?"
			args := []any{b.Type, b.Target}
			if b.ValidFrom != nil {
				clause += " AND COALESCE(tm_interaction, tm_create) >= ?"
				args = append(args, b.ValidFrom.UTC())
			}
			if b.ValidTo != nil {
				clause += " AND COALESCE(tm_interaction, tm_create) < ?"
				args = append(args, b.ValidTo.UTC())
			}
			clause += ")"
			or = append(or, sq.Expr(clause, args...))
		}
		builder = builder.Where(or)
	}

	if !since.IsZero() {
		builder = builder.Where(sq.GtOrEq{"tm_create": since.UTC()})
	}

	if token != "" {
		cursorTime, err := time.Parse("2006-01-02T15:04:05.000000Z", token)
		if err != nil {
			cursorTime, err = time.Parse(time.RFC3339Nano, token)
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse page token. InteractionListByOwnershipPeriods. err: %v", err)
		}
		builder = builder.Where(sq.Lt{"tm_create": cursorTime.UTC()})
	}

	builder = builder.
		OrderBy("tm_create DESC").
		Limit(size)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. InteractionListByOwnershipPeriods. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. InteractionListByOwnershipPeriods. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*interaction.Interaction
	for rows.Next() {
		item, scanErr := interactionGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. InteractionListByOwnershipPeriods. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. InteractionListByOwnershipPeriods. err: %v", err)
	}

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
//   - NOT auto-matched to any contact via ownership history (design §6.3:
//     rewired from live contact_addresses rows to
//     contact_address_ownership_periods, so a target's PAST owner's
//     interactions correctly stay resolved even after the target is
//     reassigned or deleted -- see design §6 for the full rationale)
//   - NOT suppressed by an unresolved (contact_id IS NULL) row's mere
//     presence (design §6.3 round-36/37/38/39: preserves today's
//     CreateUnresolvedAddress suppression-by-presence behavior exactly,
//     since unresolved rows never get a period per §4 round-10)
//   - NOT suppressed by the missing-period-skew guard (design §6.3
//     round-40/42/43: an owned row with no period of its own -- an
//     old-binary pod's AddressCreate that ran before this rewire --
//     must not resurface its interactions in the queue just because it
//     has no period row yet)
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
	//
	// Three disjuncts, each suppressing a distinct population from the
	// unresolved queue (design §6.3):
	//   1. Ownership-period match: this interaction's effective time
	//      falls inside a period some contact held this exact
	//      (type, target) during. The range check is expressed as two
	//      independently-nullable AND-terms (p.valid_from IS NULL OR ...)
	//      / (p.valid_to IS NULL OR ...) rather than the design prose's
	//      illustrative "+ INTERVAL 1 SECOND" tautology for an open
	//      upper bound -- portable across MySQL (production) and SQLite
	//      (test harness), which has no INTERVAL equivalent.
	//   2. Unresolved-row presence: a CreateUnresolvedAddress row
	//      (contact_id IS NULL) for this exact target suppresses by
	//      presence alone, time-agnostic, matching today's observable
	//      behavior (round-36).
	//   3. Missing-period-skew guard: an owned row (contact_id IS NOT
	//      NULL) for this exact target with no OPEN period for that
	//      SAME owner (round-43's final, owner-plus-open-scoped
	//      condition) suppresses by presence, exactly like an
	//      unresolved row, until the next write gives it a period.
	baseSQL := fmt.Sprintf(`
SELECT %s
FROM contact_interactions i
WHERE i.customer_id = ?
  AND i.peer_type != 'web_session'
  AND i.tm_create >= ?
  AND NOT EXISTS (
      SELECT 1 FROM contact_address_ownership_periods p
      WHERE p.customer_id = i.customer_id
        AND p.type = i.peer_type
        AND p.target = i.peer_target
        AND (p.valid_from IS NULL OR COALESCE(i.tm_interaction, i.tm_create) >= p.valid_from)
        AND (p.valid_to IS NULL   OR COALESCE(i.tm_interaction, i.tm_create) <  p.valid_to)
  )
  AND NOT EXISTS (
      SELECT 1 FROM contact_addresses a
      WHERE a.customer_id = i.customer_id
        AND a.type = i.peer_type
        AND a.target = i.peer_target
        AND a.contact_id IS NULL
  )
  AND NOT EXISTS (
      SELECT 1 FROM contact_addresses a
      WHERE a.customer_id = i.customer_id
        AND a.type = i.peer_type
        AND a.target = i.peer_target
        AND a.contact_id IS NOT NULL
        AND NOT EXISTS (
            SELECT 1 FROM contact_address_ownership_periods p2
            WHERE p2.customer_id = a.customer_id
              AND p2.type = a.type
              AND p2.target = a.target
              AND p2.contact_id = a.contact_id
              AND p2.valid_to IS NULL
        )
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
