package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_TranscribeGet(t *testing.T) {

	type test struct {
		name string

		agent        *amagent.Agent
		transcribeID uuid.UUID

		responseTranscribe *tmtranscribe.Transcribe

		expectRes *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("808d6e70-826f-11ed-8442-1702cf185b93"),

			&tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("808d6e70-826f-11ed-8442-1702cf185b93"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			&tmtranscribe.WebhookMessage{
				ID:         uuid.FromStringOrNil("808d6e70-826f-11ed-8442-1702cf185b93"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)

			res, err := h.TranscribeGet(ctx, tt.agent, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeGets(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		response []tmtranscribe.Transcribe

		expectFilters map[string]string
		expectRes     []*tmtranscribe.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2020-10-20 01:00:00.995000",
			10,

			[]tmtranscribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				},
				{
					ID: uuid.FromStringOrNil("df6c8bf0-8270-11ed-8a5a-0b5818b7baac"),
				},
			},

			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			[]*tmtranscribe.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				},
				{
					ID: uuid.FromStringOrNil("df6c8bf0-8270-11ed-8a5a-0b5818b7baac"),
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

			mockReq.EXPECT().TranscribeV1TranscribeGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)
			res, err := h.TranscribeGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeStart(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		referenceType request.TranscribeReferenceType
		referenceID   uuid.UUID
		language      string
		direction     tmtranscribe.Direction

		responseCall       *cmcall.Call
		responseRecording  *cmrecording.Recording
		responseTranscribe *tmtranscribe.Transcribe

		expectReferenceType tmtranscribe.ReferenceType
		expectRes           *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			referenceType: request.TranscribeReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("cafe48aa-8281-11ed-ae72-b7dd7e37dc39"),
			language:      "en-US",
			direction:     tmtranscribe.DirectionBoth,

			responseCall: &cmcall.Call{
				ID:         uuid.FromStringOrNil("cafe48aa-8281-11ed-ae72-b7dd7e37dc39"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Status:     cmcall.StatusProgressing,
				TMDelete:   defaultTimestamp,
			},
			responseTranscribe: &tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("2b76bad2-8282-11ed-9cde-fb9aba5fd1d7"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			expectReferenceType: tmtranscribe.ReferenceTypeCall,
			expectRes: &tmtranscribe.WebhookMessage{
				ID:         uuid.FromStringOrNil("2b76bad2-8282-11ed-9cde-fb9aba5fd1d7"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
			case request.TranscribeReferenceTypeCall:
				mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)

			case request.TranscribeReferenceTypeRecording:
				mockReq.EXPECT().CallV1RecordingGet(ctx, tt.referenceID).Return(tt.responseRecording, nil)
			}
			mockReq.EXPECT().TranscribeV1TranscribeStart(ctx, tt.agent.CustomerID, tt.expectReferenceType, tt.referenceID, tt.language, tt.direction).Return(tt.responseTranscribe, nil)

			res, err := h.TranscribeStart(ctx, tt.agent, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeStop(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent

		transcribeID uuid.UUID

		responseTranscribe *tmtranscribe.Transcribe

		expectRes *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			transcribeID: uuid.FromStringOrNil("d86d88d8-8282-11ed-b6c2-c3ac86331ed9"),

			responseTranscribe: &tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("d86d88d8-8282-11ed-b6c2-c3ac86331ed9"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			expectRes: &tmtranscribe.WebhookMessage{
				ID:         uuid.FromStringOrNil("d86d88d8-8282-11ed-b6c2-c3ac86331ed9"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			// transcribeGet
			mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)

			mockReq.EXPECT().TranscribeV1TranscribeStop(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)

			res, err := h.TranscribeStop(ctx, tt.agent, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeDelete(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent

		transcribeID uuid.UUID

		responseTranscribe *tmtranscribe.Transcribe

		expectRes *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			transcribeID: uuid.FromStringOrNil("719adccc-8283-11ed-973c-df1113145910"),

			responseTranscribe: &tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("719adccc-8283-11ed-973c-df1113145910"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			expectRes: &tmtranscribe.WebhookMessage{
				ID:         uuid.FromStringOrNil("719adccc-8283-11ed-973c-df1113145910"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			// transcribeGet
			mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)

			mockReq.EXPECT().TranscribeV1TranscribeDelete(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)

			res, err := h.TranscribeDelete(ctx, tt.agent, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
