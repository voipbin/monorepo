package callhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
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
		directionListen externalmedia.Direction
		directionSpeak  externalmedia.Direction

		responseCall          *call.Call
		responseExternalMedia *externalmedia.ExternalMedia
	}{
		{
			name: "normal",

			id:              uuid.FromStringOrNil("10320c34-b333-11ef-a431-6796e4efd447"),
			externalMediaID: uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
			externalHost:    "example.com",
			encapsulation:   externalmedia.EncapsulationRTP,
			transport:       externalmedia.TransportUDP,
			connectionType:  "client",
			format:          "ulaw",
			directionListen: externalmedia.DirectionIn,
			directionSpeak:  externalmedia.DirectionOut,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10320c34-b333-11ef-a431-6796e4efd447"),
				},
			},
			responseExternalMedia: &externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("10320c34-b333-11ef-a431-6796e4efd447"),
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

			h := &callHandler{
				utilHandler:          mockUtil,
				reqHandler:           mockReq,
				db:                   mockDB,
				externalMediaHandler: mockExternal,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockExternal.EXPECT().Start(
				ctx,
				tt.externalMediaID,
				externalmedia.ReferenceTypeCall,
				tt.id,
				tt.externalHost,
				tt.encapsulation,
				tt.transport,
				tt.connectionType,
				tt.format,
				tt.directionListen,
				tt.directionSpeak,
			).Return(tt.responseExternalMedia, nil)
			mockDB.EXPECT().CallAddExternalMediaID(ctx, tt.id, tt.responseExternalMedia.ID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)

			res, err := h.ExternalMediaStart(
				ctx,
				tt.id,
				tt.externalMediaID,
				tt.externalHost,
				tt.encapsulation,
				tt.transport,
				tt.connectionType,
				tt.format,
				tt.directionListen,
				tt.directionSpeak,
			)
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

		id              uuid.UUID
		externalMediaID uuid.UUID

		responseCall          *call.Call
		responseExternalMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("a4a11b7a-96f8-11ed-a7ff-c7b76333c22f"),
			uuid.FromStringOrNil("5338f2ba-96fa-11ed-9741-83dfb85823e4"),

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4a11b7a-96f8-11ed-a7ff-c7b76333c22f"),
				},
				ExternalMediaIDs: []uuid.UUID{uuid.FromStringOrNil("5338f2ba-96fa-11ed-9741-83dfb85823e4")},
			},
			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("77af2454-96f8-11ed-a2c6-1b0a3b96dbdd"),
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

			h := &callHandler{
				reqHandler:           mockReq,
				db:                   mockDB,
				externalMediaHandler: mockExternal,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockExternal.EXPECT().Stop(ctx, tt.externalMediaID).Return(&externalmedia.ExternalMedia{}, nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)

			res, err := h.ExternalMediaStop(ctx, tt.id, tt.externalMediaID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseCall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseCall, res)
			}
		})
	}
}

func Test_ExternalMediaStop_error(t *testing.T) {

	tests := []struct {
		name string

		id              uuid.UUID
		externalMediaID uuid.UUID

		responseCall               *call.Call
		responseCallError          error
		responseExternalMedia      *externalmedia.ExternalMedia
		responseExternalMediaError error
	}{
		{
			name: "call get returns an error",

			id:              uuid.FromStringOrNil("524db4e8-9728-11ed-8b63-33b3e4cab35f"),
			externalMediaID: uuid.FromStringOrNil("92cd6342-9728-11ed-b00b-d7edef482e40"),

			responseCallError: fmt.Errorf(""),
		},
		{
			name: "external media id not in call's list",

			id:              uuid.FromStringOrNil("7e77d07c-9727-11ed-9a5c-7f64fe264774"),
			externalMediaID: uuid.FromStringOrNil("92cd6342-9728-11ed-b00b-d7edef482e40"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7e77d07c-9727-11ed-9a5c-7f64fe264774"),
				},
				ExternalMediaIDs: []uuid.UUID{},
			},
		},
		{
			name: "external media stop returns an error",

			id:              uuid.FromStringOrNil("92a767b4-9728-11ed-885f-47623ef9293e"),
			externalMediaID: uuid.FromStringOrNil("92cd6342-9728-11ed-b00b-d7edef482e40"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("92a767b4-9728-11ed-885f-47623ef9293e"),
				},
				ExternalMediaIDs: []uuid.UUID{uuid.FromStringOrNil("92cd6342-9728-11ed-b00b-d7edef482e40")},
			},
			responseExternalMediaError: fmt.Errorf(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &callHandler{
				reqHandler:           mockReq,
				db:                   mockDB,
				externalMediaHandler: mockExternal,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, tt.responseCallError)
			if tt.responseCallError == nil {
				// check if the externalMediaID is in the call's list
				found := false
				for _, emID := range tt.responseCall.ExternalMediaIDs {
					if emID == tt.externalMediaID {
						found = true
						break
					}
				}
				if found {
					mockExternal.EXPECT().Stop(ctx, tt.externalMediaID).Return(&externalmedia.ExternalMedia{}, tt.responseExternalMediaError)
					if tt.responseExternalMediaError == nil {
						mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
					}
				}
			}

			_, err := h.ExternalMediaStop(ctx, tt.id, tt.externalMediaID)
			if err == nil {
				t.Error("Wrong match. expect: error, got: ok")
			}
		})
	}
}
