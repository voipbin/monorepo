package actioncatalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fmaction "monorepo/bin-flow-manager/models/action"
)

// ErrUnknownActionType is returned by DescribeAction when the requested action
// type is not in the catalog. The error message echoes the received value and
// lists the valid types so an LLM can self-correct.
var ErrUnknownActionType = errors.New("unknown action type")

// ActionTypeEnum returns the JSON-schema enum of all flow action types, sourced
// from the authoritative flow-manager action.TypeListAll. Both the create_call
// "actions[].type" schema and the describe_action "action_type" schema use this
// so the two enums cannot drift from each other or from flow-manager.
func ActionTypeEnum() []string {
	out := make([]string, len(fmaction.TypeListAll))
	for i, t := range fmaction.TypeListAll {
		out[i] = string(t)
	}
	return out
}

// actionOptionField describes a single option field of a flow action, for the
// describe_action tool. Name MUST match the option struct's top-level json tag
// (verified by TestActionCatalogFieldsMatchOptionStructs); Type and Description
// are human-readable text condensed from option.go comments and the RST docs.
type actionOptionField struct {
	Name        string
	Type        string
	Required    bool
	Description string
}

// actionCatalogEntry describes one flow action type for the LLM.
type actionCatalogEntry struct {
	Type    fmaction.Type
	Summary string
	Options []actionOptionField
}

