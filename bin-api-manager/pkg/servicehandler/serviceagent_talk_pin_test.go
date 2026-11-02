package servicehandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkpin "monorepo/bin-talk-manager/models/pin"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	pinTestPinID    = uuid.FromStringOrNil("6f1c1a52-8d3b-4c1e-9a5e-0b7f2d9a1c10")
	pinTestTargetID = uuid.FromStringOrNil("0d4b7a3e-2f5c-4b8e-8a21-9c3e6f1d7a42")
)

func pinTestTime() *time.Time {
	tm := time.Date(2026, 11, 2, 9, 15, 30, 0, time.UTC)
	return &tm
}

func pinTestPin(targetType tkpin.TargetType, targetID uuid.UUID) *tkpin.Pin {
	return &tkpin.Pin{
		Identity:   commonidentity.Identity{ID: pinTestPinID, CustomerID: h3CustomerID},
		Owner:      commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: h3AgentID},
		ChatID:     h3ResourceID,
		TargetType: targetType,
		TargetID:   targetID,
		TMCreate:   pinTestTime(),
	}
}

func pinTestWM(p *tkpin.Pin) *tkpin.WebhookMessage {
	wm, _ := p.ConvertWebhookMessage()
	return wm
}

func pinTestNilAgent() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{Identity: commonidentity.Identity{ID: uuid.Nil, CustomerID: h3CustomerID}, Permission: amagent.PermissionCustomerAdmin})
}

