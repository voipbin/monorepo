"""queue_agent_event_driven_routing_add_reserve_and_groupcall

Revision ID: 7c24b27a8c77
Revises: 2ebeae4fd8b5
Create Date: 2026-09-23 03:33:44.588369

Event-driven routing redesign (VOIP-1539) step 1: additive-only schema.
- agent_agents: add reserve_reference_type / reserve_reference_id / tm_reserve
  (method B reservation columns; agent-manager reserve/release/sweep RPC uses them).
- queue_queuecalls: add groupcall_id (agent-leg groupcall persistence for
  dial-then-CAS + backstop leg hangup) and a (queue_id, status) composite index
  (waiting-FIFO listing and status-scoped scans under the new CAS routing).

Additive-only and deploy-order-first: every later step (scheduler removal, the
CAS status-transition setters, reserve RPC) assumes these columns already exist,
so this migration must ship and apply BEFORE the new binaries. The destructive
`queue_queues.execute` DROP is intentionally NOT here; it is a separate 2nd
migration that runs last (after the Execute struct field is removed and the new
scheduler-less queue-manager is deployed).

Each DDL statement is guarded independently against information_schema so a
partially-applied retry (MySQL commits DDL per statement, no rollback of an
earlier statement when a later one fails) is safe.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '7c24b27a8c77'
down_revision = '2ebeae4fd8b5'
branch_labels = None
depends_on = None

ZERO_UUID = "UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))"


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

    # --- agent_agents: method B reservation columns ---
    if not _column_exists(conn, 'agent_agents', 'reserve_reference_type'):
        op.execute("""ALTER TABLE agent_agents ADD COLUMN reserve_reference_type VARCHAR(255) AFTER direct_hash;""")
        op.execute("""UPDATE agent_agents SET reserve_reference_type = '' WHERE reserve_reference_type IS NULL;""")
    if not _column_exists(conn, 'agent_agents', 'reserve_reference_id'):
        op.execute("""ALTER TABLE agent_agents ADD COLUMN reserve_reference_id BINARY(16) AFTER reserve_reference_type;""")
        op.execute("""UPDATE agent_agents SET reserve_reference_id = %s WHERE reserve_reference_id IS NULL;""" % ZERO_UUID)
    if not _column_exists(conn, 'agent_agents', 'tm_reserve'):
        op.execute("""ALTER TABLE agent_agents ADD COLUMN tm_reserve DATETIME(6) AFTER reserve_reference_id;""")

    # --- queue_queuecalls: agent-leg groupcall persistence ---
    if not _column_exists(conn, 'queue_queuecalls', 'groupcall_id'):
        op.execute("""ALTER TABLE queue_queuecalls ADD COLUMN groupcall_id BINARY(16) AFTER confbridge_id;""")
        op.execute("""UPDATE queue_queuecalls SET groupcall_id = %s WHERE groupcall_id IS NULL;""" % ZERO_UUID)

    # --- queue_queuecalls: (queue_id, status) composite index for CAS routing scans ---
    if not _index_exists(conn, 'queue_queuecalls', 'idx_queue_queuecalls_queue_id_status'):
        op.execute("""CREATE INDEX idx_queue_queuecalls_queue_id_status ON queue_queuecalls(queue_id, status);""")


def downgrade():
    conn = op.get_bind()

    if _index_exists(conn, 'queue_queuecalls', 'idx_queue_queuecalls_queue_id_status'):
        op.execute("""DROP INDEX idx_queue_queuecalls_queue_id_status ON queue_queuecalls;""")
    if _column_exists(conn, 'queue_queuecalls', 'groupcall_id'):
        op.execute("""ALTER TABLE queue_queuecalls DROP COLUMN groupcall_id;""")

    if _column_exists(conn, 'agent_agents', 'tm_reserve'):
        op.execute("""ALTER TABLE agent_agents DROP COLUMN tm_reserve;""")
    if _column_exists(conn, 'agent_agents', 'reserve_reference_id'):
        op.execute("""ALTER TABLE agent_agents DROP COLUMN reserve_reference_id;""")
    if _column_exists(conn, 'agent_agents', 'reserve_reference_type'):
        op.execute("""ALTER TABLE agent_agents DROP COLUMN reserve_reference_type;""")
