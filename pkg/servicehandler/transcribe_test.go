package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_TranscribeGet(t *testing.T) {

	type test struct {
		name string

		customer     *cscustomer.Customer
		transcribeID uuid.UUID

		responseTranscribe *tmtranscribe.Transcribe

		expectRes *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("80546666-826f-11ed-a410-ebc3c048d175"),
			},

			uuid.FromStringOrNil("808d6e70-826f-11ed-8442-1702cf185b93"),

			&tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("808d6e70-826f-11ed-8442-1702cf185b93"),
				CustomerID: uuid.FromStringOrNil("80546666-826f-11ed-a410-ebc3c048d175"),
			},

			&tmtranscribe.WebhookMessage{
				ID:         uuid.FromStringOrNil("808d6e70-826f-11ed-8442-1702cf185b93"),
				CustomerID: uuid.FromStringOrNil("80546666-826f-11ed-a410-ebc3c048d175"),
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

			res, err := h.TranscribeGet(ctx, tt.customer, tt.transcribeID)
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

		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []tmtranscribe.Transcribe
		expectRes []*tmtranscribe.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("defbf0e8-8270-11ed-bd6a-23cb6665a292"),
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

			mockReq.EXPECT().TranscribeV1TranscribeGets(ctx, tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)
			res, err := h.TranscribeGets(ctx, tt.customer, tt.pageSize, tt.pageToken)
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

		customer      *cscustomer.Customer
		referenceType tmtranscribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     tmtranscribe.Direction

		responseCall       *cmcall.Call
		responseRecording  *cmrecording.Recording
		responseTranscribe *tmtranscribe.Transcribe

		expectRes *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			referenceType: tmtranscribe.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("cafe48aa-8281-11ed-ae72-b7dd7e37dc39"),
			language:      "en-US",
			direction:     tmtranscribe.DirectionBoth,

			responseCall: &cmcall.Call{
				ID:         uuid.FromStringOrNil("cafe48aa-8281-11ed-ae72-b7dd7e37dc39"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Status:     cmcall.StatusProgressing,
				TMDelete:   defaultTimestamp,
			},
			responseTranscribe: &tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("2b76bad2-8282-11ed-9cde-fb9aba5fd1d7"),
			},

			expectRes: &tmtranscribe.WebhookMessage{
				ID: uuid.FromStringOrNil("2b76bad2-8282-11ed-9cde-fb9aba5fd1d7"),
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
			case transcribe.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)

			case transcribe.ReferenceTypeRecording:
				mockReq.EXPECT().CallV1RecordingGet(ctx, tt.referenceID).Return(tt.responseRecording, nil)
			}
			mockReq.EXPECT().TranscribeV1TranscribeStart(ctx, tt.customer.ID, tt.referenceType, tt.referenceID, tt.language, tt.direction).Return(tt.responseTranscribe, nil)

			res, err := h.TranscribeStart(ctx, tt.customer, tt.referenceType, tt.referenceID, tt.language, tt.direction)
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

		customer *cscustomer.Customer

		transcribeID uuid.UUID

		responseTranscribe *tmtranscribe.Transcribe

		expectRes *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("d83aff44-8282-11ed-851f-1f49a32483fb"),
			},
			transcribeID: uuid.FromStringOrNil("d86d88d8-8282-11ed-b6c2-c3ac86331ed9"),

			responseTranscribe: &tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("d86d88d8-8282-11ed-b6c2-c3ac86331ed9"),
				CustomerID: uuid.FromStringOrNil("d83aff44-8282-11ed-851f-1f49a32483fb"),
			},

			expectRes: &tmtranscribe.WebhookMessage{
				ID:         uuid.FromStringOrNil("d86d88d8-8282-11ed-b6c2-c3ac86331ed9"),
				CustomerID: uuid.FromStringOrNil("d83aff44-8282-11ed-851f-1f49a32483fb"),
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

			res, err := h.TranscribeStop(ctx, tt.customer, tt.transcribeID)
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

		customer *cscustomer.Customer

		transcribeID uuid.UUID

		responseTranscribe *tmtranscribe.Transcribe

		expectRes *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("7176ea4c-8283-11ed-95d3-a72987927819"),
			},
			transcribeID: uuid.FromStringOrNil("719adccc-8283-11ed-973c-df1113145910"),

			responseTranscribe: &tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("719adccc-8283-11ed-973c-df1113145910"),
				CustomerID: uuid.FromStringOrNil("7176ea4c-8283-11ed-95d3-a72987927819"),
			},

			expectRes: &tmtranscribe.WebhookMessage{
				ID:         uuid.FromStringOrNil("719adccc-8283-11ed-973c-df1113145910"),
				CustomerID: uuid.FromStringOrNil("7176ea4c-8283-11ed-95d3-a72987927819"),
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

			res, err := h.TranscribeDelete(ctx, tt.customer, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
