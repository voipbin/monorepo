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

func Test_ServiceAgentTranscriptList(t *testing.T) {

	tests := []struct {
		name string

		agent        *auth.AuthIdentity
		pageSize     uint64
		pageToken    string
		transcribeID uuid.UUID

		responseTranscribe  *tmtranscribe.Transcribe
		responseTranscripts []tmtranscript.Transcript

		expectFilters map[tmtranscript.Field]any
		expectRes     []*tmtranscript.WebhookMessage
	}{
		{
			// Plain Agent permission (not Admin/Manager) must be able to list
			// transcript lines for a transcribe belonging to its own
			// customer via the service_agents surface. This is the whole
			// point of the new endpoint.
			"agent permission, own customer's transcribe",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			10,
			"2020-10-20T01:00:00.995000Z",
			uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),

			&tmtranscribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			[]tmtranscript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					TranscribeID: uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),
					Message:      "Hello, how can I help you?",
				},
			},

			map[tmtranscript.Field]any{
				tmtranscript.FieldTranscribeID: uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),
				tmtranscript.FieldDeleted:      false,
			},
			[]*tmtranscript.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					TranscribeID: uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e"),
					Message:      "Hello, how can I help you?",
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

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)
			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseTranscripts, nil)

			res, err := h.ServiceAgentTranscriptList(ctx, tt.agent, tt.pageSize, tt.pageToken, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentTranscriptList_CrossCustomerDenied(t *testing.T) {
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

	// Agent belongs to a DIFFERENT customer than the fetched transcribe —
	// must be denied even though the transcribe_id itself is valid and
	// resolves successfully. This is the IDOR/cross-tenant boundary check.
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})
	transcribeID := uuid.FromStringOrNil("b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e")

	mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(&tmtranscribe.Transcribe{
		Identity: commonidentity.Identity{
			ID:         transcribeID,
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
	}, nil)
	mockUtil.EXPECT().TimeGetCurTime().Return("2020-10-20T01:00:00.995000Z")

	res, err := h.ServiceAgentTranscriptList(ctx, agent, 100, "", transcribeID)
	if err != serviceerrors.ErrPermissionDenied {
		t.Errorf("Wrong match. expect: ErrPermissionDenied, got: %v, res: %v", err, res)
	}
}

func Test_ServiceAgentTranscriptList_TranscribeNotFound(t *testing.T) {
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
	// layer (see server/service_agents_transcripts.go), which flows into
	// transcribeGet the same way any not-found transcribe_id would — the
	// RPC layer is expected to surface a not-found error, which must
	// propagate here rather than panic or silently succeed.
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})

	expectErr := serviceerrors.ErrNotFound

	mockUtil.EXPECT().TimeGetCurTime().Return("2020-10-20T01:00:00.995000Z")
	mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, uuid.Nil).Return(nil, expectErr)

	res, err := h.ServiceAgentTranscriptList(ctx, agent, 100, "", uuid.Nil)
	if err != expectErr {
		t.Errorf("Wrong match. expect: %v, got: %v, res: %v", expectErr, err, res)
	}
}
