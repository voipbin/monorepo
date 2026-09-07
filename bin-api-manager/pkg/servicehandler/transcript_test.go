package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
)

func Test_TranscriptList(t *testing.T) {

	type test struct {
		name string

		agent        *auth.AuthIdentity
		pageSize     uint64
		pageToken    string
		transcribeID uuid.UUID

		responseCurTime     string
		responseTranscribe  *tmtranscribe.Transcribe
		responseTranscripts []tmtranscript.Transcript

		expectToken   string
		expectFilters map[tmtranscript.Field]any
		expectRes     []*tmtranscript.WebhookMessage
	}

	adminAgent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
	transcribeID := uuid.FromStringOrNil("9eafc870-8284-11ed-92de-d74d9e2342cb")
	transcribe := &tmtranscribe.Transcribe{
		Identity: commonidentity.Identity{
			ID:         transcribeID,
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	}
	transcripts := []tmtranscript.Transcript{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9ede9632-8284-11ed-bf13-43420adb75f6")}},
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9f06037a-8284-11ed-8b1a-1f5800b90993")}},
	}
	filters := map[tmtranscript.Field]any{
		tmtranscript.FieldTranscribeID: transcribeID,
		tmtranscript.FieldDeleted:      false,
	}
	expectRes := []*tmtranscript.WebhookMessage{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9ede9632-8284-11ed-bf13-43420adb75f6")}},
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9f06037a-8284-11ed-8b1a-1f5800b90993")}},
	}

	tests := []test{
		{
			// An explicit token and size must reach transcribe-manager
			// verbatim -- this is the VOIP-1480 regression case (the admin
			// surface used to send "" and 100 regardless).
			name: "explicit token and size are passed through",

			agent:        adminAgent,
			pageSize:     10,
			pageToken:    "2020-10-20T01:00:00.995000Z",
			transcribeID: transcribeID,

			responseTranscribe:  transcribe,
			responseTranscripts: transcripts,

			expectToken:   "2020-10-20T01:00:00.995000Z",
			expectFilters: filters,
			expectRes:     expectRes,
		},
		{
			// An empty token defaults to "now" in api-manager, as
			// TranscribeList and ServiceAgentTranscriptList do.
			name: "empty token defaults to now",

			agent:        adminAgent,
			pageSize:     100,
			pageToken:    "",
			transcribeID: transcribeID,

			responseCurTime:     "2020-10-20T01:00:00.995000Z",
			responseTranscribe:  transcribe,
			responseTranscripts: transcripts,

			expectToken:   "2020-10-20T01:00:00.995000Z",
			expectFilters: filters,
			expectRes:     expectRes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			if tt.pageToken == "" {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			}
			mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)
			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, tt.expectToken, tt.pageSize, tt.expectFilters).Return(tt.responseTranscripts, nil)

			res, err := h.TranscriptList(ctx, tt.agent, tt.pageSize, tt.pageToken, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscriptList_CrossCustomerDenied(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	// A CustomerAdmin of a DIFFERENT customer than the fetched transcribe
	// must be denied even though the transcribe_id resolves. The token
	// default runs before the permission check (design 2.2), so
	// TimeGetCurTime is still called; the transcript list RPC must not be.
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
	transcribeID := uuid.FromStringOrNil("9eafc870-8284-11ed-92de-d74d9e2342cb")

	mockUtil.EXPECT().TimeGetCurTime().Return("2020-10-20T01:00:00.995000Z")
	mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(&tmtranscribe.Transcribe{
		Identity: commonidentity.Identity{
			ID:         transcribeID,
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
	}, nil)

	res, err := h.TranscriptList(ctx, agent, 100, "", transcribeID)
	if err != serviceerrors.ErrPermissionDenied {
		t.Errorf("Wrong match. expect: ErrPermissionDenied, got: %v, res: %v", err, res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func Test_TranscriptList_DirectAccessNotSupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	// A direct-scope identity (TypeDirect, not an accesskey) is rejected
	// before anything else runs: no TimeGetCurTime, no RPC.
	agent := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		AllowedResourceTypes: []string{"call"},
	})

	res, err := h.TranscriptList(ctx, agent, 100, "", uuid.FromStringOrNil("9eafc870-8284-11ed-92de-d74d9e2342cb"))
	if err != serviceerrors.ErrDirectAccessNotSupported {
		t.Errorf("Wrong match. expect: ErrDirectAccessNotSupported, got: %v, res: %v", err, res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func Test_TranscriptList_SameCustomerNoPermissionDenied(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	// An agent of the SAME customer as the fetched transcribe, but without
	// PermissionCustomerAdmin/Manager, must still be denied. The token
	// default runs before the permission check (design 2.2), so
	// TimeGetCurTime is still called; the transcript list RPC must not be.
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionNone,
	})
	transcribeID := uuid.FromStringOrNil("9eafc870-8284-11ed-92de-d74d9e2342cb")

	mockUtil.EXPECT().TimeGetCurTime().Return("2020-10-20T01:00:00.995000Z")
	mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(&tmtranscribe.Transcribe{
		Identity: commonidentity.Identity{
			ID:         transcribeID,
			CustomerID: customerID,
		},
	}, nil)

	res, err := h.TranscriptList(ctx, agent, 100, "", transcribeID)
	if err != serviceerrors.ErrPermissionDenied {
		t.Errorf("Wrong match. expect: ErrPermissionDenied, got: %v, res: %v", err, res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func Test_TranscriptList_TranscribeNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	// A malformed/missing transcribe_id resolves to uuid.Nil at the HTTP
	// layer (see server/transcripts.go), which flows into transcribeGet the
	// same way any not-found transcribe_id would -- any error transcribeGet
	// returns must propagate here unwrapped rather than panic or silently
	// succeed. The mock returns an api-manager sentinel (serviceerrors.ErrNotFound)
	// purely so the test can assert the returned error's identity; it is not
	// asserting that transcribe-manager's RPC layer specifically returns a
	// not-found error.
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	expectErr := serviceerrors.ErrNotFound

	mockUtil.EXPECT().TimeGetCurTime().Return("2020-10-20T01:00:00.995000Z")
	mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, uuid.Nil).Return(nil, expectErr)

	res, err := h.TranscriptList(ctx, agent, 100, "", uuid.Nil)
	if err != expectErr {
		t.Errorf("Wrong match. expect: %v, got: %v, res: %v", expectErr, err, res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}
