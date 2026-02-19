package externalmediahandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id              uuid.UUID
		asteriskID      string
		channelID       string
		bridgeID        string
		playbackID      string
		referenceType   externalmedia.ReferenceType
		referenceID     uuid.UUID
		localIP         string
		localPort       int
		externalHost    string
		encapsulation   externalmedia.Encapsulation
		transport       externalmedia.Transport
		connectionType  string
		format          string
		directionListen externalmedia.Direction
		directionSpeak  externalmedia.Direction

		responseUUID uuid.UUID

		expectExternalMedia *externalmedia.ExternalMedia
	}{
		{
			name: "normal",

			id:              uuid.FromStringOrNil("a59fb054-b32f-11ef-8b11-6395ff848e6c"),
			asteriskID:      "3e:50:6b:43:bb:30",
			channelID:       "6295f80e-97b6-11ed-88e0-ffb00420fb1a",
			bridgeID:        "7f34278a-7d9e-11f0-ba9f-97cef5bed809",
			playbackID:      "7f571894-7d9e-11f0-9702-67d856ab2a53",
			referenceType:   externalmedia.ReferenceTypeCall,
			referenceID:     uuid.FromStringOrNil("da311f40-97b8-11ed-99f4-13108e8918da"),
			localIP:         "127.0.0.1",
			localPort:       8080,
			externalHost:    "127.0.0.1:8090",
			encapsulation:   defaultEncapsulation,
			transport:       defaultTransport,
			connectionType:  defaultConnectionType,
			format:          defaultFormat,
			directionListen: externalmedia.DirectionIn,
			directionSpeak:  externalmedia.DirectionOut,

			expectExternalMedia: &externalmedia.ExternalMedia{
				ID:              uuid.FromStringOrNil("a59fb054-b32f-11ef-8b11-6395ff848e6c"),
				Type:            externalmedia.TypeNormal,
				AsteriskID:      "3e:50:6b:43:bb:30",
				ChannelID:       "6295f80e-97b6-11ed-88e0-ffb00420fb1a",
				BridgeID:        "7f34278a-7d9e-11f0-ba9f-97cef5bed809",
				PlaybackID:      "7f571894-7d9e-11f0-9702-67d856ab2a53",
				ReferenceType:   externalmedia.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("da311f40-97b8-11ed-99f4-13108e8918da"),
				Status:          externalmedia.StatusRunning,
				LocalIP:         "127.0.0.1",
				LocalPort:       8080,
				ExternalHost:    "127.0.0.1:8090",
				Encapsulation:   defaultEncapsulation,
				Transport:       defaultTransport,
				ConnectionType:  defaultConnectionType,
				Format:          defaultFormat,
				DirectionListen: externalmedia.DirectionIn,
				DirectionSpeak:  externalmedia.DirectionOut,
			},
		},
		{
			name: "has no id given",

			id:              uuid.Nil,
			asteriskID:      "3e:50:6b:43:bb:30",
			channelID:       "09d1321e-b330-11ef-ad3e-170da10752c8",
			bridgeID:        "96b5355c-7d9e-11f0-a2ca-b3875a58d870",
			playbackID:      "96df44be-7d9e-11f0-a0fd-bb11489c28cf",
			referenceType:   externalmedia.ReferenceTypeCall,
			referenceID:     uuid.FromStringOrNil("09ec9a0e-b330-11ef-b9dc-6701b7b5d325"),
			localIP:         "127.0.0.1",
			localPort:       8080,
			externalHost:    "127.0.0.1:8090",
			encapsulation:   defaultEncapsulation,
			transport:       defaultTransport,
			connectionType:  defaultConnectionType,
			format:          defaultFormat,
			directionListen: externalmedia.DirectionIn,
			directionSpeak:  externalmedia.DirectionOut,

			responseUUID: uuid.FromStringOrNil("0a05e55e-b330-11ef-a24c-07a8913a053b"),
			expectExternalMedia: &externalmedia.ExternalMedia{
				ID:              uuid.FromStringOrNil("0a05e55e-b330-11ef-a24c-07a8913a053b"),
				Type:            externalmedia.TypeNormal,
				AsteriskID:      "3e:50:6b:43:bb:30",
				ChannelID:       "09d1321e-b330-11ef-ad3e-170da10752c8",
				BridgeID:        "96b5355c-7d9e-11f0-a2ca-b3875a58d870",
				PlaybackID:      "96df44be-7d9e-11f0-a0fd-bb11489c28cf",
				ReferenceType:   externalmedia.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("09ec9a0e-b330-11ef-b9dc-6701b7b5d325"),
				Status:          externalmedia.StatusRunning,
				LocalIP:         "127.0.0.1",
				LocalPort:       8080,
				ExternalHost:    "127.0.0.1:8090",
				Encapsulation:   defaultEncapsulation,
				Transport:       defaultTransport,
				ConnectionType:  defaultConnectionType,
				Format:          defaultFormat,
				DirectionListen: externalmedia.DirectionIn,
				DirectionSpeak:  externalmedia.DirectionOut,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &externalMediaHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			if tt.id == uuid.Nil {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			}
			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia).Return(nil)
			res, err := h.Create(
				ctx,
				tt.id,
				externalmedia.TypeNormal,
				tt.asteriskID,
				tt.channelID,
				tt.bridgeID,
				tt.playbackID,
				tt.referenceType,
				tt.referenceID,
				tt.localIP,
				tt.localPort,
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

			if !reflect.DeepEqual(res, tt.expectExternalMedia) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectExternalMedia, res)
			}

		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseExternalMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("70818138-97b9-11ed-9ea2-2bfbad9f1911"),

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("70818138-97b9-11ed-9ea2-2bfbad9f1911"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &externalMediaHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id).Return(tt.responseExternalMedia, nil)
			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseExternalMedia) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseExternalMedia, res)
			}

		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		filters map[externalmedia.Field]any

		responseExternalMedia *externalmedia.ExternalMedia

		expectReferenceID uuid.UUID
		expectRes         []*externalmedia.ExternalMedia
	}{
		{
			"normal",

			map[externalmedia.Field]any{
				externalmedia.FieldReferenceID: uuid.FromStringOrNil("d441ee34-e825-11ee-b63c-47956736e3ae"),
			},

			&externalmedia.ExternalMedia{
				ID:          uuid.FromStringOrNil("e14699ae-e825-11ee-90b5-970e6c9b40c8"),
				ReferenceID: uuid.FromStringOrNil("d441ee34-e825-11ee-b63c-47956736e3ae"),
			},

			uuid.FromStringOrNil("d441ee34-e825-11ee-b63c-47956736e3ae"),
			[]*externalmedia.ExternalMedia{
				{
					ID:          uuid.FromStringOrNil("e14699ae-e825-11ee-90b5-970e6c9b40c8"),
					ReferenceID: uuid.FromStringOrNil("d441ee34-e825-11ee-b63c-47956736e3ae"),
				},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &externalMediaHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ExternalMediaGetByReferenceID(ctx, tt.expectReferenceID).Return(tt.responseExternalMedia, nil)
			res, err := h.List(ctx, 1, "", tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseExternalMedia, res)
			}

		})
	}
}
