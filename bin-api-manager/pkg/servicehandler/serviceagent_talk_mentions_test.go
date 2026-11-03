package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tkmessage "monorepo/bin-talk-manager/models/message"
)

func Test_ServiceAgentTalkMentionList(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111")
	agentID := uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: agentID, CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})

	responseMessages := []*tkmessage.Message{
		{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"), CustomerID: customerID},
			Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222")},
			ChatID:   uuid.FromStringOrNil("e1111111-1111-1111-1111-111111111111"),
			Type:     tkmessage.TypeNormal,
			Text:     "hi [@me](mention:a1111111-1111-1111-1111-111111111111)",
			TMCreate: timePtr("2024-01-01T00:00:00.000000Z"),
		},
	}
	expectRes := []*tkmessage.WebhookMessage{
		{
			Identity: responseMessages[0].Identity,
			Owner:    responseMessages[0].Owner,
			ChatID:   responseMessages[0].ChatID,
			Type:     tkmessage.TypeNormal,
			Text:     responseMessages[0].Text,
			TMCreate: responseMessages[0].TMCreate,
		},
	}

	t.Run("the calling agent's own mentions are requested with its own ids", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		// the plain Agent permission is enough, and the ids come from the identity, never from the caller
		mockReq.EXPECT().TalkV1MentionList(ctx, customerID, agentID, "tok", uint64(10)).Return(responseMessages, nil)

		res, err := h.ServiceAgentTalkMentionList(ctx, agent, 10, "tok")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(res, expectRes) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
		}
	})

	t.Run("an empty result is an empty list", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().TalkV1MentionList(ctx, customerID, agentID, "", uint64(10)).Return(nil, nil)

		res, err := h.ServiceAgentTalkMentionList(ctx, agent, 10, "")
		if err != nil || res == nil || len(res) != 0 {
			t.Errorf("Wrong match. expect: empty list, got: %v, %v", res, err)
		}
	})

	t.Run("a non agent identity is unauthenticated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

		_, err := h.ServiceAgentTalkMentionList(context.Background(), h3Direct(), 10, "")
		if !errors.Is(err, serviceerrors.ErrAuthenticationRequired) {
			t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrAuthenticationRequired, err)
		}
	})

	t.Run("an rpc error is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().TalkV1MentionList(ctx, customerID, agentID, "", uint64(10)).Return(nil, errors.New("rpc failed"))

		res, err := h.ServiceAgentTalkMentionList(ctx, agent, 10, "")
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
		}
	})
}
