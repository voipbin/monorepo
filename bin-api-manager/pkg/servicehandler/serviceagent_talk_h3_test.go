package servicehandler

import (
	"context"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
)

var (
	h3TalkMessageID = uuid.FromStringOrNil("77777777-0000-0000-0000-000000000007")
	h3TalkPartID    = uuid.FromStringOrNil("88888888-0000-0000-0000-000000000008")
	h3TalkParentID  = uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-00000000000a")
	h3TalkOtherID   = uuid.FromStringOrNil("99999999-0000-0000-0000-000000000009")
)

type h3Mock = requesthandler.MockRequestHandler

func h3TalkChat(t tkchat.Type, customerID uuid.UUID) *tkchat.Chat {
	return &tkchat.Chat{
		Identity: commonidentity.Identity{ID: h3ResourceID, CustomerID: customerID},
		Type:     t,
		Name:     "h3 chat",
	}
}

func h3TalkParticipants(agentIDs ...uuid.UUID) []*tkparticipant.Participant {
	res := []*tkparticipant.Participant{}
	for _, id := range agentIDs {
		res = append(res, &tkparticipant.Participant{
			Identity: commonidentity.Identity{ID: h3TalkPartID, CustomerID: h3CustomerID},
			Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: id},
			ChatID:   h3ResourceID,
		})
	}
	return res
}

// access granted through the public (talk-type, same customer) rule.
func h3TalkAccessOK(m *h3Mock, ctx context.Context) {
	m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(h3TalkChat(tkchat.TypeTalk, h3CustomerID), nil)
}

// access denied: a group chat in which the agent is not a participant.
func h3TalkAccessDenied(m *h3Mock, ctx context.Context) {
	m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(h3TalkChat(tkchat.TypeGroup, h3CustomerID), nil)
	m.EXPECT().TalkV1ParticipantList(ctx, h3ResourceID).Return(h3TalkParticipants(h3TalkOtherID), nil)
}

func h3TalkPartOK(m *h3Mock, ctx context.Context) {
	m.EXPECT().TalkV1ParticipantList(ctx, h3ResourceID).Return(h3TalkParticipants(h3AgentID), nil)
}

func h3TalkPartDenied(m *h3Mock, ctx context.Context) {
	m.EXPECT().TalkV1ParticipantList(ctx, h3ResourceID).Return(h3TalkParticipants(h3TalkOtherID), nil)
}

func h3TalkMsg() *tkmessage.Message {
	return &tkmessage.Message{
		Identity: commonidentity.Identity{ID: h3TalkMessageID, CustomerID: h3CustomerID},
		Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: h3AgentID},
		ChatID:   h3ResourceID,
		Type:     tkmessage.Type(tkmessage.TypeNormal),
		Text:     "hello",
	}
}

func h3TalkMsgWM(m *tkmessage.Message) *tkmessage.WebhookMessage {
	wm, _ := m.ConvertWebhookMessage()
	return wm
}

// h3TalkAuthRow is the shared unauthenticated row: a non-agent identity.
func h3TalkAuthRow[T any](call func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (T, error)) h3ResRow[T] {
	return h3ResRow[T]{name: "non-agent is unauthenticated", a: h3Direct(), call: call, wantErr: serviceerrors.ErrAuthenticationRequired}
}

func Test_ServiceAgentTalkChatGet_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkchat.WebhookMessage, error) {
		return h.ServiceAgentTalkChatGet(ctx, a, h3ResourceID)
	}
	chat := h3TalkChat(tkchat.TypeTalk, h3CustomerID)
	h3RunRes(t, []h3ResRow[*tkchat.WebhookMessage]{
		h3TalkAuthRow(call),
		{name: "no access", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: h3TalkAccessDenied},
		{
			name: "chat fetch error after access check", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3InOrder(
					m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(chat, nil),
					m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(nil, errH3),
				)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: chat.ConvertWebhookMessage(),
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(chat, nil).Times(2)
			},
		},
	})
}

