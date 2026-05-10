# bin-api-manager Routing Table

All routes are served under the base URL `https://api.voipbin.net/v1.0`. Routes are generated from `bin-openapi-manager/openapi/openapi.yaml` via `oapi-codegen`. The generated router file is `gens/openapi_server/gen.go`.

Backend service column shows the `bin-*-manager` that handles each request via RabbitMQ RPC.

---

## Auth

Public endpoints — no authentication required.

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| POST | `/auth/boot` | bin-customer-manager | Initial bootstrap / login |
| POST | `/auth/signup` | bin-customer-manager | New account registration |
| POST | `/auth/email-verify` | bin-customer-manager | Email verification |
| POST | `/auth/password-forgot` | bin-customer-manager | Send password reset email |
| GET | `/auth/password-reset` | bin-customer-manager | Render reset form |
| POST | `/auth/password-reset` | bin-customer-manager | Submit new password |
| DELETE | `/auth/unregister` | bin-customer-manager | Delete account (also allowed while frozen) |
| POST | `/auth/unregister` | bin-customer-manager | Unregister request |

---

## Customer

`/customer` (singular) is self-service for the authenticated user. `/customers` (plural) is admin-only.

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/customer` | bin-customer-manager | Get own customer profile |
| PUT | `/customer` | bin-customer-manager | Update own profile (webhook URL, metadata) |
| PUT | `/customer/billing_account_id` | bin-customer-manager | Set billing account |
| PUT | `/customer/metadata` | bin-customer-manager | Update metadata |
| GET | `/customers` | bin-customer-manager | Admin: list all customers |
| POST | `/customers` | bin-customer-manager | Admin: create customer |
| GET | `/customers/:id` | bin-customer-manager | Admin: get customer by ID |
| PUT | `/customers/:id` | bin-customer-manager | Admin: update customer |
| DELETE | `/customers/:id` | bin-customer-manager | Admin: delete customer |
| PUT | `/customers/:id/billing_account_id` | bin-customer-manager | Admin: set billing account |
| PUT | `/customers/:id/metadata` | bin-customer-manager | Admin: update metadata |
| POST | `/customers/:id/freeze` | bin-customer-manager | Admin: freeze account |
| POST | `/customers/:id/recover` | bin-customer-manager | Admin: recover frozen account |

---

## Agents

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/agents` | bin-agent-manager | List agents |
| POST | `/agents` | bin-agent-manager | Create agent |
| GET | `/agents/:id` | bin-agent-manager | Get agent |
| PUT | `/agents/:id` | bin-agent-manager | Update agent |
| DELETE | `/agents/:id` | bin-agent-manager | Delete agent |
| PUT | `/agents/:id/addresses` | bin-agent-manager | Set SIP/contact addresses |
| POST | `/agents/:id/direct-hash-regenerate` | bin-agent-manager | Regenerate direct access hash |
| PUT | `/agents/:id/password` | bin-agent-manager | Change password |
| PUT | `/agents/:id/permission` | bin-agent-manager | Update permissions |
| PUT | `/agents/:id/status` | bin-agent-manager | Set availability status |
| PUT | `/agents/:id/tag_ids` | bin-agent-manager | Update tags |
| GET | `/me` | bin-agent-manager | Get own agent profile |

---

## Extensions

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/extensions` | bin-agent-manager | List extensions |
| POST | `/extensions` | bin-agent-manager | Create extension |
| GET | `/extensions/:id` | bin-agent-manager | Get extension |
| PUT | `/extensions/:id` | bin-agent-manager | Update extension |
| DELETE | `/extensions/:id` | bin-agent-manager | Delete extension |
| POST | `/extensions/:id/direct-hash-regenerate` | bin-agent-manager | Regenerate direct hash |

---

## Teams

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/teams` | bin-agent-manager | List teams |
| POST | `/teams` | bin-agent-manager | Create team |
| GET | `/teams/:id` | bin-agent-manager | Get team |
| PUT | `/teams/:id` | bin-agent-manager | Update team |
| DELETE | `/teams/:id` | bin-agent-manager | Delete team |
| POST | `/teams/:id/direct-hash-regenerate` | bin-agent-manager | Regenerate direct hash |

---