// actionCatalog is the authoritative hand-authored catalog: one entry per
// flow-manager action type. It is condensed from
// bin-flow-manager/models/action/option.go (field names + comments) and the
// user-facing bin-api-manager/docsdev/source/flow_struct_action.rst.
//
// DRIFT GUARDS:
//   - TestActionCatalogMatchesTypeListAll: one entry per action.TypeListAll
//     (no missing / extra / duplicate type).
//   - TestActionCatalogFieldsMatchOptionStructs: each entry's option field
//     Names equal the real option struct's top-level json fields (via
//     action.OptionStructByType). Field TYPE/Description text is NOT machine
//     checked; keep it in sync by hand when option.go changes.
var actionCatalog = []actionCatalogEntry{
	{Type: fmaction.TypeAMD, Summary: "Detect whether a human or an answering machine answered the call.", Options: []actionOptionField{
		{Name: "machine_handle", Type: "string (hangup|continue)", Required: false, Description: "What to do if a machine answered: \"hangup\" or \"continue\"."},
		{Name: "async", Type: "bool", Required: false, Description: "If false, the flow waits until AMD finishes before continuing."},
	}},
	{Type: fmaction.TypeAnswer, Summary: "Answer the incoming call.", Options: nil},
	{Type: fmaction.TypeAISummary, Summary: "Generate an AI summary of a reference (e.g. a recording or transcribe).", Options: []actionOptionField{
		{Name: "on_end_flow_id", Type: "uuid", Required: false, Description: "Flow id to run when the summary finishes."},
		{Name: "reference_type", Type: "string (call|conference|transcribe|recording)", Required: true, Description: "Type of the resource to summarize."},
		{Name: "reference_id", Type: "uuid", Required: true, Description: "Id of the resource to summarize."},
		{Name: "language", Type: "string", Required: false, Description: "Output language (IETF locale, e.g. en-US)."},
	}},
	{Type: fmaction.TypeAITalk, Summary: "Start an AI voice conversation on the call.", Options: []actionOptionField{
		{Name: "ai_id", Type: "uuid", Required: false, Description: "Deprecated; use assistance_type + assistance_id."},
		{Name: "assistance_type", Type: "string", Required: true, Description: "\"ai\" or \"team\"."},
		{Name: "assistance_id", Type: "uuid", Required: true, Description: "Id of the AI or team to converse with."},
		{Name: "duration", Type: "int (seconds)", Required: false, Description: "Maximum AI talk duration in seconds."},
	}},
	{Type: fmaction.TypeAITask, Summary: "Run an AI task (non-conversational) on the call.", Options: []actionOptionField{
		{Name: "ai_id", Type: "uuid", Required: false, Description: "Deprecated; use assistance_type + assistance_id."},
		{Name: "assistance_type", Type: "string", Required: true, Description: "\"ai\" or \"team\"."},
		{Name: "assistance_id", Type: "uuid", Required: true, Description: "Id of the AI or team to run the task."},
	}},
	{Type: fmaction.TypeBeep, Summary: "Play a beep tone on the call.", Options: nil},
	{Type: fmaction.TypeBlock, Summary: "Internal grouping/block action.", Options: nil},
	{Type: fmaction.TypeBranch, Summary: "Branch the flow to different actions based on a variable / received DTMF.", Options: []actionOptionField{
		{Name: "variable", Type: "string", Required: false, Description: "Variable to branch on (defaults to received digits)."},
		{Name: "default_target_id", Type: "uuid", Required: false, Description: "Action id to go to when no target matches."},
		{Name: "target_ids", Type: "object (map of value -> action id)", Required: true, Description: "Map of matched value to the action id to jump to."},
	}},
	{Type: fmaction.TypeCall, Summary: "Originate one or more new outbound calls (each runs its own flow or actions).", Options: []actionOptionField{
		{Name: "source", Type: "address object {type,target,target_name}", Required: false, Description: "Source endpoint / caller id."},
		{Name: "destinations", Type: "array of address objects {type,target,target_name}", Required: true, Description: "Destinations to call."},
		{Name: "flow_id", Type: "uuid", Required: false, Description: "Pre-built flow the new call runs (use flow_id OR actions)."},
		{Name: "actions", Type: "array of action objects", Required: false, Description: "Inline actions the new call runs (use flow_id OR actions)."},
		{Name: "chained", Type: "bool", Required: false, Description: "If true, the created calls hang up when the master call hangs up."},
		{Name: "early_execution", Type: "bool", Required: false, Description: "If true, the created call runs its flow before being answered."},
		{Name: "anonymous", Type: "string (yes|no|auto)", Required: false, Description: "Anonymous caller id on outbound PSTN."},
	}},
	{Type: fmaction.TypeConditionCallDigits, Summary: "Branch to a false target unless the call's received digits meet the condition.", Options: []actionOptionField{
		{Name: "length", Type: "int", Required: false, Description: "Required digit length."},
		{Name: "key", Type: "string", Required: false, Description: "Required finishing digit key."},
		{Name: "false_target_id", Type: "uuid", Required: true, Description: "Action id to jump to when the condition is false."},
	}},
	{Type: fmaction.TypeConditionCallStatus, Summary: "Branch to a false target unless the call's status matches.", Options: []actionOptionField{
		{Name: "status", Type: "string (dialing|ringing|progressing|terminating|canceling|hangup)", Required: true, Description: "Call status to test for."},
		{Name: "false_target_id", Type: "uuid", Required: true, Description: "Action id to jump to when the condition is false."},
	}},
	{Type: fmaction.TypeConditionDatetime, Summary: "Branch to a false target unless the current date/time matches the condition.", Options: []actionOptionField{
		{Name: "condition", Type: "string (==|!=|>|>=|<|<=)", Required: true, Description: "Comparison operator."},
		{Name: "minute", Type: "int (0-59)", Required: false, Description: "Minute component."},
		{Name: "hour", Type: "int (0-23)", Required: false, Description: "Hour component."},
		{Name: "day", Type: "int (1-31)", Required: false, Description: "Day of month."},
		{Name: "month", Type: "int (1-12)", Required: false, Description: "Month."},
		{Name: "weekdays", Type: "array of int (Sun=0..Sat=6)", Required: false, Description: "Allowed weekdays."},
		{Name: "false_target_id", Type: "uuid", Required: true, Description: "Action id to jump to when the condition is false."},
	}},
	{Type: fmaction.TypeConditionVariable, Summary: "Branch to a false target unless a flow variable matches the condition.", Options: []actionOptionField{
		{Name: "condition", Type: "string (==|!=|>|>=|<|<=)", Required: true, Description: "Comparison operator."},
		{Name: "variable", Type: "string", Required: true, Description: "Variable name to test."},
		{Name: "value_type", Type: "string (string|number|length)", Required: true, Description: "Type of the value to compare."},
		{Name: "value_string", Type: "string", Required: false, Description: "String value to compare against."},
		{Name: "value_number", Type: "number", Required: false, Description: "Numeric value to compare against."},
		{Name: "value_length", Type: "int", Required: false, Description: "Length value to compare against."},
		{Name: "false_target_id", Type: "uuid", Required: true, Description: "Action id to jump to when the condition is false."},
	}},
	{Type: fmaction.TypeConfbridgeJoin, Summary: "Join the call into a confbridge (advanced/internal).", Options: []actionOptionField{
		{Name: "confbridge_id", Type: "uuid", Required: true, Description: "Confbridge id to join."},
	}},
	{Type: fmaction.TypeConferenceJoin, Summary: "Join the call into a conference.", Options: []actionOptionField{
		{Name: "conference_id", Type: "uuid", Required: true, Description: "Conference id to join."},
	}},
	{Type: fmaction.TypeConnect, Summary: "Originate new call(s) and bridge them into the current call (transfer/connect).", Options: []actionOptionField{
		{Name: "source", Type: "address object {type,target,target_name}", Required: false, Description: "Source endpoint / caller id."},
		{Name: "destinations", Type: "array of address objects {type,target,target_name}", Required: true, Description: "Endpoints to connect to."},
		{Name: "early_media", Type: "bool", Required: false, Description: "If true, get early media from the destination."},
		{Name: "relay_reason", Type: "bool", Required: false, Description: "If true, hang up the master call with the destination's hangup reason."},
		{Name: "anonymous", Type: "string (yes|no|auto)", Required: false, Description: "Anonymous caller id on outbound PSTN."},
	}},
	{Type: fmaction.TypeConversationSend, Summary: "Send a message into a conversation (chat/SNS).", Options: []actionOptionField{
		{Name: "conversation_id", Type: "uuid", Required: true, Description: "Conversation id to send into."},
		{Name: "text", Type: "string", Required: true, Description: "Message text."},
		{Name: "sync", Type: "bool", Required: false, Description: "Whether to send synchronously."},
	}},
	{Type: fmaction.TypeDigitsReceive, Summary: "Receive DTMF digits from the caller.", Options: []actionOptionField{
		{Name: "duration", Type: "int (ms)", Required: false, Description: "DTMF receiving duration in milliseconds."},
		{Name: "key", Type: "string", Required: false, Description: "Finishing key; not included in the resulting variable. If unset, no key finishes."},
		{Name: "length", Type: "int", Required: false, Description: "Max number of DTMF events to gather before continuing."},
	}},
	{Type: fmaction.TypeDigitsSend, Summary: "Send DTMF tones on the call.", Options: []actionOptionField{
		{Name: "digits", Type: "string", Required: true, Description: "Keys to send (0-9, A-D, #, *; max 100)."},
		{Name: "duration", Type: "int (ms)", Required: false, Description: "DTMF tone duration per key (100-1000)."},
		{Name: "interval", Type: "int (ms)", Required: false, Description: "Interval between keys (0-5000)."},
	}},
	{Type: fmaction.TypeEcho, Summary: "Echo the caller's audio back to them.", Options: []actionOptionField{
		{Name: "duration", Type: "int", Required: false, Description: "Echo duration."},
	}},
	{Type: fmaction.TypeEmailSend, Summary: "Send an email.", Options: []actionOptionField{
		{Name: "destinations", Type: "array of address objects {type,target,target_name}", Required: true, Description: "Email recipients."},
		{Name: "subject", Type: "string", Required: true, Description: "Email subject."},
		{Name: "content", Type: "string", Required: true, Description: "Email body (HTML or plain text)."},
		{Name: "attachments", Type: "array of attachment objects {reference_type,reference_id}", Required: false, Description: "Optional attachments."},
	}},
	{Type: fmaction.TypeExternalMediaStart, Summary: "Start streaming external media (advanced/infra).", Options: []actionOptionField{
		{Name: "external_host", Type: "string", Required: true, Description: "External media target host address."},
		{Name: "encapsulation", Type: "string", Required: false, Description: "Encapsulation (default rtp)."},
		{Name: "transport", Type: "string", Required: false, Description: "Transport (default udp)."},
		{Name: "transport_data", Type: "string", Required: false, Description: "Transport-specific data."},
		{Name: "connection_type", Type: "string", Required: false, Description: "Connection type (default client)."},
		{Name: "format", Type: "string", Required: false, Description: "Audio format (default ulaw)."},
		{Name: "direction_listen", Type: "string", Required: false, Description: "Listen direction."},
		{Name: "direction_speak", Type: "string", Required: false, Description: "Speak direction."},
		{Name: "data", Type: "string", Required: false, Description: "Data."},
	}},
	{Type: fmaction.TypeExternalMediaStop, Summary: "Stop external media streaming (advanced/infra).", Options: nil},
	{Type: fmaction.TypeFetch, Summary: "Fetch the next actions from an external HTTP endpoint.", Options: []actionOptionField{
		{Name: "event_url", Type: "string", Required: true, Description: "URL to fetch actions from."},
		{Name: "event_method", Type: "string", Required: false, Description: "HTTP method."},
	}},
	{Type: fmaction.TypeFetchFlow, Summary: "Fetch and run the actions of another flow.", Options: []actionOptionField{
		{Name: "flow_id", Type: "uuid", Required: true, Description: "Flow id whose actions to run."},
	}},
	{Type: fmaction.TypeGoto, Summary: "Jump to another action in the flow (optionally looping).", Options: []actionOptionField{
		{Name: "target_id", Type: "uuid", Required: true, Description: "Action id to jump to."},
		{Name: "loop_count", Type: "int", Required: false, Description: "Number of times to loop."},
	}},
	{Type: fmaction.TypeHangup, Summary: "Hang up the call.", Options: []actionOptionField{
		{Name: "reason", Type: "string", Required: false, Description: "Hangup reason code."},
		{Name: "reference_id", Type: "uuid", Required: false, Description: "Hang up with the same reason as this referenced call id (overrides reason)."},
	}},
	{Type: fmaction.TypeMessageSend, Summary: "Send an SMS text message.", Options: []actionOptionField{
		{Name: "source", Type: "address object {type,target,target_name}", Required: false, Description: "Source phone number."},
		{Name: "destinations", Type: "array of address objects {type,target,target_name}", Required: true, Description: "Destination phone numbers."},
		{Name: "text", Type: "string", Required: true, Description: "Message text."},
	}},
	{Type: fmaction.TypeMute, Summary: "Mute audio on the call.", Options: nil},
	{Type: fmaction.TypePlay, Summary: "Play audio from one or more media URLs.", Options: []actionOptionField{
		{Name: "stream_urls", Type: "array of string", Required: true, Description: "Media URLs to play."},
	}},
	{Type: fmaction.TypeQueueJoin, Summary: "Place the call into a queue.", Options: []actionOptionField{
		{Name: "queue_id", Type: "uuid", Required: true, Description: "Queue id to join."},
	}},
	{Type: fmaction.TypeRecordingStart, Summary: "Start recording the call.", Options: []actionOptionField{
		{Name: "format", Type: "string", Required: false, Description: "Audio format: wav, mp3, ogg."},
		{Name: "end_of_silence", Type: "int (seconds)", Required: false, Description: "Max silence duration; 0 for no limit."},
		{Name: "end_of_key", Type: "string", Required: false, Description: "DTMF input to terminate recording: none, any, *, #."},
		{Name: "duration", Type: "int (seconds)", Required: false, Description: "Max recording duration; 0 for no limit."},
		{Name: "beep_start", Type: "bool", Required: false, Description: "Play a beep when recording begins."},
		{Name: "on_end_flow_id", Type: "uuid", Required: false, Description: "Flow id to run when recording ends."},
	}},
	{Type: fmaction.TypeRecordingStop, Summary: "Stop the current recording.", Options: nil},
	{Type: fmaction.TypeSleep, Summary: "Pause the flow for a duration.", Options: []actionOptionField{
		{Name: "duration", Type: "int (ms)", Required: true, Description: "Sleep duration in milliseconds."},
	}},
	{Type: fmaction.TypeStop, Summary: "Stop the flow execution.", Options: nil},
	{Type: fmaction.TypeStreamEcho, Summary: "Stream-echo the caller's audio (advanced).", Options: []actionOptionField{
		{Name: "duration", Type: "int", Required: false, Description: "Echo duration."},
	}},
	{Type: fmaction.TypeTalk, Summary: "Speak text to the call using TTS (SSML or plain text).", Options: []actionOptionField{
		{Name: "text", Type: "string", Required: true, Description: "Text to read (SSML or plain text)."},
		{Name: "language", Type: "string", Required: false, Description: "IETF locale, e.g. ko-KR, en-US."},
		{Name: "provider", Type: "string", Required: false, Description: "TTS provider (gcp/aws)."},
		{Name: "voice_id", Type: "string", Required: false, Description: "Provider-specific voice ID."},
		{Name: "digits_handle", Type: "string (next or empty)", Required: false, Description: "What to do when DTMF digits are received during talk: \"next\" moves to the next action; empty does nothing."},
		{Name: "async", Type: "bool", Required: false, Description: "If true, the flow continues without waiting for talk to finish."},
	}},
	{Type: fmaction.TypeTranscribeStart, Summary: "Start live transcription of the call.", Options: []actionOptionField{
		{Name: "language", Type: "string", Required: false, Description: "BCP47 language, e.g. en-US."},
		{Name: "on_end_flow_id", Type: "uuid", Required: false, Description: "Flow id to run when transcription ends."},
		{Name: "provider", Type: "string", Required: false, Description: "Transcribe provider (gcp/aws)."},
		{Name: "direction", Type: "string", Required: false, Description: "in|out|both (default both)."},
	}},
	{Type: fmaction.TypeTranscribeStop, Summary: "Stop live transcription.", Options: nil},
	{Type: fmaction.TypeTranscribeRecording, Summary: "Transcribe a recording.", Options: []actionOptionField{
		{Name: "language", Type: "string", Required: false, Description: "BCP47 language, e.g. en-US."},
		{Name: "on_end_flow_id", Type: "uuid", Required: false, Description: "Flow id to run when transcription ends."},
		{Name: "provider", Type: "string", Required: false, Description: "Transcribe provider (gcp/aws)."},
		{Name: "direction", Type: "string", Required: false, Description: "in|out|both (default both)."},
	}},
	{Type: fmaction.TypeVariableSet, Summary: "Set a flow variable.", Options: []actionOptionField{
		{Name: "key", Type: "string", Required: true, Description: "Variable name."},
		{Name: "value", Type: "string", Required: true, Description: "Variable value."},
	}},
	{Type: fmaction.TypeWebhookSend, Summary: "Send an HTTP webhook request.", Options: []actionOptionField{
		{Name: "sync", Type: "bool", Required: false, Description: "Whether to wait for the response."},
		{Name: "uri", Type: "string", Required: true, Description: "Target URL."},
		{Name: "method", Type: "string", Required: false, Description: "POST/GET/PUT/DELETE."},
		{Name: "data_type", Type: "string", Required: false, Description: "Content type, e.g. application/json."},
		{Name: "data", Type: "string", Required: false, Description: "Request body."},
	}},
}

