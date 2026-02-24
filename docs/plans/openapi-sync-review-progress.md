# OpenAPI Sync Review Progress

| # | Service | OpenAPI Schemas | Status | Notes |
|---|---------|----------------|--------|-------|
| 1 | bin-agent-manager | AgentManagerAgent | done - in sync | All 13 fields and 3 enums match |
| 2 | bin-ai-manager | AIManagerAI, AIManagerAIcall, AIManagerMessage, AIManagerSummary | done - fixed 8 mismatches | EngineType/EngineModel enum stale, tts_type/stt_type missing $ref, Gender/Direction/Role missing empty string, Message missing customer_id/tool_calls/tool_call_id and had extra tm_delete |
| 3 | bin-call-manager | CallManagerCall, CallManagerGroupcall, CallManagerRecording | done - fixed 2 mismatches | CallManagerCall had extra external_media_ids (not in WebhookMessage), CallManagerRecordingFormat x-enum-varnames missing space after dash |
| 4 | bin-campaign-manager | CampaignManagerCampaign, CampaignManagerCampaigncall, CampaignManagerOutplan | pending | |
| 5 | bin-conference-manager | ConferenceManagerConference, ConferenceManagerConferencecall | pending | |
| 6 | bin-contact-manager | ContactManagerContact | pending | |
| 7 | bin-conversation-manager | ConversationManagerAccount, ConversationManagerConversation, ConversationManagerMessage | pending | |
| 8 | bin-customer-manager | CustomerManagerAccesskey, CustomerManagerCustomer | pending | |
| 9 | bin-email-manager | EmailManagerEmail | pending | |
| 10 | bin-flow-manager | FlowManagerFlow, FlowManagerActiveflow | pending | |
| 11 | bin-message-manager | MessageManagerMessage | pending | |
| 12 | bin-number-manager | NumberManagerNumber, NumberManagerAvailableNumber | pending | |
| 13 | bin-outdial-manager | OutdialManagerOutdial, OutdialManagerOutdialtarget | pending | |
| 14 | bin-queue-manager | QueueManagerQueue, QueueManagerQueuecall | pending | |
| 15 | bin-registrar-manager | RegistrarManagerExtension, RegistrarManagerTrunk | pending | |
| 16 | bin-route-manager | RouteManagerProvider, RouteManagerRoute | pending | |
| 17 | bin-storage-manager | StorageManagerAccount, StorageManagerFile | pending | |
| 18 | bin-tag-manager | TagManagerTag | pending | |
| 19 | bin-talk-manager | TalkManagerMessage, TalkManagerParticipant | pending | |
| 20 | bin-transcribe-manager | TranscribeManagerTranscribe, TranscribeManagerTranscript | pending | |
| 21 | bin-transfer-manager | TransferManagerTransfer | pending | |
| 22 | bin-tts-manager | TtsManagerSpeaking | pending | |
