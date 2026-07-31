"""scheduler_schedules_create_table

Revision ID: 02c54988d77f
Revises: 27a91e200854
Create Date: 2026-08-01 07:49:39.541766

Creates the scheduler_schedules table for bin-scheduler-manager
(VOIP-1283). See docs/plans/2026-08-01-bin-scheduler-manager-design.md
§4.1.

A schedule row is a cron-driven RPC dispatch definition: cron expression
(UTC), target RabbitMQ queue/uri/method/payload, timeout, and retry
budget. customer_id is the nil UUID for platform housekeeping jobs;
multi-tenant customer schedules reuse the same table (Phase 3).

Deliberately NO unique key on (customer_id, name): active-name
uniqueness is enforced in schedulehandler over tm_delete IS NULL rows —
a DB unique index would either block name reuse after soft delete or
(with tm_delete in the key) not constrain active rows at all, since
MySQL permits multiple NULLs in unique indexes.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '02c54988d77f'
down_revision = '27a91e200854'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE scheduler_schedules (
            id          BINARY(16) NOT NULL,
            customer_id BINARY(16) NOT NULL,   -- nil UUID = platform job

            name   VARCHAR(255) NOT NULL,   -- stable cross-environment handle for seeds/doctor
            detail VARCHAR(1024),

            type VARCHAR(32)  NOT NULL,   -- 'rpc' (Phase 1: only value; enum lives in Go)
            cron VARCHAR(128) NOT NULL,   -- 5-field cron expression, evaluated in UTC

            target_queue     VARCHAR(255)  NOT NULL,   -- e.g. bin-manager.number-manager.request
            target_uri       VARCHAR(1024) NOT NULL,   -- e.g. /v1/numbers/renew
            target_method    VARCHAR(16)   NOT NULL,   -- POST/GET/PUT/DELETE
            target_data_type VARCHAR(255),             -- application/json
            target_data      JSON,                     -- request payload

            timeout_ms INT NOT NULL,             -- RPC timeout for the dispatch
            retry_max  INT NOT NULL DEFAULT 0,   -- in-run immediate retries on failure

            enabled TINYINT(1) NOT NULL DEFAULT 1,

            tm_next_run DATETIME(6),   -- NULL = compute on next scan
            tm_last_run DATETIME(6),   -- last successful claim time (fire time, not completion)

            tm_create DATETIME(6),
            tm_update DATETIME(6),
            tm_delete DATETIME(6),     -- NULL = active

            PRIMARY KEY (id),

            INDEX idx_scheduler_schedules_customer_name (customer_id, name),   -- non-unique, lookup only
            INDEX idx_scheduler_schedules_next_run (enabled, tm_next_run)      -- due-scan
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS scheduler_schedules;""")
