package listenhandler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

type directHandlerFunc func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)

// Test_handlers_malformedURI checks that a handler which parses the request URI itself
// returns the parse error and calls no business handler. The mocks have no
// expectations, so any call into them fails the test.
func Test_handlers_malformedURI(t *testing.T) {
	tests := []struct {
		name string
		fn   directHandlerFunc
		uri  string
	}{
		{"campaigns list", (*listenHandler).v1CampaignsGet, "/v1/campaigns%zz?page_size=10"},
		{"campaigns get", (*listenHandler).v1CampaignsIDGet, "/v1/campaigns/" + testCampaignURIID + "%zz"},
		{"campaigns delete", (*listenHandler).v1CampaignsIDDelete, "/v1/campaigns/" + testCampaignURIID + "%zz"},
		{"outplans list", (*listenHandler).v1OutplansGet, "/v1/outplans%zz?page_size=10"},
		{"outplans get", (*listenHandler).v1OutplansIDGet, "/v1/outplans/" + testOutplanURIID + "%zz"},
		{"outplans delete", (*listenHandler).v1OutplansIDDelete, "/v1/outplans/" + testOutplanURIID + "%zz"},
		{"outplans put", (*listenHandler).v1OutplansIDPut, "/v1/outplans/" + testOutplanURIID + "%zz"},
		{"outplans dials put", (*listenHandler).v1OutplansIDDialsPut, "/v1/outplans/" + testOutplanURIID + "/dials%zz"},
		{"campaigncalls list", (*listenHandler).v1CampaigncallsGet, "/v1/campaigncalls%zz?page_size=10"},
		{"campaigncalls get", (*listenHandler).v1CampaigncallsIDGet, "/v1/campaigncalls/" + testCampaigncallURIID + "%zz"},
		{"campaigncalls delete", (*listenHandler).v1CampaigncallsIDDelete, "/v1/campaigncalls/" + testCampaigncallURIID + "%zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newErrorPathListenHandler(t)

			// the URI must really be unparsable, otherwise the case would not reach the branch
			_, expectErr := url.Parse(tt.uri)
			if expectErr == nil {
				t.Fatalf("The test uri must be unparsable. uri: %s", tt.uri)
			}

			res, err := tt.fn(h, context.Background(), &sock.Request{
				URI:    tt.uri,
				Method: sock.RequestMethodGet,
			})
			if err == nil {
				t.Fatalf("Wrong match. expect: parse error, got: nil, res: %v", res)
			}
			var urlErr *url.Error
			if !errors.As(err, &urlErr) {
				t.Errorf("Wrong match. expect: *url.Error, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil response, got: %v", res)
			}
		})
	}
}

// Test_handlers_shortURI checks that a handler which takes the id from the URI path
// answers 400 when the path has no id segment, without calling any business handler.
func Test_handlers_shortURI(t *testing.T) {
	tests := []struct {
		name string
		fn   directHandlerFunc
		uri  string
		data []byte
	}{
		{"campaigns put", (*listenHandler).v1CampaignsIDPut, "/v1/campaigns", []byte(`{}`)},
		{"campaigns execute", (*listenHandler).v1CampaignsIDExecutePost, "/v1/campaigns", nil},
		{"campaigns status", (*listenHandler).v1CampaignsIDStatusPut, "/v1/campaigns", []byte(`{}`)},
		{"campaigns service level", (*listenHandler).v1CampaignsIDServiceLevelPut, "/v1/campaigns", []byte(`{}`)},
		{"campaigns actions", (*listenHandler).v1CampaignsIDActionsPut, "/v1/campaigns", []byte(`{}`)},
		{"campaigns resource info", (*listenHandler).v1CampaignsIDResourceInfoPut, "/v1/campaigns", []byte(`{}`)},
		{"campaigns next campaign id", (*listenHandler).v1CampaignsIDNextCampaignIDPut, "/v1/campaigns", []byte(`{}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newErrorPathListenHandler(t)

			res, err := tt.fn(h, context.Background(), &sock.Request{
				URI:    tt.uri,
				Method: sock.RequestMethodPut,
				Data:   tt.data,
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

// Test_v1CampaignsFlowsReconcilePost_invalidBody checks that a malformed body is
// returned as an error and the reconcile is not started.
func Test_v1CampaignsFlowsReconcilePost_invalidBody(t *testing.T) {
	h, _ := newErrorPathListenHandler(t)

	res, err := h.v1CampaignsFlowsReconcilePost(context.Background(), &sock.Request{
		URI:    "/v1/campaigns/flows/reconcile",
		Method: sock.RequestMethodPost,
		Data:   []byte(`{"broken`),
	})
	if err == nil {
		t.Fatalf("Wrong match. expect: unmarshal error, got: nil, res: %v", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}