// catalogByType is the O(1) lookup map, built once at package load from
// actionCatalog. Duplicate Types would be a programming error; the build panics
// so they are caught at startup (and by TestActionCatalogMatchesTypeListAll).
var catalogByType = buildCatalogByType()

func buildCatalogByType() map[fmaction.Type]actionCatalogEntry {
	m := make(map[fmaction.Type]actionCatalogEntry, len(actionCatalog))
	for _, e := range actionCatalog {
		if _, dup := m[e.Type]; dup {
			panic(fmt.Sprintf("duplicate action catalog entry for type %s", e.Type))
		}
		m[e.Type] = e
	}
	return m
}

// renderActionCatalogEntry renders a catalog entry to the compact readable text
// returned by describe_action. This format is a load-bearing contract consumed
// by the LLM; it is pinned by a golden test.
func renderActionCatalogEntry(e actionCatalogEntry) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("action: %s\n", e.Type))
	b.WriteString(fmt.Sprintf("summary: %s\n", e.Summary))
	if len(e.Options) == 0 {
		b.WriteString("options: this action takes no options.")
		return b.String()
	}
	b.WriteString("options:\n")
	lines := make([]string, 0, len(e.Options))
	for _, o := range e.Options {
		req := "optional"
		if o.Required {
			req = "required"
		}
		lines = append(lines, fmt.Sprintf("  - %s (%s, %s): %s", o.Name, o.Type, req, o.Description))
	}
	b.WriteString(strings.Join(lines, "\n"))
	return b.String()
}

// sortedActionTypeList returns the action types as a sorted, comma-joined string
// for use in error messages (so a wrong action_type self-corrects).
func sortedActionTypeList() string {
	out := ActionTypeEnum()
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// DescribeAction returns the rendered option-field description for a flow action
// type. On an unknown/empty type it returns an error wrapping ErrUnknownActionType
// whose message echoes the received value and lists the valid types so the LLM
// can self-correct. This is a pure, customer-agnostic, read-only lookup.
func DescribeAction(actionType string) (string, error) {
	if actionType == "" {
		return "", fmt.Errorf("%w: action_type is required. valid types: %s", ErrUnknownActionType, sortedActionTypeList())
	}
	entry, ok := catalogByType[fmaction.Type(actionType)]
	if !ok {
		return "", fmt.Errorf("%w: %q. valid types: %s", ErrUnknownActionType, actionType, sortedActionTypeList())
	}
	return renderActionCatalogEntry(entry), nil
}