## Calls

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/calls` | bin-call-manager | List calls |
| POST | `/calls` | bin-call-manager | Create outbound call |
| GET | `/calls/:id` | bin-call-manager | Get call |
| DELETE | `/calls/:id` | bin-call-manager | Delete call record |
| POST | `/calls/:id/hangup` | bin-call-manager | Hang up call |
| POST | `/calls/:id/hold` | bin-call-manager | Place call on hold |
| DELETE | `/calls/:id/hold` | bin-call-manager | Remove hold |
| POST | `/calls/:id/mute` | bin-call-manager | Mute call |
| DELETE | `/calls/:id/mute` | bin-call-manager | Unmute call |
| POST | `/calls/:id/moh` | bin-call-manager | Play music on hold |
| DELETE | `/calls/:id/moh` | bin-call-manager | Stop music on hold |
| POST | `/calls/:id/silence` | bin-call-manager | Silence call |
| DELETE | `/calls/:id/silence` | bin-call-manager | Remove silence |
| POST | `/calls/:id/talk` | bin-call-manager | Inject TTS audio |
| POST | `/calls/:id/recording_start` | bin-call-manager | Start recording |
| POST | `/calls/:id/recording_stop` | bin-call-manager | Stop recording |
| GET | `/calls/:id/media_stream` | bin-call-manager | WebSocket media stream URL |

---

## Group Calls

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/groupcalls` | bin-call-manager | List group calls |
| POST | `/groupcalls` | bin-call-manager | Create group call |
| GET | `/groupcalls/:id` | bin-call-manager | Get group call |
| DELETE | `/groupcalls/:id` | bin-call-manager | Delete group call |
| POST | `/groupcalls/:id/hangup` | bin-call-manager | Hang up group call |

---

## Conferences

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/conferences` | bin-conference-manager | List conferences |
| POST | `/conferences` | bin-conference-manager | Create conference |
| GET | `/conferences/:id` | bin-conference-manager | Get conference |
| PUT | `/conferences/:id` | bin-conference-manager | Update conference |
| DELETE | `/conferences/:id` | bin-conference-manager | Delete conference |
| POST | `/conferences/:id/direct-hash-regenerate` | bin-conference-manager | Regenerate direct hash |
| POST | `/conferences/:id/recording_start` | bin-conference-manager | Start recording |
| POST | `/conferences/:id/recording_stop` | bin-conference-manager | Stop recording |
| POST | `/conferences/:id/transcribe_start` | bin-conference-manager | Start transcription |
| POST | `/conferences/:id/transcribe_stop` | bin-conference-manager | Stop transcription |
| GET | `/conferences/:id/media_stream` | bin-conference-manager | WebSocket media stream URL |
| GET | `/conferencecalls` | bin-conference-manager | List conference call participants |
| GET | `/conferencecalls/:id` | bin-conference-manager | Get conference call participant |
| DELETE | `/conferencecalls/:id` | bin-conference-manager | Remove participant |

---

## Queues

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/queues` | bin-queue-manager | List queues |
| POST | `/queues` | bin-queue-manager | Create queue |
| GET | `/queues/:id` | bin-queue-manager | Get queue |
| PUT | `/queues/:id` | bin-queue-manager | Update queue |
| DELETE | `/queues/:id` | bin-queue-manager | Delete queue |
| POST | `/queues/:id/direct-hash-regenerate` | bin-queue-manager | Regenerate direct hash |
| PUT | `/queues/:id/routing_method` | bin-queue-manager | Set routing method |
| PUT | `/queues/:id/tag_ids` | bin-queue-manager | Update tags |
| GET | `/queuecalls` | bin-queue-manager | List queued calls |
| GET | `/queuecalls/:id` | bin-queue-manager | Get queued call |
| POST | `/queuecalls/:id/kick` | bin-queue-manager | Kick call from queue |
| POST | `/queuecalls/reference_id/:id/kick` | bin-queue-manager | Kick by reference ID |

---

