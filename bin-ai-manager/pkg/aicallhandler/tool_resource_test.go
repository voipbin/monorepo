package aicallhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	amsummary "monorepo/bin-ai-manager/models/summary"
)

const (
	trCustomerID = "22222222-0000-4000-8000-000000000001"
	trForeignID  = "22222222-0000-4000-8000-000000000002"
	trResourceID = "44444444-0000-4000-8000-000000000001"
)

func trNewAicall() *aicall.AIcall {
	return &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil(trCustomerID),
		},
	}
}

func trNewTool(args string) *message.ToolCall {
	return &message.ToolCall{
		ID:   "tool-1",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameGetResource,
			Arguments: args,
		},
	}
}

func trTime(s string) *time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return &t
}

func trIdentity(customerID string) commonidentity.Identity {
	return commonidentity.Identity{
		ID:         uuid.FromStringOrNil(trResourceID),
		CustomerID: uuid.FromStringOrNil(customerID),
	}
}

func summaryModelForeign() *amsummary.Summary {
	return &amsummary.Summary{
		Identity: trIdentity(trForeignID),
		Status:   amsummary.StatusDone,
		Content:  "foreign content",
	}
}

func summaryModelOwnWithContent(content string) *amsummary.Summary {
	return &amsummary.Summary{
		Identity: trIdentity(trCustomerID),
		Status:   amsummary.StatusDone,
		Content:  content,
	}
}

