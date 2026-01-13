package servicehandler

import (
	"context"
	"reflect"
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_RecordingGets(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		size  uint64
		token string

		responseRecording []cmrecording.Recording

		expectFilters map[string]string
		expectRes     []*cmrecording.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-10-20 01:00:00.995000",

			[]cmrecording.Recording{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					Filenames: []string{
						"call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("43259aa4-6146-11eb-acb2-6b996101131d"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					Filenames: []string{
						"call_2f167946-b2b4-4370-94fa-d6c2c57c84da_2020-12-04T18:48:03Z",
					},
				},
			},

			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			[]*cmrecording.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("34a87712-6146-11eb-be45-83bc6e54dfb9"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("43259aa4-6146-11eb-acb2-6b996101131d"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGets(ctx, tt.token, tt.size, tt.expectFilters.Return(tt.responseRecording, nil)

			res, err := h.RecordingGets(ctx, tt.agent, tt.size, tt.token)

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res[0])
			}
		})
	}
}

func Test_RecordingDelete(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		recordingID uuid.UUID

		responseRecording *cmrecording.Recording
		expectRes         *cmrecording.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("8f7a8b7e-8f1d-11ed-be94-07c28fd4c979"),

			&cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8f7a8b7e-8f1d-11ed-be94-07c28fd4c979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},

			&cmrecording.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8f7a8b7e-8f1d-11ed-be94-07c28fd4c979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGet(ctx, tt.recordingID.Return(tt.responseRecording, nil)
			mockReq.EXPECT().CallV1RecordingDelete(ctx, tt.recordingID.Return(tt.responseRecording, nil)

			res, err := h.RecordingDelete(ctx, tt.agent, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