## Campaigns

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/campaigns` | bin-campaign-manager | List campaigns |
| POST | `/campaigns` | bin-campaign-manager | Create campaign |
| GET | `/campaigns/:id` | bin-campaign-manager | Get campaign |
| PUT | `/campaigns/:id` | bin-campaign-manager | Update campaign |
| DELETE | `/campaigns/:id` | bin-campaign-manager | Delete campaign |
| PUT | `/campaigns/:id/actions` | bin-campaign-manager | Set campaign actions |
| GET | `/campaigns/:id/campaigncalls` | bin-campaign-manager | List campaign calls |
| PUT | `/campaigns/:id/next_campaign_id` | bin-campaign-manager | Set next campaign |
| PUT | `/campaigns/:id/resource_info` | bin-campaign-manager | Update resource info |
| PUT | `/campaigns/:id/service_level` | bin-campaign-manager | Set service level |
| PUT | `/campaigns/:id/status` | bin-campaign-manager | Start/stop campaign |
| GET | `/campaigncalls` | bin-campaign-manager | List all campaign calls |
| GET | `/campaigncalls/:id` | bin-campaign-manager | Get campaign call |
| DELETE | `/campaigncalls/:id` | bin-campaign-manager | Delete campaign call |

---

## Flows

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/flows` | bin-flow-manager | List flows |
| POST | `/flows` | bin-flow-manager | Create flow |
| GET | `/flows/:id` | bin-flow-manager | Get flow |
| PUT | `/flows/:id` | bin-flow-manager | Update flow |
| DELETE | `/flows/:id` | bin-flow-manager | Delete flow |
| POST | `/flows/:id/direct-hash-regenerate` | bin-flow-manager | Regenerate direct hash |
| GET | `/activeflows` | bin-flow-manager | List active flow executions |
| POST | `/activeflows` | bin-flow-manager | Start flow execution |
| GET | `/activeflows/:id` | bin-flow-manager | Get active flow state |
| DELETE | `/activeflows/:id` | bin-flow-manager | Delete active flow record |
| POST | `/activeflows/:id/stop` | bin-flow-manager | Stop running flow |

---

## Billing

`/billing_account` (singular) = self-service. `/billing_accounts` (plural) = admin-only.

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/billing_account` | bin-billing-manager | Get own billing account |
| PUT | `/billing_account` | bin-billing-manager | Update own billing account |
| POST | `/billing_account/paddle_portal_session` | bin-billing-manager | Create Paddle billing portal session |
| PUT | `/billing_account/payment_info` | bin-billing-manager | Update payment info |
| GET | `/billing_accounts` | bin-billing-manager | Admin: list billing accounts |
| GET | `/billing_accounts/:id` | bin-billing-manager | Admin: get billing account |
| PUT | `/billing_accounts/:id` | bin-billing-manager | Admin: update billing account |
| POST | `/billing_accounts/:id/balance_add_force` | bin-billing-manager | Admin: force add balance |
| POST | `/billing_accounts/:id/balance_subtract_force` | bin-billing-manager | Admin: force subtract balance |
| PUT | `/billing_accounts/:id/payment_info` | bin-billing-manager | Admin: update payment info |
| GET | `/billings` | bin-billing-manager | List billing records |
| GET | `/billings/:billing-id` | bin-billing-manager | Get billing record |

---

## Numbers

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/numbers` | bin-number-manager | List phone numbers |
| POST | `/numbers` | bin-number-manager | Purchase phone number |
| POST | `/numbers/renew` | bin-number-manager | Renew number |
| GET | `/numbers/:id` | bin-number-manager | Get number |
| PUT | `/numbers/:id` | bin-number-manager | Update number |
| DELETE | `/numbers/:id` | bin-number-manager | Release number |
| PUT | `/numbers/:id/flow_ids` | bin-number-manager | Assign flows to number |
| PUT | `/numbers/:id/metadata` | bin-number-manager | Update metadata |
| GET | `/available_numbers` | bin-number-manager | List available numbers to purchase |

---

## Messages

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/messages` | bin-message-manager | List messages |
| POST | `/messages` | bin-message-manager | Send message |
| GET | `/messages/:id` | bin-message-manager | Get message |
| DELETE | `/messages/:id` | bin-message-manager | Delete message |

---

## Emails

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/emails` | bin-email-manager | List emails |
| POST | `/emails` | bin-email-manager | Send email |
| GET | `/emails/:id` | bin-email-manager | Get email |
| DELETE | `/emails/:id` | bin-email-manager | Delete email |

---

