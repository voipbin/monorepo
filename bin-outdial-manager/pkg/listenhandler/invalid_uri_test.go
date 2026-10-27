package listenhandler

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-outdial-manager/pkg/outdialhandler"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
)

// Test_handlers_invalidURI checks that a request URI that cannot be parsed is returned as an
// error before any handler is called. The handlers are called directly because the router
// only passes URIs that match its patterns.
func Test_handlers_invalidURI(t *testing.T) {
	const badURI = "/v1/outdials/%zz"

	mc := gomock.NewController(t)
	defer mc.Finish()

	// none of the mocks has an expectation: a call to any of them fails the test
	h := &listenHandler{
		sockHandler:          sockhandler.NewMockSockHandler(mc),
		outdialHandler:       outdialhandler.NewMockOutdialHandler(mc),
		outdialTargetHandler: outdialtargethandler.NewMockOutdialTargetHandler(mc),
	}

	tests := []struct {
		name    string
		handler func(context.Context, *sock.Request) (*sock.Response, error)
	}{
		{"v1OutdialsGet", h.v1OutdialsGet},
		{"v1OutdialsIDGet", h.v1OutdialsIDGet},
		{"v1OutdialsIDDelete", h.v1OutdialsIDDelete},
		{"v1OutdialsIDPut", h.v1OutdialsIDPut},
		{"v1OutdialsIDCampaignIDPut", h.v1OutdialsIDCampaignIDPut},
		{"v1OutdialsIDDataPut", h.v1OutdialsIDDataPut},
		{"v1OutdialsIDAvailableGet", h.v1OutdialsIDAvailableGet},
		{"v1OutdialsIDTargetsPost", h.v1OutdialsIDTargetsPost},
		{"v1OutdialsIDTargetsGet", h.v1OutdialsIDTargetsGet},
		{"v1OutdialtargetsIDGet", h.v1OutdialtargetsIDGet},
		{"v1OutdialtargetsIDDelete", h.v1OutdialtargetsIDDelete},
		{"v1OutdialtargetsIDProgressingPost", h.v1OutdialtargetsIDProgressingPost},
		{"v1OutdialtargetsIDStatusPut", h.v1OutdialtargetsIDStatusPut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.handler(context.Background(), &sock.Request{URI: badURI})

			var urlErr *url.Error
			if !errors.As(err, &urlErr) {
				t.Fatalf("expected a URL parse error, got: %v", err)
			}
			if res != nil {
				t.Errorf("expected no response with a parse error, got: %v", res)
			}
		})
	}
}
