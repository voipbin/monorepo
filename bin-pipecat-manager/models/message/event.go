package message

const (
	EventTypeBotTranscription  string = "message_bot_transcription"
	EventTypeUserTranscription string = "message_user_transcription"

	EventTypeBotLLM  string = "message_bot_llm"
	EventTypeUserLLM string = "message_user_llm"

	EventTypeTeamMemberSwitched string = "team_member_switched"
)
