package action

import "testing"

// Test_MetaGolden pins the exposure and flow kind of every action type.
// Test_MetaCoversAllTypes only checks that a type has an entry, so without
// this table a one-word change (for example fetch back to sensitive, which
// would offer an action with no SSRF guard to the builder) passes every test.
// Changing a row here is a deliberate, reviewed decision (VOIP-1573 OQ1, OQ8).
func Test_MetaGolden(t *testing.T) {
	tests := []struct {
		ty   Type
		exp  Exposure
		flow FlowKind
	}{
		{Type("ai_summary"), ExposureInternal, FlowKindContinue},
		{Type("ai_talk"), ExposureSensitive, FlowKindContinue},
		{Type("ai_task"), ExposureSensitive, FlowKindContinue},
		{Type("amd"), ExposureCore, FlowKindContinue},
		{Type("answer"), ExposureCore, FlowKindContinue},
		{Type("beep"), ExposureCore, FlowKindContinue},
		{Type("block"), ExposureInternal, FlowKindContinue},
		{Type("branch"), ExposureCore, FlowKindJump},
		{Type("call"), ExposureSensitive, FlowKindContinue},
		{Type("case_create"), ExposureCore, FlowKindContinue},
		{Type("condition_call_digits"), ExposureCore, FlowKindContinue},
		{Type("condition_call_status"), ExposureCore, FlowKindContinue},
		{Type("condition_datetime"), ExposureCore, FlowKindContinue},
		{Type("condition_variable"), ExposureCore, FlowKindContinue},
		{Type("confbridge_join"), ExposureInternal, FlowKindContinue},
		{Type("conference_join"), ExposureCore, FlowKindContinue},
		{Type("connect"), ExposureSensitive, FlowKindContinue},
		{Type("conversation_send"), ExposureSensitive, FlowKindContinue},
		{Type("digits_receive"), ExposureCore, FlowKindContinue},
		{Type("digits_send"), ExposureCore, FlowKindContinue},
		{Type("echo"), ExposureInternal, FlowKindContinue},
		{Type("email_send"), ExposureSensitive, FlowKindContinue},
		{Type("external_media_start"), ExposureInternal, FlowKindContinue},
		{Type("external_media_stop"), ExposureInternal, FlowKindContinue},
		{Type("fetch"), ExposureInternal, FlowKindContinue},
		{Type("fetch_flow"), ExposureCore, FlowKindContinue},
		{Type("goto"), ExposureInternal, FlowKindContinue},
		{Type("hangup"), ExposureCore, FlowKindTerminate},
		{Type("message_send"), ExposureSensitive, FlowKindContinue},
		{Type("mute"), ExposureCore, FlowKindContinue},
		{Type("play"), ExposureCore, FlowKindContinue},
		{Type("queue_join"), ExposureCore, FlowKindContinue},
		{Type("recording_start"), ExposureCore, FlowKindContinue},
		{Type("recording_stop"), ExposureCore, FlowKindContinue},
		{Type("sleep"), ExposureCore, FlowKindContinue},
		{Type("stop"), ExposureCore, FlowKindTerminate},
		{Type("stream_echo"), ExposureInternal, FlowKindContinue},
		{Type("talk"), ExposureCore, FlowKindContinue},
		{Type("transcribe_recording"), ExposureSensitive, FlowKindContinue},
		{Type("transcribe_start"), ExposureSensitive, FlowKindContinue},
		{Type("transcribe_stop"), ExposureCore, FlowKindContinue},
		{Type("variable_set"), ExposureCore, FlowKindContinue},
		{Type("webhook_send"), ExposureSensitive, FlowKindContinue},
	}

	if len(tests) != len(MetaByType) {
		t.Errorf("Wrong match. the golden table has %d rows, MetaByType has %d: add or remove the row of the changed type", len(tests), len(MetaByType))
	}
	for _, tt := range tests {
		t.Run(string(tt.ty), func(t *testing.T) {
			m, ok := MetaByType[tt.ty]
			if !ok {
				t.Fatalf("Wrong match. %s is not in MetaByType", tt.ty)
			}
			if m.Exposure != tt.exp || m.Flow != tt.flow {
				t.Errorf("Wrong match.\nexpect: %s/%s\ngot: %s/%s", tt.exp, tt.flow, m.Exposure, m.Flow)
			}
		})
	}
}
