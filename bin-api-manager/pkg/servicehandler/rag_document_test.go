package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	rmdocument "monorepo/bin-rag-manager/models/document"
	rmrag "monorepo/bin-rag-manager/models/rag"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_RagDocumentCreate(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		ragID         uuid.UUID
		docName       string
		docType       rmdocument.DocType
		sourceURL     string
		storageFileID uuid.UUID

		responseRag *rmrag.Rag
		responseDoc *rmdocument.Document
		expectRes   *rmdocument.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			ragID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			docName:       "test document",
			docType:       rmdocument.DocTypeURL,
			sourceURL:     "https://example.com/doc.pdf",
			storageFileID: uuid.Nil,

			responseRag: &rmrag.Rag{
				ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			responseDoc: &rmdocument.Document{
				ID:         uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				Name:       "test document",
				DocType:    rmdocument.DocTypeURL,
				SourceURL:  "https://example.com/doc.pdf",
				Status:     rmdocument.StatusPending,
			},
			expectRes: &rmdocument.WebhookMessage{
				ID:         uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				Name:       "test document",
				DocType:    rmdocument.DocTypeURL,
				SourceURL:  "https://example.com/doc.pdf",
				Status:     rmdocument.StatusPending,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RagV1RagGet(ctx, tt.ragID).Return(tt.responseRag, nil)
			mockReq.EXPECT().RagV1DocumentCreate(
				ctx,
				tt.agent.CustomerID,
				tt.ragID,
				tt.docName,
				tt.docType,
				tt.sourceURL,
				tt.storageFileID,
			).Return(tt.responseDoc, nil)

			res, err := h.RagDocumentCreate(
				ctx,
				tt.agent,
				tt.ragID,
				tt.docName,
				tt.docType,
				tt.sourceURL,
				tt.storageFileID,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RagDocumentGet(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		docID uuid.UUID

		response  *rmdocument.Document
		expectRes *rmdocument.WebhookMessage
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
			uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),

			&rmdocument.Document{
				ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},
			&rmdocument.WebhookMessage{
				ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RagV1DocumentGet(ctx, tt.docID).Return(tt.response, nil)

			res, err := h.RagDocumentGet(ctx, tt.agent, tt.docID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RagDocumentGets(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		ragID   uuid.UUID
		size    uint64
		token   string
		filters map[rmdocument.Field]any

		response  []*rmdocument.Document
		expectRes []*rmdocument.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			ragID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			size:  10,
			token: "2020-09-20T03:23:20.995000Z",
			filters: map[rmdocument.Field]any{
				rmdocument.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				rmdocument.FieldRagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},

			response: []*rmdocument.Document{
				{
					ID:         uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				},
			},
			expectRes: []*rmdocument.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RagV1DocumentGets(ctx, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.RagDocumentGets(ctx, tt.agent, tt.ragID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RagDocumentDelete(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		docID uuid.UUID

		responseDoc *rmdocument.Document
		expectRes   *rmdocument.WebhookMessage
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
			uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			&rmdocument.Document{
				ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},
			&rmdocument.WebhookMessage{
				ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RagV1DocumentGet(ctx, tt.docID).Return(tt.responseDoc, nil)
			mockReq.EXPECT().RagV1DocumentDelete(ctx, tt.docID).Return(nil)

			res, err := h.RagDocumentDelete(ctx, tt.agent, tt.docID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
