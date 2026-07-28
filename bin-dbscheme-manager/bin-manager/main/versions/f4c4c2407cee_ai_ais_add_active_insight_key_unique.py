"""ai_ais_add_active_insight_key_unique

Revision ID: f4c4c2407cee
Revises: d8b04ef3ddd0
Create Date: 2026-07-29 04:14:32.140037

SQUARE-23: adds active_insight_key to ai_ais as a STORED generated column
to enforce "at most one active (non-deleted) type=insight AI per
customer" at the DB level, mirroring the ai_aicalls.active_reference_key
pattern (a5a40c93d3e6_ai_aicalls_add_active_reference_key_.py, VOIP-1234).

active_insight_key carries customer_id only when type='insight' AND
tm_delete IS NULL; all other rows (type='normal', or soft-deleted
type='insight' rows) compute NULL, which MySQL treats as distinct under
UNIQUE, so any number of such rows may coexist while at most one
genuinely-active Insight AI may exist per customer.

Before applying this migration to any non-local environment: confirm no
customer currently has 2+ non-deleted type='insight' AIs (see SQUARE-23
design doc docs/plans/2026-07-29-insight-ai-one-per-customer-design.md
section 3 for the audit query and cleanup approach -- extras must be
removed via the existing DELETE /ais/{id} API, not raw SQL, before this
migration's CREATE UNIQUE INDEX can succeed), and confirm ai_ais's row
count / the expected ALTER TABLE ... ADD COLUMN ... STORED rewrite
duration are acceptable for an in-place lock, per the precedent's
documented incident (see design doc section 1 "Operational cost").
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f4c4c2407cee'
down_revision = 'd8b04ef3ddd0'
branch_labels = None
depends_on = None


def _column_exists(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    return result.scalar() > 0


def _index_exists(conn, table, index):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.statistics "
        "WHERE table_schema = DATABASE() AND table_name = :table AND index_name = :idx"
    ), {'table': table, 'idx': index})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()

    if not _column_exists(conn, 'ai_ais', 'active_insight_key'):
        op.execute("""
            ALTER TABLE ai_ais
            ADD COLUMN active_insight_key BINARY(16) GENERATED ALWAYS AS (
                IF(type = 'insight' AND tm_delete IS NULL, customer_id, NULL)
            ) STORED
        """)

    if not _index_exists(conn, 'ai_ais', 'uq_ai_active_insight_key'):
        op.execute("""
            CREATE UNIQUE INDEX uq_ai_active_insight_key
            ON ai_ais(active_insight_key)
        """)


def downgrade():
    conn = op.get_bind()

    if _index_exists(conn, 'ai_ais', 'uq_ai_active_insight_key'):
        op.execute("""
            DROP INDEX uq_ai_active_insight_key ON ai_ais
        """)

    if _column_exists(conn, 'ai_ais', 'active_insight_key'):
        op.execute("""
            ALTER TABLE ai_ais
            DROP COLUMN active_insight_key
        """)
