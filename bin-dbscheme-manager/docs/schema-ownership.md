# Schema Ownership

This document maps each database table to the service that owns it. "Owns" means the service is the authoritative writer for that table — it creates, updates, and deletes rows. Other services may read but should not write to tables they do not own.

All tables below are in the `bin_manager` MySQL database. For Asterisk PJSIP tables (in the `asterisk` database), see the bottom section.

## Service-Owned Schemas

Every application table in the `bin_manager` database (except the unused legacy `users` table, see below) is owned by a service that keeps its own `migrations/` directory (golang-migrate, SQL embedded with `go:embed`) and applies it at startup. See [docs/workflows/service-owned-migrations.md](../../docs/workflows/service-owned-migrations.md) for the rules. Each baseline (`000001_baseline`) creates the schema of the Alembic head and differs from it only in what the service accepts through `E1_ALLOWED_DIFF` (collation, foreign key names, CHECK constraint names, column order), used where a baseline keeps the names or collation of existing deployments. Bookkeeping tables (`alembic_version` and each service's `<prefix>_schema_migrations`) are not listed.

The `bin-manager` Alembic stream is frozen read-only history (last head `b0307529af4a`), used for install catch-up and the verify E1 reference: do not add new migrations to it (`asterisk_config` is not frozen). The directory stays because the install scripts still run this chain to catch up existing installations and older pins, `scripts/verify-service-migrations.sh` builds its E1 (Alembic head) reference from it, and `tests/` uses it as real revisions. The migration hooks and scripts that check it behave as before. The `users` table that the stream created is not used by any service and is not created by any service baseline. Four owned tables (`contact_activities`, `contact_lists`, `contact_list_members`, `queue_queuecallreferences`) are kept in the baselines but are not referenced by current code.

| service | tables | schema source |
|---|---|---|
| bin-agent-manager | agent_addresses, agent_agents | `bin-agent-manager/migrations/` |
| bin-ai-manager | ai_ai_audits, ai_ai_prompt_histories, ai_ai_prompt_proposals, ai_aicall_participants, ai_aicalls, ai_ais, ai_mcp_oauth_states, ai_mcp_servers, ai_messages, ai_summaries, ai_teams | `bin-ai-manager/migrations/` |
| bin-billing-manager | billing_accounts, billing_billings, billing_failed_events | `bin-billing-manager/migrations/` |
| bin-call-manager | call_bridges, call_calls, call_channels, call_confbridges, call_groupcalls, call_outbound_configs, call_recordings | `bin-call-manager/migrations/` |
| bin-campaign-manager | campaign_campaigncalls, campaign_campaigns, campaign_outplans | `bin-campaign-manager/migrations/` |
| bin-conference-manager | conference_conferencecalls, conference_conferences | `bin-conference-manager/migrations/` |
| bin-contact-manager | contact_activities, contact_address_ownership_periods, contact_addresses, contact_case_notes, contact_cases, contact_contacts, contact_list_members, contact_lists, contact_tag_assignments | `bin-contact-manager/migrations/` |
| bin-conversation-manager | conversation_accounts, conversation_conversations, conversation_medias, conversation_messages | `bin-conversation-manager/migrations/` |
| bin-customer-manager | customer_accesskeys, customer_customers | `bin-customer-manager/migrations/` |
| bin-direct-manager | direct_directs | `bin-direct-manager/migrations/` |
| bin-email-manager | email_emails | `bin-email-manager/migrations/` |
| bin-flow-manager | flow_activeflows, flow_flows | `bin-flow-manager/migrations/` |
| bin-message-manager | message_messages | `bin-message-manager/migrations/` |
| bin-number-manager | number_numbers, number_providers | `bin-number-manager/migrations/` |
| bin-outdial-manager | outdial_outdials, outdial_outdialtargetcalls, outdial_outdialtargets | `bin-outdial-manager/migrations/` |
| bin-pipecat-manager | pipecat_pipecatcalls | `bin-pipecat-manager/migrations/` |
| bin-queue-manager | queue_queuecallreferences, queue_queuecalls, queue_queues | `bin-queue-manager/migrations/` |
| bin-registrar-manager | registrar_customer_domains, registrar_extensions, registrar_sip_auths, registrar_trunks | `bin-registrar-manager/migrations/` |
| bin-route-manager | route_providercalls, route_providers, route_routes | `bin-route-manager/migrations/` |
| bin-schedule-manager | schedule_executions, schedule_schedules | `bin-schedule-manager/migrations/` |
| bin-storage-manager | storage_accounts, storage_files | `bin-storage-manager/migrations/` |
| bin-tag-manager | tag_tags | `bin-tag-manager/migrations/` |
| bin-talk-manager | talk_chats, talk_messages, talk_participants | `bin-talk-manager/migrations/` |
| bin-timeline-manager | timeline_analyses | `bin-timeline-manager/migrations/mysql/` |
| bin-transcribe-manager | transcribe_transcribes, transcribe_transcripts | `bin-transcribe-manager/migrations/` |
| bin-tts-manager | tts_manager_speaking | `bin-tts-manager/migrations/` |
| bin-webchat-manager | webchat_messages, webchat_sessions, webchat_widgets | `bin-webchat-manager/migrations/` |

This list is derived from the `CREATE TABLE` statements in each service's `migrations/` directory (`bin-timeline-manager` keeps its MySQL migrations under `migrations/mysql/` next to its ClickHouse migrations; `bin-rag-manager` uses PostgreSQL and is not part of `bin_manager`). A PR that adds or drops a table updates this document.

## Table Ownership

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
| contact_tag_assignments | bin-contact-manager | Tag-to-contact associations |
| conversation_accounts | bin-conversation-manager | Messaging channel accounts |
| conversation_conversations | bin-conversation-manager | Conversation threads |
| conversation_medias | bin-conversation-manager | Media attachments in conversations |
| conversation_messages | bin-conversation-manager | Individual conversation messages |
| customer_accesskeys | bin-customer-manager | API access key credentials |
| customer_customers | bin-customer-manager | Customer account records |
| direct_directs | bin-direct-manager | Direct SIP trunk definitions |
| email_emails | bin-email-manager | Outbound email records |
| flow_activeflows | bin-flow-manager | Currently executing flow instances |
| flow_flows | bin-flow-manager | Flow definitions |
| message_messages | bin-message-manager | Outbound SMS/messaging records |
| number_numbers | bin-number-manager | Purchased DID numbers |
| number_providers | bin-number-manager | Number provider settings (priority, enabled flag, encrypted secret) |
| outdial_outdials | bin-outdial-manager | Outbound dialer sessions |
| outdial_outdialtargetcalls | bin-outdial-manager | Calls placed to outdialtargets |
| outdial_outdialtargets | bin-outdial-manager | Individual dial targets |
| pipecat_pipecatcalls | bin-pipecat-manager | Pipecat AI voice sessions |
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
| talk_messages | bin-talk-manager | Messages in talk chats |
| talk_participants | bin-talk-manager | Participants in talk sessions |
| timeline_analyses | bin-timeline-manager | Timeline analysis records |
| transcribe_transcribes | bin-transcribe-manager | Transcription job records |
| transcribe_transcripts | bin-transcribe-manager | STT transcription results (linked to transcribes) |
| tts_manager_speaking | bin-tts-manager | Text-to-speech speaking sessions |
| webchat_messages | bin-webchat-manager | Individual inbound/outbound webchat messages |
| webchat_sessions | bin-webchat-manager | Visitor conversation sessions per widget |
| webchat_widgets | bin-webchat-manager | Website chat widget configuration (flow_id, theme, direct-access hash) |

### Asterisk Database Tables (`asterisk` database)

These tables are managed by `asterisk_config/` and owned by the Asterisk PBX configuration subsystem. They are written by Kamailio/Asterisk at runtime, not by Go services.

| table_name | owning_service | notes |
|---|---|---|
| ps_endpoints | asterisk-config | PJSIP endpoint definitions |
| ps_auths | asterisk-config | PJSIP authentication records |
| ps_aors | asterisk-config | PJSIP address-of-record bindings |
| ps_contacts | asterisk-config | PJSIP contact registrations |
| ps_domain_aliases | asterisk-config | PJSIP domain alias mappings |
| ps_globals | asterisk-config | PJSIP global settings |
| ps_identify | asterisk-config | PJSIP endpoint identification rules |
| ps_transports | asterisk-config | PJSIP transport configurations |

> The Asterisk schema is sourced from `asterisk/contrib/ast-db-manage`. When upgrading Asterisk, copy the new version files from the Asterisk repo into `asterisk_config/config/versions/`.
