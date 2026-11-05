package activeflowhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
)

func Test_actionHandleMessageSend_medias(t *testing.T) {
	customerID := uuid.FromStringOrNil("184d60ac-a2cf-11ec-a800-fb524059f338")
	fileID1 := uuid.FromStringOrNil("5f0b3c52-1c2d-11f1-8a11-0b1c2d3e4f51")
	fileID2 := uuid.FromStringOrNil("5f0b3c52-1c2d-11f1-8a11-0b1c2d3e4f52")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+155****0002"}}
	errSend := errors.New("send failed")

	tests := []struct {
		name string

		// option is the raw json of the action option, as a customer or the flow builder writes it.
		option string

		expectSMS  bool
		expectIDs  []uuid.UUID
		expectErr  bool
		sendErr    error
		unmarshErr bool
	}{
		{
			name:      "no medias key is an sms",
			option:    `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi"}`,
			expectSMS: true,
		},
		{
			name:      "empty medias array is an sms",
			option:    `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":[]}`,
			expectSMS: true,
		},
		{
			name:      "null medias is an sms",
			option:    `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":null}`,
			expectSMS: true,
		},
		{
			name:      "file ids are an mms with the ids in order",
			option:    `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":["5f0b3c52-1c2d-11f1-8a11-0b1c2d3e4f51","5f0b3c52-1c2d-11f1-8a11-0b1c2d3e4f52"]}`,
			expectIDs: []uuid.UUID{fileID1, fileID2},
		},
		{
			name:      "a null element reaches message-manager as a nil id",
			option:    `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":[null]}`,
			expectIDs: []uuid.UUID{uuid.Nil},
		},
		{
			name:      "mms error fails the action",
			option:    `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":["5f0b3c52-1c2d-11f1-8a11-0b1c2d3e4f51"]}`,
			expectIDs: []uuid.UUID{fileID1},
			sendErr:   errSend,
			expectErr: true,
		},
		{
			name:       "a url is not accepted and nothing is sent",
			option:     `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":["https://example.com/a.jpg"]}`,
			expectErr:  true,
			unmarshErr: true,
		},
		{
			name:       "an empty string is not accepted and nothing is sent",
			option:     `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":[""]}`,
			expectErr:  true,
			unmarshErr: true,
		},
		{
			name:       "a malformed id is not accepted and nothing is sent",
			option:     `{"source":{"type":"tel","target":"+155****0001"},"destinations":[{"type":"tel","target":"+155****0002"}],"text":"hi","medias":["abc"]}`,
			expectErr:  true,
			unmarshErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &activeflowHandler{reqHandler: mockReq}
			ctx := context.Background()

			var opt map[string]any
			if err := json.Unmarshal([]byte(tt.option), &opt); err != nil {
				t.Fatalf("bad test option: %v", err)
			}
			af := &activeflow.Activeflow{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9332ed50-dc86-11ec-ba14-8b96908d546b"), CustomerID: customerID},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
					Type:   action.TypeMessageSend,
					Option: opt,
				},
			}

			switch {
			case tt.unmarshErr:
				// no request is expected
			case tt.expectSMS:
				mockReq.EXPECT().MessageV1MessageSend(ctx, uuid.Nil, customerID, source, destinations, "hi").Return(&mmmessage.Message{}, nil)
			case tt.sendErr != nil:
				mockReq.EXPECT().MessageV1MessageSendMMS(ctx, uuid.Nil, customerID, source, destinations, "hi", tt.expectIDs).Return(nil, tt.sendErr)
			default:
				mockReq.EXPECT().MessageV1MessageSendMMS(ctx, uuid.Nil, customerID, source, destinations, "hi", tt.expectIDs).Return(&mmmessage.Message{}, nil)
			}

			err := h.actionHandleMessageSend(ctx, af)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.sendErr != nil && !errors.Is(err, tt.sendErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.sendErr, err)
			}
		})
	}
}
