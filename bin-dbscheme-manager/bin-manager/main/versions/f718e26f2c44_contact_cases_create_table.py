"""contact_cases_create_table

Revision ID: f718e26f2c44
Revises: 7e981e3aa8e9
Create Date: 2026-07-07 09:22:07.278837

Creates the Case entity table (VOIP-1228). See
docs/plans/2026-07-07-contact-case-management-design.md §3.1.

Case is a thin, per-channel session header that groups related
contact_interactions rows into a start/end unit agents can pick up,
work, and close. Get-or-create keyed by
(customer_id, peer_type, peer_target, reference_type).

open_peer_uk enforces "at most one OPEN case per (customer, peer,
reference_type)" via a STORED generated column, since MySQL has no
native partial/filtered unique index. SHA2(..., 256) is used instead
of a raw CONCAT of the key fields to avoid silent truncation of an
oversized value into a fixed-size BINARY column, which could create
false-positive uniqueness collisions between genuinely different
tuples -- see §3.1 for the full rationale (round-2 design review).
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f718e26f2c44'
down_revision = '7e981e3aa8e9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE contact_cases (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,

            peer_type     VARCHAR(255) NOT NULL DEFAULT '',
            peer_target   VARCHAR(255) NOT NULL DEFAULT '',   -- normalized, matches
                                                                -- contact_addresses.target
                                                                -- and contact_interactions.peer_target
            reference_type VARCHAR(255) NOT NULL DEFAULT '',  -- reuses contact_interactions.reference_type's
                                                                -- EXISTING vocabulary ("call",
                                                                -- "conversation_message", ...) -- NOT
                                                                -- bin-conversation-manager's separate
                                                                -- message.ReferenceType enum.

            contact_id BINARY(16) DEFAULT NULL,   -- nullable; denormalized cache, derived
                                                    -- from contact_resolutions (§3.4)

            owner_type VARCHAR(255) DEFAULT NULL,
            owner_id   BINARY(16)   DEFAULT NULL,   -- commonidentity.Owner pattern, reused
                                                       -- as-is from assignable-conversation-design.md

            status         VARCHAR(32) NOT NULL DEFAULT 'open',   -- open | closed
            opened_at      DATETIME(6) DEFAULT NULL,
            closed_at      DATETIME(6) DEFAULT NULL,
            closed_reason  VARCHAR(32) DEFAULT NULL,   -- agent_closed | timeout | merged (reserved)
            closed_by_type VARCHAR(32) DEFAULT NULL,   -- agent | system
            closed_by_id   BINARY(16)  DEFAULT NULL,

            previous_case_id BINARY(16) DEFAULT NULL,   -- re-contact chain

            tm_create DATETIME(6),
            tm_update DATETIME(6),

            PRIMARY KEY (id),

            -- open_peer_uk carries a value only when status='open'; closed rows compute
            -- NULL, which MySQL treats as distinct under UNIQUE, so any number of closed
            -- rows for the same peer/reference_type may coexist, while at most one open
            -- row may exist. CONCAT_WS (not CONCAT) so a NULL component (defensive; none
            -- of these fields are nullable today) produces a distinguishable hash input
            -- rather than collapsing the whole expression to NULL.
            open_peer_uk BINARY(32) GENERATED ALWAYS AS (
                IF(status = 'open',
                   UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)),
                   NULL)
            ) STORED,

            UNIQUE INDEX uq_case_open_peer (open_peer_uk),

            -- Supporting indexes for the hot-path queries this design specifies:
            INDEX idx_case_unresolved (customer_id, status, contact_id),   -- backs CaseListUnresolved (§6)
            INDEX idx_case_owner      (customer_id, owner_type, owner_id), -- backs "my cases" list (§7)
            INDEX idx_case_customer_reftype (customer_id, reference_type)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS contact_cases;""")
