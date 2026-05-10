# Schema Ownership

This document maps each database table to the service that owns it. "Owns" means the service is the authoritative writer for that table — it creates, updates, and deletes rows. Other services may read but should not write to tables they do not own.

All tables below are in the `voipbin` MySQL database managed by `bin-manager/`. For Asterisk PJSIP tables (in the `asterisk` database), see the bottom section.

## Table Ownership

| table_name | owning_service | notes |
|---|---|---|
| agent_agents | bin-agent-manager | Agent roster and state |
| ai_ais | bin-ai-manager | AI configuration records |
| ai_aicalls | bin-ai-manager | Active AI call sessions |
| ai_messages | bin-ai-manager | AI conversation messages |
| ai_summaries | bin-ai-manager | AI-generated call summaries |
| ai_teams | bin-ai-manager | AI team groupings |
| billing_accounts | bin-billing-manager | Billing account balances |
| billing_billings | bin-billing-manager | Individual billing events |
| billing_failed_events | bin-billing-manager | Retry queue for failed billing events |
| billing_allowances | bin-billing-manager | Per-account usage allowances |
| call_calls | bin-call-manager | Active and historical calls |
| call_confbridges | bin-call-manager | Conference bridge legs |
| call_groupcalls | bin-call-manager | Group call sessions |
| call_recordings | bin-call-manager | Call recording metadata |
| call_outbound_configs | bin-call-manager | Outbound call configuration |
| campaign_campaigns | bin-campaign-manager | Campaign definitions |
| campaign_campaigncalls | bin-campaign-manager | Calls originated by campaigns |
| campaign_outplans | bin-campaign-manager | Campaign outbound dial plans |
| conference_conferences | bin-conference-manager | Conference room definitions |
| conference_conferencecalls | bin-conference-manager | Per-participant conference legs |
| contact_contacts | bin-contact-manager | Contact book entries |
| contact_emails | bin-contact-manager | Email addresses for contacts |
| contact_phone_numbers | bin-contact-manager | Phone numbers for contacts |
| contact_tag_assignments | bin-contact-manager | Tag-to-contact associations |
| conversation_accounts | bin-conversation-manager | Messaging channel accounts |
| conversation_conversations | bin-conversation-manager | Conversation threads |
| conversation_medias | bin-conversation-manager | Media attachments in conversations |
| conversation_messages | bin-conversation-manager | Individual conversation messages |
| customer_customers | bin-customer-manager | Customer account records |
| customer_accesskeys | bin-customer-manager | API access key credentials |
| direct_directs | bin-direct-manager | Direct SIP trunk definitions |
| email_emails | bin-email-manager | Outbound email records |
| flow_flows | bin-flow-manager | Flow definitions |
| flow_activeflows | bin-flow-manager | Currently executing flow instances |
| message_messages | bin-message-manager | Outbound SMS/messaging records |
| number_numbers | bin-number-manager | Purchased DID numbers |
| outdial_outdials | bin-outdial-manager | Outbound dialer sessions |
| outdial_outdialtargets | bin-outdial-manager | Individual dial targets |
| outdial_outdialtargetcalls | bin-outdial-manager | Calls placed to outdialtargets |
| pipecat_pipecatcalls | bin-pipecat-manager | Pipecat AI voice sessions |
| queue_queues | bin-queue-manager | Queue definitions |
| queue_queuecalls | bin-queue-manager | Calls waiting in or processed by queues |
| registrar_extensions | bin-registrar-manager | SIP extension registrations |
| registrar_sip_auths | bin-registrar-manager | SIP authentication credentials |
| registrar_trunks | bin-registrar-manager | SIP trunk registrations |
| route_routes | bin-route-manager | Outbound call routing rules |
| route_providers | bin-route-manager | SIP provider definitions |
| route_providercalls | bin-route-manager | Per-provider call tracking |
| storage_accounts | bin-storage-manager | Object storage bucket accounts |
| storage_files | bin-storage-manager | Stored file metadata |
| tag_tags | bin-tag-manager | Tag definitions |
| transcripts | bin-ai-manager | STT transcription results (linked to transcribes) |
| transcribes | TBD | Transcription job records |
| talk_chats | bin-talk-manager | Talk (agent UI) chat sessions |
| talk_messages | bin-talk-manager | Messages in talk chats |
| talk_participants | bin-talk-manager | Participants in talk sessions |

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
