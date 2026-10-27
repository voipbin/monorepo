package messagehandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	commonerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

type covMocks struct {
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
	req    *requesthandler.MockRequestHandler
}

// newCovHandler builds a handler on strict mocks: any call that a test did
// not declare fails the test.
func newCovHandler(t *testing.T) (*messageHandler, *covMocks) {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &covMocks{
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
	}
	h := &messageHandler{
		db:            m.db,
		notifyHandler: m.notify,
		reqHandler:    m.req,
		utilHandler:   utilhandler.NewUtilHandler(),
	}
	return h, m
}

func Test_Create_dbErrors(t *testing.T) {
	errBoom := errors.New("boom")

	tests := []struct {
		name string

		createErr error
		getErr    error
		expectGet bool
	}{
		{name: "insert fails", createErr: errBoom},
		{name: "read back fails", getErr: errBoom, expectGet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.expectGet {
				m.db.EXPECT().MessageGet(ctx, gomock.Any()).Return(nil, tt.getErr)
			}
			// no webhook may be published on failure

			res, err := h.Create(ctx, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, message.DirectionIncoming, message.RoleUser, "hi", nil, "")
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			if !errors.Is(err, errBoom) {
				t.Errorf("error = %v, expect: %v", err, errBoom)
			}
		})
	}
}

func Test_Get_errorMapping(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	errBoom := errors.New("boom")

	tests := []struct {
		name string

		dbErr error

		expectNotFound bool
		expectMsg      string
	}{
		{name: "not found becomes NotFound", dbErr: dbhandler.ErrNotFound, expectNotFound: true},
		{name: "other error is wrapped", dbErr: errBoom, expectMsg: "could not get data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			m.db.EXPECT().MessageGet(ctx, id).Return(nil, tt.dbErr)

			res, err := h.Get(ctx, id)
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if !errors.Is(err, tt.dbErr) {
				t.Errorf("cause lost: %v", err)
			}
			var ve *commonerrors.VoipbinError
			isVoipbinNotFound := errors.As(err, &ve) && ve.Status == commonerrors.StatusNotFound && ve.Reason == "MESSAGE_NOT_FOUND"
			if tt.expectNotFound != isVoipbinNotFound {
				t.Errorf("not-found mapping = %v, expect: %v (err: %v)", isVoipbinNotFound, tt.expectNotFound, err)
			}
			if tt.expectMsg != "" && !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
		})
	}
}

func Test_List_dbError(t *testing.T) {
	h, m := newCovHandler(t)
	ctx := context.Background()
	errBoom := errors.New("boom")
	filters := map[message.Field]any{message.FieldAIcallID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")}

	m.db.EXPECT().MessageList(ctx, uint64(10), "tok", filters).Return(nil, errBoom)

	res, err := h.List(ctx, 10, "tok", filters)
	if res != nil || !errors.Is(err, errBoom) {
		t.Errorf("result = %v, err = %v", res, err)
	}
}

func Test_NewMessageHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	req := requesthandler.NewMockRequestHandler(mc)
	notify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewMockDBHandler(mc)

	res := NewMessageHandler(req, notify, db, nil, nil, nil)
	impl, ok := res.(*messageHandler)
	if !ok {
		t.Fatalf("unexpected type %T", res)
	}
	if impl.reqHandler != req || impl.notifyHandler != notify || impl.db != db || impl.utilHandler == nil {
		t.Errorf("dependencies not wired: %+v", impl)
	}
}

func Test_ResolveOriginForTest(t *testing.T) {
	tests := []struct {
		name string
		opts []CreateOption

		expect message.Origin
	}{
		{name: "no options", expect: message.OriginNone},
		{name: "proactive", opts: []CreateOption{WithOrigin(message.OriginProactive)}, expect: message.OriginProactive},
		{name: "last option wins", opts: []CreateOption{WithOrigin(message.OriginProactive), WithOrigin(message.OriginNone)}, expect: message.OriginNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveOriginForTest(tt.opts...); got != tt.expect {
				t.Errorf("origin = %q, expect: %q", got, tt.expect)
			}
		})
	}
}

