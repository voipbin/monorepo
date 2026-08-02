"""schedule_schedules_seed_phase2_ticker_jobs

Revision ID: 0c037bf0a362
Revises: a5e6f559299c
Create Date: 2026-08-02 09:02:26.971220

Seeds the five Phase 2 ticker-migration schedules for
bin-schedule-manager (VOIP-1284). See
docs/plans/2026-08-01-schedule-manager-phase2-tickers-design.md §7.

All five rows use the nil-UUID customer_id (platform jobs) and
UNHEX(REPLACE(UUID(),'-','')) ids per docs/conventions/database.md
§7.0a. The name column, not the id, is the stable cross-environment
handle (the dbscheme image bakes migration output into a schema dump at
build time, so the UUID() values are fixed per image build).

tm_next_run is seeded NULL = compute on next scan.

target_data uses JSON_OBJECT() (empty object; none of these jobs take a
payload) instead of a raw '{}' string literal: alembic op.execute()
passes raw SQL through sqlalchemy.text(), which treats a bare colon as
a bind-parameter marker (e.g. ':28' -> "A value is required for bind
parameter '28'"). JSON_OBJECT produces the identical JSON value with no
colon in the SQL text -- this is the VOIP-1283 hotfix pitfall.

All five ship enabled (enabled=1), matching the tickers they replace,
with retry_max=0 throughout (Phase 1's number-renew precedent -- a
failed run waits for the next natural slot rather than retrying
in-run).
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0c037bf0a362'
down_revision = 'a5e6f559299c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        INSERT INTO schedule_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'billing-monthly-topup',
            'Monthly token top-up for due billing accounts',
            'rpc',
            '0 * * * *',
            'bin-manager.billing-manager.request',
            '/v1/accounts/top_up',
            'POST',
            'application/json',
            JSON_OBJECT(),
            300000,
            0,
            1,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)
    op.execute("""
        INSERT INTO schedule_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'billing-failed-event-retry',
            'Retry billing events that failed initial processing',
            'rpc',
            '* * * * *',
            'bin-manager.billing-manager.request',
            '/v1/failed_events/retry',
            'POST',
            'application/json',
            JSON_OBJECT(),
            60000,
            0,
            1,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)
    op.execute("""
        INSERT INTO schedule_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'ai-proposal-expiry',
            'Expire completed AI prompt proposals past the drift window',
            'rpc',
            '0 * * * *',
            'bin-manager.ai-manager.request',
            '/v1/aipromptproposals/expire',
            'POST',
            'application/json',
            JSON_OBJECT(),
            300000,
            0,
            1,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)
    op.execute("""
        INSERT INTO schedule_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'customer-unverified-cleanup',
            'Soft-delete unverified customers past the grace period',
            'rpc',
            '*/15 * * * *',
            'bin-manager.customer-manager.request',
            '/v1/customers/cleanup_unverified',
            'POST',
            'application/json',
            JSON_OBJECT(),
            60000,
            0,
            1,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)
    op.execute("""
        INSERT INTO schedule_schedules (
            id, customer_id, name, detail, type, cron,
            target_queue, target_uri, target_method, target_data_type, target_data,
            timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
        ) VALUES (
            UNHEX(REPLACE(UUID(), '-', '')),
            UNHEX('00000000000000000000000000000000'),
            'customer-frozen-expiry',
            'Anonymize and soft-delete frozen customers past their deletion grace period',
            'rpc',
            '0 4 * * *',
            'bin-manager.customer-manager.request',
            '/v1/customers/cleanup_frozen_expired',
            'POST',
            'application/json',
            JSON_OBJECT(),
            600000,
            0,
            1,
            NULL,
            UTC_TIMESTAMP(6),
            UTC_TIMESTAMP(6)
        );
    """)


def downgrade():
    # No executions cleanup needed: there is no FK from
    # schedule_executions.schedule_id to schedule_schedules.id, so a
    # plain DELETE leaves at most orphaned audit rows, consistent with
    # how schedule_executions already outlives soft-deleted schedules
    # via the app-level DELETE /v1/schedules/<id> API path.
    op.execute("""
        DELETE FROM schedule_schedules
        WHERE name IN (
            'billing-monthly-topup',
            'billing-failed-event-retry',
            'ai-proposal-expiry',
            'customer-unverified-cleanup',
            'customer-frozen-expiry'
        )
          AND customer_id = UNHEX('00000000000000000000000000000000');
    """)
