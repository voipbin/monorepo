package aicallhandler

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_toolHandleMessageSend_medias(t *testing.T) {
	customerID := uuid.FromStringOrNil("4c0ab9fc-bbb3-11f0-99a8-8f2a21cdfa73")
	msgID := uuid.FromStringOrNil("4c8ad7cc-bbb3-11f0-a7ed-4f8dd1e691ca")
	fileID1 := uuid.FromStringOrNil("6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61")
	fileID2 := uuid.FromStringOrNil("6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a62")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+123****6789"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+111****1111"}}
	errSend := errors.New("send failed")

	base := `"source":{"type":"tel","target":"+123****6789"},"destinations":[{"type":"tel","target":"+111****1111"}],"text":"hi"`

	tests := []struct {
		name string

		arguments string

		expectSMS    bool
		expectMMS    bool
		expectIDs    []uuid.UUID
		responseErr  error
		expectResult string
		expectNoUUID bool
	}{
		{
			name:         "no medias is an sms",
			arguments:    `{` + base + `}`,
			expectSMS:    true,
			expectResult: "success",
		},
		{
			name:         "empty medias is an sms",
			arguments:    `{` + base + `,"medias":[]}`,
			expectSMS:    true,
			expectResult: "success",
		},
		{
			name:         "file ids are an mms with the ids in order",
			arguments:    `{` + base + `,"medias":["6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61","6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a62"]}`,
			expectMMS:    true,
			expectIDs:    []uuid.UUID{fileID1, fileID2},
			expectResult: "success",
		},
		{
			name:         "a null element reaches message-manager as a nil id",
			arguments:    `{` + base + `,"medias":[null]}`,
			expectMMS:    true,
			expectIDs:    []uuid.UUID{uuid.Nil},
			expectResult: "success",
		},
		{
			name:         "mms error is a failed tool result",
			arguments:    `{` + base + `,"medias":["6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61"]}`,
			expectMMS:    true,
			expectIDs:    []uuid.UUID{fileID1},
			responseErr:  errSend,
			expectResult: "failed",
		},
		{
			name:         "a url fails the arguments and nothing is sent",
			arguments:    `{` + base + `,"medias":["https://example.com/a.jpg"]}`,
			expectNoUUID: true,
			expectResult: "failed",
		},
		{
			name:         "a malformed id fails the arguments and nothing is sent",
			arguments:    `{` + base + `,"medias":["abc"]}`,
			expectNoUUID: true,
			expectResult: "failed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{utilHandler: mockUtil, reqHandler: mockReq}
			ctx := context.Background()

			c := &aicall.AIcall{Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("4bc61c34-bbb3-11f0-9fe7-73c103bcd202"),
				CustomerID: customerID,
			}}
			tool := &message.ToolCall{
				ID:   "4c620144-bbb3-11f0-823e-77b11be82ade",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameSendMessage,
					Arguments: tt.arguments,
				},
			}

			if !tt.expectNoUUID {
				mockUtil.EXPECT().UUIDCreate().Return(msgID)
			}
			response := &mmmessage.Message{Identity: commonidentity.Identity{ID: msgID}}
			switch {
			case tt.expectSMS:
				mockReq.EXPECT().MessageV1MessageSend(ctx, msgID, customerID, source, destinations, "hi").Return(response, nil)
			case tt.expectMMS && tt.responseErr != nil:
				mockReq.EXPECT().MessageV1MessageSendMMS(ctx, msgID, customerID, source, destinations, "hi", tt.expectIDs).Return(nil, tt.responseErr)
			case tt.expectMMS:
				mockReq.EXPECT().MessageV1MessageSendMMS(ctx, msgID, customerID, source, destinations, "hi", tt.expectIDs).Return(response, nil)
			}

			res := h.toolHandleMessageSend(ctx, c, tool)
			if res.Result != tt.expectResult {
				t.Errorf("Wrong result. expect: %s, got: %s (%s)", tt.expectResult, res.Result, res.Message)
			}
			if tt.expectResult == "success" && res.ResourceID != msgID.String() {
				t.Errorf("Wrong resource id. got: %s", res.ResourceID)
			}
		})
	}
}
