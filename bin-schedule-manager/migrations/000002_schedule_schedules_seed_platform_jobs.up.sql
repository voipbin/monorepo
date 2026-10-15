-- DESTRUCTIVE: INSERT SELECT seed guarded by an empty table check
-- Seeds the nine platform schedules (nil customer) that the former Alembic stream inserted.
-- Rows are inserted only when schedule_schedules has no row at all: a database that already has
-- the table content (renamed, disabled or soft deleted seeds included) is left untouched.
INSERT INTO schedule_schedules (
  id, customer_id, name, detail, type, cron,
  target_queue, target_uri, target_method, target_data_type, target_data,
  timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update
)
SELECT
  UNHEX(REPLACE(UUID(), '-', '')), UNHEX('00000000000000000000000000000000'), s.name, s.detail, 'rpc', s.cron,
  s.target_queue, s.target_uri, 'POST', 'application/json', s.target_data,
  s.timeout_ms, 0, s.enabled, NULL, UTC_TIMESTAMP(6), UTC_TIMESTAMP(6)
FROM (
  SELECT 'number-renew' AS name, 'Daily phone number renewal sweep' AS detail, '0 1 * * *' AS cron, 'bin-manager.number-manager.request' AS target_queue, '/v1/numbers/renew' AS target_uri, JSON_OBJECT('days', 28) AS target_data, 300000 AS timeout_ms, 1 AS enabled
  UNION ALL
  SELECT 'database-backup', 'In-stack MySQL logical backup for self-hosted deployments', '0 2 * * *', 'bin-manager.schedule-manager.request', '/v1/backups', JSON_OBJECT(), 1800000, 0
  UNION ALL
  SELECT 'execution-retention', 'Prune schedule execution audit rows past retention', '30 2 * * *', 'bin-manager.schedule-manager.request', '/v1/executions/prune', JSON_OBJECT(), 600000, 1
  UNION ALL
  SELECT 'billing-monthly-topup', 'Monthly token top-up for due billing accounts', '0 * * * *', 'bin-manager.billing-manager.request', '/v1/accounts/top_up', JSON_OBJECT(), 300000, 1
  UNION ALL
  SELECT 'billing-failed-event-retry', 'Retry billing events that failed initial processing', '* * * * *', 'bin-manager.billing-manager.request', '/v1/failed_events/retry', JSON_OBJECT(), 60000, 1
  UNION ALL
  SELECT 'ai-proposal-expiry', 'Expire completed AI prompt proposals past the drift window', '0 * * * *', 'bin-manager.ai-manager.request', '/v1/aipromptproposals/expire', JSON_OBJECT(), 300000, 1
  UNION ALL
  SELECT 'customer-unverified-cleanup', 'Soft-delete unverified customers past the grace period', '*/15 * * * *', 'bin-manager.customer-manager.request', '/v1/customers/cleanup_unverified', JSON_OBJECT(), 60000, 1
  UNION ALL
  SELECT 'customer-frozen-expiry', 'Anonymize and soft-delete frozen customers past their deletion grace period', '0 4 * * *', 'bin-manager.customer-manager.request', '/v1/customers/cleanup_frozen_expired', JSON_OBJECT(), 600000, 1
  UNION ALL
  SELECT 'campaign-flow-reconcile', 'Delete orphaned flows left behind by campaigns deleted within the last 7 days', '0 */6 * * *', 'bin-manager.campaign-manager.request', '/v1/campaigns/flows/reconcile', JSON_OBJECT('recent_interval_sec', 21600), 120000, 0
) AS s
WHERE NOT EXISTS (SELECT 1 FROM schedule_schedules);
