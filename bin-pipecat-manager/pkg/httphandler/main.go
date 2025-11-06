package httphandler

import (
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HttpHandler interface {
	Run() error
}

const (
	defaultListenAddress = "localhost:8001"
)

type httpHandler struct {
	requestHandler     requesthandler.RequestHandler
	pipecatcallHandler pipecatcallhandler.PipecatcallHandler
}

func NewHttpHandler(
	requestHandler requesthandler.RequestHandler,
	pipecatcallHandler pipecatcallhandler.PipecatcallHandler,
) HttpHandler {
	return &httpHandler{
		requestHandler:     requestHandler,
		pipecatcallHandler: pipecatcallHandler,
	}
}

func (h *httpHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/:id/ws", h.wsHandle)
	router.POST("/:id/tools", h.toolHandle)

	server := &http.Server{
		Handler: router,
	}

	listener, err := net.Listen("tcp", defaultListenAddress)
	if err != nil {
		log.Errorf("Failed to listen on ephemeral port: %v", err)
		return errors.Wrapf(err, "failed to listen on ephemeral port")
	}

	go func() {
		log.Debugf("Starting Gin server on %s", listener.Addr().String())
		if errServe := server.Serve(listener); errServe != nil && errServe != http.ErrServerClosed {
			log.Errorf("Could not start Gin server: %v", errServe)
		}
		log.Debugf("Gin server stopped")
	}()

	return nil
}

func (h *httpHandler) wsHandle(c *gin.Context) {
	log := logrus.WithField("func", "wsHandle")

	id := uuid.FromStringOrNil(c.Param("id"))
	log.WithField("id", id).Debug("WebSocket handle called")

	h.pipecatcallHandler.RunnerWebsocketHandle(id, c)
}

func (h *httpHandler) toolHandle(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "toolHandle",
	})

	id := uuid.FromStringOrNil(c.Param("id"))
	log.WithField("id", id).Debug("Tool handle called")

	h.pipecatcallHandler.RunnerToolHandle(id, c)
}
