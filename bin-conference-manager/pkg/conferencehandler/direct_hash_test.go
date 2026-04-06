package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_DirectHashRegenerate_ExistingDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseConference        *conference.Conference
		responseDirect            *dmdirect.Direct
		responseUpdatedConference *conference.Conference

		expectUpdateFields map[conference.Field]any
	}

	tests := []test{
		{
			name: "conference with existing direct_id regenerates",

			id: uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),

			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "oldhash123",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				ResourceType: "conference",
				ResourceID:   uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
				Hash:         "newhash456",
			},
			responseUpdatedConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "newhash456",
			},

			expectUpdateFields: map[conference.Field]any{
				conference.FieldDirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				conference.FieldDirectHash: "newhash456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockReq.EXPECT().DirectV1DirectRegenerate(ctx, tt.responseConference.DirectID).Return(tt.responseDirect, nil)
			mockDB.EXPECT().ConferenceUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseUpdatedConference, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedConference, res)
			}
		})
	}
}

func Test_DirectHashRegenerate_NoDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseConference        *conference.Conference
		responseDirect            *dmdirect.Direct
		responseUpdatedConference *conference.Conference

		expectUpdateFields map[conference.Field]any
	}

	tests := []test{
		{
			name: "conference with no direct_id creates new direct",

			id: uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),

			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.Nil,
				DirectHash: "",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				ResourceType: "conference",
				ResourceID:   uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
				Hash:         "createdhash789",
			},
			responseUpdatedConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				DirectHash: "createdhash789",
			},

			expectUpdateFields: map[conference.Field]any{
				conference.FieldDirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				conference.FieldDirectHash: "createdhash789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, tt.responseConference.CustomerID, dmdirect.ResourceTypeConference, tt.id).Return(tt.responseDirect, nil)
			mockDB.EXPECT().ConferenceUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseUpdatedConference, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedConference, res)
			}
		})
	}
}
