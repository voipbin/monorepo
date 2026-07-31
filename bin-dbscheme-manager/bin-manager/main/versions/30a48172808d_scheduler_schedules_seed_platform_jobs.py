"""scheduler_schedules_seed_platform_jobs

Revision ID: 30a48172808d
Revises: bf4977e8a6a7
Create Date: 2026-08-01 07:49:40.049679

Seeds the three platform housekeeping schedules for
bin-scheduler-manager (VOIP-1283). See
docs/plans/2026-08-01-bin-scheduler-manager-design.md §8.

All three rows use the nil-UUID customer_id (platform jobs) and
UNHEX(REPLACE(UUID(),'-','')) ids per docs/conventions/database.md
§7.0a. The name column, not the id, is the stable cross-environment
handle (the dbscheme image bakes migration output into a schema dump at
build time, so the UUID() values are fixed per image build).

tm_next_run is seeded NULL = compute on next scan.

database-backup ships disabled (enabled=0): production uses managed
Cloud SQL backups; self-hosted installs enable it (VOIP-1281).
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '30a48172808d'
down_revision = 'bf4977e8a6a7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        INSERT INTO scheduler_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'number-renew',
            'Daily phone number renewal sweep',
            'rpc',
            '0 1 * * *',
            'bin-manager.number-manager.request',
            '/v1/numbers/renew',
            'POST',
            'application/json',
            '{"days":28}',
            300000,
            0,
            1,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)
    op.execute("""
        INSERT INTO scheduler_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'database-backup',
            'In-stack MySQL logical backup for self-hosted deployments',
            'rpc',
            '0 2 * * *',
            'bin-manager.scheduler-manager.request',
            '/v1/backups',
            'POST',
            'application/json',
            '{}',
            1800000,
            0,
            0,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)
    op.execute("""
        INSERT INTO scheduler_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'execution-retention',
            'Prune scheduler execution audit rows past retention',
            'rpc',
            '30 2 * * *',
            'bin-manager.scheduler-manager.request',
            '/v1/executions/prune',
            'POST',
            'application/json',
            '{}',
            600000,
            0,
            1,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)


def downgrade():
    # Executions cleanup is unnecessary: this seed migration runs before
    # Phase 1 use, so no execution rows reference these schedules yet.
    op.execute("""
        DELETE FROM scheduler_schedules
        WHERE name IN ('number-renew', 'database-backup', 'execution-retention')
          AND customer_id = UNHEX('00000000000000000000000000000000');
    """)
