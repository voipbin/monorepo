package listenhandler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-rag-manager/models/query"
	"monorepo/bin-rag-manager/models/rag"
	"monorepo/bin-rag-manager/pkg/raghandler"
)

var (
	covRagID      = uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
	covCustomerID = uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222")
	covFileID     = uuid.FromStringOrNil("33333333-3333-4333-8333-333333333333")
	covSourceID   = uuid.FromStringOrNil("44444444-4444-4444-8444-444444444444")

	covRagJSON = `{"id":"11111111-1111-4111-8111-111111111111","customer_id":"22222222-2222-4222-8222-222222222222","name":"kb","description":"desc"}`
)

func covRag() *rag.Rag {
	return &rag.Rag{
		ID:          covRagID,
		CustomerID:  covCustomerID,
		Name:        "kb",
		Description: "desc",
	}
}

func Test_processV1RagsPost(t *testing.T) {
	tests := []struct {
		name string

		data []byte

		expectRagCreate bool
		ragCreateErr    error
		expectStatus    int
		expectBody      string
	}{
		{
			name:         "invalid json",
			data:         []byte(`{"name":`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "missing customer_id",
			data:         []byte(`{"name":"kb","source_urls":["https://example.invalid/a.txt"]}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "missing name",
			data:         []byte(`{"customer_id":"22222222-2222-4222-8222-222222222222","source_urls":["https://example.invalid/a.txt"]}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "no sources",
			data:         []byte(`{"customer_id":"22222222-2222-4222-8222-222222222222","name":"kb"}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:            "success",
			data:            []byte(`{"customer_id":"22222222-2222-4222-8222-222222222222","name":"kb","description":"desc","storage_file_ids":["33333333-3333-4333-8333-333333333333"],"source_urls":["https://example.invalid/a.txt"]}`),
			expectRagCreate: true,
			expectStatus:    http.StatusOK,
			expectBody:      covRagJSON,
		},
		{
			name:            "handler error",
			data:            []byte(`{"customer_id":"22222222-2222-4222-8222-222222222222","name":"kb","description":"desc","storage_file_ids":["33333333-3333-4333-8333-333333333333"],"source_urls":["https://example.invalid/a.txt"]}`),
			expectRagCreate: true,
			ragCreateErr:    errors.New("boom"),
			expectStatus:    http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			ctx := context.Background()

			if tt.expectRagCreate {
				var res *rag.Rag
				if tt.ragCreateErr == nil {
					res = covRag()
				}
				mockRag.EXPECT().RagCreate(ctx, covCustomerID, "kb", "desc", []uuid.UUID{covFileID}, []string{"https://example.invalid/a.txt"}).Return(res, tt.ragCreateErr)
			}

			res, err := h.processV1RagsPost(ctx, &sock.Request{URI: "/v1/rags", Method: sock.RequestMethodPost, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
			if string(res.Data) != tt.expectBody {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBody, string(res.Data))
			}
		})
	}
}

func Test_processV1RagsGet(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data []byte

		expectList       bool
		expectPageSize   uint64
		expectPageToken  string
		expectFilters    map[rag.Field]any
		listErr          error
		expectStatus     int
		expectBody       string
		listResponseRags []*rag.Rag
	}{
		{
			name:         "uri parse error",
			uri:          "/v1/rags?\x01",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "filters are not json",
			uri:          "/v1/rags",
			data:         []byte(`{"customer_id":`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "filter value cannot be converted",
			uri:          "/v1/rags",
			data:         []byte(`{"customer_id":"not-a-uuid"}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:             "page params and customer filter",
			uri:              "/v1/rags?page_size=10&page_token=2026-01-01T00:00:00.000000Z",
			data:             []byte(`{"customer_id":"22222222-2222-4222-8222-222222222222"}`),
			expectList:       true,
			expectPageSize:   10,
			expectPageToken:  "2026-01-01T00:00:00.000000Z",
			expectFilters:    map[rag.Field]any{rag.FieldCustomerID: covCustomerID},
			listResponseRags: []*rag.Rag{covRag()},
			expectStatus:     http.StatusOK,
			expectBody:       "[" + covRagJSON + "]",
		},
		{
			name:             "no params and no filters",
			uri:              "/v1/rags",
			expectList:       true,
			expectFilters:    map[rag.Field]any{},
			listResponseRags: []*rag.Rag{},
			expectStatus:     http.StatusOK,
			expectBody:       "[]",
		},
		{
			name:             "unknown filter key is ignored",
			uri:              "/v1/rags",
			data:             []byte(`{"unknown_key":"x"}`),
			expectList:       true,
			expectFilters:    map[rag.Field]any{},
			listResponseRags: []*rag.Rag{},
			expectStatus:     http.StatusOK,
			expectBody:       "[]",
		},
		{
			name:          "handler error",
			uri:           "/v1/rags",
			expectList:    true,
			expectFilters: map[rag.Field]any{},
			listErr:       errors.New("boom"),
			expectStatus:  http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			ctx := context.Background()

			if tt.expectList {
				mockRag.EXPECT().RagList(ctx, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.listResponseRags, tt.listErr)
			}

			res, err := h.processV1RagsGet(ctx, &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
			if string(res.Data) != tt.expectBody {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBody, string(res.Data))
			}
		})
	}
}

func Test_processV1RagsID_invalidURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		call func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)
		data []byte
	}{
		{"get short uri", "/v1/rags", (*listenHandler).processV1RagsIDGet, nil},
		{"get invalid uuid", "/v1/rags/not-a-uuid", (*listenHandler).processV1RagsIDGet, nil},
		{"put short uri", "/v1/rags", (*listenHandler).processV1RagsIDPut, []byte(`{"name":"n"}`)},
		{"put invalid uuid", "/v1/rags/not-a-uuid", (*listenHandler).processV1RagsIDPut, []byte(`{"name":"n"}`)},
		{"delete short uri", "/v1/rags", (*listenHandler).processV1RagsIDDelete, nil},
		{"delete invalid uuid", "/v1/rags/not-a-uuid", (*listenHandler).processV1RagsIDDelete, nil},
		{"sources post short uri", "/v1/rags/" + covRagID.String(), (*listenHandler).processV1RagsIDSourcesPost, []byte(`{"source_urls":["https://example.invalid/a.txt"]}`)},
		{"sources post invalid rag uuid", "/v1/rags/not-a-uuid/sources", (*listenHandler).processV1RagsIDSourcesPost, []byte(`{"source_urls":["https://example.invalid/a.txt"]}`)},
		{"source delete short uri", "/v1/rags/" + covRagID.String() + "/sources", (*listenHandler).processV1RagsIDSourcesIDDelete, nil},
		{"source delete invalid rag uuid", "/v1/rags/not-a-uuid/sources/" + covSourceID.String(), (*listenHandler).processV1RagsIDSourcesIDDelete, nil},
		{"source delete invalid source uuid", "/v1/rags/" + covRagID.String() + "/sources/not-a-uuid", (*listenHandler).processV1RagsIDSourcesIDDelete, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation is registered, so any rag handler call fails the test.
			h := newTestHandler(t, raghandler.NewMockRagHandler(mc))

			res, err := tt.call(h, context.Background(), &sock.Request{URI: tt.uri, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %v, got: %v", http.StatusBadRequest, res.StatusCode)
			}
		})
	}
}