func Test_ServiceAgentTalkPinCreate(t *testing.T) {
	chatPin := pinTestPin(tkpin.TargetTypeChat, h3ResourceID)
	msgPin := pinTestPin(tkpin.TargetTypeMessage, pinTestTargetID)

	msg := h3TalkMsg()
	msg.ID = pinTestTargetID
	deletedMsg := h3TalkMsg()
	deletedMsg.ID = pinTestTargetID
	deletedMsg.TMDelete = pinTestTime()

	callChat := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkpin.WebhookMessage, error) {
		return h.ServiceAgentTalkPinCreate(ctx, a, tkpin.TargetTypeChat, h3ResourceID)
	}
	callMsg := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkpin.WebhookMessage, error) {
		return h.ServiceAgentTalkPinCreate(ctx, a, tkpin.TargetTypeMessage, pinTestTargetID)
	}

	h3RunRes(t, []h3ResRow[*tkpin.WebhookMessage]{
		h3TalkAuthRow(callChat),
		{name: "agent without an id", a: pinTestNilAgent(), call: callChat, wantErr: serviceerrors.ErrPermissionDenied},
		{
			name: "unknown target type", a: h3Admin(), wantErr: serviceerrors.ErrInvalidArgument,
			call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkpin.WebhookMessage, error) {
				return h.ServiceAgentTalkPinCreate(ctx, a, tkpin.TargetType("call"), h3ResourceID)
			},
		},
		{
			name: "nil target id", a: h3Admin(), wantErr: serviceerrors.ErrInvalidArgument,
			call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkpin.WebhookMessage, error) {
				return h.ServiceAgentTalkPinCreate(ctx, a, tkpin.TargetTypeChat, uuid.Nil)
			},
		},
		{name: "chat without access", a: h3Admin(), call: callChat, wantErr: serviceerrors.ErrPermissionDenied, setup: h3TalkAccessDenied},
		{
			name: "chat pin", a: h3Admin(), call: callChat, want: pinTestWM(chatPin),
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkAccessOK(m, ctx)
				m.EXPECT().TalkV1PinCreate(ctx, h3CustomerID, "agent", h3AgentID, tkpin.TargetTypeChat, h3ResourceID).Return(chatPin, nil)
			},
		},
		{
			name: "chat pin rpc error", a: h3Admin(), call: callChat, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkAccessOK(m, ctx)
				m.EXPECT().TalkV1PinCreate(ctx, h3CustomerID, "agent", h3AgentID, tkpin.TargetTypeChat, h3ResourceID).Return(nil, errH3)
			},
		},
		{
			name: "message lookup error", a: h3Admin(), call: callMsg, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, pinTestTargetID).Return(nil, errH3)
			},
		},
		{
			name: "deleted message is not found", a: h3Admin(), call: callMsg, wantErr: serviceerrors.ErrNotFound,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, pinTestTargetID).Return(deletedMsg, nil)
			},
		},
		{
			name: "message in a chat without access", a: h3Admin(), call: callMsg, wantErr: serviceerrors.ErrPermissionDenied,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, pinTestTargetID).Return(msg, nil)
				h3TalkAccessDenied(m, ctx)
			},
		},
		{
			name: "message pin", a: h3Admin(), call: callMsg, want: pinTestWM(msgPin),
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, pinTestTargetID).Return(msg, nil)
				h3TalkAccessOK(m, ctx)
				m.EXPECT().TalkV1PinCreate(ctx, h3CustomerID, "agent", h3AgentID, tkpin.TargetTypeMessage, pinTestTargetID).Return(msgPin, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkPinList(t *testing.T) {
	msg := h3TalkMsg()
	msgPin := pinTestPin(tkpin.TargetTypeMessage, msg.ID)
	msgPin.Message = msg
	chatPin := pinTestPin(tkpin.TargetTypeChat, h3ResourceID)

	call := func(targetType tkpin.TargetType, chatID uuid.UUID, size uint64, token string) func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tkpin.WebhookMessage, error) {
		return func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tkpin.WebhookMessage, error) {
			return h.ServiceAgentTalkPinList(ctx, a, targetType, chatID, size, token)
		}
	}

	h3RunRes(t, []h3ResRow[[]*tkpin.WebhookMessage]{
		h3TalkAuthRow(call("", uuid.Nil, 10, "tok")),
		{name: "agent without an id", a: pinTestNilAgent(), call: call("", uuid.Nil, 10, "tok"), wantErr: serviceerrors.ErrPermissionDenied},
		{name: "unknown target type", a: h3Admin(), call: call(tkpin.TargetType("call"), uuid.Nil, 10, "tok"), wantErr: serviceerrors.ErrInvalidArgument},
		{
			name: "rpc error", a: h3Admin(), call: call("", uuid.Nil, 10, "tok"), wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1PinList(ctx, h3CustomerID, "agent", h3AgentID, tkpin.TargetType(""), uuid.Nil, "tok", uint64(10)).Return(nil, errH3)
			},
		},
		{
			name: "filters are passed and the message is converted", a: h3Admin(),
			call: call(tkpin.TargetTypeMessage, h3ResourceID, 5, "tok"),
			want: []*tkpin.WebhookMessage{pinTestWM(msgPin), pinTestWM(chatPin)},
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1PinList(ctx, h3CustomerID, "agent", h3AgentID, tkpin.TargetTypeMessage, h3ResourceID, "tok", uint64(5)).Return([]*tkpin.Pin{msgPin, chatPin}, nil)
			},
		},
		{
			name: "empty list is an empty slice", a: h3Admin(), call: call("", uuid.Nil, 10, "tok"), want: []*tkpin.WebhookMessage{},
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1PinList(ctx, h3CustomerID, "agent", h3AgentID, tkpin.TargetType(""), uuid.Nil, "tok", uint64(10)).Return(nil, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkPinList_defaultToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq, mockUtil := h3NewHandler(mc)
	ctx := context.Background()

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-11-02T09:15:30.000000Z")
	mockReq.EXPECT().TalkV1PinList(ctx, h3CustomerID, "agent", h3AgentID, tkpin.TargetType(""), uuid.Nil, "2026-11-02T09:15:30.000000Z", uint64(10)).Return([]*tkpin.Pin{}, nil)

	if _, err := h.ServiceAgentTalkPinList(ctx, h3Admin(), "", uuid.Nil, 10, ""); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ServiceAgentTalkPinDelete(t *testing.T) {
	chatPin := pinTestPin(tkpin.TargetTypeChat, h3ResourceID)
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkpin.WebhookMessage, error) {
		return h.ServiceAgentTalkPinDelete(ctx, a, pinTestPinID)
	}

	h3RunRes(t, []h3ResRow[*tkpin.WebhookMessage]{
		h3TalkAuthRow(call),
		{name: "agent without an id", a: pinTestNilAgent(), call: call, wantErr: serviceerrors.ErrPermissionDenied},
		{
			name: "nil pin id", a: h3Admin(), wantErr: serviceerrors.ErrInvalidArgument,
			call: func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkpin.WebhookMessage, error) {
				return h.ServiceAgentTalkPinDelete(ctx, a, uuid.Nil)
			},
		},
		{
			name: "rpc error is passed on", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1PinDelete(ctx, pinTestPinID, h3CustomerID, "agent", h3AgentID).Return(nil, errH3)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: pinTestWM(chatPin),
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1PinDelete(ctx, pinTestPinID, h3CustomerID, "agent", h3AgentID).Return(chatPin, nil)
			},
		},
	})
}

