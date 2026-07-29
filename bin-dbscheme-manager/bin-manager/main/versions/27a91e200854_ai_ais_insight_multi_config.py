"""ai_ais_insight_multi_config

Revision ID: 27a91e200854
Revises: f4c4c2407cee
Create Date: 2026-07-29 17:30:23.649099

Allows a customer to keep multiple non-deleted type='insight' AI records
while exactly one of them may be marked active.

SQUARE-23 (f4c4c2407cee) enforced "at most one non-deleted type='insight'
AI per customer" by keying active_insight_key on customer_id whenever
type='insight' AND tm_delete IS NULL. That blocked the legitimate
workflow of preparing a second Insight AI configuration before switching
production traffic to it.

This migration adds an explicit is_insight_active flag and redefines
active_insight_key to only carry customer_id when that flag is TRUE, so
any number of inactive insight configs may coexist while the unique
index still guarantees a single active one per customer.

is_insight_active is BOOLEAN NOT NULL DEFAULT FALSE (not nullable): the
Go side (models/ai/main.go) declares it as a plain bool, which cannot
represent NULL and fails the row scan outright if one is ever returned.
NOT NULL DEFAULT FALSE is what makes that plain-bool mapping safe, and a
tri-state NULL/false split would be unrepresentable there anyway. The
generated column's own type='insight' guard is what scopes the semantics.

This migration deliberately chains on f4c4c2407cee rather than amending
it in place. Alembic keys on revision ID, not file content, so any
database that already ran f4c4c2407cee would silently skip an amended
upgrade() and permanently miss is_insight_active. The cost is a second
ai_ais table rewrite (ALTER TABLE ... ADD COLUMN ... STORED rewrites the
table in place under a lock); confirm ai_ais's row count and the expected
rewrite duration are acceptable for the maintenance window before
applying this to any non-local environment.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '27a91e200854'
down_revision = 'f4c4c2407cee'
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

    if not _column_exists(conn, 'ai_ais', 'is_insight_active'):
        op.execute("""
            ALTER TABLE ai_ais
            ADD COLUMN is_insight_active BOOLEAN NOT NULL DEFAULT FALSE
        """)

    # A generated column's expression cannot be altered in place, so
    # SQUARE-23's index and column are dropped and rebuilt below.
    if _index_exists(conn, 'ai_ais', 'uq_ai_active_insight_key'):
        op.execute("""
            DROP INDEX uq_ai_active_insight_key ON ai_ais
        """)

    if _column_exists(conn, 'ai_ais', 'active_insight_key'):
        op.execute("""
            ALTER TABLE ai_ais
            DROP COLUMN active_insight_key
        """)

    if not _column_exists(conn, 'ai_ais', 'active_insight_key'):
        op.execute("""
            ALTER TABLE ai_ais
            ADD COLUMN active_insight_key BINARY(16) GENERATED ALWAYS AS (
                IF(type = 'insight' AND tm_delete IS NULL AND is_insight_active = TRUE,
                   customer_id, NULL)
            ) STORED
        """)

    if not _index_exists(conn, 'ai_ais', 'uq_ai_active_insight_key'):
        op.execute("""
            CREATE UNIQUE INDEX uq_ai_active_insight_key
            ON ai_ais(active_insight_key)
        """)


def downgrade():
    """Restore SQUARE-23's "at most one non-deleted insight AI per customer".

    WARNING: this is a one-way door in practice. Once any customer has 2+
    non-deleted type='insight' AI records -- exactly what upgrade() above
    exists to permit -- the CREATE UNIQUE INDEX below fails with error
    1062 (duplicate entry), leaving ai_ais without its unique index and
    without active_insight_key. Do NOT run this blindly against a
    populated table. Audit first, and remove extras via the
    DELETE /ais/{id} API, never raw SQL:

        SELECT customer_id, COUNT(*) FROM ai_ais
        WHERE type = 'insight' AND tm_delete IS NULL
        GROUP BY customer_id HAVING COUNT(*) > 1;
    """
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

    if _column_exists(conn, 'ai_ais', 'is_insight_active'):
        op.execute("""
            ALTER TABLE ai_ais
            DROP COLUMN is_insight_active
        """)