func Test_ServiceAgentTalkChatList_unauth_error_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tkchat.WebhookMessage, error) {
		return h.ServiceAgentTalkChatList(ctx, a, 10, "tok")
	}
	filters := map[string]any{"owner_type": "agent", "owner_id": h3AgentID.String(), "deleted": false}
	h3RunRes(t, []h3ResRow[[]*tkchat.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "rpc error is wrapped", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatList(ctx, filters, "tok", uint64(10)).Return(nil, errH3)
			},
		},
	})
}

func Test_ServiceAgentTalkChannelList_unauth_error_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tkchat.WebhookMessage, error) {
		return h.ServiceAgentTalkChannelList(ctx, a, 10, "tok")
	}
	filters := map[string]any{"customer_id": h3CustomerID.String(), "type": "talk", "deleted": false}
	h3RunRes(t, []h3ResRow[[]*tkchat.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "rpc error is wrapped", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatList(ctx, filters, "tok", uint64(10)).Return(nil, errH3)
			},
		},
	})
}

func Test_ServiceAgentTalkChatCreate_h3(t *testing.T) {
	parts := []tkparticipant.ParticipantInput{{OwnerType: "agent", OwnerID: h3TalkOtherID}}
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkchat.WebhookMessage, error) {
		return h.ServiceAgentTalkChatCreate(ctx, a, tkchat.TypeGroup, "n", "d", parts)
	}
	chat := h3TalkChat(tkchat.TypeGroup, h3CustomerID)
	create := func(m *h3Mock, ctx context.Context) *h3GomockCall {
		return m.EXPECT().TalkV1ChatCreate(ctx, h3CustomerID, tkchat.TypeGroup, "n", "d", "agent", h3AgentID, parts)
	}
	h3RunRes(t, []h3ResRow[*tkchat.WebhookMessage]{
		h3TalkAuthRow(call),
		{name: "rpc error", a: h3Admin(), call: call, wantErr: errH3, setup: func(m *h3Mock, ctx context.Context) { create(m, ctx).Return(nil, errH3) }},
		{name: "success", a: h3Admin(), call: call, want: chat.ConvertWebhookMessage(), setup: func(m *h3Mock, ctx context.Context) { create(m, ctx).Return(chat, nil) }},
	})
}

func Test_ServiceAgentTalkChatUpdateDelete_h3(t *testing.T) {
	name := "renamed"
	chat := h3TalkChat(tkchat.TypeGroup, h3CustomerID)
	ops := []struct {
		name string
		call func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkchat.WebhookMessage, error)
		fail func(m *h3Mock, ctx context.Context)
		ok   func(m *h3Mock, ctx context.Context)
	}{
		{
			"update",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkchat.WebhookMessage, error) {
				return h.ServiceAgentTalkChatUpdate(ctx, a, h3ResourceID, &name, nil)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatUpdate(ctx, h3ResourceID, &name, nil).Return(nil, errH3)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatUpdate(ctx, h3ResourceID, &name, nil).Return(chat, nil)
			},
		},
		{
			"delete",
			func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkchat.WebhookMessage, error) {
				return h.ServiceAgentTalkChatDelete(ctx, a, h3ResourceID)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatDelete(ctx, h3ResourceID).Return(nil, errH3)
			},
			func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatDelete(ctx, h3ResourceID).Return(chat, nil)
			},
		},
	}
	for _, o := range ops {
		t.Run(o.name, func(t *testing.T) {
			h3RunRes(t, []h3ResRow[*tkchat.WebhookMessage]{
				h3TalkAuthRow(o.call),
				{name: "not a participant", a: h3Admin(), call: o.call, wantErr: h3ErrDenied, setup: h3TalkPartDenied},
				{name: "rpc error", a: h3Admin(), call: o.call, wantErr: errH3, setup: func(m *h3Mock, ctx context.Context) { h3TalkPartOK(m, ctx); o.fail(m, ctx) }},
				{name: "success", a: h3Admin(), call: o.call, want: chat.ConvertWebhookMessage(), setup: func(m *h3Mock, ctx context.Context) { h3TalkPartOK(m, ctx); o.ok(m, ctx) }},
			})
		})
	}
}

