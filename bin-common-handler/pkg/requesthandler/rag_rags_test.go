package requesthandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	rmdocument "monorepo/bin-rag-manager/models/document"
	rmquery "monorepo/bin-rag-manager/models/query"
	rmrag "monorepo/bin-rag-manager/models/rag"
)

const ragQueue = "bin-manager.rag-manager.request"

func Test_RagV1RagWrappers(t *testing.T) {
	ragID := uuid.FromStringOrNil("5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59")
	customerID := uuid.FromStringOrNil("a2c4e6f8-1b3d-4f57-9a8c-6e0d2b4f6a81")
	fileID := uuid.FromStringOrNil("0b9d7f5e-3c1a-4e86-8d24-1a5c7e9b3d62")
	sourceID := uuid.FromStringOrNil("7d3f1b9c-5a2e-4c48-b6e0-9f1d3b5a7c24")

	cases := []rpcCase{
		{
			name:           "query_with_top_k",
			queue:          ragQueue,
			expectURI:      "/v1/query",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"rag_id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59","query":"how to reset","top_k":3}`,
			responseData:   `{"sources":[{"document_id":"0b9d7f5e-3c1a-4e86-8d24-1a5c7e9b3d62","document_name":"guide","section_title":"reset","relevance_score":0.5,"text":"chunk"}]}`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagQuery(context.Background(), ragID, "how to reset", 3)
			},
			expectRes: &rmquery.Response{Sources: []rmquery.Source{
				{DocumentID: fileID, DocumentName: "guide", SectionTitle: "reset", RelevanceScore: 0.5, Text: "chunk"},
			}},
		},
		{
			name:           "query_without_top_k",
			queue:          ragQueue,
			expectURI:      "/v1/query",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"rag_id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59","query":"q"}`,
			responseData:   `{"sources":[]}`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagQuery(context.Background(), ragID, "q", 0)
			},
			expectRes: &rmquery.Response{Sources: []rmquery.Source{}},
		},
		{
			name:           "create",
			queue:          ragQueue,
			expectURI:      "/v1/rags",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"a2c4e6f8-1b3d-4f57-9a8c-6e0d2b4f6a81","name":"kb","description":"d","storage_file_ids":["0b9d7f5e-3c1a-4e86-8d24-1a5c7e9b3d62"],"source_urls":["https://docs.example.invalid/a"]}`,
			responseData:   `{"id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59","name":"kb"}`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagCreate(context.Background(), customerID, "kb", "d", []uuid.UUID{fileID}, []string{"https://docs.example.invalid/a"})
			},
			expectRes: &rmrag.Rag{ID: ragID, Name: "kb"},
		},
		{
			name:           "add_sources",
			queue:          ragQueue,
			expectURI:      "/v1/rags/5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59/sources",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"storage_file_ids":null,"source_urls":["https://docs.example.invalid/b"]}`,
			responseData:   `{"id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59","sources":[{"id":"7d3f1b9c-5a2e-4c48-b6e0-9f1d3b5a7c24","source_url":"https://docs.example.invalid/b","status":"pending"}]}`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagAddSources(context.Background(), ragID, nil, []string{"https://docs.example.invalid/b"})
			},
			expectRes: &rmrag.Rag{ID: ragID, Sources: []rmrag.Source{
				{Identity: identity.Identity{ID: sourceID}, SourceURL: "https://docs.example.invalid/b", Status: rmdocument.StatusPending},
			}},
		},
		{
			name:           "get",
			queue:          ragQueue,
			expectURI:      "/v1/rags/5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			responseData:   `{"id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59","description":"desc"}`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagGet(context.Background(), ragID)
			},
			expectRes: &rmrag.Rag{ID: ragID, Description: "desc"},
		},
		{
			name:           "gets",
			queue:          ragQueue,
			expectURI:      "/v1/rags?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=5",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"a2c4e6f8-1b3d-4f57-9a8c-6e0d2b4f6a81","name":"kb"}`,
			responseData:   `[{"id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59"}]`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagGets(context.Background(), "2020-09-20T03:23:20.995000Z", 5, map[rmrag.Field]any{rmrag.FieldCustomerID: customerID, rmrag.FieldName: "kb"})
			},
			expectRes: []*rmrag.Rag{{ID: ragID}},
		},
		{
			name:           "update",
			queue:          ragQueue,
			expectURI:      "/v1/rags/5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{"description":"new","name":"renamed"}`,
			responseData:   `{"id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59","name":"renamed","description":"new"}`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagUpdate(context.Background(), ragID, map[rmrag.Field]any{rmrag.FieldName: "renamed", rmrag.FieldDescription: "new"})
			},
			expectRes: &rmrag.Rag{ID: ragID, Name: "renamed", Description: "new"},
		},
		{
			name:           "remove_source",
			queue:          ragQueue,
			expectURI:      "/v1/rags/5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59/sources/7d3f1b9c-5a2e-4c48-b6e0-9f1d3b5a7c24",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeJSON,
			responseData:   `{"id":"5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59"}`,
			call: func(h *requestHandler) (any, error) {
				return h.RagV1RagRemoveSource(context.Background(), ragID, sourceID)
			},
			expectRes: &rmrag.Rag{ID: ragID},
		},
		{
			name:           "delete",
			queue:          ragQueue,
			expectURI:      "/v1/rags/5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeJSON,
			responseData:   `{}`,
			call: func(h *requestHandler) (any, error) {
				return nil, h.RagV1RagDelete(context.Background(), ragID)
			},
			expectRes:  nil,
			statusMode: statusCustom,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "gets_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.RagV1RagGets(ctx, "", 5, map[rmrag.Field]any{rmrag.FieldName: make(chan int)})
	})
	runMarshalErrorCase(t, "update_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.RagV1RagUpdate(ctx, ragID, map[rmrag.Field]any{rmrag.FieldName: make(chan int)})
	})
}

func Test_RagV1RagDelete_onlyStatus200IsSuccess(t *testing.T) {
	ragID := uuid.FromStringOrNil("5e1f3b7a-9c24-4d68-b0a1-3f7d2c8e4a59")

	tests := []struct {
		name       string
		statusCode int
		expectErr  bool
	}{
		{name: "ok", statusCode: 200, expectErr: false},
		{name: "no content is not accepted", statusCode: 204, expectErr: true},
		{name: "server error", statusCode: 500, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), ragQueue, gomock.Any()).Return(&sock.Response{StatusCode: tt.statusCode}, nil)

			err := h.RagV1RagDelete(context.Background(), ragID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if err != nil {
				if errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "status") {
					t.Errorf("Wrong match. unexpected error: %v", err)
				}
			}
		})
	}
}
