package action

// OptionStructByType maps each action Type to a zero-value of its option struct.
//
// It exists so that downstream consumers can reflect over an action's option
// fields by type. Its primary consumer today is bin-ai-manager's describe_action
// tool, whose hand-authored option catalog is verified against these structs by
// the test TestActionCatalogFieldsMatchOptionStructs (in bin-ai-manager).
//
// SYNC NOTE (read before editing option.go):
//   - Adding/removing/renaming an OptionXxx field changes what the LLM should be
//     told. After such a change, update the ai-manager describe_action catalog
//     (bin-ai-manager/pkg/actioncatalog/main.go). The reflection test
//     TestActionCatalogFieldsMatchOptionStructs will FAIL until the catalog's
//     option field names match again.
//   - Field TYPE / meaning changes (e.g. string->int, or a reworded purpose) are
//     NOT caught by that test (reflection cannot read comments); update the
//     catalog's human-readable Type/Description text by hand.
//   - The same fields are documented for users in
//     bin-api-manager/docsdev/source/flow_struct_action.rst, which may also need
//     updating.
//   - Adding a NEW action Type: add it to TypeListAll AND to this map, or the
//     test TestActionCatalogMatchesTypeListAll / the field-parity test will fail.
//
// Actions that take no options (e.g. mute, stop) map to an empty struct{}{} —
// they have zero top-level json fields, so their catalog entry has no options.
var OptionStructByType = map[Type]any{
	TypeAMD:                 OptionAMD{},
	TypeAnswer:              OptionAnswer{},
	TypeAISummary:           OptionAISummary{},
	TypeAITalk:              OptionAITalk{},
	TypeAITask:              OptionAITask{},
	TypeBeep:                OptionBeep{},
	TypeBlock:               OptionBlock{},
	TypeBranch:              OptionBranch{},
	TypeCall:                OptionCall{},
	TypeConditionCallDigits: OptionConditionCallDigits{},
	TypeConditionCallStatus: OptionConditionCallStatus{},
	TypeConditionDatetime:   OptionConditionDatetime{},
	TypeConditionVariable:   OptionConditionVariable{},
	TypeConfbridgeJoin:      OptionConfbridgeJoin{},
	TypeConferenceJoin:      OptionConferenceJoin{},
	TypeConnect:             OptionConnect{},
	TypeConversationSend:    OptionConversationSend{},
	TypeDigitsReceive:       OptionDigitsReceive{},
	TypeDigitsSend:          OptionDigitsSend{},
	TypeEcho:                OptionEcho{},
	TypeEmailSend:           OptionEmailSend{},
	TypeExternalMediaStart:  OptionExternalMediaStart{},
	TypeExternalMediaStop:   OptionExternalMediaStop{},
	TypeFetch:               OptionFetch{},
	TypeFetchFlow:           OptionFetchFlow{},
	TypeGoto:                OptionGoto{},
	TypeHangup:              OptionHangup{},
	TypeMessageSend:         OptionMessageSend{},
	TypeMute:                struct{}{}, // no options
	TypePlay:                OptionPlay{},
	TypeQueueJoin:           OptionQueueJoin{},
	TypeRecordingStart:      OptionRecordingStart{},
	TypeRecordingStop:       OptionRecordingStop{},
	TypeSleep:               OptionSleep{},
	TypeStop:                struct{}{}, // no options
	TypeStreamEcho:          OptionStreamEcho{},
	TypeTalk:                OptionTalk{},
	TypeTranscribeStart:     OptionTranscribeStart{},
	TypeTranscribeStop:      OptionTranscribeStop{},
	TypeTranscribeRecording: OptionTranscribeRecording{},
	TypeVariableSet:         OptionVariableSet{},
	TypeWebhookSend:         OptionWebhookSend{},
}