## Contacts

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/contacts` | bin-contact-manager | List contacts |
| POST | `/contacts` | bin-contact-manager | Create contact |
| GET | `/contacts/lookup` | bin-contact-manager | Lookup contact by phone/email |
| GET | `/contacts/:id` | bin-contact-manager | Get contact |
| PUT | `/contacts/:id` | bin-contact-manager | Update contact |
| DELETE | `/contacts/:id` | bin-contact-manager | Delete contact |
| POST | `/contacts/:id/emails` | bin-contact-manager | Add email to contact |
| PUT | `/contacts/:id/emails/:email_id` | bin-contact-manager | Update contact email |
| DELETE | `/contacts/:id/emails/:email_id` | bin-contact-manager | Remove contact email |
| POST | `/contacts/:id/phone-numbers` | bin-contact-manager | Add phone number |
| PUT | `/contacts/:id/phone-numbers/:phone_number_id` | bin-contact-manager | Update phone number |
| DELETE | `/contacts/:id/phone-numbers/:phone_number_id` | bin-contact-manager | Remove phone number |
| POST | `/contacts/:id/tags` | bin-contact-manager | Add tag to contact |
| DELETE | `/contacts/:id/tags/:tag_id` | bin-contact-manager | Remove tag from contact |

---

## Conversations

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/conversation_accounts` | bin-conversation-manager | List conversation accounts |
| POST | `/conversation_accounts` | bin-conversation-manager | Create conversation account |
| GET | `/conversation_accounts/:id` | bin-conversation-manager | Get conversation account |
| PUT | `/conversation_accounts/:id` | bin-conversation-manager | Update conversation account |
| DELETE | `/conversation_accounts/:id` | bin-conversation-manager | Delete conversation account |
| GET | `/conversations` | bin-conversation-manager | List conversations |
| GET | `/conversations/:id` | bin-conversation-manager | Get conversation |
| PUT | `/conversations/:id` | bin-conversation-manager | Update conversation |
| GET | `/conversations/:id/messages` | bin-conversation-manager | List conversation messages |
| POST | `/conversations/:id/messages` | bin-conversation-manager | Send message in conversation |
| POST | `/conversations/:id/unassign` | bin-conversation-manager | Unassign conversation |

---

## AI

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/ais` | bin-ai-manager | List AI agents |
| POST | `/ais` | bin-ai-manager | Create AI agent |
| GET | `/ais/:id` | bin-ai-manager | Get AI agent |
| PUT | `/ais/:id` | bin-ai-manager | Update AI agent |
| DELETE | `/ais/:id` | bin-ai-manager | Delete AI agent |
| POST | `/ais/:id/direct-hash-regenerate` | bin-ai-manager | Regenerate direct hash |
| GET | `/aicalls` | bin-ai-manager | List AI call sessions |
| POST | `/aicalls` | bin-ai-manager | Create AI call session |
| GET | `/aicalls/:id` | bin-ai-manager | Get AI call session |
| DELETE | `/aicalls/:id` | bin-ai-manager | Delete AI call session |
| GET | `/aisummaries` | bin-ai-manager | List AI summaries |
| POST | `/aisummaries` | bin-ai-manager | Create AI summary |
| GET | `/aisummaries/:id` | bin-ai-manager | Get AI summary |
| DELETE | `/aisummaries/:id` | bin-ai-manager | Delete AI summary |
| GET | `/aimessages` | bin-ai-manager | List AI messages |
| POST | `/aimessages` | bin-ai-manager | Create AI message |
| GET | `/aimessages/:id` | bin-ai-manager | Get AI message |
| DELETE | `/aimessages/:id` | bin-ai-manager | Delete AI message |
| GET | `/aggregated-events` | bin-hook-manager | List aggregated webhook events |

---

## RAG (Retrieval-Augmented Generation)

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/rags` | bin-rag-manager | List RAG knowledge bases |
| POST | `/rags` | bin-rag-manager | Create RAG knowledge base |
| GET | `/rags/:id` | bin-rag-manager | Get RAG knowledge base |
| PUT | `/rags/:id` | bin-rag-manager | Update RAG knowledge base |
| DELETE | `/rags/:id` | bin-rag-manager | Delete RAG knowledge base |
| POST | `/rags/:id/sources` | bin-rag-manager | Add source document |
| DELETE | `/rags/:id/sources/:source_id` | bin-rag-manager | Remove source document |

