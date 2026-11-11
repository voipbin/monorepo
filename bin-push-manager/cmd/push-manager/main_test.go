package main

import (
	"sync"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	// the packages below register their metrics in init: importing them here makes the registry hold them
	_ "monorepo/bin-push-manager/pkg/eventhandler"
	_ "monorepo/bin-push-manager/pkg/listenhandler"
	_ "monorepo/bin-push-manager/pkg/notificationhandler"
	_ "monorepo/bin-push-manager/pkg/subscribehandler"
)

// requestHandlerOnce keeps a repeated run of the test (go test -count=N) from building a second request handler:
// the request handler registers its metrics, and a second one panics by design.
var requestHandlerOnce sync.Once

// The service builds one request handler, whose metrics share the namespace of the metrics of the service. A
// duplicate name makes MustRegister panic when the service starts, and nothing else would show it before a
// deploy. The packages of the service are imported above, so their metrics are registered first, in the same
// order as in the service.
func Test_requestHandlerMetricsDoNotCollide(t *testing.T) {
	requestHandlerOnce.Do(func() {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// must not panic
		_ = requesthandler.NewRequestHandler(sockhandler.NewMockSockHandler(mc), commonoutline.ServiceNamePushManager)
	})

	// the registry still gathers: a registration problem would show as an error here
	if _, err := prometheus.DefaultGatherer.Gather(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
