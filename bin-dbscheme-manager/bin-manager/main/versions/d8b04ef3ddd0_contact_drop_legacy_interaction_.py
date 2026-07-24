"""contact_drop_legacy_interaction_resolution_tables

Revision ID: d8b04ef3ddd0
Revises: abfdbef47552
Create Date: 2026-07-25 07:14:40.482183

Drops contact_interactions and contact_resolutions (VOIP CPO decision,
2026-07-25, per PR #1137 / design doc
bin-contact-manager/docs/plans/2026-07-25-contact-interaction-retire-to-peer-events-design.md).

PR #1137 (merged 2026-07-24, commit 036587a19) already retired the Go
code that reads/writes these two tables: contacthandler.InteractionList
now proxies bin-timeline-manager's peer_events read API instead of
querying contact_interactions, and every write path
(EventCallCreated/EventConversationMessageCreated projection handlers,
ResolutionCreate/Delete) was deleted. This migration is the follow-up
DB-level cleanup: no Go code anywhere in the monorepo references either
table after that PR (grep-confirmed: dbhandler.InteractionCreate/Get/
List/ListByOwnershipPeriods/ListByIDs/ListUnresolved and
dbhandler.ResolutionCreate/Delete/ListByInteraction/ListByContact, plus
the two OwnershipPeriodsListByContactID/MissingPeriodOwnedAddresses
read-side helpers that existed solely to support the old
InteractionList's ownership-period matching, are all removed in the
same change as this migration).

NOTE: contact_address_ownership_periods is explicitly OUT OF SCOPE and
NOT touched here -- that table's WRITE path (AddressCreateTx/
AddressUpdateTx/AddressDeleteTx/AddressClaimTx via
OwnershipPeriodsLockAndResolveTx) is still live and used by Address
CRUD independent of Interaction; only its two Interaction-only READ
helpers were removed at the Go level.

No data migration: this is a pure retirement of dead tables. Any data
still physically present is now permanently unreachable by any Go code
path (the read/write handlers that touched it no longer exist), so
there is nothing left to preserve by copying it elsewhere.

Downgrade recreates the tables at their FINAL pre-drop schema shape
(after every prior migration in this chain -- ac5d4e18060c's original
create, 8d5a344905e7/99e7e955a149's case_id/case support additions,
ce27eeb27456's case_id removal from contact_interactions,
adb8daac2bb0/b41d1b2317af's local/peer JSON + generated-column
conversion), NOT the original create-time shape. Data is NOT restored
(same policy as 1d0f4d07ff58's contact_phone_numbers/contact_emails
drop) -- a downgrade only recreates the empty table structure.

PRE-DEPLOY SAFETY CHECK (manual, before running this migration against
staging/production): run
    SELECT COUNT(*) FROM contact_interactions;
    SELECT COUNT(*) FROM contact_resolutions;
first. Any non-zero counts represent now-orphaned historical event data
with no remaining Go read path -- confirm this is acceptable to the CPO/
CEO before the DROP is applied, since upgrade() is irreversible on data.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd8b04ef3ddd0'
down_revision = 'abfdbef47552'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""DROP TABLE IF EXISTS contact_resolutions;""")
    op.execute("""DROP TABLE IF EXISTS contact_interactions;""")


def downgrade():
    # Recreate contact_interactions at its FINAL pre-drop shape (post
    # ac5d4e18060c create -> 8d5a344905e7 add case_id ->
    # ce27eeb27456 drop case_id -> adb8daac2bb0 add local_type/
    # local_target -> b41d1b2317af convert peer_type/peer_target/
    # local_type/local_target to STORED generated columns backed by new
    # peer/local JSON columns). Data is NOT restored.
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_interactions (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,

            direction   VARCHAR(255) NOT NULL DEFAULT '',

            peer  JSON NOT NULL,
            local JSON NULL,

            peer_type VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type'))) STORED NOT NULL,
            peer_target VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.target'))) STORED NOT NULL,
            local_type VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.type'))) STORED,
            local_target VARCHAR(255)
                GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.target'))) STORED,

            reference_type VARCHAR(255) NOT NULL DEFAULT '',
            reference_id   BINARY(16)   NOT NULL,

            tm_interaction DATETIME(6),
            tm_create      DATETIME(6),

            PRIMARY KEY (id),
            UNIQUE INDEX idx_contact_interactions_idem (reference_type, reference_id, peer_target),
            INDEX        idx_contact_interactions_peer (customer_id, peer_type, peer_target),
            INDEX        idx_contact_interactions_cursor (customer_id, tm_create)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)

    # Recreate contact_resolutions at its FINAL pre-drop shape (post
    # ac5d4e18060c create -> 99e7e955a149 add case_id/case_positive_uk/
    # interaction_id-now-nullable/CHECK constraint). Data is NOT restored.
    op.execute("""
        CREATE TABLE IF NOT EXISTS contact_resolutions (
            id             BINARY(16) NOT NULL,
            customer_id    BINARY(16) NOT NULL,
            contact_id     BINARY(16) NOT NULL,
            interaction_id BINARY(16) DEFAULT NULL,
            case_id        BINARY(16) DEFAULT NULL,

            resolution_type  VARCHAR(255) NOT NULL DEFAULT '',
            resolved_by_type VARCHAR(255) NOT NULL DEFAULT '',
            resolved_by_id   BINARY(16)   DEFAULT NULL,

            case_positive_uk BINARY(16) GENERATED ALWAYS AS (
                IF(resolution_type = 'positive' AND interaction_id IS NULL AND tm_delete IS NULL,
                   case_id, NULL)
            ) STORED,

            tm_create DATETIME(6),
            tm_update DATETIME(6),
            tm_delete DATETIME(6),

            PRIMARY KEY (id),
            CONSTRAINT chk_resolution_case_or_interaction
                CHECK (interaction_id IS NOT NULL OR case_id IS NOT NULL),
            UNIQUE INDEX uq_resolution_case_positive (case_positive_uk),
            INDEX idx_contact_resolutions_contact (customer_id, contact_id, tm_delete),
            INDEX idx_contact_resolutions_interaction (customer_id, interaction_id, tm_delete),
            INDEX idx_contact_resolutions_case_id (case_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)