func Test_EventPMMessageUserTranscription_createFails(t *testing.T) {
	h, m := newCovHandler(t)
	h.reqHandler = nil
	ctx := context.Background()

	m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(errors.New("boom"))
	// the failed insert must not be read back or published (strict mocks)

	h.EventPMMessageUserTranscription(ctx, &pmmessage.Message{
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		Text:                     "hello",
	})
}

func Test_EventPMMessageUserLLM_createFails(t *testing.T) {
	h, m := newCovHandler(t)
	ctx := context.Background()
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	m.req.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(nil, errors.New("rpc down"))
	m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(errors.New("boom"))

	h.EventPMMessageUserLLM(ctx, &pmmessage.Message{PipecatcallReferenceID: aicallID, Text: "hello"})
}

func Test_EventPMMessageBotLLM_createFails(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	pcID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	errBoom := errors.New("boom")

	tests := []struct {
		name string

		refType pmpipecatcall.ReferenceType
		aicall  *aicall.AIcall // nil means no aicall lookup is expected
	}{
		{name: "non aicall reference", refType: pmpipecatcall.ReferenceTypeCall},
		{
			name:    "voice aicall",
			refType: pmpipecatcall.ReferenceTypeAICall,
			aicall:  &aicall.AIcall{Identity: identity.Identity{ID: aicallID}, ReferenceType: aicall.ReferenceTypeCall, PipecatcallID: pcID},
		},
		{
			name:    "conversation aicall",
			refType: pmpipecatcall.ReferenceTypeAICall,
			aicall:  &aicall.AIcall{Identity: identity.Identity{ID: aicallID}, ReferenceType: aicall.ReferenceTypeConversation, PipecatcallID: pcID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			if tt.aicall != nil {
				m.req.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(tt.aicall, nil)
			}
			m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(errBoom)
			// after a failed insert: no read back, no second aicall lookup,
			// no conversation send, no webhook (strict mocks)

			h.EventPMMessageBotLLM(ctx, &pmmessage.Message{
				PipecatcallID:            pcID,
				PipecatcallReferenceType: tt.refType,
				PipecatcallReferenceID:   aicallID,
				Text:                     "reply",
			})
		})
	}
}

func Test_EventPMMessageBotLLM_foreignReReadFails(t *testing.T) {
	h, m := newCovHandler(t)
	ctx := context.Background()
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	m.req.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(&aicall.AIcall{
		Identity:      identity.Identity{ID: aicallID},
		ReferenceType: aicall.ReferenceTypeContactCase,
		PipecatcallID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
	}, nil)
	m.req.EXPECT().AIV1AIcallGetSkipCache(ctx, aicallID).Return(nil, errors.New("rpc down"))
	// the reply is dropped: nothing is persisted (strict mocks)

	before := testutil.ToFloat64(promForeignPipecatcallDroppedTotal.WithLabelValues("bot_llm"))

	h.EventPMMessageBotLLM(ctx, &pmmessage.Message{
		PipecatcallID:            uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		Text:                     "reply",
	})

	after := testutil.ToFloat64(promForeignPipecatcallDroppedTotal.WithLabelValues("bot_llm"))
	if after-before != 1 {
		t.Errorf("dropped counter moved by %v, expect: 1", after-before)
	}
}

