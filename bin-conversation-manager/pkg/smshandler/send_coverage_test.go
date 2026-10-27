package smshandler

import (
	"context"
	"errors"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
)

func Test_NewSMSHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	res, ok := NewSMSHandler(mockReq, mockAccount).(*smsHandler)
	if !ok {
		t.Fatalf("NewSMSHandler() did not return an *smsHandler")
	}
	if res.reqHandler != mockReq || res.accountHandler != mockAccount {
		t.Errorf("NewSMSHandler() did not keep the given handlers")
	}
}

func Test_Send_requestFailureIsReturned(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	errSend := errors.New("message service unavailable")

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &smsHandler{reqHandler: mockReq, accountHandler: accounthandler.NewMockAccountHandler(mc)}

	customerID := uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e60")
	messageID := uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e61")
	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e62"), CustomerID: customerID},
		Self:     commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
		Peer:     commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000002"},
	}

	ctx := context.Background()
	mockReq.EXPECT().MessageV1MessageSend(ctx, messageID, customerID, &cv.Self, []commonaddress.Address{cv.Peer}, "hello").Return(nil, errSend)

	if err := h.Send(ctx, cv, messageID, "hello"); !errors.Is(err, errSend) {
		t.Errorf("Send() error = %v, want %v", err, errSend)
	}
}