func Test_ServiceAgentTalkChatJoin_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkparticipant.WebhookMessage, error) {
		return h.ServiceAgentTalkChatJoin(ctx, a, h3ResourceID)
	}
	part := h3TalkParticipants(h3AgentID)[0]
	chatGet := func(c *tkchat.Chat) func(m *h3Mock, ctx context.Context) {
		return func(m *h3Mock, ctx context.Context) { m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(c, nil) }
	}
	create := func(m *h3Mock, ctx context.Context) *h3GomockCall {
		return m.EXPECT().TalkV1ParticipantCreate(ctx, h3ResourceID, "agent", h3AgentID)
	}
	h3RunRes(t, []h3ResRow[*tkparticipant.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "chat fetch error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) { m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(nil, errH3) },
		},
		{name: "only talk-type chats can be joined", a: h3Admin(), call: call, wantErr: serviceerrors.ErrInvalidArgument, setup: chatGet(h3TalkChat(tkchat.TypeGroup, h3CustomerID))},
		{name: "chat of another customer", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: chatGet(h3TalkChat(tkchat.TypeTalk, h3OtherCustomerID))},
		{
			name: "participant create error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				chatGet(h3TalkChat(tkchat.TypeTalk, h3CustomerID))(m, ctx)
				create(m, ctx).Return(nil, errH3)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: part.ConvertWebhookMessage(),
			setup: func(m *h3Mock, ctx context.Context) {
				chatGet(h3TalkChat(tkchat.TypeTalk, h3CustomerID))(m, ctx)
				create(m, ctx).Return(part, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkParticipantList_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tkparticipant.WebhookMessage, error) {
		return h.ServiceAgentTalkParticipantList(ctx, a, h3ResourceID)
	}
	parts := h3TalkParticipants(h3AgentID, h3TalkOtherID)
	want := []*tkparticipant.WebhookMessage{parts[0].ConvertWebhookMessage(), parts[1].ConvertWebhookMessage()}
	h3RunRes(t, []h3ResRow[[]*tkparticipant.WebhookMessage]{
		h3TalkAuthRow(call),
		{name: "no access", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: h3TalkAccessDenied},
		{
			name: "rpc error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkAccessOK(m, ctx)
				m.EXPECT().TalkV1ParticipantList(ctx, h3ResourceID).Return(nil, errH3)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: want,
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkAccessOK(m, ctx)
				m.EXPECT().TalkV1ParticipantList(ctx, h3ResourceID).Return(parts, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkParticipantCreate_guards_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkparticipant.WebhookMessage, error) {
		return h.ServiceAgentTalkParticipantCreate(ctx, a, h3ResourceID, "agent", h3TalkOtherID)
	}
	part := h3TalkParticipants(h3TalkOtherID)[0]
	create := func(m *h3Mock, ctx context.Context) *h3GomockCall {
		return m.EXPECT().TalkV1ParticipantCreate(ctx, h3ResourceID, "agent", h3TalkOtherID)
	}
	h3RunRes(t, []h3ResRow[*tkparticipant.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "create rpc error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				// allowed: the caller already participates (second call of the participant list is not made)
				m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(h3TalkChat(tkchat.TypeGroup, h3CustomerID), nil)
				h3TalkPartOK(m, ctx)
				create(m, ctx).Return(nil, errH3)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: part.ConvertWebhookMessage(),
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1ChatGet(ctx, h3ResourceID).Return(h3TalkChat(tkchat.TypeGroup, h3CustomerID), nil)
				h3TalkPartOK(m, ctx)
				create(m, ctx).Return(part, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkParticipantDelete_guards_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkparticipant.WebhookMessage, error) {
		return h.ServiceAgentTalkParticipantDelete(ctx, a, h3ResourceID, h3TalkPartID)
	}
	part := h3TalkParticipants(h3AgentID)[0]
	del := func(m *h3Mock, ctx context.Context) *h3GomockCall {
		return m.EXPECT().TalkV1ParticipantDelete(ctx, h3ResourceID, h3TalkPartID)
	}
	h3RunRes(t, []h3ResRow[*tkparticipant.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "rpc error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkPartOK(m, ctx)
				del(m, ctx).Return(nil, errH3)
			},
		},
		{
			name: "already deleted (nil) returns nil", a: h3Admin(), call: call,
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkPartOK(m, ctx)
				del(m, ctx).Return(nil, nil)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: part.ConvertWebhookMessage(),
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkPartOK(m, ctx)
				del(m, ctx).Return(part, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkMessageGet_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkmessage.WebhookMessage, error) {
		return h.ServiceAgentTalkMessageGet(ctx, a, h3TalkMessageID)
	}
	msg := h3TalkMsg()
	h3RunRes(t, []h3ResRow[*tkmessage.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "message lookup error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, h3TalkMessageID).Return(nil, errH3)
			},
		},
		{
			name: "no access to the message's chat", a: h3Admin(), call: call, wantErr: h3ErrDenied,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, h3TalkMessageID).Return(msg, nil)
				h3TalkAccessDenied(m, ctx)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: h3TalkMsgWM(msg),
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, h3TalkMessageID).Return(msg, nil)
				h3TalkAccessOK(m, ctx)
			},
		},
	})
}

func Test_ServiceAgentTalkMessageList_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) ([]*tkmessage.WebhookMessage, error) {
		return h.ServiceAgentTalkMessageList(ctx, a, h3ResourceID, 10, "tok")
	}
	msg := h3TalkMsg()
	filters := map[string]any{"chat_id": h3ResourceID.String()}
	h3RunRes(t, []h3ResRow[[]*tkmessage.WebhookMessage]{
		h3TalkAuthRow(call),
		{name: "no access", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: h3TalkAccessDenied},
		{
			name: "list rpc error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkAccessOK(m, ctx)
				m.EXPECT().TalkV1MessageListWithFilters(ctx, filters, "tok", uint64(10)).Return(nil, errH3)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: []*tkmessage.WebhookMessage{h3TalkMsgWM(msg)},
			setup: func(m *h3Mock, ctx context.Context) {
				h3TalkAccessOK(m, ctx)
				m.EXPECT().TalkV1MessageListWithFilters(ctx, filters, "tok", uint64(10)).Return([]*tkmessage.Message{msg}, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkMessageCreate_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkmessage.WebhookMessage, error) {
		return h.ServiceAgentTalkMessageCreate(ctx, a, h3ResourceID, &h3TalkParentID, tkmessage.Type(tkmessage.TypeNormal), "hello", nil)
	}
	msg := h3TalkMsg()
	create := func(m *h3Mock, ctx context.Context) *h3GomockCall {
		return m.EXPECT().TalkV1MessageCreate(ctx, h3ResourceID, &h3TalkParentID, "agent", h3AgentID, tkmessage.Type(tkmessage.TypeNormal), "hello", []tkmessage.Media(nil))
	}
	h3RunRes(t, []h3ResRow[*tkmessage.WebhookMessage]{
		h3TalkAuthRow(call),
		{name: "not a participant", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: h3TalkPartDenied},
		{name: "rpc error", a: h3Admin(), call: call, wantErr: errH3, setup: func(m *h3Mock, ctx context.Context) { h3TalkPartOK(m, ctx); create(m, ctx).Return(nil, errH3) }},
		{name: "success", a: h3Admin(), call: call, want: h3TalkMsgWM(msg), setup: func(m *h3Mock, ctx context.Context) { h3TalkPartOK(m, ctx); create(m, ctx).Return(msg, nil) }},
	})
}

func Test_ServiceAgentTalkMessageDelete_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkmessage.WebhookMessage, error) {
		return h.ServiceAgentTalkMessageDelete(ctx, a, h3TalkMessageID)
	}
	msg := h3TalkMsg()
	foreign := h3TalkMsg()
	foreign.OwnerID = h3TalkOtherID
	get := func(mm *tkmessage.Message) func(m *h3Mock, ctx context.Context) {
		return func(m *h3Mock, ctx context.Context) {
			m.EXPECT().TalkV1MessageGet(ctx, h3TalkMessageID).Return(mm, nil)
		}
	}
	h3RunRes(t, []h3ResRow[*tkmessage.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "message lookup error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, h3TalkMessageID).Return(nil, errH3)
			},
		},
		{name: "only the owner may delete", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: get(foreign)},
		{
			name: "delete rpc error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				get(msg)(m, ctx)
				m.EXPECT().TalkV1MessageDelete(ctx, h3TalkMessageID).Return(nil, errH3)
			},
		},
		{
			name: "success", a: h3Admin(), call: call, want: h3TalkMsgWM(msg),
			setup: func(m *h3Mock, ctx context.Context) {
				get(msg)(m, ctx)
				m.EXPECT().TalkV1MessageDelete(ctx, h3TalkMessageID).Return(msg, nil)
			},
		},
	})
}

