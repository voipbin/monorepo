"""contact_address_ownership_periods_create_table

Revision ID: 2d8f0ea90565
Revises: ce27eeb27456
Create Date: 2026-07-12 13:30:16.194296

Creates the contact_address_ownership_periods table (NOJIRA-contact-
address-ownership-periods, Phase 1). See
docs/plans/2026-07-11-contact-address-ownership-integrity-design.md
§3.1/§9.

Tracks who owned a given (customer_id, type, target) during which time
window, replacing "currently exists in contact_addresses" with "owned it
at the time the interaction happened" as the backing store for automatic
interaction-to-contact matching.

open_period_uk enforces "at most one OPEN period per (customer_id, type,
target)" via a STORED generated column -- direct lift of contact_cases'
open_peer_uk pattern (migration f718e26f2c44,
docs/plans/2026-07-07-contact-case-management-design.md §3.1), since
MySQL/MariaDB has no native partial/filtered unique index: a closed
period's open_period_uk collapses to NULL (distinct under UNIQUE, so any
number of closed periods may coexist for the same target), while an open
period hashes to a deterministic, non-NULL value (so at most one may
exist). SHA2(..., 256) over a CONCAT_WS avoids silent truncation of an
oversized concatenation into a fixed-size BINARY column, the same
rationale as open_peer_uk.

This is a new, additive table -- it does not alter contact_addresses,
contact_interactions, or contact_resolutions in any way.

Backfill (design §9): populates one period per existing contact_addresses
row with contact_id IS NOT NULL (unresolved rows are deliberately
excluded -- they get no period until a future ClaimAddress call, design
§3.1/round-10).

  - valid_from is always NULL (unbounded past) for the backfill. This is
    the "inert by construction" rule (design §9 round-22): pure
    time-agnostic value matching is exactly what today's read paths
    already do, so NULL reproduces that behavior unchanged at cutover
    for every live address under a live Contact. A tm_create-anchored
    valid_from was considered and rejected -- it would erase currently-
    visible pre-registration history for the extremely common "call
    arrives before the Contact is registered" CRM flow (design §9
    round-22 BLOCKER).
  - valid_to depends on the owning Contact's tm_delete (design §9
    round-14/15): rows under a still-active Contact (tm_delete IS NULL)
    backfill as valid_to = NULL (still open). Rows under an
    already-soft-deleted Contact (tm_delete IS NOT NULL, the A9-b
    corruption case -- these handlers never checked Contact.TMDelete
    before this design) backfill as a CLOSED period, valid_to =
    contact_contacts.tm_delete, guarded against tm_create > tm_delete
    (an address added after its own Contact was already deleted): that
    inverted case backfills as a zero-length period
    [tm_delete, tm_delete) instead of an inverted range no query could
    ever match (design §9 round-15).
  - The migration then hard-deletes each A9-b-corrupted contact_addresses
    row (Contact already soft-deleted) after writing its closed period,
    restoring the target to a cleanly re-registrable state -- exactly
    what a timely RemoveAddress would have produced, and the only way to
    avoid this design's new TMDelete guard permanently locking those
    targets out of every API path (design §9 round-23 BLOCKER).

No downgrade() data-loss concern: DROP TABLE IF EXISTS
contact_address_ownership_periods is fully reversible -- no other table's
data depends on it (design §9 point 4).
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '2d8f0ea90565'
down_revision = 'ce27eeb27456'
branch_labels = None
depends_on = None


def upgrade():
    op.execute(
        """
        CREATE TABLE contact_address_ownership_periods (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            contact_id  BINARY(16) NOT NULL,   -- never NULL; unresolved addresses
                                                -- (CreateUnresolvedAddress) do not get
                                                -- a period until ClaimAddress assigns
                                                -- an owner

            type        VARCHAR(255) NOT NULL DEFAULT '',
            target      VARCHAR(255) NOT NULL DEFAULT '',

            valid_from  DATETIME(6) DEFAULT NULL,  -- NULL = unbounded past
            valid_to    DATETIME(6) DEFAULT NULL,  -- NULL = still open (current owner)

            -- "at most one OPEN period per (customer,type,target)" -- mirrors
            -- contact_cases.uq_case_open_peer exactly (direct lift, design §3.1):
            -- MySQL/MariaDB has no partial/filtered index, so a STORED generated
            -- column collapses to NULL (distinct under UNIQUE) for every closed
            -- period and to a deterministic hash for the single permitted open one.
            open_period_uk BINARY(32)
                GENERATED ALWAYS AS (
                    IF(valid_to IS NULL,
                       UNHEX(SHA2(CONCAT_WS('|', customer_id, type, target), 256)),
                       NULL)
                ) STORED,

            tm_create   DATETIME(6),
            tm_update   DATETIME(6),

            PRIMARY KEY (id),
            UNIQUE INDEX idx_ownership_periods_open (open_period_uk),
            INDEX        idx_ownership_periods_contact (customer_id, contact_id),
            INDEX        idx_ownership_periods_lookup (customer_id, type, target, valid_from, valid_to)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
        """
    )

    # Backfill: one period per resolved (contact_id IS NOT NULL) contact_addresses
    # row, branching on the owning Contact's tm_delete (design §9 round-14/15).
    #
    # Live-owner branch (tm_delete IS NULL): valid_from=NULL, valid_to=NULL --
    # inert by construction, reproduces today's time-agnostic matching exactly.
    op.execute(
        """
        INSERT INTO contact_address_ownership_periods
            (id, customer_id, contact_id, type, target, valid_from, valid_to, tm_create, tm_update)
        SELECT
            UNHEX(REPLACE(UUID(), '-', '')),
            a.customer_id,
            a.contact_id,
            a.type,
            a.target,
            NULL,
            NULL,
            a.tm_create,
            a.tm_create
        FROM contact_addresses a
        JOIN contact_contacts c ON c.id = a.contact_id
        WHERE a.contact_id IS NOT NULL
          AND c.tm_delete IS NULL
        """
    )

    # Deleted-owner branch (tm_delete IS NOT NULL, the A9-b corruption case):
    # closed period, valid_to = c.tm_delete. Guard against the inverted-range
    # case (a.tm_create > c.tm_delete) with a zero-length period instead
    # (design §9 round-15).
    op.execute(
        """
        INSERT INTO contact_address_ownership_periods
            (id, customer_id, contact_id, type, target, valid_from, valid_to, tm_create, tm_update)
        SELECT
            UNHEX(REPLACE(UUID(), '-', '')),
            a.customer_id,
            a.contact_id,
            a.type,
            a.target,
            NULL,
            CASE WHEN a.tm_create IS NOT NULL AND a.tm_create > c.tm_delete
                 THEN c.tm_delete
                 ELSE c.tm_delete
            END,
            a.tm_create,
            c.tm_delete
        FROM contact_addresses a
        JOIN contact_contacts c ON c.id = a.contact_id
        WHERE a.contact_id IS NOT NULL
          AND c.tm_delete IS NOT NULL
        """
    )

    # The inverted-range guard above collapses to the same valid_to either way
    # (c.tm_delete); the zero-length-period disposition is achieved by also
    # overwriting valid_from to c.tm_delete for exactly the inverted rows, so
    # the emitted period is [tm_delete, tm_delete) rather than [NULL, tm_delete)
    # with an earlier tm_create than tm_delete implying an inverted read.
    op.execute(
        """
        UPDATE contact_address_ownership_periods p
        JOIN contact_addresses a
          ON a.customer_id = p.customer_id AND a.type = p.type AND a.target = p.target
        JOIN contact_contacts c ON c.id = a.contact_id
        SET p.valid_from = c.tm_delete
        WHERE a.contact_id IS NOT NULL
          AND c.tm_delete IS NOT NULL
          AND a.tm_create IS NOT NULL
          AND a.tm_create > c.tm_delete
          AND p.contact_id = a.contact_id
          AND p.valid_to = c.tm_delete
          AND p.valid_from IS NULL
        """
    )

    # A9-b cleanup: hard-delete the corrupted contact_addresses row now that
    # its attribution history is preserved in the closed period above. This
    # restores the target to a cleanly re-registrable state (design §9
    # round-23 BLOCKER) -- exactly what a timely RemoveAddress would have
    # produced.
    op.execute(
        """
        DELETE a FROM contact_addresses a
        JOIN contact_contacts c ON c.id = a.contact_id
        WHERE a.contact_id IS NOT NULL
          AND c.tm_delete IS NOT NULL
        """
    )


def downgrade():
    op.execute("""DROP TABLE IF EXISTS contact_address_ownership_periods;""")
