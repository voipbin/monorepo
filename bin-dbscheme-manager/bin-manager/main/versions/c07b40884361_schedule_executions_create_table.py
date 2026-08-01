"""schedule_executions_create_table

Revision ID: c07b40884361
Revises: 93d8469841bc
Create Date: 2026-08-01 14:47:50.317336

Creates the schedule_executions table for bin-schedule-manager
(VOIP-1283). See docs/plans/2026-08-01-bin-schedule-manager-design.md
§4.2.

Audit trail / dead-letter record: one row per dispatch run. The column
is named trigger_type, not trigger — TRIGGER is a MySQL reserved word /
SQLite keyword, and the shared data path
(GetDBFields/PrepareFields/squirrel) emits identifiers unquoted.

result is MEDIUMTEXT because number-renew returns the full
renewed-number list, which can exceed TEXT's 65,535-byte cap; the
dispatcher truncates stored payloads to 60,000 bytes (audit aid, not a
parseable contract).

uq_schedule_executions_schedule_scheduled is a hard DB-level backstop
against double-fire: two claimants of the same (schedule_id,
trigger_type, tm_scheduled) slot cannot both INSERT.

Retention hard-DELETEs rows past SCHEDULE_EXECUTION_RETENTION_DAYS via
idx_schedule_executions_tm_create; tm_delete exists solely for
convention/tooling uniformity and is never set by the service.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c07b40884361'
down_revision = '93d8469841bc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE schedule_executions (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,   -- copied from schedule
            schedule_id BINARY(16) NOT NULL,

            trigger_type VARCHAR(16) NOT NULL,   -- 'cron' / 'manual'
            status       VARCHAR(32) NOT NULL,   -- 'running' / 'success' / 'failed' / 'abandoned'

            status_code INT,          -- RPC response status code (0 if transport error)
            error       TEXT,         -- error string on failure/abandonment
            result      MEDIUMTEXT,   -- RPC response body on success, truncated to 60,000 bytes

            attempt_count INT NOT NULL DEFAULT 0,   -- 1 + retries actually used
            duration_ms   INT,                      -- total wall time of the run

            tm_scheduled DATETIME(6) NOT NULL,   -- the tm_next_run this row consumed (idempotency key part)
            tm_deadline  DATETIME(6) NOT NULL,   -- reap deadline, computed in Go at INSERT

            tm_start DATETIME(6),
            tm_end   DATETIME(6),

            tm_create DATETIME(6),
            tm_update DATETIME(6),
            tm_delete DATETIME(6),   -- convention only; never set by the service

            PRIMARY KEY (id),

            INDEX idx_schedule_executions_schedule (schedule_id, tm_create),   -- per-schedule history
            INDEX idx_schedule_executions_reap (status, tm_deadline),          -- per-tick reap
            INDEX idx_schedule_executions_tm_create (tm_create),               -- age-based retention DELETE

            UNIQUE KEY uq_schedule_executions_schedule_scheduled (schedule_id, trigger_type, tm_scheduled)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS schedule_executions;""")
