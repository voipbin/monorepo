"""queue_queues_drop_execute

Revision ID: 9cc9fded2d17
Revises: 7c24b27a8c77
Create Date: 2026-09-23 12:08:57.022550

Event-driven routing redesign (VOIP-1539) step 5 (final, destructive): drop
the queue_queues.execute column.

This is the SECOND of two migrations for the scheduler-removal redesign. The
FIRST (7c24b27a8c77, additive-only) shipped the reservation/groupcall columns
that the new CAS-routing code needs and deployed BEFORE the new binaries.
This one runs LAST, only after:
  1. The Go `queue.Execute` struct field, `ExecuteRun`/`ExecuteStop` consts,
     and every scheduler code path were removed (PR #1330, merged) -- so no
     binary's reflection-based SELECT (`GetDBFields(&queue.Queue{})`,
     bin-common-handler/pkg/databasehandler/mapping.go) still requests this
     column. Removing the struct field before dropping the column is the
     mandatory ordering (a reflection-driven SELECT column list has no
     compile-time signal that a struct field went unused).
  2. The scheduler-less queue-manager (no `/v1/queues/{id}/execute[_run]`
     RPC, no `execute` ticker) is deployed to every environment that runs
     this migration.
  3. The matching backstop (PR #1331, §5.2 reconcile) that replaces the old
     scheduler's safety-net role is live.

`execute` was a nullable VARCHAR(255) (added by e93a0bfa95fe, back when this
table was still named `queues`; it was later renamed to `queue_queues`) --
dropping a nullable column with no dependent index/constraint is a pure
metadata operation, not a data-shape change that needs a backfill.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9cc9fded2d17'
down_revision = '7c24b27a8c77'
branch_labels = None
depends_on = None


def _column_exists(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()
    if _column_exists(conn, 'queue_queues', 'execute'):
        op.execute("""ALTER TABLE queue_queues DROP COLUMN execute;""")


def downgrade():
    conn = op.get_bind()
    if not _column_exists(conn, 'queue_queues', 'execute'):
        op.execute("""ALTER TABLE queue_queues ADD COLUMN execute varchar(255) AFTER tag_ids;""")
