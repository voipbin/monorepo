package streamhandler

import (
	"context"
	"monorepo/bin-api-manager/models/stream"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

func Test_Start(t *testing.T) {
	type test struct {
		name string

		ws            *websocket.Conn
		referenceType cmexternalmedia.ReferenceType
		referenceID   uuid.UUID
		encapsulation stream.Encapsulation

		responseUUID          uuid.UUID
		responseExternalMedia *cmexternalmedia.ExternalMedia

		expectedRes *stream.Stream
	}

	tests := []test{
		{
			name: "normal",

			ws:            nil,
			referenceType: cmexternalmedia.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("718466c8-b7d2-11ef-868e-9fb64395f0d3"),
			encapsulation: stream.EncapsulationAudiosocket,

			responseUUID: uuid.FromStringOrNil("720c7bb2-b7d2-11ef-abc2-23c5599e4029"),
			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("7232599a-b7d2-11ef-af1b-8384c0b4170b"),
			},

			expectedRes: &stream.Stream{
				ID:            uuid.FromStringOrNil("720c7bb2-b7d2-11ef-abc2-23c5599e4029"),
				ConnWebsocket: nil,
				Encapsulation: stream.EncapsulationAudiosocket,
				ExternalMedia: &cmexternalmedia.ExternalMedia{
					ID: uuid.FromStringOrNil("7232599a-b7d2-11ef-af1b-8384c0b4170b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			localhost := "127.0.0.1:9000"

			h := &streamHandler{
				reqHandler:    mockReq,
				utilHandler:   mockUtil,
				listenAddress: localhost,
				streamData:    make(map[string]*stream.Stream),
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUID)
			mockReq.EXPECT().CallV1ExternalMediaStart(
				ctx,
				tt.responseUUID,
				tt.referenceType,
				tt.referenceID,
				localhost,
				defaultExternalMediaEncapsulation,
				defaultExternalMediaTransport,
				defaultExternalMediaConnectionType,
				defaultExternalMediaFormat,
				defaultExternalMediaDirectionListen,
				defaultExternalMediaDirectionSpeak,
			.Return(tt.responseExternalMedia, nil)

			res, err := h.Start(ctx, tt.ws, tt.referenceType, tt.referenceID, tt.encapsulation)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedRes, res)
			}

		})
	}
}