func Test_EventPMMessageBotLLMIntermediate_aicallResolution(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	boundPC := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	otherPC := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	aiID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	customerID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")

	tests := []struct {
		name string

		noRequestHandler bool
		aicall           *aicall.AIcall
		aicallErr        error
		eventPC          uuid.UUID

		expectPublish  bool
		expectActiveAI uuid.UUID
		expectDropped  bool
	}{
		{name: "no request handler publishes without active ai", noRequestHandler: true, eventPC: boundPC, expectPublish: true, expectActiveAI: uuid.Nil},
		{name: "aicall lookup failure still publishes", aicallErr: errors.New("rpc down"), eventPC: boundPC, expectPublish: true, expectActiveAI: uuid.Nil},
		{
			name:           "contact case with the bound pipecatcall publishes with the active ai",
			aicall:         &aicall.AIcall{ReferenceType: aicall.ReferenceTypeContactCase, AssistanceType: aicall.AssistanceTypeAI, AssistanceID: aiID, PipecatcallID: boundPC},
			eventPC:        boundPC,
			expectPublish:  true,
			expectActiveAI: aiID,
		},
		{
			name:          "contact case with a foreign pipecatcall is dropped",
			aicall:        &aicall.AIcall{ReferenceType: aicall.ReferenceTypeContactCase, AssistanceType: aicall.AssistanceTypeAI, AssistanceID: aiID, PipecatcallID: boundPC},
			eventPC:       otherPC,
			expectDropped: true,
		},
		{
			name:           "other reference types are never treated as foreign",
			aicall:         &aicall.AIcall{ReferenceType: aicall.ReferenceTypeConversation, AssistanceType: aicall.AssistanceTypeAI, AssistanceID: aiID, PipecatcallID: boundPC},
			eventPC:        otherPC,
			expectPublish:  true,
			expectActiveAI: aiID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			if tt.noRequestHandler {
				h.reqHandler = nil
			} else {
				m.req.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(tt.aicall, tt.aicallErr)
			}

			var published *message.IntermediateWebhookMessage
			if tt.expectPublish {
				m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageIntermediate, gomock.Any()).
					Do(func(_ context.Context, _ uuid.UUID, _ string, data notifyhandler.WebhookEventMessage) {
						published, _ = data.(*message.IntermediateWebhookMessage)
					})
			}

			before := testutil.ToFloat64(promForeignPipecatcallDroppedTotal.WithLabelValues("bot_llm_intermediate"))

			evt := &pmmessage.Message{
				ID:                       uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
				PipecatcallID:            tt.eventPC,
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   aicallID,
				Text:                     "delta",
				Sequence:                 3,
			}
			evt.CustomerID = customerID
			h.EventPMMessageBotLLMIntermediate(ctx, evt)

			dropped := testutil.ToFloat64(promForeignPipecatcallDroppedTotal.WithLabelValues("bot_llm_intermediate")) - before
			if tt.expectDropped != (dropped == 1) {
				t.Errorf("dropped counter moved by %v, expect dropped: %v", dropped, tt.expectDropped)
			}
			if !tt.expectPublish {
				return
			}
			if published == nil {
				t.Fatal("no intermediate message captured")
			}
			if published.ActiveAIID != tt.expectActiveAI || published.Content != "delta" || published.Sequence != 3 || published.AIcallID != aicallID {
				t.Errorf("published = %+v", published)
			}
		})
	}
}

func Test_EventPMTeamMemberSwitched_createFails(t *testing.T) {
	h, m := newCovHandler(t)
	ctx := context.Background()
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	// both member lookups fail softly, then the notification insert fails
	m.req.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(nil, errors.New("rpc down")).Times(2)
	m.db.EXPECT().MessageCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, msg *message.Message) error {
		if msg.Role != message.RoleNotification || msg.ActiveAIID != uuid.Nil || !strings.Contains(msg.Content, `"type":"member_switched"`) {
			t.Errorf("unexpected notification row: %+v", msg)
		}
		return errors.New("boom")
	})
	// no participant write and no webhook after a failed insert (strict mocks; participantHandler is nil)

	h.EventPMTeamMemberSwitched(ctx, &pmmessage.MemberSwitchedEvent{
		PipecatcallReferenceID: aicallID,
		TransitionFunctionName: "escalate",
	})
}
