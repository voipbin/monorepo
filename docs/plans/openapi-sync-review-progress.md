# OpenAPI Sync Review Progress

| # | Service | OpenAPI Schemas | Status | Notes |
|---|---------|----------------|--------|-------|
| 1 | bin-agent-manager | AgentManagerAgent | done - in sync | All 13 fields and 3 enums match |
| 2 | bin-ai-manager | AIManagerAI, AIManagerAIcall, AIManagerMessage, AIManagerSummary | done - fixed 8 mismatches | EngineType/EngineModel enum stale, tts_type/stt_type missing $ref, Gender/Direction/Role missing empty string, Message missing customer_id/tool_calls/tool_call_id and had extra tm_delete |
| 3 | bin-call-manager | CallManagerCall, CallManagerGroupcall, CallManagerRecording | done - fixed 2 mismatches | CallManagerCall had extra external_media_ids (not in WebhookMessage), CallManagerRecordingFormat x-enum-varnames missing space after dash |
| 4 | bin-campaign-manager | CampaignManagerCampaign, CampaignManagerCampaigncall, CampaignManagerOutplan | done - in sync | All fields and enums match across all 3 schemas |
| 5 | bin-conference-manager | ConferenceManagerConference, ConferenceManagerConferencecall | done - in sync | All fields and enums match across both schemas |
| 6 | bin-contact-manager | ContactManagerContact | done - in sync | All fields match |
| 7 | bin-conversation-manager | ConversationManagerAccount, ConversationManagerConversation, ConversationManagerMessage | done - fixed 3 mismatches | Conversation had wrong fields (reference_type/reference_id/source/participants replaced with type/dialog_id/self/peer), renamed enum ConversationManagerConversationReferenceType to ConversationManagerConversationType, Message had extra source field |
| 8 | bin-customer-manager | CustomerManagerAccesskey, CustomerManagerCustomer | done - in sync | All fields match |
| 9 | bin-email-manager | EmailManagerEmail | done - fixed 1 mismatch | Removed extra activeflow_id field not in WebhookMessage |
| 10 | bin-flow-manager | FlowManagerFlow, FlowManagerActiveflow | done - fixed 1 mismatch | FlowManagerReferenceType enum had only 3 values (none/call/message), updated to match Go's 8 values (none/ai/api/call/campaign/conversation/transcribe/recording) |
| 11 | bin-message-manager | MessageManagerMessage | done - in sync | All fields match |
| 12 | bin-number-manager | NumberManagerNumber, NumberManagerAvailableNumber | done - in sync | All fields match |
| 13 | bin-outdial-manager | OutdialManagerOutdial, OutdialManagerOutdialtarget | done - in sync | All fields match |
| 14 | bin-queue-manager | QueueManagerQueue, QueueManagerQueuecall | done - in sync | All fields match |
| 15 | bin-registrar-manager | RegistrarManagerExtension, RegistrarManagerTrunk | done - in sync | All fields match |
| 16 | bin-route-manager | RouteManagerProvider, RouteManagerRoute | done - in sync | All fields match |
| 17 | bin-storage-manager | StorageManagerAccount, StorageManagerFile | done - fixed 1 mismatch | StorageManagerFile missing owner_type field (from commonidentity.Owner embed) |
| 18 | bin-tag-manager | TagManagerTag | done - fixed 1 mismatch | Missing customer_id field (from commonidentity.Identity embed) |
| 19 | bin-talk-manager | TalkManagerMessage, TalkManagerParticipant | done - in sync | All fields match |
| 20 | bin-transcribe-manager | TranscribeManagerTranscribe, TranscribeManagerTranscript | done - fixed 5 mismatches | Transcribe missing activeflow_id and on_end_flow_id, ReferenceType enum had extra "conference", Provider enum missing empty string "", Transcript missing customer_id |
| 21 | bin-transfer-manager | TransferManagerTransfer | done - in sync | All fields match |
| 22 | bin-tts-manager | TtsManagerSpeaking | done - fixed 3 mismatches | reference_type/direction/status were plain strings instead of $ref to enum schemas, created TtsManagerSpeakingReferenceType/Direction/Status enums |