// Test_toolHandleGetResource_success exercises the success path per resource
// type and asserts curated output: customer-meaningful fields present,
// internal fields absent.
func Test_toolHandleGetResource_success(t *testing.T) {
	tests := []struct {
		name string
		args string

		mockSetup func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler)

		expectContains    []string
		expectNotContains []string
	}{
		{
			name: "call",
			args: `{"resource_type": "call", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				mockReq.EXPECT().CallV1CallGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(&cmcall.Call{
					Identity:     trIdentity(trCustomerID),
					ChannelID:    "internal-channel-id",
					BridgeID:     "internal-bridge-id",
					Status:       cmcall.StatusHangup,
					Direction:    cmcall.DirectionOutgoing,
					Source:       commonaddress.Address{Target: "+821000000001"},
					Destination:  commonaddress.Address{Target: "+821000000002"},
					HangupBy:     cmcall.HangupByRemote,
					HangupReason: cmcall.HangupReasonNormal,
					TMCreate:     trTime("2026-06-11T01:00:00Z"),
					TMHangup:     trTime("2026-06-11T01:05:00Z"),
				}, nil)
			},
			expectContains:    []string{"status: hangup", "direction: outgoing", "source: +821000000001", "destination: +821000000002", "hangup_by: remote"},
			expectNotContains: []string{"internal-channel-id", "internal-bridge-id"},
		},
		{
			name: "groupcall",
			args: `{"resource_type": "groupcall", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				mockReq.EXPECT().CallV1GroupcallGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(&cmgroupcall.Groupcall{
					Identity:     trIdentity(trCustomerID),
					Status:       cmgroupcall.StatusProgressing,
					RingMethod:   cmgroupcall.RingMethodRingAll,
					Source:       &commonaddress.Address{Target: "+821****0001"},
					Destinations: []commonaddress.Address{{Target: "+821000000002"}, {Target: "+821000000003"}},
					CallIDs:      []uuid.UUID{uuid.FromStringOrNil("55555555-0000-4000-8000-000000000001")},
				}, nil)
			},
			expectContains: []string{"status: progressing", "ring_method: ring_all", "destinations: 2", "55555555-0000-4000-8000-000000000001"},
		},
		{
			name: "recording",
			args: `{"resource_type": "recording", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				mockReq.EXPECT().CallV1RecordingGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(&cmrecording.Recording{
					Identity:      trIdentity(trCustomerID),
					Status:        cmrecording.StatusEnded,
					Format:        cmrecording.FormatWAV,
					RecordingName: "my-recording",
					Filenames:     []string{"/internal/storage/path.wav"},
					AsteriskID:    "internal-asterisk-id",
					TMStart:       trTime("2026-06-11T01:00:00Z"),
					TMEnd:         trTime("2026-06-11T01:03:00Z"),
				}, nil)
			},
			expectContains:    []string{"recording_name: my-recording", "format: wav"},
			expectNotContains: []string{"/internal/storage/path.wav", "internal-asterisk-id"},
		},
		{
			name: "transcribe with transcripts",
			args: `{"resource_type": "transcribe", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				mockReq.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(&tmtranscribe.Transcribe{
					Identity: trIdentity(trCustomerID),
					Status:   tmtranscribe.StatusDone,
					Language: "en-US",
					HostID:   uuid.FromStringOrNil("66666666-0000-4000-8000-000000000001"),
				}, nil)
				// dbhandler order: tm_create DESC (most recent first)
				offset1 := time.Time{}.Add(3 * time.Second)
				offset2 := time.Time{}.Add(8 * time.Second)
				mockReq.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
					tmtranscript.FieldTranscribeID: uuid.FromStringOrNil(trResourceID),
					tmtranscript.FieldDeleted:      false,
				}).Return([]tmtranscript.Transcript{
					{Direction: tmtranscript.DirectionOut, Message: "how can I help", TMTranscript: &offset2},
					{Direction: tmtranscript.DirectionIn, Message: "hello there", TMTranscript: &offset1},
					{Direction: tmtranscript.DirectionIn, Message: "very first words", TMTranscript: nil},
				}, nil)
			},
			// chronological order after reversal; nil TMTranscript renders --:--:--
			expectContains:    []string{"status: done", "language: en-US", "[in --:--:--] very first words", "[in 00:00:03] hello there", "[out 00:00:08] how can I help"},
			expectNotContains: []string{"66666666-0000-4000-8000-000000000001"},
		},
		{
			name: "summary",
			args: `{"resource_type": "summary", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				s := summaryModelOwnWithContent("the caller asked about billing")
				s.Language = "en-US"
				mockReq.EXPECT().AIV1SummaryGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(s, nil)
			},
			expectContains: []string{"status: done", "language: en-US", "content: the caller asked about billing"},
		},
		{
			name: "aicall with conversation history",
			args: `{"resource_type": "aicall", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(&aicall.AIcall{
					Identity:      trIdentity(trCustomerID),
					Status:        aicall.StatusTerminated,
					ReferenceType: aicall.ReferenceTypeCall,
				}, nil)
				// dbhandler order: tm_create DESC (most recent first)
				mockMsg.EXPECT().List(gomock.Any(), uint64(resourceListPageSize+1), "", map[message.Field]any{
					message.FieldAIcallID: uuid.FromStringOrNil(trResourceID),
					message.FieldDeleted:  false,
				}).Return([]*message.Message{
					{Role: message.RoleAssistant, Content: "goodbye"},
					{Role: message.RoleAssistant, Content: "sending it now", ToolCalls: []message.ToolCall{{Function: message.FunctionCall{Name: message.FunctionCallNameSendMessage}}}},
					{Role: message.RoleAssistant, Content: "", ToolCalls: []message.ToolCall{{Function: message.FunctionCall{Name: message.FunctionCallNameSendEmail}}}},
					{Role: message.RoleTool, Content: `{"tool_call_id": "x", "result": "success"}`},
					{Role: message.RoleNotification, Content: "internal notification body"},
					{Role: message.RoleFunction, Content: "function role body"},
					{Role: message.RoleNone, Content: "empty role body"},
					{Role: message.RoleUser, Content: "send me the report"},
					{Role: message.RoleSystem, Content: "SECRET PROMPT SNAPSHOT"},
				}, nil)
			},
			expectContains: []string{
				"status: terminated",
				"[user] send me the report",
				"[assistant called send_email]",
				"[assistant] sending it now",
				"[assistant called send_message]",
				"[assistant] goodbye",
			},
			expectNotContains: []string{"SECRET PROMPT SNAPSHOT", "tool_call_id", "internal notification body", "function role body", "empty role body"},
		},
		{
			name: "conferencecall",
			args: `{"resource_type": "conferencecall", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				mockReq.EXPECT().ConferenceV1ConferencecallGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(&cfconferencecall.Conferencecall{
					Identity:     trIdentity(trCustomerID),
					Status:       cfconferencecall.StatusJoined,
					ConferenceID: uuid.FromStringOrNil("77777777-0000-4000-8000-000000000001"),
				}, nil)
			},
			expectContains: []string{"status: joined", "conference_id: 77777777-0000-4000-8000-000000000001"},
		},
		{
			name: "queuecall with nil timestamps omitted",
			args: `{"resource_type": "queuecall", "resource_id": "` + trResourceID + `"}`,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler, mockMsg *messagehandler.MockMessageHandler) {
				mockReq.EXPECT().QueueV1QueuecallGet(gomock.Any(), uuid.FromStringOrNil(trResourceID)).Return(&qmqueuecall.Queuecall{
					Identity:        trIdentity(trCustomerID),
					Status:          qmqueuecall.StatusDone,
					QueueID:         uuid.FromStringOrNil("88888888-0000-4000-8000-000000000001"),
					DurationWaiting: 12000,
					// TMService / TMEnd nil: must be omitted
				}, nil)
			},
			expectContains:    []string{"status: done", "duration_waiting_ms: 12000"},
			expectNotContains: []string{"serviced:", "ended:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockMsg := messagehandler.NewMockMessageHandler(mc)
			h := &aicallHandler{reqHandler: mockReq, messageHandler: mockMsg}

			tt.mockSetup(mockReq, mockMsg)

			res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(tt.args))

			if res.Result != "success" {
				t.Fatalf("expected success, got %s (message: %s)", res.Result, res.Message)
			}
			for _, want := range tt.expectContains {
				if !strings.Contains(res.Message, want) {
					t.Errorf("expected summary to contain %q. got:\n%s", want, res.Message)
				}
			}
			for _, ban := range tt.expectNotContains {
				if strings.Contains(res.Message, ban) {
					t.Errorf("summary must NOT contain %q. got:\n%s", ban, res.Message)
				}
			}
		})
	}
}

// Test_toolHandleGetResource_maskingInvariant locks the existence-oracle
// property per resource type: absent and cross-customer produce
// byte-identical messageContent for the same fixed inputs.
func Test_toolHandleGetResource_maskingInvariant(t *testing.T) {
	rid := uuid.FromStringOrNil(trResourceID)

	// per-type EXPECT setups: absent (ErrNotFound) and foreign owner
	types := []struct {
		resourceType string
		setupAbsent  func(mockReq *requesthandler.MockRequestHandler)
		setupForeign func(mockReq *requesthandler.MockRequestHandler)
	}{
		{
			"call",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().CallV1CallGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().CallV1CallGet(gomock.Any(), rid).Return(&cmcall.Call{Identity: trIdentity(trForeignID)}, nil)
			},
		},
		{
			"groupcall",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().CallV1GroupcallGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().CallV1GroupcallGet(gomock.Any(), rid).Return(&cmgroupcall.Groupcall{Identity: trIdentity(trForeignID)}, nil)
			},
		},
		{
			"recording",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().CallV1RecordingGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().CallV1RecordingGet(gomock.Any(), rid).Return(&cmrecording.Recording{Identity: trIdentity(trForeignID)}, nil)
			},
		},
		{
			"transcribe",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				// NOTE: no TranscribeV1TranscriptList EXPECT — strict gomock
				// fails the test if the enrichment fetch runs for a foreign
				// transcribe (post-ownership contract).
				m.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), rid).Return(&tmtranscribe.Transcribe{Identity: trIdentity(trForeignID)}, nil)
			},
		},
		{
			"summary",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1SummaryGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1SummaryGet(gomock.Any(), rid).Return(summaryModelForeign(), nil)
			},
		},
		{
			"aicall",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				// NOTE: no messageHandler.List EXPECT — strict gomock fails
				// if the message fetch runs for a foreign aicall.
				m.EXPECT().AIV1AIcallGet(gomock.Any(), rid).Return(&aicall.AIcall{Identity: trIdentity(trForeignID)}, nil)
			},
		},
		{
			"conferencecall",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().ConferenceV1ConferencecallGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().ConferenceV1ConferencecallGet(gomock.Any(), rid).Return(&cfconferencecall.Conferencecall{Identity: trIdentity(trForeignID)}, nil)
			},
		},
		{
			"queuecall",
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().QueueV1QueuecallGet(gomock.Any(), rid).Return(nil, requesthandler.ErrNotFound)
			},
			func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().QueueV1QueuecallGet(gomock.Any(), rid).Return(&qmqueuecall.Queuecall{Identity: trIdentity(trForeignID)}, nil)
			},
		},
	}

	for _, tc := range types {
		t.Run(tc.resourceType, func(t *testing.T) {
			args := `{"resource_type": "` + tc.resourceType + `", "resource_id": "` + trResourceID + `"}`

			run := func(setup func(*requesthandler.MockRequestHandler)) *messageContent {
				mc := gomock.NewController(t)
				defer mc.Finish()
				mockReq := requesthandler.NewMockRequestHandler(mc)
				mockMsg := messagehandler.NewMockMessageHandler(mc)
				h := &aicallHandler{reqHandler: mockReq, messageHandler: mockMsg}
				setup(mockReq)
				return h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(args))
			}

			resAbsent := run(tc.setupAbsent)
			resForeign := run(tc.setupForeign)

			// Both must be the masked success response.
			if resAbsent.Result != "success" || resAbsent.Message != msgResourceNotFound {
				t.Fatalf("absent path not masked: %+v", resAbsent)
			}
			// Byte-identical: same fixed tc.ID + resource_type + resource_id.
			if !messageContentEqual(resAbsent, resForeign) {
				t.Errorf("masking invariant violated.\nabsent:  %+v\nforeign: %+v", resAbsent, resForeign)
			}
			// IDOR fall-through guard: no summary fragment in the foreign response.
			if strings.Contains(resForeign.Message, "status:") {
				t.Errorf("foreign response leaked a summary fragment: %s", resForeign.Message)
			}
		})
	}
}

func messageContentEqual(a, b *messageContent) bool {
	return reflect.DeepEqual(a, b)
}

// Test_toolHandleGetResource_errors covers transient failure, argument
// validation, and graceful enrichment degradation.
func Test_toolHandleGetResource_errors(t *testing.T) {
	rid := uuid.FromStringOrNil(trResourceID)

	t.Run("transient RPC error is honest, not masked", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: messagehandler.NewMockMessageHandler(mc)}

		mockReq.EXPECT().CallV1CallGet(gomock.Any(), rid).Return(nil, stderrors.New("rabbitmq timeout"))

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "call", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "failed" {
			t.Fatalf("expected failed, got %+v", res)
		}
		if res.Message != "resource lookup failed" {
			t.Errorf("expected sanitized failure message, got %q", res.Message)
		}
	})

	t.Run("unsupported resource_type", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h := &aicallHandler{reqHandler: requesthandler.NewMockRequestHandler(mc), messageHandler: messagehandler.NewMockMessageHandler(mc)}

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "channel", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "failed" || !strings.Contains(res.Message, "unsupported resource_type") || !strings.Contains(res.Message, "transcribe") {
			t.Errorf("expected unsupported-type failure listing supported set, got %+v", res)
		}
	})

	t.Run("empty resource_type", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h := &aicallHandler{reqHandler: requesthandler.NewMockRequestHandler(mc), messageHandler: messagehandler.NewMockMessageHandler(mc)}

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_id": "`+trResourceID+`"}`))
		if res.Result != "failed" || !strings.Contains(res.Message, "resource_type is required") {
			t.Errorf("expected resource_type-required failure, got %+v", res)
		}
	})

	t.Run("invalid resource_id", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h := &aicallHandler{reqHandler: requesthandler.NewMockRequestHandler(mc), messageHandler: messagehandler.NewMockMessageHandler(mc)}

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "call", "resource_id": "not-a-uuid"}`))
		if res.Result != "failed" || res.Message != "invalid resource_id" {
			t.Errorf("expected invalid resource_id failure, got %+v", res)
		}
	})

	t.Run("malformed JSON args", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h := &aicallHandler{reqHandler: requesthandler.NewMockRequestHandler(mc), messageHandler: messagehandler.NewMockMessageHandler(mc)}

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{not json`))
		if res.Result != "failed" || !strings.Contains(res.Message, "invalid arguments") {
			t.Errorf("expected invalid-arguments failure, got %+v", res)
		}
	})

	t.Run("transcript list failure degrades gracefully", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: messagehandler.NewMockMessageHandler(mc)}

		mockReq.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), rid).Return(&tmtranscribe.Transcribe{
			Identity: trIdentity(trCustomerID),
			Status:   tmtranscribe.StatusDone,
		}, nil)
		mockReq.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(resourceListPageSize+1), gomock.Any()).Return(nil, stderrors.New("boom"))

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "transcribe", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" || !strings.Contains(res.Message, "(transcripts unavailable)") {
			t.Errorf("expected graceful degradation, got %+v", res)
		}
	})

	t.Run("aicall message list failure degrades gracefully", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockMsg := messagehandler.NewMockMessageHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: mockMsg}

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), rid).Return(&aicall.AIcall{Identity: trIdentity(trCustomerID), Status: aicall.StatusTerminated}, nil)
		mockMsg.EXPECT().List(gomock.Any(), gomock.Any(), "", gomock.Any()).Return(nil, stderrors.New("db down"))

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "aicall", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" || !strings.Contains(res.Message, "(messages unavailable)") {
			t.Errorf("expected graceful degradation, got %+v", res)
		}
	})

	t.Run("empty transcript list renders no transcripts", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: messagehandler.NewMockMessageHandler(mc)}

		mockReq.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), rid).Return(&tmtranscribe.Transcribe{Identity: trIdentity(trCustomerID)}, nil)
		mockReq.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(resourceListPageSize+1), gomock.Any()).Return([]tmtranscript.Transcript{}, nil)

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "transcribe", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" || !strings.Contains(res.Message, "(no transcripts)") {
			t.Errorf("expected (no transcripts), got %+v", res)
		}
	})

	t.Run("nil owner customer id fails closed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: messagehandler.NewMockMessageHandler(mc)}

		// resource with unset customer_id; caller also crafted to Nil must NOT match
		mockReq.EXPECT().CallV1CallGet(gomock.Any(), rid).Return(&cmcall.Call{}, nil)

		caller := trNewAicall()
		caller.CustomerID = uuid.Nil
		res := h.toolHandleGetResource(context.Background(), caller, trNewTool(`{"resource_type": "call", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" || res.Message != msgResourceNotFound {
			t.Errorf("expected masked fail-closed response, got %+v", res)
		}
	})
}

// Test_toolHandleGetResource_truncation locks the cap mechanics: most recent
// lines kept, marker at the top of the block.
func Test_toolHandleGetResource_truncation(t *testing.T) {
	rid := uuid.FromStringOrNil(trResourceID)

	t.Run("long summary content is hard-truncated with marker", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: messagehandler.NewMockMessageHandler(mc)}

		long := strings.Repeat("x", maxResourceSummaryRunes+500)
		mockReq.EXPECT().AIV1SummaryGet(gomock.Any(), rid).Return(summaryModelOwnWithContent(long), nil)

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "summary", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" {
			t.Fatalf("expected success, got %+v", res)
		}
		if got := len([]rune(res.Message)); got > maxResourceSummaryRunes {
			t.Errorf("message exceeds cap: %d > %d", got, maxResourceSummaryRunes)
		}
		if !strings.Contains(res.Message, "...(truncated)") {
			t.Errorf("expected truncation marker, got tail: %s", res.Message[len(res.Message)-40:])
		}
	})

	t.Run("aicall message page cap keeps most recent with marker", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockMsg := messagehandler.NewMockMessageHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: mockMsg}

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), rid).Return(&aicall.AIcall{Identity: trIdentity(trCustomerID)}, nil)

		// 101 rows returned (DESC: index 0 is the newest) → page cap fires.
		msgs := make([]*message.Message, resourceListPageSize+1)
		for i := range msgs {
			msgs[i] = &message.Message{Role: message.RoleUser, Content: "m" + string(rune('A'+i%26))}
		}
		msgs[0].Content = "NEWEST"
		msgs[len(msgs)-1].Content = "OLDEST"
		mockMsg.EXPECT().List(gomock.Any(), uint64(resourceListPageSize+1), "", gomock.Any()).Return(msgs, nil)

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "aicall", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" {
			t.Fatalf("expected success, got %+v", res)
		}
		if !strings.Contains(res.Message, "earlier messages omitted; showing the most recent") {
			t.Errorf("expected page-cap marker, got:\n%s", res.Message)
		}
		if !strings.Contains(res.Message, "NEWEST") {
			t.Errorf("newest message must be kept")
		}
		if strings.Contains(res.Message, "OLDEST") {
			t.Errorf("oldest (101st) message must be dropped by the page cap")
		}
		// marker must precede the kept lines (top of block)
		if strings.Index(res.Message, "omitted") > strings.Index(res.Message, "NEWEST") {
			t.Errorf("marker must be at the top of the block")
		}
	})
}

// Test_renderBodyLines_capMechanics locks the rune-budget truncation
// arithmetic directly: most-recent-kept line dropping, top-of-block marker,
// the degenerate single-line hard-cut, and the absolute cap guarantee.
func Test_renderBodyLines_capMechanics(t *testing.T) {
	t.Run("rune budget drops oldest lines, marker at top", func(t *testing.T) {
		header := "status: done"
		// 200 lines x ~30 runes ≈ 6000 runes > 4000 cap
		lines := make([]string, 200)
		for i := range lines {
			lines[i] = strings.Repeat("x", 25) + "L" + fmt.Sprintf("%03d", i)
		}
		out := renderBodyLines(header, lines, false, "messages")

		if got := len([]rune(out)); got > maxResourceSummaryRunes {
			t.Fatalf("output exceeds cap: %d > %d", got, maxResourceSummaryRunes)
		}
		if !strings.Contains(out, "earlier messages omitted; showing the most recent") {
			t.Fatalf("expected truncation marker, got head: %.120s", out)
		}
		if !strings.Contains(out, "L199") {
			t.Errorf("newest line must be kept")
		}
		if strings.Contains(out, "L000") {
			t.Errorf("oldest line must be dropped")
		}
		if strings.Index(out, "omitted") > strings.Index(out, "L199") {
			t.Errorf("marker must precede the kept lines")
		}
	})

	t.Run("degenerate single oversized line is hard-cut and capped", func(t *testing.T) {
		header := "status: done"
		lines := []string{strings.Repeat("y", maxResourceSummaryRunes*2)}
		out := renderBodyLines(header, lines, false, "transcripts")

		if got := len([]rune(out)); got > maxResourceSummaryRunes {
			t.Fatalf("degenerate output exceeds cap: %d > %d", got, maxResourceSummaryRunes)
		}
		if !strings.Contains(out, "showing the most recent 1, truncated") {
			t.Errorf("expected degenerate marker, got head: %.150s", out)
		}
		if !strings.Contains(out, "yyyy") {
			t.Errorf("hard-cut line content must still be shown")
		}
	})

	t.Run("everything fits, no marker", func(t *testing.T) {
		out := renderBodyLines("h: v", []string{"[user] hi", "[assistant] hello"}, false, "messages")
		if strings.Contains(out, "omitted") {
			t.Errorf("no marker expected when everything fits: %s", out)
		}
		if out != "h: v\n[user] hi\n[assistant] hello" {
			t.Errorf("unexpected output: %q", out)
		}
	})
}

// Test_toolHandleGetResource_transcribePageCap exercises the transcribe leg of
// the page-cap detection (duplicated per type, so the aicall test does not
// protect this copy).
func Test_toolHandleGetResource_transcribePageCap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq, messageHandler: messagehandler.NewMockMessageHandler(mc)}

	rid := uuid.FromStringOrNil(trResourceID)
	mockReq.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), rid).Return(&tmtranscribe.Transcribe{Identity: trIdentity(trCustomerID)}, nil)

	// 101 rows (DESC: index 0 newest) → page cap fires.
	rows := make([]tmtranscript.Transcript, resourceListPageSize+1)
	for i := range rows {
		rows[i] = tmtranscript.Transcript{Direction: tmtranscript.DirectionIn, Message: "t" + fmt.Sprintf("%03d", i)}
	}
	rows[0].Message = "NEWEST"
	rows[len(rows)-1].Message = "OLDEST"
	mockReq.EXPECT().TranscribeV1TranscriptList(gomock.Any(), "", uint64(resourceListPageSize+1), gomock.Any()).Return(rows, nil)

	res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "transcribe", "resource_id": "`+trResourceID+`"}`))
	if res.Result != "success" {
		t.Fatalf("expected success, got %+v", res)
	}
	if !strings.Contains(res.Message, "earlier transcripts omitted; showing the most recent") {
		t.Errorf("expected page-cap marker, got:\n%.200s", res.Message)
	}
	if !strings.Contains(res.Message, "NEWEST") || strings.Contains(res.Message, "OLDEST") {
		t.Errorf("page cap must keep newest and drop the 101st oldest")
	}
}

