package confbridgehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
)

func Test_ExternalMediaStart(t *testing.T) {

	tests := []struct {
		name string

		id              uuid.UUID
		externalMediaID uuid.UUID
		externalHost    string
		encapsulation   externalmedia.Encapsulation
		transport       externalmedia.Transport
		connectionType  string
		format          string
		direction       string

		responseCall          *confbridge.Confbridge
		responseExternalMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("45c1a114-996f-11ed-b089-575b5e0a4a0d"),
			uuid.FromStringOrNil("620b555c-b332-11ef-af24-271a7cd9ab2a"),
			"example.com",
			externalmedia.EncapsulationRTP,
			externalmedia.TransportUDP,
			"client",
			"ulaw",
			"both",

			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("45c1a114-996f-11ed-b089-575b5e0a4a0d"),
			},
			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("ae01d90e-96e2-11ed-8b03-f31329c0298c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &confbridgeHandler{
				utilHandler:          mockUtil,
				reqHandler:           mockReq,
				db:                   mockDB,
				externalMediaHandler: mockExternal,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockExternal.EXPECT().Start(ctx, tt.externalMediaID, externalmedia.ReferenceTypeConfbridge, tt.id, true, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction).Return(tt.responseExternalMedia, nil)
			mockDB.EXPECT().ConfbridgeSetExternalMediaID(ctx, tt.id, tt.responseExternalMedia.ID).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseCall, nil)

			res, err := h.ExternalMediaStart(ctx, tt.id, tt.externalMediaID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseCall, res) {
				t.Errorf("Wrong match.\nexpect: %vgot: %v", tt.responseExternalMedia, res)
			}
		})
	}
}

func Test_ExternalMediaStop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge    *confbridge.Confbridge
		responseExternalMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("46086f68-996f-11ed-a311-3bbc19b864b5"),

			&confbridge.Confbridge{
				ID:              uuid.FromStringOrNil("46086f68-996f-11ed-a311-3bbc19b864b5"),
				ExternalMediaID: uuid.FromStringOrNil("462aa10a-996f-11ed-8cd9-47103a32e558"),
			},
			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("462aa10a-996f-11ed-8cd9-47103a32e558"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &confbridgeHandler{
				reqHandler:           mockReq,
				db:                   mockDB,
				externalMediaHandler: mockExternal,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			mockExternal.EXPECT().Stop(ctx, tt.responseConfbridge.ExternalMediaID).Return(&externalmedia.ExternalMedia{}, nil)
			mockDB.EXPECT().ConfbridgeSetExternalMediaID(ctx, tt.id, uuid.Nil).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)

			res, err := h.ExternalMediaStop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConfbridge, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConfbridge, res)
			}
		})
	}
}