// Verify (7), the api side: canAccessChat is the access check for creating a pin; the talk-manager
// list applies rule R (cf. Test_PinList_ruleR in bin-talk-manager/pkg/dbhandler). Both are evaluated
// over the same rows. They are the same verdict except on two axes: a deleted chat and a customer
// mismatch, where canAccessChat can say yes and rule R says no.
func Test_canAccessChat_vsPinRuleR(t *testing.T) {
	for _, chatType := range []tkchat.Type{tkchat.TypeTalk, tkchat.TypeDirect, tkchat.TypeGroup} {
		for _, isParticipant := range []bool{true, false} {
			for _, chatDeleted := range []bool{false, true} {
				for _, customerMatch := range []bool{true, false} {
					name := fmt.Sprintf("%s participant=%t deleted=%t customerMatch=%t", chatType, isParticipant, chatDeleted, customerMatch)
					t.Run(name, func(t *testing.T) {
						mc := gomock.NewController(t)
						defer mc.Finish()
						h, mockReq, _ := h3NewHandler(mc)
						ctx := context.Background()

						chatCustomer := h3CustomerID
						if !customerMatch {
							chatCustomer = h3OtherCustomerID
						}
						c := h3TalkChat(chatType, chatCustomer)
						if chatDeleted {
							c.TMDelete = pinTestTime()
						}
						participants := h3TalkParticipants(h3TalkOtherID)
						if isParticipant {
							participants = h3TalkParticipants(h3AgentID)
						}

						mockReq.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(c, nil)
						if chatType != tkchat.TypeTalk || !customerMatch {
							mockReq.EXPECT().TalkV1ParticipantList(ctx, h3ResourceID).Return(participants, nil)
						}

						access := h.canAccessChat(ctx, h3AgentID, h3CustomerID, h3ResourceID)
						ruleR := !chatDeleted && customerMatch && (chatType == tkchat.TypeTalk || isParticipant)

						// the expected canAccessChat verdict, stated independently of the code
						expectAccess := (chatType == tkchat.TypeTalk && customerMatch) || isParticipant
						if access != expectAccess {
							t.Errorf("Wrong match. canAccessChat expect: %v, got: %v", expectAccess, access)
						}

						// they agree unless the chat is deleted or the customer differs
						if !chatDeleted && customerMatch && access != ruleR {
							t.Errorf("Wrong match. verdicts differ outside the two axes. canAccessChat: %v, rule R: %v", access, ruleR)
						}
						// and where they differ, canAccessChat is the more permissive one
						if access != ruleR && !access {
							t.Errorf("Wrong match. rule R allowed what canAccessChat denies")
						}
						if access != ruleR && !chatDeleted && customerMatch {
							t.Errorf("Wrong match. verdicts differ on another axis")
						}
					})
				}
			}
		}
	}
}