---

## TTS / Speakings

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/speakings` | bin-tts-manager / bin-pipecat-manager | List speaking sessions |
| POST | `/speakings` | bin-tts-manager / bin-pipecat-manager | Create speaking session |
| GET | `/speakings/:id` | bin-tts-manager / bin-pipecat-manager | Get speaking session |
| DELETE | `/speakings/:id` | bin-tts-manager / bin-pipecat-manager | Delete speaking session |
| POST | `/speakings/:id/say` | bin-tts-manager / bin-pipecat-manager | Speak text |
| POST | `/speakings/:id/flush` | bin-tts-manager / bin-pipecat-manager | Flush TTS queue |
| POST | `/speakings/:id/stop` | bin-tts-manager / bin-pipecat-manager | Stop speaking |

---

## Transcribe

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/transcribes` | bin-transcribe-manager | List transcription sessions |
| POST | `/transcribes` | bin-transcribe-manager | Create transcription session |
| GET | `/transcribes/:id` | bin-transcribe-manager | Get transcription session |
| DELETE | `/transcribes/:id` | bin-transcribe-manager | Delete transcription session |
| POST | `/transcribes/:id/stop` | bin-transcribe-manager | Stop transcription |
| GET | `/transcripts` | bin-transcribe-manager | List transcript records |

---

## Storage

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/storage_account` | bin-storage-manager | Get own storage account |
| GET | `/storage_accounts` | bin-storage-manager | Admin: list storage accounts |
| POST | `/storage_accounts` | bin-storage-manager | Admin: create storage account |
| GET | `/storage_accounts/:id` | bin-storage-manager | Admin: get storage account |
| DELETE | `/storage_accounts/:id` | bin-storage-manager | Admin: delete storage account |
| GET | `/storage_files` | bin-storage-manager | List files |
| POST | `/storage_files` | bin-storage-manager | Upload file metadata |
| GET | `/storage_files/:id` | bin-storage-manager | Get file metadata |
| GET | `/storage_files/:id/file` | bin-storage-manager | Download file content (GCS) |
| DELETE | `/storage_files/:id` | bin-storage-manager | Delete file |

---

## Tags

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/tags` | bin-tag-manager | List tags |
| POST | `/tags` | bin-tag-manager | Create tag |
| GET | `/tags/:id` | bin-tag-manager | Get tag |
| PUT | `/tags/:id` | bin-tag-manager | Update tag |
| DELETE | `/tags/:id` | bin-tag-manager | Delete tag |

---

## Timeline

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/timelines/:resource_type/:resource_id/events` | bin-timeline-manager | List timeline events for a resource |
| GET | `/timelines/calls/:call_id/pcap` | bin-timeline-manager | Download SIP PCAP capture |
| GET | `/timelines/calls/:call_id/sip-analysis` | bin-timeline-manager | Get SIP flow analysis |

---

## Recordings

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/recordings` | bin-call-manager | List recordings |
| GET | `/recordings/:id` | bin-call-manager | Get recording metadata |
| DELETE | `/recordings/:id` | bin-call-manager | Delete recording |
| GET | `/recordingfiles/:id` | bin-storage-manager | Download recording file content |

---

## Outdial

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/outdials` | bin-outdial-manager | List outdial lists |
| POST | `/outdials` | bin-outdial-manager | Create outdial list |
| GET | `/outdials/:id` | bin-outdial-manager | Get outdial list |
| PUT | `/outdials/:id` | bin-outdial-manager | Update outdial list |
| DELETE | `/outdials/:id` | bin-outdial-manager | Delete outdial list |
| PUT | `/outdials/:id/campaign_id` | bin-outdial-manager | Assign campaign |
| PUT | `/outdials/:id/data` | bin-outdial-manager | Update dial data |
| GET | `/outdials/:id/targets` | bin-outdial-manager | List dial targets |
| POST | `/outdials/:id/targets` | bin-outdial-manager | Add dial target |
| GET | `/outdials/:id/targets/:target_id` | bin-outdial-manager | Get dial target |
| DELETE | `/outdials/:id/targets/:target_id` | bin-outdial-manager | Delete dial target |
| GET | `/outplans` | bin-outdial-manager | List outbound plans |
| POST | `/outplans` | bin-outdial-manager | Create outbound plan |
| GET | `/outplans/:id` | bin-outdial-manager | Get outbound plan |
| PUT | `/outplans/:id` | bin-outdial-manager | Update outbound plan |
| DELETE | `/outplans/:id` | bin-outdial-manager | Delete outbound plan |
| PUT | `/outplans/:id/dial_info` | bin-outdial-manager | Update dial info |

---

## Outbound Config

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/outbound_config` | bin-call-manager | Get own outbound config |
| PUT | `/outbound_config` | bin-call-manager | Update own outbound config |
| GET | `/outbound_configs` | bin-call-manager | List outbound configs |
| POST | `/outbound_configs` | bin-call-manager | Create outbound config |
| GET | `/outbound_configs/:id` | bin-call-manager | Get outbound config |
| PUT | `/outbound_configs/:id` | bin-call-manager | Update outbound config |
| DELETE | `/outbound_configs/:id` | bin-call-manager | Delete outbound config |

