package listenhandler

import (
	"context"
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-rtpengine-proxy/pkg/ngclient"
	"monorepo/voip-rtpengine-proxy/pkg/processmanager"
)

// ListenHandler processes incoming RabbitMQ requests.
type ListenHandler interface {
	Run() error
}

type listenHandler struct {
	sockHandler                       sockhandler.SockHandler
	rabbitQueueListenRequestPermanent string
	rabbitQueueListenRequestVolatile  string
	ngClient                          ngclient.NGClient
	procMgr                           processmanager.ProcessManager
}

var regCommandPost = regexp.MustCompile(`^/v1/commands$`)

func simpleResponse(code int) *sock.Response {
	return &sock.Response{StatusCode: code}
}

// NewListenHandler creates a ListenHandler.
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	permanentQueue string,
	volatileQueue string,
	ng ngclient.NGClient,
	procMgr processmanager.ProcessManager,
) ListenHandler {
	return &listenHandler{
		sockHandler:                       sockHandler,
		rabbitQueueListenRequestPermanent: permanentQueue,
		rabbitQueueListenRequestVolatile:  volatileQueue,
		ngClient:                          ng,
		procMgr:                           procMgr,
	}
}

// Run starts consuming both queues.
func (h *listenHandler) Run() error {
	go func() {
		if err := h.listenRun(); err != nil {
			logrus.WithError(err).Error("Listen handler error")
		}
	}()
	return nil
}

func (h *listenHandler) listenRun() error {
	queues := []struct {
		name   string
		qType  string
	}{
		{h.rabbitQueueListenRequestPermanent, "normal"},
		{h.rabbitQueueListenRequestVolatile, "volatile"},
	}

	for _, q := range queues {
		if err := h.sockHandler.QueueCreate(q.name, q.qType); err != nil {
			return fmt.Errorf("create queue %q: %w", q.name, err)
		}
		logrus.WithField("queue", q.name).Info("Listening on queue")
		go func(queue string) {
			if err := h.sockHandler.ConsumeRPC(context.Background(), queue, "rtpengine-proxy", false, false, false, 10, h.processRequest); err != nil {
				logrus.WithError(err).Errorf("ConsumeRPC failed for queue %s", queue)
			}
		}(q.name)
	}

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "processRequest",
		"uri":    m.URI,
		"method": m.Method,
	})

	switch {
	case regCommandPost.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		return h.processCommandPost(m)
	default:
		log.Warnf("No handler for request")
		return simpleResponse(404), nil
	}
}
