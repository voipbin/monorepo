"""schedule_schedules_seed_campaign_flow_reconcile

Revision ID: 9f7413ad1d07
Revises: 75bf4fd0682a
Create Date: 2026-09-02 16:32:50.293038

Seeds the "campaign-flow-reconcile" schedule for bin-schedule-manager
(VOIP-1444). See
docs/plans/2026-09-02-voip-1444-orphaned-flow-reconciliation-design.md
and the paired -plan.md's "Tasks -- bin-dbscheme-manager" / task 8.

Every column both precedent migrations set is set explicitly here,
never left to a DB default -- most critically timeout_ms, which is
NOT NULL with no DEFAULT: under a strict sql_mode an INSERT omitting
it fails loudly at apply time; under a relaxed sql_mode it can
silently insert 0, which requesthandler.SendRequest turns into an
already-expired context.WithTimeout on every dispatch, forever.

Primary model: 0c037bf0a362 (schedule_schedules_seed_phase2_ticker_
jobs) -- the closer precedent for this ticket's cross-service
dispatch shape (bin-campaign-manager's own /v1/... route) than
a5e6f559299c's self-RPC housekeeping jobs. Both precedents happen to
set an identical column list.

target_data carries recent_interval_sec via JSON_OBJECT('recent_
interval_sec', 21600) -- a bound SQL parameter, not a raw JSON string
literal like '{"recent_interval_sec": 21600}': op.execute() passes
raw SQL through sqlalchemy.text(), which would parse the literal's
bare colon as a bind-parameter marker (the VOIP-1283 hotfix pitfall,
documented in 0c037bf0a362). recent_interval_sec is seeded here
alongside cron (both derive from the same 6-hour interval decision)
so the *initial* values can only drift apart by missing a line in
this same diff. This protects the initial seed only -- it does NOT
protect against later drift: a post-seed edit to cron (via
schedule-control or PUT /v1/schedules/{id}, including the
RecentSaturated "shorten the interval" remedy documented in
docs/operations.md) does not automatically update
target_data.recent_interval_sec, since the two are edited through
entirely different operational paths once the schedule exists. Any
future change to the cron interval must update both in the same
operational change and re-verify reconcilePassTimeout (see below)
against the new interval -- see bin-campaign-manager/docs/
operations.md's "Changing the schedule's cron interval" section.

This row ships enabled=0, unlike every row in the 0c037bf0a362
precedent (which all ship enabled=1). This is deliberate, not an
oversight: the schedule would otherwise dispatch into a route that
does not exist yet the moment this migration is applied, ahead of
the bin-campaign-manager deploy carrying it (the "schedule seeded
before the route exists" ordering hazard). The rollout sequence
(apply migration -> deploy the image -> manual
POST /v1/schedules/{id}/execute smoke test, polled to a terminal
`success` execution -> `schedule-control schedule enable
campaign-flow-reconcile`) is documented in the PR description, not
performed by this migration or by this AI session -- per this
repo's Alembic AI-authorship boundary, this session creates and
commits the migration file only, and never runs `alembic upgrade` /
`alembic downgrade`, nor the post-deploy enable step.

timeout_ms / retry_max / cron sizing (worked example, re-verify
against real measured RPC latency before the schedule is enabled):
  - reconcilePassTimeout (pkg/campaignhandler/reconcile.go) = 90s
  - timeout_ms = 120000 (120s) -- strictly above reconcilePassTimeout,
    with 30s margin covering real message-delivery delay (not just
    processing latency; the two timeouts' clocks start at different
    points -- see bin-campaign-manager/docs/operations.md). This is
    the actual mutual-exclusion guarantee: it keeps
    bin-schedule-manager's "Forbid overlap" guard (ExecutionHasRunning,
    checked before the tm_next_run CAS on every claim path) covering
    this pass's entire physical duration.
  - retry_max = 0 -- a failed pass is naturally retried by the next
    scheduled run; no schedule-level retry needed. Also keeps the
    reap-deadline formula below simple (no in-execution retry loop).
  - cron = '0 */6 * * *' (every 6 hours) -- reconcilePassTimeout (90s)
    is well below this interval (cadence/liveness requirement, not a
    concurrency guard: keeps the schedule from chronically observing
    dispatch-level skipped_overlap skips). timeout_ms + 60s (180s,
    the reap deadline at retry_max=0, bounding how long a
    bin-schedule-manager dispatch goroutine itself crashing --
    "abandon-not-drain" -- can leave this schedule's execution row
    stuck running) is also comfortably below the 6-hour interval.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9f7413ad1d07'
down_revision = '75bf4fd0682a'
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
            'campaign-flow-reconcile',
            'Delete orphaned flows left behind by campaigns deleted within the last 7 days',
            'rpc',
            '0 */6 * * *',
            'bin-manager.campaign-manager.request',
            '/v1/campaigns/flows/reconcile',
            'POST',
            'application/json',
            JSON_OBJECT('recent_interval_sec', 21600),
            120000,
            0,
            0,
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
        WHERE name = 'campaign-flow-reconcile'
          AND customer_id = UNHEX('00000000000000000000000000000000');
    """)
