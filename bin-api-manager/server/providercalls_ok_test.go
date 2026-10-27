package server

import (
	"net/http"
	"testing"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	rmprovidercall "monorepo/bin-route-manager/models/providercall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ProvidercallsHandlers_Success(t *testing.T) {
	providerID := uuid.FromStringOrNil("11111111-0000-4000-8000-000000000001")
	callID := uuid.FromStringOrNil("11111111-0000-4000-8000-000000000002")
	tmCreate := time.Date(2024, 1, 2, 3, 4, 5, 678000000, time.UTC)

	item := &rmprovidercall.WebhookMessage{ID: callID, ProviderID: providerID, Anonymous: "auto", TMCreate: &tmCreate}
	itemJSON := `{"id":"11111111-0000-4000-8000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","provider_id":"11111111-0000-4000-8000-000000000001","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"anonymous":"auto","call_ids":null,"groupcall_ids":null,"tm_create":"2024-01-02T03:04:05.678Z","tm_update":null,"tm_delete":null}`

	tests := []okCase{
		{
			name: "list with defaults", method: http.MethodGet, path: "/providercalls", wantStatus: 200,
			wantJSON: `{"result":[` + itemJSON + `],"next_page_token":"2024-01-02T03:04:05.678000Z"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderCallGets(gomock.Any(), a, uint64(100), "", uuid.Nil).Return([]*rmprovidercall.WebhookMessage{item}, nil)
			},
		},
		{
			name: "list with page size and provider filter", method: http.MethodGet,
			path: "/providercalls?page_size=7&page_token=tok&provider_id=11111111-0000-4000-8000-000000000001", wantStatus: 200,
			wantJSON: `{"result":[],"next_page_token":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderCallGets(gomock.Any(), a, uint64(7), "tok", providerID).Return(nil, nil)
			},
		},
		{
			name: "list clamps oversized page size", method: http.MethodGet, path: "/providercalls?page_size=1000", wantStatus: 200,
			wantJSON: `{"result":[],"next_page_token":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderCallGets(gomock.Any(), a, uint64(100), "", uuid.Nil).Return(nil, nil)
			},
		},
		{
			name: "create forwards parsed fields", method: http.MethodPost, path: "/providercalls", wantStatus: 200, wantJSON: itemJSON,
			body: `{"provider_id":"11111111-0000-4000-8000-000000000001","source":{"type":"tel","target":"+15550100001"},"destinations":[{"type":"tel","target":"+15550100002"}],"anonymous":"yes"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				src := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}
				dst := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100002"}}
				m.EXPECT().ProviderCallCreate(gomock.Any(), a, providerID, uuid.Nil, gomock.Len(0), &src, dst, "yes").Return(item, nil)
			},
		},
		{
			name: "create defaults anonymous to auto and keeps flow id", method: http.MethodPost, path: "/providercalls", wantStatus: 200, wantJSON: itemJSON,
			body: `{"provider_id":"11111111-0000-4000-8000-000000000001","flow_id":"11111111-0000-4000-8000-000000000003","destinations":[{"type":"tel","target":"+15550100002"}]}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				dst := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100002"}}
				m.EXPECT().ProviderCallCreate(gomock.Any(), a, providerID, uuid.FromStringOrNil("11111111-0000-4000-8000-000000000003"), gomock.Len(0), (*commonaddress.Address)(nil), dst, "auto").Return(item, nil)
			},
		},
		{
			name: "get", method: http.MethodGet, path: "/providercalls/11111111-0000-4000-8000-000000000002", wantStatus: 200, wantJSON: itemJSON,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderCallGet(gomock.Any(), a, callID).Return(item, nil)
			},
		},
		{
			name: "delete", method: http.MethodDelete, path: "/providercalls/11111111-0000-4000-8000-000000000002", wantStatus: 200, wantJSON: itemJSON,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderCallDelete(gomock.Any(), a, callID).Return(item, nil)
			},
		},
	}

	okRunCases(t, tests)
}

func Test_ProvidercallsHandlers_CreateValidation(t *testing.T) {
	tests := []s2Case{
		{name: "nil provider id", method: http.MethodPost, path: "/providercalls", body: `{"provider_id":"00000000-0000-0000-0000-000000000000","destinations":[{"type":"tel","target":"+15550100002"}]}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT"},
		{name: "empty destinations", method: http.MethodPost, path: "/providercalls", body: `{"provider_id":"11111111-0000-4000-8000-000000000001","destinations":[]}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT"},
		{name: "create service error", method: http.MethodPost, path: "/providercalls", body: `{"provider_id":"11111111-0000-4000-8000-000000000001","destinations":[{"type":"tel","target":"+15550100002"}]}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderCallCreate(gomock.Any(), a, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "auto").Return(nil, s2Err)
			}},
		{name: "list service error", method: http.MethodGet, path: "/providercalls", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderCallGets(gomock.Any(), a, uint64(100), "", gomock.Any()).Return(nil, s2Err)
			}},
	}
	s2RunCases(t, tests)
}