---

## Providers (SIP Trunks)

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/providers` | bin-direct-manager | List SIP providers |
| POST | `/providers` | bin-direct-manager | Create SIP provider |
| POST | `/providers/setup` | bin-direct-manager | Setup provider (provisioning) |
| GET | `/providers/:id` | bin-direct-manager | Get provider |
| PUT | `/providers/:id` | bin-direct-manager | Update provider |
| DELETE | `/providers/:id` | bin-direct-manager | Delete provider |
| GET | `/providercalls` | bin-direct-manager | List provider calls |
| POST | `/providercalls` | bin-direct-manager | Create provider call |
| GET | `/providercalls/:id` | bin-direct-manager | Get provider call |
| DELETE | `/providercalls/:id` | bin-direct-manager | Delete provider call |

---

## Route

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/routes` | bin-route-manager | List routing rules |
| POST | `/routes` | bin-route-manager | Create routing rule |
| GET | `/routes/:id` | bin-route-manager | Get routing rule |
| PUT | `/routes/:id` | bin-route-manager | Update routing rule |
| DELETE | `/routes/:id` | bin-route-manager | Delete routing rule |

---

## Trunks / Transfer

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/trunks` | bin-transfer-manager | List trunks |
| POST | `/trunks` | bin-transfer-manager | Create trunk |
| GET | `/trunks/:id` | bin-transfer-manager | Get trunk |
| PUT | `/trunks/:id` | bin-transfer-manager | Update trunk |
| DELETE | `/trunks/:id` | bin-transfer-manager | Delete trunk |
| POST | `/transfers` | bin-call-manager | Transfer call |

---

## Access Keys

Managed locally by api-manager (no RabbitMQ RPC to a backend service).

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/accesskeys` | local (api-manager db) | List access keys |
| POST | `/accesskeys` | local (api-manager db) | Create access key |
| GET | `/accesskeys/:id` | local (api-manager db) | Get access key |
| PUT | `/accesskeys/:id` | local (api-manager db) | Update access key |
| DELETE | `/accesskeys/:id` | local (api-manager db) | Delete access key |

---

## Service Agents (Agent-Scoped Proxy)

