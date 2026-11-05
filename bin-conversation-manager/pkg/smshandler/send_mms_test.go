package smshandler

import (
	"context"
	"errors"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
)

func Test_Send_medias(t *testing.T) {
	customerID := uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e60")
	messageID := uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e61")
	fileID1 := uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e71")
	fileID2 := uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e72")
	errSend := errors.New("send failed")

	newConversation := func() *conversation.Conversation {
		return &conversation.Conversation{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("7e1c4b20-9d3a-4f58-8c6e-1b2a3c4d5e62"), CustomerID: customerID},
			Self:     commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"},
			Peer:     commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
		}
	}
	response := &mmmessage.Message{}
	response.ID = messageID

	tests := []struct {
		name string

		medias []media.Media

		expectSMS  bool
		expectIDs  []uuid.UUID
		responseEr error
	}{
		{
			name:      "nil medias use the sms method",
			medias:    nil,
			expectSMS: true,
		},
		{
			name:      "empty medias use the sms method",
			medias:    []media.Media{},
			expectSMS: true,
		},
		{
			name: "medias use the mms method with the file ids in order",
			medias: []media.Media{
				{Identity: commonidentity.Identity{ID: fileID1}, Type: media.TypeImage, Filename: "a.jpg"},
				{Identity: commonidentity.Identity{ID: fileID2}, Type: media.TypeFile, Filename: "b.pdf"},
			},
			expectIDs: []uuid.UUID{fileID1, fileID2},
		},
		{
			name: "mms error is returned unchanged",
			medias: []media.Media{
				{Identity: commonidentity.Identity{ID: fileID1}},
			},
			expectIDs:  []uuid.UUID{fileID1},
			responseEr: errSend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &smsHandler{reqHandler: mockReq, accountHandler: accounthandler.NewMockAccountHandler(mc)}

			cv := newConversation()
			ctx := context.Background()
			destinations := []commonaddress.Address{cv.Peer}

			if tt.expectSMS {
				mockReq.EXPECT().MessageV1MessageSend(ctx, messageID, customerID, &cv.Self, destinations, "hello").Return(response, nil)
			} else if tt.responseEr != nil {
				mockReq.EXPECT().MessageV1MessageSendMMS(ctx, messageID, customerID, &cv.Self, destinations, "hello", tt.expectIDs).Return(nil, tt.responseEr)
			} else {
				mockReq.EXPECT().MessageV1MessageSendMMS(ctx, messageID, customerID, &cv.Self, destinations, "hello", tt.expectIDs).Return(response, nil)
			}

			err := h.Send(ctx, cv, messageID, "hello", tt.medias)
			if !errors.Is(err, tt.responseEr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseEr, err)
			}
		})
	}
}