func Test_ServiceAgentTalkMessageReactionCreate_h3(t *testing.T) {
	call := func(h *serviceHandler, ctx context.Context, a *auth.AuthIdentity) (*tkmessage.WebhookMessage, error) {
		return h.ServiceAgentTalkMessageReactionCreate(ctx, a, h3TalkMessageID, ":+1:")
	}
	msg := h3TalkMsg()
	get := func(m *h3Mock, ctx context.Context) {
		m.EXPECT().TalkV1MessageGet(ctx, h3TalkMessageID).Return(msg, nil)
	}
	react := func(m *h3Mock, ctx context.Context) *h3GomockCall {
		return m.EXPECT().TalkV1MessageReactionCreate(ctx, h3TalkMessageID, "agent", h3AgentID, ":+1:")
	}
	h3RunRes(t, []h3ResRow[*tkmessage.WebhookMessage]{
		h3TalkAuthRow(call),
		{
			name: "message lookup error", a: h3Admin(), call: call, wantErr: errH3,
			setup: func(m *h3Mock, ctx context.Context) {
				m.EXPECT().TalkV1MessageGet(ctx, h3TalkMessageID).Return(nil, errH3)
			},
		},
		{name: "not a participant", a: h3Admin(), call: call, wantErr: h3ErrDenied, setup: func(m *h3Mock, ctx context.Context) { get(m, ctx); h3TalkPartDenied(m, ctx) }},
		{name: "rpc error", a: h3Admin(), call: call, wantErr: errH3, setup: func(m *h3Mock, ctx context.Context) {
			get(m, ctx)
			h3TalkPartOK(m, ctx)
			react(m, ctx).Return(nil, errH3)
		}},
		{name: "success", a: h3Admin(), call: call, want: h3TalkMsgWM(msg), setup: func(m *h3Mock, ctx context.Context) {
			get(m, ctx)
			h3TalkPartOK(m, ctx)
			react(m, ctx).Return(msg, nil)
		}},
	})
}

func Test_talkMessageListByChatIDs_empty_h3(t *testing.T) {
	mc := h3NewController(t)
	h, _, _ := h3NewHandler(mc)

	res, err := h.talkMessageListByChatIDs(context.Background(), nil, 10, "tok")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || len(res) != 0 {
		t.Errorf("Wrong match. expect: empty non-nil slice without any rpc, got: %v", res)
	}
}
