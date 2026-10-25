package listenhandler

import (
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

const (
	testCampaignURIID     = "7a8b9c10-0001-11ee-8a45-000000000001"
	testOutplanURIID      = "7a8b9c10-0002-11ee-8a45-000000000002"
	testCampaigncallURIID = "7a8b9c10-0003-11ee-8a45-000000000003"
)

type errorPathMocks struct {
	campaign     *campaignhandler.MockCampaignHandler
	outplan      *outplanhandler.MockOutplanHandler
	campaigncall *campaigncallhandler.MockCampaigncallHandler
}

func newErrorPathListenHandler(t *testing.T) (*listenHandler, *errorPathMocks) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &errorPathMocks{
		campaign:     campaignhandler.NewMockCampaignHandler(mc),
		outplan:      outplanhandler.NewMockOutplanHandler(mc),
		campaigncall: campaigncallhandler.NewMockCampaigncallHandler(mc),
	}

	h := &listenHandler{
		sockHandler:         sockhandler.NewMockSockHandler(mc),
		campaignHandler:     m.campaign,
		outplanHandler:      m.outplan,
		campaigncallHandler: m.campaigncall,
	}

	return h, m
}

// Test_processRequest_invalidBody checks that a malformed request body is rejected
// with a 400 response before any business handler is called. The mocks have no
// expectations, so any call into them fails the test.
func Test_processRequest_invalidBody(t *testing.T) {
	const invalid = `{"broken`

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"campaign create", sock.RequestMethodPost, "/v1/campaigns"},
		{"campaign list filters", sock.RequestMethodGet, "/v1/campaigns?page_size=10&page_token=t"},
		{"campaign update basic info", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID},
		{"campaign update status", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/status"},
		{"campaign update service level", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/service_level"},
		{"campaign update actions", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/actions"},
		{"campaign update resource info", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/resource_info"},
		{"campaign update next campaign id", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/next_campaign_id"},
		{"outplan create", sock.RequestMethodPost, "/v1/outplans"},
		{"outplan list filters", sock.RequestMethodGet, "/v1/outplans?page_size=10&page_token=t"},
		{"outplan update basic info", sock.RequestMethodPut, "/v1/outplans/" + testOutplanURIID},
		{"outplan update dial info", sock.RequestMethodPut, "/v1/outplans/" + testOutplanURIID + "/dials"},
		{"campaigncall list filters", sock.RequestMethodGet, "/v1/campaigncalls?page_size=10&page_token=t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newErrorPathListenHandler(t)

			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   tt.method,
				DataType: "application/json",
				Data:     []byte(invalid),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: nil error, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %v", http.StatusBadRequest, res)
			}
		})
	}
}

// Test_processRequest_invalidFilterValue checks that a filter whose value can not be
// converted to the typed field map is rejected with 400 and no list call is made.
func Test_processRequest_invalidFilterValue(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"campaign", "/v1/campaigns?page_size=10"},
		{"outplan", "/v1/outplans?page_size=10"},
		{"campaigncall", "/v1/campaigncalls?page_size=10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newErrorPathListenHandler(t)

			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"not-a-uuid"}`),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: nil error, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %v", http.StatusBadRequest, res)
			}
		})
	}
}

// Test_processRequest_handlerError checks how an error from the business handler is
// mapped to the response: unclassified errors become 400, record-not-found and typed
// not-found errors become 404.
func Test_processRequest_handlerError(t *testing.T) {
	anyArg := gomock.Any()
	campaignID := uuid.FromStringOrNil(testCampaignURIID)
	outplanID := uuid.FromStringOrNil(testOutplanURIID)
	campaigncallID := uuid.FromStringOrNil(testCampaigncallURIID)

	routes := []struct {
		name   string
		method sock.RequestMethod
		uri    string
		data   string
		expect func(m *errorPathMocks, err error)
	}{
		{"campaign create", sock.RequestMethodPost, "/v1/campaigns", `{}`,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().Create(anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg).Return(nil, err)
			}},
		{"campaign list", sock.RequestMethodGet, "/v1/campaigns?page_size=10&page_token=t", ``,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().List(anyArg, "t", uint64(10), anyArg).Return(nil, err)
			}},
		{"campaign get", sock.RequestMethodGet, "/v1/campaigns/" + testCampaignURIID, ``,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().Get(anyArg, campaignID).Return(nil, err)
			}},
		{"campaign delete", sock.RequestMethodDelete, "/v1/campaigns/" + testCampaignURIID, ``,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().Delete(anyArg, campaignID).Return(nil, err)
			}},
		{"campaign update basic info", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID, `{}`,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().UpdateBasicInfo(anyArg, campaignID, anyArg, anyArg, anyArg, anyArg, anyArg).Return(nil, err)
			}},
		{"campaign update status", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/status", `{}`,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().UpdateStatus(anyArg, campaignID, anyArg).Return(nil, err)
			}},
		{"campaign update service level", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/service_level", `{}`,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().UpdateServiceLevel(anyArg, campaignID, anyArg).Return(nil, err)
			}},
		{"campaign update actions", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/actions", `{}`,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().UpdateActions(anyArg, campaignID, anyArg).Return(nil, err)
			}},
		{"campaign update resource info", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/resource_info", `{}`,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().UpdateResourceInfo(anyArg, campaignID, anyArg, anyArg, anyArg, anyArg).Return(nil, err)
			}},
		{"campaign update next campaign id", sock.RequestMethodPut, "/v1/campaigns/" + testCampaignURIID + "/next_campaign_id", `{}`,
			func(m *errorPathMocks, err error) {
				m.campaign.EXPECT().UpdateNextCampaignID(anyArg, campaignID, anyArg).Return(nil, err)
			}},
		{"outplan create", sock.RequestMethodPost, "/v1/outplans", `{}`,
			func(m *errorPathMocks, err error) {
				m.outplan.EXPECT().Create(anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg).Return(nil, err)
			}},
		{"outplan list", sock.RequestMethodGet, "/v1/outplans?page_size=10&page_token=t", ``,
			func(m *errorPathMocks, err error) {
				m.outplan.EXPECT().List(anyArg, "t", uint64(10), anyArg).Return(nil, err)
			}},
		{"outplan get", sock.RequestMethodGet, "/v1/outplans/" + testOutplanURIID, ``,
			func(m *errorPathMocks, err error) {
				m.outplan.EXPECT().Get(anyArg, outplanID).Return(nil, err)
			}},
		{"outplan delete", sock.RequestMethodDelete, "/v1/outplans/" + testOutplanURIID, ``,
			func(m *errorPathMocks, err error) {
				m.outplan.EXPECT().Delete(anyArg, outplanID).Return(nil, err)
			}},
		{"outplan update basic info", sock.RequestMethodPut, "/v1/outplans/" + testOutplanURIID, `{}`,
			func(m *errorPathMocks, err error) {
				m.outplan.EXPECT().UpdateBasicInfo(anyArg, outplanID, anyArg, anyArg).Return(nil, err)
			}},
		{"outplan update dial info", sock.RequestMethodPut, "/v1/outplans/" + testOutplanURIID + "/dials", `{}`,
			func(m *errorPathMocks, err error) {
				m.outplan.EXPECT().UpdateDialInfo(anyArg, outplanID, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, anyArg).Return(nil, err)
			}},
		{"campaigncall list", sock.RequestMethodGet, "/v1/campaigncalls?page_size=10&page_token=t", ``,
			func(m *errorPathMocks, err error) {
				m.campaigncall.EXPECT().List(anyArg, "t", uint64(10), anyArg).Return(nil, err)
			}},
		{"campaigncall get", sock.RequestMethodGet, "/v1/campaigncalls/" + testCampaigncallURIID, ``,
			func(m *errorPathMocks, err error) {
				m.campaigncall.EXPECT().Get(anyArg, campaigncallID).Return(nil, err)
			}},
		{"campaigncall delete", sock.RequestMethodDelete, "/v1/campaigncalls/" + testCampaigncallURIID, ``,
			func(m *errorPathMocks, err error) {
				m.campaigncall.EXPECT().Delete(anyArg, campaigncallID).Return(nil, err)
			}},
	}

	errors := []struct {
		name       string
		err        error
		expectCode int
	}{
		{"unclassified", fmt.Errorf("test error"), http.StatusBadRequest},
		{"record not found", fmt.Errorf("wrapped: %w", dbhandler.ErrNotFound), http.StatusNotFound},
		{"typed not found", cerrors.NotFound(commonoutline.ServiceNameCampaignManager, "TEST_NOT_FOUND", "test"), http.StatusNotFound},
	}

	for _, rt := range routes {
		for _, e := range errors {
			t.Run(rt.name+"/"+e.name, func(t *testing.T) {
				h, m := newErrorPathListenHandler(t)
				rt.expect(m, e.err)

				res, err := h.processRequest(&sock.Request{
					URI:      rt.uri,
					Method:   rt.method,
					DataType: "application/json",
					Data:     []byte(rt.data),
				})
				if err != nil {
					t.Fatalf("Wrong match. expect: nil error, got: %v", err)
				}
				if res == nil || res.StatusCode != e.expectCode {
					t.Errorf("Wrong match. expect: %d, got: %v", e.expectCode, res)
				}
			})
		}
	}
}

func Test_processRequest_unknownRoute(t *testing.T) {
	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"unknown uri", sock.RequestMethodGet, "/v1/unknown"},
		{"unsupported method", sock.RequestMethod("PATCH"), "/v1/campaigns/" + testCampaignURIID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newErrorPathListenHandler(t)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method})
			if err != nil {
				t.Fatalf("Wrong match. expect: nil error, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: %d, got: %v", http.StatusNotFound, res)
			}
		})
	}
}
