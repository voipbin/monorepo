package listenhandler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
)

// processConfig answers the config requests (the keys of the service, the write of a secret, the
// removal of a stored value and the change history). The body of a secret request holds the
// secret, so nothing of the request is logged here, and a panic is answered with a fixed text:
// the recovered value is never formatted, because it can carry what the code was working on.
//
// It is also the answer to a request that looks like a config request (see processRequest) but is
// not one of them: a fixed 404, so that its body is not dispatched to another route.
//
// err is always nil on return.
func (h *listenHandler) processConfig(m *sock.Request) (response *sock.Response, err error) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			// r is deliberately not logged or formatted.
			logrus.WithField("func", "processConfig").Error("A config request panicked.")
			response = simpleResponse(500)
			err = nil
		}

		elapsed := time.Since(start)
		promReceivedRequestProcessTime.WithLabelValues("/v1/configs", string(m.Method)).Observe(float64(elapsed.Milliseconds()))
	}()

	if !h.configServer.Match(m) {
		return simpleResponse(404), nil
	}

	return h.configServer.Handle(context.Background(), m)
}