func Test_processV1RagsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRag := raghandler.NewMockRagHandler(mc)
	h := newTestHandler(t, mockRag)
	ctx := context.Background()

	mockRag.EXPECT().RagGet(ctx, covRagID).Return(covRag(), nil)

	res, err := h.processV1RagsIDGet(ctx, &sock.Request{URI: "/v1/rags/" + covRagID.String(), Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Wrong match. expect: %v, got: %v", http.StatusOK, res.StatusCode)
	}
	if string(res.Data) != covRagJSON {
		t.Errorf("Wrong match. expect: %v, got: %v", covRagJSON, string(res.Data))
	}
}

func Test_processV1RagsIDPut(t *testing.T) {
	tests := []struct {
		name string

		data []byte

		expectUpdate bool
		expectFields map[rag.Field]any
		updateErr    error
		expectStatus int
		expectBody   string
	}{
		{
			name:         "invalid json",
			data:         []byte(`{"name":`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "no fields",
			data:         []byte(`{}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "name only",
			data:         []byte(`{"name":"kb"}`),
			expectUpdate: true,
			expectFields: map[rag.Field]any{rag.FieldName: "kb"},
			expectStatus: http.StatusOK,
			expectBody:   covRagJSON,
		},
		{
			name:         "description only, empty string is still an update",
			data:         []byte(`{"description":""}`),
			expectUpdate: true,
			expectFields: map[rag.Field]any{rag.FieldDescription: ""},
			expectStatus: http.StatusOK,
			expectBody:   covRagJSON,
		},
		{
			name:         "name and description",
			data:         []byte(`{"name":"kb","description":"desc"}`),
			expectUpdate: true,
			expectFields: map[rag.Field]any{rag.FieldName: "kb", rag.FieldDescription: "desc"},
			expectStatus: http.StatusOK,
			expectBody:   covRagJSON,
		},
		{
			name:         "handler error",
			data:         []byte(`{"name":"kb"}`),
			expectUpdate: true,
			expectFields: map[rag.Field]any{rag.FieldName: "kb"},
			updateErr:    errors.New("boom"),
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			ctx := context.Background()

			if tt.expectUpdate {
				var res *rag.Rag
				if tt.updateErr == nil {
					res = covRag()
				}
				mockRag.EXPECT().RagUpdate(ctx, covRagID, tt.expectFields).Return(res, tt.updateErr)
			}

			res, err := h.processV1RagsIDPut(ctx, &sock.Request{URI: "/v1/rags/" + covRagID.String(), Method: sock.RequestMethodPut, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
			if string(res.Data) != tt.expectBody {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBody, string(res.Data))
			}
		})
	}
}

func Test_processV1RagsIDDelete(t *testing.T) {
	tests := []struct {
		name         string
		deleteErr    error
		expectStatus int
	}{
		{"success", nil, http.StatusOK},
		{"handler error", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			ctx := context.Background()

			mockRag.EXPECT().RagDelete(ctx, covRagID).Return(tt.deleteErr)

			res, err := h.processV1RagsIDDelete(ctx, &sock.Request{URI: "/v1/rags/" + covRagID.String(), Method: sock.RequestMethodDelete})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: empty body, got: %v", string(res.Data))
			}
		})
	}
}

func Test_processV1RagsIDSourcesPost(t *testing.T) {
	tests := []struct {
		name string

		data []byte

		expectAdd    bool
		expectFiles  []uuid.UUID
		expectURLs   []string
		addErr       error
		expectStatus int
		expectBody   string
	}{
		{
			name:         "invalid json",
			data:         []byte(`{"source_urls":`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "no sources",
			data:         []byte(`{"storage_file_ids":[],"source_urls":[]}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "success",
			data:         []byte(`{"storage_file_ids":["33333333-3333-4333-8333-333333333333"],"source_urls":["https://example.invalid/a.txt"]}`),
			expectAdd:    true,
			expectFiles:  []uuid.UUID{covFileID},
			expectURLs:   []string{"https://example.invalid/a.txt"},
			expectStatus: http.StatusOK,
			expectBody:   covRagJSON,
		},
		{
			name:         "handler error",
			data:         []byte(`{"source_urls":["https://example.invalid/a.txt"]}`),
			expectAdd:    true,
			expectURLs:   []string{"https://example.invalid/a.txt"},
			addErr:       errors.New("boom"),
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			ctx := context.Background()

			if tt.expectAdd {
				var res *rag.Rag
				if tt.addErr == nil {
					res = covRag()
				}
				mockRag.EXPECT().RagAddSources(ctx, covRagID, tt.expectFiles, tt.expectURLs).Return(res, tt.addErr)
			}

			res, err := h.processV1RagsIDSourcesPost(ctx, &sock.Request{URI: "/v1/rags/" + covRagID.String() + "/sources", Method: sock.RequestMethodPost, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
			if string(res.Data) != tt.expectBody {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBody, string(res.Data))
			}
		})
	}
}

func Test_processV1RagsIDSourcesIDDelete(t *testing.T) {
	tests := []struct {
		name         string
		removeErr    error
		expectStatus int
		expectBody   string
	}{
		{"success", nil, http.StatusOK, covRagJSON},
		{"handler error", errors.New("boom"), http.StatusInternalServerError, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			ctx := context.Background()

			var ret *rag.Rag
			if tt.removeErr == nil {
				ret = covRag()
			}
			mockRag.EXPECT().RagRemoveSource(ctx, covRagID, covSourceID).Return(ret, tt.removeErr)

			res, err := h.processV1RagsIDSourcesIDDelete(ctx, &sock.Request{
				URI:    "/v1/rags/" + covRagID.String() + "/sources/" + covSourceID.String(),
				Method: sock.RequestMethodDelete,
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
			if string(res.Data) != tt.expectBody {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBody, string(res.Data))
			}
		})
	}
}

func Test_processV1QueryPost(t *testing.T) {
	tests := []struct {
		name string

		data []byte

		expectQuery  bool
		expectTopK   int
		queryErr     error
		expectStatus int
		expectBody   string
	}{
		{
			name:         "invalid json",
			data:         []byte(`{"rag_id":`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "missing rag_id",
			data:         []byte(`{"query":"hello"}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "missing query",
			data:         []byte(`{"rag_id":"11111111-1111-4111-8111-111111111111"}`),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "success passes top_k through",
			data:         []byte(`{"rag_id":"11111111-1111-4111-8111-111111111111","query":"hello","top_k":3}`),
			expectQuery:  true,
			expectTopK:   3,
			expectStatus: http.StatusOK,
			expectBody:   `{"sources":[{"document_id":"33333333-3333-4333-8333-333333333333","document_name":"doc","section_title":"sec","relevance_score":0.5,"text":"body"}]}`,
		},
		{
			name:         "success without top_k",
			data:         []byte(`{"rag_id":"11111111-1111-4111-8111-111111111111","query":"hello"}`),
			expectQuery:  true,
			expectTopK:   0,
			expectStatus: http.StatusOK,
			expectBody:   `{"sources":[{"document_id":"33333333-3333-4333-8333-333333333333","document_name":"doc","section_title":"sec","relevance_score":0.5,"text":"body"}]}`,
		},
		{
			name:         "handler error",
			data:         []byte(`{"rag_id":"11111111-1111-4111-8111-111111111111","query":"hello"}`),
			expectQuery:  true,
			queryErr:     errors.New("boom"),
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			ctx := context.Background()

			if tt.expectQuery {
				var res *query.Response
				if tt.queryErr == nil {
					res = &query.Response{Sources: []query.Source{{
						DocumentID:     covFileID,
						DocumentName:   "doc",
						SectionTitle:   "sec",
						RelevanceScore: 0.5,
						Text:           "body",
					}}}
				}
				mockRag.EXPECT().QueryRag(ctx, covRagID, "hello", tt.expectTopK).Return(res, tt.queryErr)
			}

			res, err := h.processV1QueryPost(ctx, &sock.Request{URI: "/v1/query", Method: sock.RequestMethodPost, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.StatusCode)
			}
			if string(res.Data) != tt.expectBody {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBody, string(res.Data))
			}
		})
	}
}

// Test_processRequest_dispatch checks that every registered route reaches its own rag handler call.
func Test_processRequest_dispatch(t *testing.T) {
	ragURI := "/v1/rags/" + covRagID.String()

	tests := []struct {
		name string

		method sock.RequestMethod
		uri    string
		data   []byte

		// setup registers the single rag handler call that the route must reach.
		setup func(ctx context.Context, m *raghandler.MockRagHandler)
	}{
		{
			name:   "sources id delete",
			method: sock.RequestMethodDelete,
			uri:    ragURI + "/sources/" + covSourceID.String(),
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().RagRemoveSource(ctx, covRagID, covSourceID).Return(covRag(), nil)
			},
		},
		{
			name:   "sources post",
			method: sock.RequestMethodPost,
			uri:    ragURI + "/sources",
			data:   []byte(`{"source_urls":["https://example.invalid/a.txt"]}`),
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().RagAddSources(ctx, covRagID, []uuid.UUID(nil), []string{"https://example.invalid/a.txt"}).Return(covRag(), nil)
			},
		},
		{
			name:   "id get",
			method: sock.RequestMethodGet,
			uri:    ragURI,
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().RagGet(ctx, covRagID).Return(covRag(), nil)
			},
		},
		{
			name:   "id delete",
			method: sock.RequestMethodDelete,
			uri:    ragURI,
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().RagDelete(ctx, covRagID).Return(nil)
			},
		},
		{
			name:   "id put",
			method: sock.RequestMethodPut,
			uri:    ragURI,
			data:   []byte(`{"name":"kb"}`),
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().RagUpdate(ctx, covRagID, map[rag.Field]any{rag.FieldName: "kb"}).Return(covRag(), nil)
			},
		},
		{
			name:   "rags post",
			method: sock.RequestMethodPost,
			uri:    "/v1/rags",
			data:   []byte(`{"customer_id":"22222222-2222-4222-8222-222222222222","name":"kb","source_urls":["https://example.invalid/a.txt"]}`),
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().RagCreate(ctx, covCustomerID, "kb", "", []uuid.UUID(nil), []string{"https://example.invalid/a.txt"}).Return(covRag(), nil)
			},
		},
		{
			name:   "rags get with query string",
			method: sock.RequestMethodGet,
			uri:    "/v1/rags?page_size=5",
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().RagList(ctx, uint64(5), "", map[rag.Field]any{}).Return([]*rag.Rag{covRag()}, nil)
			},
		},
		{
			name:   "query post",
			method: sock.RequestMethodPost,
			uri:    "/v1/query",
			data:   []byte(`{"rag_id":"11111111-1111-4111-8111-111111111111","query":"hello"}`),
			setup: func(ctx context.Context, m *raghandler.MockRagHandler) {
				m.EXPECT().QueryRag(ctx, covRagID, "hello", 0).Return(&query.Response{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRag := raghandler.NewMockRagHandler(mc)
			h := newTestHandler(t, mockRag)
			tt.setup(context.Background(), mockRag)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Errorf("Wrong match. expect: %v, got: %v", http.StatusOK, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_unroutable(t *testing.T) {
	ragURI := "/v1/rags/" + covRagID.String()

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"query with get", sock.RequestMethodGet, "/v1/query"},
		{"query with query string", sock.RequestMethodPost, "/v1/query?x=1"},
		{"rags with put", sock.RequestMethodPut, "/v1/rags"},
		{"rags with delete", sock.RequestMethodDelete, "/v1/rags"},
		{"rag id with post", sock.RequestMethodPost, ragURI},
		{"sources with get", sock.RequestMethodGet, ragURI + "/sources"},
		{"source id with get", sock.RequestMethodGet, ragURI + "/sources/" + covSourceID.String()},
		{"source id with put", sock.RequestMethodPut, ragURI + "/sources/" + covSourceID.String()},
		{"rag id with a non uuid", sock.RequestMethodGet, "/v1/rags/not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation is registered, so any rag handler call fails the test.
			h := newTestHandler(t, raghandler.NewMockRagHandler(mc))

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: %v, got: %v", http.StatusNotFound, res.StatusCode)
			}
		})
	}
}

func Test_jsonResponse_unmarshalable(t *testing.T) {
	res := jsonResponse(http.StatusOK, make(chan int))
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %v, got: %v", http.StatusInternalServerError, res.StatusCode)
	}
	if len(res.Data) != 0 {
		t.Errorf("Wrong match. expect: empty body, got: %v", string(res.Data))
	}
}
