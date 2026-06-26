"""contact_crm_create_tables

Revision ID: ac5d4e18060c
Revises: 4579f211877c
Create Date: 2026-06-26 17:58:15.407482

Creates the 3 new tables for the contact-axis unified interaction timeline
(lightweight CRM). See docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md
(VOIP-1204). This is implementation step 1 (VOIP-1206), the prerequisite for
M1 address migration (VOIP-1207), projection handlers (VOIP-1208), and the
read API (VOIP-1209).

Tables:
  1. contact_addresses     - permanent identifier mapping (merges
                             contact_phone_numbers + contact_emails). Hard-delete,
                             no tm_delete, mirrors agent_addresses (2026-06-21).
  2. contact_interactions  - immutable append-only fact log. No tm_delete in v1
                             (append-only; deletion added later via expand-contract).
  3. contact_resolutions   - manual attribution (positive/negative). Append-only
                             with tm_delete retraction (NULL = active).

is_primary uniqueness (§3.1): "one primary per CONTACT" (not per type) is enforced
by a STORED generated column `primary_contact_uk = IF(is_primary=1, contact_id, NULL)`
with a UNIQUE index on (customer_id, primary_contact_uk). Non-primary rows store
NULL, which is distinct under MySQL UNIQUE, so only one is_primary=true row is
allowed per (customer_id, contact_id). A CHECK (is_primary=0 OR contact_id IS NOT
NULL) forbids an unresolved (contact_id NULL) address from being primary, which
would otherwise compute primary_contact_uk=NULL and escape the UNIQUE. This is a
NEW pattern for VoIPBin; the MariaDB-build -> mysqldump -> MySQL-8.0-import round
trip (generated column + UNIQUE + CHECK) is verified at implementation (VOIP-1206).

NOTE on cross-engine data dumps: contact_addresses has a STORED generated column.
A `mysqldump` of POPULATED data from MariaDB into MySQL 8.0 must use
`--complete-insert` (explicit column list), otherwise MySQL 8.0 rejects the
generated column value with ERROR 3105. The build pipeline dumps an EMPTY schema,
so this does not affect the build; it matters only for future data migrations /
backups crossing the MariaDB->MySQL boundary.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ac5d4e18060c'
down_revision = '4579f211877c'
branch_labels = None
depends_on = None


def upgrade():
    # ------------------------------------------------------------------
    # contact_addresses - permanent identifier mapping (hard-delete)
    # ------------------------------------------------------------------
    op.execute("""
        CREATE TABLE contact_addresses (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,
            contact_id  BINARY(16) DEFAULT NULL,   -- NULL = unresolved

            -- address (commonaddress.Address): target is normalized (join key)
            type        VARCHAR(255) NOT NULL DEFAULT '',
            target      VARCHAR(255) NOT NULL DEFAULT '',
            target_name VARCHAR(255) NOT NULL DEFAULT '',
            is_primary  TINYINT(1)   NOT NULL DEFAULT 0,

            -- "one primary per contact" enforcement: only is_primary=1 rows carry
            -- a value; non-primary rows are NULL (distinct under UNIQUE).
            primary_contact_uk BINARY(16)
                GENERATED ALWAYS AS (IF(is_primary = 1, contact_id, NULL)) STORED,

            -- hard-delete (no tm_delete), mirrors agent_addresses
            tm_create   DATETIME(6),
            tm_update   DATETIME(6),

            PRIMARY KEY (id),
            INDEX        idx_contact_addresses_contact (contact_id),
            UNIQUE INDEX idx_contact_addresses_identifier (customer_id, type, target),
            UNIQUE INDEX idx_contact_addresses_primary (customer_id, primary_contact_uk),

            -- an unresolved address (contact_id NULL) must not be primary, else
            -- primary_contact_uk computes to NULL and escapes the UNIQUE above.
            CONSTRAINT chk_contact_addresses_primary_resolved
                CHECK (is_primary = 0 OR contact_id IS NOT NULL)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)

    # ------------------------------------------------------------------
    # contact_interactions - immutable append-only fact log
    # ------------------------------------------------------------------
    op.execute("""
        CREATE TABLE contact_interactions (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,

            direction   VARCHAR(255) NOT NULL DEFAULT '',   -- inbound / outbound

            -- raw remote endpoint as the event carried it (match key). Identity
            -- is computed at read time against contact_addresses, never stored.
            peer_type   VARCHAR(255) NOT NULL DEFAULT '',
            peer_target VARCHAR(255) NOT NULL DEFAULT '',   -- normalized

            -- origin channel discriminator + origin record id (state/body fetched
            -- at read time via (reference_type, reference_id)). reference_id is
            -- NOT NULL: it is half the idempotency key, and MySQL treats NULL as
            -- distinct under UNIQUE, so a nullable reference_id would defeat
            -- dedup. Every projected channel carries an origin id (call_id,
            -- message_id, conversation_message_id, aicall_id).
            reference_type VARCHAR(255) NOT NULL DEFAULT '',
            reference_id   BINARY(16)   NOT NULL,

            tm_interaction DATETIME(6),   -- origin event time (display sort / sessionize)
            tm_create      DATETIME(6),   -- projection insert time (pagination cursor)

            PRIMARY KEY (id),
            -- idempotency (at-least-once); the bare triple distinguishes SMS fan-out.
            UNIQUE INDEX idx_contact_interactions_idem (reference_type, reference_id, peer_target),
            -- automatic peer-match lookup (read-time IN-match)
            INDEX        idx_contact_interactions_peer (customer_id, peer_type, peer_target),
            -- pagination cursor (tm_create DESC + id tie-breaker)
            INDEX        idx_contact_interactions_cursor (customer_id, tm_create)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)

    # ------------------------------------------------------------------
    # contact_resolutions - manual attribution (append-only, tm_delete retraction)
    # ------------------------------------------------------------------
    op.execute("""
        CREATE TABLE contact_resolutions (
            id             BINARY(16) NOT NULL,
            customer_id    BINARY(16) NOT NULL,
            contact_id     BINARY(16) NOT NULL,   -- the contact this is attributed to
            interaction_id BINARY(16) NOT NULL,   -- single-row grain

            resolution_type  VARCHAR(255) NOT NULL DEFAULT '',  -- positive / negative
            resolved_by_type VARCHAR(255) NOT NULL DEFAULT '',  -- agent / system / rule
            resolved_by_id   BINARY(16)   DEFAULT NULL,          -- nil for system

            tm_create DATETIME(6),
            tm_update DATETIME(6),
            tm_delete DATETIME(6),   -- soft-delete = attribution retraction (NULL = active)

            PRIMARY KEY (id),
            INDEX idx_contact_resolutions_contact (customer_id, contact_id, tm_delete),
            INDEX idx_contact_resolutions_interaction (customer_id, interaction_id, tm_delete)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS contact_resolutions;""")
    op.execute("""DROP TABLE IF EXISTS contact_interactions;""")
    op.execute("""DROP TABLE IF EXISTS contact_addresses;""")
