# Service-owned migrations

A service that owns its tables applies its schema migrations itself at startup (golang-migrate, SQL files embedded with `go:embed`) instead of through Alembic. The shared code lives in `bin-common-handler/pkg/migrationhandler`; `bin-tag-manager` is the reference service. Tested engines: MariaDB 12.3 and MySQL 8.0, so migration files avoid engine specific syntax. The tables of each service are listed under "Service-owned tables" below; `bin-dbscheme-manager` manages only the `asterisk` database.

## What a service holds

- `migrations/embed.go`: `const VersionTable = "<prefix>_schema_migrations"` and `//go:embed *.sql` `var FS embed.FS`
- `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits from 000001 without gaps; the first pair is `000001_baseline`)
- `cmd/<name>-manager/main.go`, right after the database connection and before the cache and the message broker:
  `migrationhandler.Run(cfg.DatabaseDSN, migrations.FS, migrations.VersionTable)`
- `cmd/<name>-control/main.go`: `migrationhandler.NewMigrateCommand("migrate", open)`, where `open` returns `migrationhandler.New(dsn, migrations.FS, migrations.VersionTable)`. The command (`status`, `up`, `down`, `force <version>`) opens only the database, so it works while the cache or the broker are down. A service whose control binary already has a `migrate` command registers it under another name.
- `migrations/migrations_test.go`: `migrationcheck.Check(FS, migrationcheck.Options{TableOptions: ..., ColumnCharsets: ...})` and a test that no non-.sql file sits next to the embedded files. For a migration directory below the service root pass `fs.Sub(FS, "<dir>")`.

## Rules for the files

- One DDL statement per file. Only the baseline may hold several `CREATE TABLE IF NOT EXISTS` statements.
- Every up has a down. A down that cannot restore data starts with `-- DOWN-LOSSY: <reason>`; an up with a destructive operation starts with `-- DESTRUCTIVE: <reason>` and its PR body starts with `[DESTRUCTIVE]`.
- Merged files are immutable and numbers are never reused. Fix a mistake with a new number.
- Expand/contract: the old code must work on the new schema. Drop a column in a later PR than the one that stops using it; replace instead of rename or retype.
- The baseline down is the no-op `SELECT 1`.
- Do not compare the result of a string function (`JSON_TYPE`, `LOWER`, `CONCAT`, ...) with a literal in a `CHECK` or default. The result carries the system collation and the literal the database default, so the statement fails with Error 1267 on a database whose default differs (MariaDB 12.x resolves `CHARACTER SET utf8` to `utf8mb3_uca1400_ai_ci`). Cast both sides to binary: `CAST(JSON_TYPE(c) AS BINARY) <> CAST('NULL' AS BINARY)`. `scripts/verify-service-migrations.sh` (V13) applies the files to a database created with the engine default collation, for every service; CI runs the same check on the changed services: the job `service-migration-collation` (`.circleci/scripts/check-service-migration-default-collation.sh`, triggered by a change under `bin-*/migrations/`, MariaDB 12.3 only, up, down, up on a database created with `CHARACTER SET utf8` and no collation); `bin-email-manager/migrations` additionally has a static test for the function calls.

## History

The schema of these tables was first created by an Alembic history that no longer exists in this repository. Each service baseline (`000001_baseline`) is the schema of the last head of that history.

## Baseline text rules

`migrationcheck` is a deny list for the baseline text. The items marked (convention) are not checked by it:

- The baseline uses `CREATE TABLE IF NOT EXISTS <name> (` with an unquoted name (strip the backticks and the database prefix of `SHOW CREATE TABLE` output); the words of the deny list are rejected even as unquoted column names (quote such a column with backticks). `ENGINE=InnoDB` is dropped (database default).
- Table options are only these, written as one of the five strings of `migrationcheck` (`OptionUTF8MB4`, `OptionUTF8MB4UnicodeCI`, `OptionUTF8MB4UnicodeCIDynamic` (with `ROW_FORMAT=DYNAMIC`), `OptionUTF8MB4Collate0900AI` (`COLLATE=utf8mb4_0900_ai_ci` written out), `OptionUTF8MB3`). The service test lists the table names that use one; a table that is not listed must have no table option. A collation that exists only in MariaDB (`*_uca1400_*`) is not used.
- A column level `CHARACTER SET` or `COLLATE` is rejected unless the service declares the column in `Options.ColumnCharsets` (`"<table>.<column>"` mapped to `ColumnCharsetUTF8MB4GeneralCI`, the only allowed string); the column definition must then contain exactly that string outside quotes and parentheses, and a declaration for a table or column the baseline does not have is an error. Any text outside the deny list that changes a collation (for example the `BINARY` column attribute) is not caught.
- (convention) JSON columns are written as `json`; expression defaults are written as `DEFAULT (expr)`. Generation expressions are not taken from `SHOW CREATE TABLE` (MariaDB and MySQL print them differently). Foreign keys without a name stay without a name (MariaDB names them `1`, `2`, which MySQL rejects); `CONSTRAINT <name> FOREIGN KEY` and `CONSTRAINT <name> CHECK (...)` are accepted where a baseline has to keep the names of existing tables.
- Seed rows are not part of the baseline; a seed is its own file with one statement.
- (convention) A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci`, like the install scripts do.

## Checks

- `scripts/check-service-migrations.sh <service-dir> [<migrations-subdir>]` (after `git fetch origin main`): merged `.sql` files are not modified, deleted, renamed or changed in mode; the added files are regular files `NNNNNN_<description>.(up|down).sql` in pairs whose numbers continue origin/main without gaps (from 000001 when origin/main has none there). `--self-test` runs it on synthetic branches. A changed `VersionTable` is not detected.
- `scripts/verify-service-migrations.sh <service-dir> [<migrations-subdir>]` starts scratch containers of both engines and checks, on an empty database: the extended normalized dump after up, down, up is unchanged; up, down, up; concurrent start; dirty refusal and force; a database ahead of the image; a restricted account; a password with special characters; the daemon applies the migrations before it connects the cache; a database created with the engine default collation (V13); the structure test. `HELPER_TESTS=1` also runs the `migrationhandler` integration tests; run it for every change of the helper. Output with SKIP lines or with `ENGINES_OVERRIDE` is not evidence for the default engines.
  Variables for services that differ: `CONTROL_BIN`, `SERVICE_BIN`, `VERSION_TABLE`, `MIGRATE_CMD`, `DSN_VAR`, `DAEMON_ENV`, `DAEMON_BLOCKS` (see the header of the script). A service without a cache keeps retrying the broker after it migrated and never exits by itself: add `DAEMON_BLOCKS=1` so the daemon start check accepts a daemon that the timeout stops, provided it migrated first.
- `scripts/verify-service-migrations.sh --self-test` exercises the dump queries (generated column, comments, foreign key rules, check expressions) on a synthetic schema.

## Impact analysis for a migration PR

Every PR that changes a migration file or the migration runner states, with evidence, for each point: destructive operations one by one (DROP of any object, TRUNCATE, data changing UPDATE/DELETE/INSERT SELECT, type narrowing or change, default removal or change, rename, NOT NULL/UNIQUE/FK/CHECK on existing data, enum value removal, NOT NULL column without default); compatibility of the old code with the new schema and of the new code with the old schema (grep results); lock and duration with the table size source; rollback (the helper skips a database version that is ahead of the image, so an older image still starts; a rollback of a baseline is the previous image); data loss. A PR with a destructive or ambiguous operation starts its body with `[DESTRUCTIVE]`. Attach the output of the two engine verification.

## Service-owned tables

"Owns" means the service is the authoritative writer for that table: it creates, updates, and deletes rows. Other services may read but should not write to tables they do not own. All tables below are in the `bin_manager` MySQL database. Every application table of that database is owned by a service that keeps its own `migrations/` directory (golang-migrate, SQL embedded with `go:embed`) and applies it at startup, under the rules of this document. Bookkeeping tables (each service's `<prefix>_schema_migrations`) are not listed. The `users` table that the former Alembic history created is not used by any service and is not created by any service baseline. Four owned tables (`contact_activities`, `contact_lists`, `contact_list_members`, `queue_queuecallreferences`) are kept in the baselines but are not referenced by current code. The `asterisk` database is managed by `bin-dbscheme-manager` (Alembic, `asterisk_config/`) and is described in its README.

| service | tables | schema source |
|---|---|---|
| bin-agent-manager | agent_addresses, agent_agents | `bin-agent-manager/migrations/` |
| bin-ai-manager | ai_ai_audits, ai_ai_prompt_histories, ai_ai_prompt_proposals, ai_aicall_participants, ai_aicalls, ai_ais, ai_config_histories, ai_configs, ai_mcp_oauth_states, ai_mcp_servers, ai_messages, ai_summaries, ai_teams | `bin-ai-manager/migrations/` |
| bin-billing-manager | billing_accounts, billing_billings, billing_failed_events | `bin-billing-manager/migrations/` |
| bin-call-manager | call_bridges, call_calls, call_channels, call_confbridges, call_groupcalls, call_outbound_configs, call_recordings | `bin-call-manager/migrations/` |
| bin-campaign-manager | campaign_campaigncalls, campaign_campaigns, campaign_outplans | `bin-campaign-manager/migrations/` |
| bin-conference-manager | conference_conferencecalls, conference_conferences | `bin-conference-manager/migrations/` |
| bin-contact-manager | contact_activities, contact_address_ownership_periods, contact_addresses, contact_case_notes, contact_cases, contact_contacts, contact_list_members, contact_lists, contact_stars, contact_tag_assignments | `bin-contact-manager/migrations/` |
| bin-conversation-manager | conversation_accounts, conversation_conversations, conversation_medias, conversation_messages | `bin-conversation-manager/migrations/` |
| bin-customer-manager | customer_accesskeys, customer_customers | `bin-customer-manager/migrations/` |
| bin-direct-manager | direct_directs | `bin-direct-manager/migrations/` |
| bin-email-manager | email_config_histories, email_configs, email_emails | `bin-email-manager/migrations/` |
| bin-flow-manager | flow_activeflows, flow_flows | `bin-flow-manager/migrations/` |
| bin-message-manager | message_config_histories, message_configs, message_messages | `bin-message-manager/migrations/` |
| bin-number-manager | number_numbers, number_providers | `bin-number-manager/migrations/` |
| bin-outdial-manager | outdial_outdials, outdial_outdialtargetcalls, outdial_outdialtargets | `bin-outdial-manager/migrations/` |
| bin-pipecat-manager | pipecat_pipecatcalls | `bin-pipecat-manager/migrations/` |
| bin-push-manager | push_devices | `bin-push-manager/migrations/` |
| bin-queue-manager | queue_queuecallreferences, queue_queuecalls, queue_queues | `bin-queue-manager/migrations/` |
| bin-registrar-manager | registrar_customer_domains, registrar_extensions, registrar_sip_auths, registrar_trunks | `bin-registrar-manager/migrations/` |
| bin-route-manager | route_providercalls, route_providers, route_routes | `bin-route-manager/migrations/` |
| bin-schedule-manager | schedule_executions, schedule_schedules | `bin-schedule-manager/migrations/` |
| bin-storage-manager | storage_accounts, storage_files | `bin-storage-manager/migrations/` |
| bin-tag-manager | tag_tags | `bin-tag-manager/migrations/` |
| bin-talk-manager | talk_chats, talk_mentions, talk_messages, talk_participants, talk_pins | `bin-talk-manager/migrations/` |
| bin-timeline-manager | timeline_analyses | `bin-timeline-manager/migrations/mysql/` |
| bin-transcribe-manager | transcribe_transcribes, transcribe_transcripts | `bin-transcribe-manager/migrations/` |
| bin-tts-manager | tts_manager_speaking | `bin-tts-manager/migrations/` |
| bin-webchat-manager | webchat_messages, webchat_sessions, webchat_widgets | `bin-webchat-manager/migrations/` |

This list is derived from the `CREATE TABLE` statements in each service's `migrations/` directory (`bin-timeline-manager` keeps its MySQL migrations under `migrations/mysql/` next to its ClickHouse migrations; `bin-rag-manager` uses PostgreSQL and is not part of `bin_manager`). A PR that adds or drops a table updates this document.

### Table ownership

| table_name | owning_service | notes |
|---|---|---|
| agent_addresses | bin-agent-manager | Agent contact addresses |
| agent_agents | bin-agent-manager | Agent roster and state |
| ai_ai_audits | bin-ai-manager | On-demand evaluation of an AI in one call |
| ai_ai_prompt_histories | bin-ai-manager | Prompt version history of AI configurations |
| ai_ai_prompt_proposals | bin-ai-manager | Proposed prompt changes for AI configurations |
| ai_aicall_participants | bin-ai-manager | Participants of AI call sessions |
| ai_aicalls | bin-ai-manager | Active AI call sessions |
| ai_ais | bin-ai-manager | AI configuration records |
| ai_config_histories | bin-ai-manager | Change history of the platform-wide stored settings (no values) |
| ai_configs | bin-ai-manager | Platform-wide stored settings (platform LLM API keys, encrypted) |
| ai_mcp_oauth_states | bin-ai-manager | OAuth state for MCP server authorization |
| ai_mcp_servers | bin-ai-manager | MCP server registrations used by AI configurations |
| ai_messages | bin-ai-manager | AI conversation messages |
| ai_summaries | bin-ai-manager | AI-generated call summaries |
| ai_teams | bin-ai-manager | AI team groupings |
| billing_accounts | bin-billing-manager | Billing account balances |
| billing_billings | bin-billing-manager | Individual billing events |
| billing_failed_events | bin-billing-manager | Retry queue for failed billing events |
| call_bridges | bin-call-manager | Bridges joining call channels |
| call_calls | bin-call-manager | Active and historical calls |
| call_channels | bin-call-manager | Call channel legs |
| call_confbridges | bin-call-manager | Conference bridge legs |
| call_groupcalls | bin-call-manager | Group call sessions |
| call_outbound_configs | bin-call-manager | Outbound call configuration |
| call_recordings | bin-call-manager | Call recording metadata |
| campaign_campaigncalls | bin-campaign-manager | Calls originated by campaigns |
| campaign_campaigns | bin-campaign-manager | Campaign definitions |
| campaign_outplans | bin-campaign-manager | Campaign outbound dial plans |
| conference_conferencecalls | bin-conference-manager | Per-participant conference legs |
| conference_conferences | bin-conference-manager | Conference room definitions |
| contact_activities | bin-contact-manager | Legacy, not referenced by current code |
| contact_address_ownership_periods | bin-contact-manager | Ownership periods of contact addresses |
| contact_addresses | bin-contact-manager | Unified contact identifier mapping (CRM, merges phone/email) |
| contact_case_notes | bin-contact-manager | Notes attached to contact cases |
| contact_cases | bin-contact-manager | Per-channel Case session header grouping related Interactions (CRM) |
| contact_contacts | bin-contact-manager | Contact book entries |
| contact_list_members | bin-contact-manager | Legacy, not referenced by current code |
| contact_lists | bin-contact-manager | Legacy, not referenced by current code |
| contact_stars | bin-contact-manager | Contacts starred by an agent (private to the agent) |
| contact_tag_assignments | bin-contact-manager | Tag-to-contact associations |
| conversation_accounts | bin-conversation-manager | Messaging channel accounts |
| conversation_conversations | bin-conversation-manager | Conversation threads |
| conversation_medias | bin-conversation-manager | Media attachments in conversations |
| conversation_messages | bin-conversation-manager | Individual conversation messages |
| customer_accesskeys | bin-customer-manager | API access key credentials |
| customer_customers | bin-customer-manager | Customer account records |
| direct_directs | bin-direct-manager | Direct SIP trunk definitions |
| email_config_histories | bin-email-manager | Change history of the stored settings (never a value) |
| email_configs | bin-email-manager | Platform-wide stored settings (provider credentials, encrypted) |
| email_emails | bin-email-manager | Outbound email records |
| flow_activeflows | bin-flow-manager | Currently executing flow instances |
| flow_flows | bin-flow-manager | Flow definitions |
| message_config_histories | bin-message-manager | Change history of the stored settings (no values) |
| message_configs | bin-message-manager | Stored settings (the SMS and MMS provider tokens, encrypted) |
| message_messages | bin-message-manager | Outbound SMS/messaging records |
| number_numbers | bin-number-manager | Purchased DID numbers |
| number_providers | bin-number-manager | Number provider settings (priority, enabled flag, encrypted secret) |
| outdial_outdials | bin-outdial-manager | Outbound dialer sessions |
| outdial_outdialtargetcalls | bin-outdial-manager | Calls placed to outdialtargets |
| outdial_outdialtargets | bin-outdial-manager | Individual dial targets |
| pipecat_pipecatcalls | bin-pipecat-manager | Pipecat AI voice sessions |
| push_devices | bin-push-manager | Agent device push tokens (FCM) |
| queue_queuecallreferences | bin-queue-manager | Legacy, not referenced by current code |
| queue_queuecalls | bin-queue-manager | Calls waiting in or processed by queues |
| queue_queues | bin-queue-manager | Queue definitions |
| registrar_customer_domains | bin-registrar-manager | Customer SIP domains |
| registrar_extensions | bin-registrar-manager | SIP extension registrations |
| registrar_sip_auths | bin-registrar-manager | SIP authentication credentials |
| registrar_trunks | bin-registrar-manager | SIP trunk registrations |
| route_providercalls | bin-route-manager | Per-provider call tracking |
| route_providers | bin-route-manager | SIP provider definitions |
| route_routes | bin-route-manager | Outbound call routing rules |
| schedule_executions | bin-schedule-manager | Execution records of scheduled jobs |
| schedule_schedules | bin-schedule-manager | Scheduled job definitions (platform jobs are seeded when the table is empty) |
| storage_accounts | bin-storage-manager | Object storage bucket accounts |
| storage_files | bin-storage-manager | Stored file metadata |
| tag_tags | bin-tag-manager | Tag definitions |
| talk_chats | bin-talk-manager | Talk (agent UI) chat sessions |
| talk_mentions | bin-talk-manager | Agent mention index of talk messages |
| talk_messages | bin-talk-manager | Messages in talk chats |
| talk_participants | bin-talk-manager | Participants in talk sessions |
| talk_pins | bin-talk-manager | Chats and messages pinned by an agent |
| timeline_analyses | bin-timeline-manager | Timeline analysis records |
| transcribe_transcribes | bin-transcribe-manager | Transcription job records |
| transcribe_transcripts | bin-transcribe-manager | STT transcription results (linked to transcribes) |
| tts_manager_speaking | bin-tts-manager | Text-to-speech speaking sessions |
| webchat_messages | bin-webchat-manager | Individual inbound/outbound webchat messages |
| webchat_sessions | bin-webchat-manager | Visitor conversation sessions per widget |
| webchat_widgets | bin-webchat-manager | Website chat widget configuration (flow_id, theme, direct-access hash) |