`/service_agents/*` routes are a scoped, agent-facing subset of the platform. Authentication uses agent JWT tokens. Resources are automatically scoped to the authenticated agent's customer.

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/service_agents/me` | bin-agent-manager | Get own agent profile |
| PUT | `/service_agents/me` | bin-agent-manager | Update own profile |
| PUT | `/service_agents/me/addresses` | bin-agent-manager | Update addresses |
| PUT | `/service_agents/me/password` | bin-agent-manager | Change password |
| PUT | `/service_agents/me/status` | bin-agent-manager | Set status |
| GET | `/service_agents/agents` | bin-agent-manager | List agents (scoped) |
| GET | `/service_agents/agents/:id` | bin-agent-manager | Get agent |
| GET | `/service_agents/calls` | bin-call-manager | List calls (scoped) |
| GET | `/service_agents/calls/:id` | bin-call-manager | Get call |
| GET | `/service_agents/contacts` | bin-contact-manager | List contacts |
| POST | `/service_agents/contacts` | bin-contact-manager | Create contact |
| GET | `/service_agents/contacts/lookup` | bin-contact-manager | Lookup contact |
| GET | `/service_agents/contacts/:id` | bin-contact-manager | Get contact |
| PUT | `/service_agents/contacts/:id` | bin-contact-manager | Update contact |
| DELETE | `/service_agents/contacts/:id` | bin-contact-manager | Delete contact |
| POST | `/service_agents/contacts/:id/emails` | bin-contact-manager | Add email |
| PUT | `/service_agents/contacts/:id/emails/:email_id` | bin-contact-manager | Update email |
| DELETE | `/service_agents/contacts/:id/emails/:email_id` | bin-contact-manager | Remove email |
| POST | `/service_agents/contacts/:id/phone_numbers` | bin-contact-manager | Add phone number |
| PUT | `/service_agents/contacts/:id/phone_numbers/:phone_number_id` | bin-contact-manager | Update phone number |
| DELETE | `/service_agents/contacts/:id/phone_numbers/:phone_number_id` | bin-contact-manager | Remove phone number |
| POST | `/service_agents/contacts/:id/tags` | bin-contact-manager | Add tag |
| DELETE | `/service_agents/contacts/:id/tags/:tag_id` | bin-contact-manager | Remove tag |
| GET | `/service_agents/conversations` | bin-conversation-manager | List conversations |
| GET | `/service_agents/conversations/:id` | bin-conversation-manager | Get conversation |
| PUT | `/service_agents/conversations/:id` | bin-conversation-manager | Update conversation |
| GET | `/service_agents/conversations/:id/messages` | bin-conversation-manager | List messages |
| POST | `/service_agents/conversations/:id/messages` | bin-conversation-manager | Send message |
| POST | `/service_agents/conversations/:id/unassign` | bin-conversation-manager | Unassign conversation |
| GET | `/service_agents/customer` | bin-customer-manager | Get customer info |
| GET | `/service_agents/extensions` | bin-agent-manager | List extensions |
| GET | `/service_agents/extensions/:id` | bin-agent-manager | Get extension |
| GET | `/service_agents/files` | bin-storage-manager | List files |
| POST | `/service_agents/files` | bin-storage-manager | Upload file |
| GET | `/service_agents/files/:id` | bin-storage-manager | Get file metadata |
| GET | `/service_agents/files/:id/file` | bin-storage-manager | Download file |
| DELETE | `/service_agents/files/:id` | bin-storage-manager | Delete file |
| GET | `/service_agents/tags` | bin-tag-manager | List tags |
| GET | `/service_agents/tags/:id` | bin-tag-manager | Get tag |
| GET | `/service_agents/talk_channels` | bin-talk-manager | List talk channels |
| GET | `/service_agents/talk_chats` | bin-talk-manager | List talk chats |
| POST | `/service_agents/talk_chats` | bin-talk-manager | Create talk chat |
| GET | `/service_agents/talk_chats/:id` | bin-talk-manager | Get talk chat |
| PUT | `/service_agents/talk_chats/:id` | bin-talk-manager | Update talk chat |
| DELETE | `/service_agents/talk_chats/:id` | bin-talk-manager | Delete talk chat |
| POST | `/service_agents/talk_chats/:id/join` | bin-talk-manager | Join talk chat |
| GET | `/service_agents/talk_chats/:id/participants` | bin-talk-manager | List participants |
| POST | `/service_agents/talk_chats/:id/participants` | bin-talk-manager | Add participant |
| DELETE | `/service_agents/talk_chats/:id/participants/:participant_id` | bin-talk-manager | Remove participant |
| GET | `/service_agents/talk_messages` | bin-talk-manager | List talk messages |
| POST | `/service_agents/talk_messages` | bin-talk-manager | Send talk message |
| GET | `/service_agents/talk_messages/:id` | bin-talk-manager | Get talk message |
| DELETE | `/service_agents/talk_messages/:id` | bin-talk-manager | Delete talk message |
| POST | `/service_agents/talk_messages/:id/reactions` | bin-talk-manager | Add reaction |
| GET | `/service_agents/ws` | local (websockhandler) | Agent WebSocket endpoint |

---

## WebSocket

| HTTP Method | Path | Backend Service | Notes |
|------------|------|----------------|-------|
| GET | `/ws` | local (websockhandler) | Customer WebSocket endpoint — real-time event push |
