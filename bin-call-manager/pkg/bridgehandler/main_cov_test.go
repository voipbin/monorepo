package bridgehandler

import (
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/dbhandler"
)

var errCov = fmt.Errorf("cov error")

type covMocks struct {
	db  *dbhandler.MockDBHandler
	req *requesthandler.MockRequestHandler
}

func newCovHandler(t *testing.T) (*bridgeHandler, *covMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &covMocks{
		db:  dbhandler.NewMockDBHandler(mc),
		req: requesthandler.NewMockRequestHandler(mc),
	}
	return &bridgeHandler{reqHandler: m.req, db: m.db}, m
}

// checkNoGetWithTimeoutGoroutine waits up to 2 seconds for every
// getWithTimeout retry goroutine to exit, and reports the first one left.
func checkNoGetWithTimeoutGoroutine(t *testing.T) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		count, first := countGetWithTimeoutGoroutines()
		if count == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("Wrong match. expect: no getWithTimeout goroutine, got: %d left\n%s", count, first)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