// Test_resolveResource_nilResultFailsClosed locks the broken-fetcher-contract
// guard: a fetcher returning (nil, nil) must mask, not panic (round-2 L1).
func Test_resolveResource_nilResultFailsClosed(t *testing.T) {
	h := &aicallHandler{}
	stub := func(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
		return nil, nil
	}
	_, err := h.resolveResource(context.Background(), uuid.FromStringOrNil(trCustomerID), stub, uuid.FromStringOrNil(trResourceID))
	if !stderrors.Is(err, ErrResourceNotAccessible) {
		t.Errorf("expected ErrResourceNotAccessible for nil fetch result, got %v", err)
	}
}

// Test_renderAIcall_emptyRenders locks the empty-render branches: genuine
// "(no messages)" vs pagedOut-but-all-dropped disclosure (round-2 L2).
func Test_renderAIcall_emptyRenders(t *testing.T) {
	rid := uuid.FromStringOrNil(trResourceID)

	t.Run("no messages at all", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockMsg := messagehandler.NewMockMessageHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: mockMsg}

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), rid).Return(&aicall.AIcall{Identity: trIdentity(trCustomerID)}, nil)
		mockMsg.EXPECT().List(gomock.Any(), gomock.Any(), "", gomock.Any()).Return([]*message.Message{}, nil)

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "aicall", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" || !strings.Contains(res.Message, "(no messages)") {
			t.Errorf("expected (no messages), got %+v", res)
		}
	})

	t.Run("paged out but newest page all allowlist-dropped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockMsg := messagehandler.NewMockMessageHandler(mc)
		h := &aicallHandler{reqHandler: mockReq, messageHandler: mockMsg}

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), rid).Return(&aicall.AIcall{Identity: trIdentity(trCustomerID)}, nil)
		// 101 rows, all tool-role → every line dropped, pagedOut true.
		msgs := make([]*message.Message, resourceListPageSize+1)
		for i := range msgs {
			msgs[i] = &message.Message{Role: message.RoleTool, Content: "tool body"}
		}
		mockMsg.EXPECT().List(gomock.Any(), gomock.Any(), "", gomock.Any()).Return(msgs, nil)

		res := h.toolHandleGetResource(context.Background(), trNewAicall(), trNewTool(`{"resource_type": "aicall", "resource_id": "`+trResourceID+`"}`))
		if res.Result != "success" || !strings.Contains(res.Message, "(earlier messages exist beyond the fetched page)") {
			t.Errorf("expected paged-out disclosure, got %+v", res)
		}
		if strings.Contains(res.Message, "(no messages)") {
			t.Errorf("must not claim (no messages) when older rows exist")
		}
	})
}
