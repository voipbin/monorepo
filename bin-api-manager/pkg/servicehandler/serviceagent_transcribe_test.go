package servicehandler

import (
	"context"
	"reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ServiceAgentTranscribeList(t *testing.T) {

	tests := []struct {
		name string

		agent         *auth.AuthIdentity
		pageToken     string
		pageSize      uint64
		referenceType string
		referenceID   uuid.UUID

		response []tmtranscribe.Transcribe

		expectFilters map[tmtranscribe.Field]any
		expectRes     []*tmtranscribe.WebhookMessage
	}{
		{
			// Plain Agent permission (not Admin/Manager) must be able to list
			// its own customer's transcribes via the service_agents surface.
			// This is the whole point of the new endpoint.
			"agent permission, no reference filter",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"",
			uuid.Nil,

			[]tmtranscribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[tmtranscribe.Field]any{
				tmtranscribe.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				tmtranscribe.FieldDeleted:    false,
			},
			[]*tmtranscribe.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},
		},
		{
			"agent permission, filtered by reference_type and reference_id",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"call",
			uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),

			[]tmtranscribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[tmtranscribe.Field]any{
				tmtranscribe.FieldCustomerID:    uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				tmtranscribe.FieldDeleted:       false,
				tmtranscribe.FieldReferenceType: "call",
				tmtranscribe.FieldReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			},
			[]*tmtranscribe.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
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

			mockReq.EXPECT().TranscribeV1TranscribeList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)
			res, err := h.ServiceAgentTranscribeList(ctx, tt.agent, tt.pageSize, tt.pageToken, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentTranscribeStart(t *testing.T) {

	type test struct {
		name string

		agent         *auth.AuthIdentity
		referenceType string
		referenceID   uuid.UUID
		language      string
		direction     tmtranscribe.Direction
		onEndFlowID   uuid.UUID
		provider      tmtranscribe.Provider

		responseCall       *cmcall.Call
		responseRecording  *cmrecording.Recording
		responseTranscribe *tmtranscribe.Transcribe

		expectReferenceType tmtranscribe.ReferenceType
		expectRes           *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			// Plain Agent permission must be able to start a transcribe on a
			// call belonging to its own customer via the service_agents
			// surface.
			name: "agent permission, call reference",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			referenceType: "call",
			referenceID:   uuid.FromStringOrNil("cafe48aa-8281-11ed-ae72-b7dd7e37dc39"),
			language:      "en-US",
			direction:     tmtranscribe.DirectionBoth,
			onEndFlowID:   uuid.FromStringOrNil("9772a0da-0943-11f0-879f-47ce2d322564"),
			provider:      tmtranscribe.ProviderGCP,

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cafe48aa-8281-11ed-ae72-b7dd7e37dc39"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Status:   cmcall.StatusProgressing,
				TMDelete: nil,
			},
			responseTranscribe: &tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2b76bad2-8282-11ed-9cde-fb9aba5fd1d7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectReferenceType: tmtranscribe.ReferenceTypeCall,
			expectRes: &tmtranscribe.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2b76bad2-8282-11ed-9cde-fb9aba5fd1d7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			switch tt.referenceType {
			case "call":
				mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)

			case "recording":
				mockReq.EXPECT().CallV1RecordingGet(ctx, tt.referenceID).Return(tt.responseRecording, nil)
			}
			mockReq.EXPECT().TranscribeV1TranscribeStart(
				ctx,
				tt.agent.CustomerID,
				uuid.Nil,
				tt.onEndFlowID,
				tt.expectReferenceType,
				tt.referenceID,
				tt.language,
				tt.direction,
				tt.provider,
				60000,
			).Return(tt.responseTranscribe, nil)

			res, err := h.ServiceAgentTranscribeStart(ctx, tt.agent, tt.referenceType, tt.referenceID, tt.language, tt.direction, tt.onEndFlowID, tt.provider)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
