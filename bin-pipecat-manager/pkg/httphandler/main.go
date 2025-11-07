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
		log.Errorf("Failed to listen on %s: %v", defaultListenAddress, err)
		return errors.Wrapf(err, "failed to listen on %s", defaultListenAddress)
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
	log := logrus.WithFields(logrus.Fields{
		"func": "wsHandle",
	})

	id := uuid.FromStringOrNil(c.Param("id"))
	if id == uuid.Nil {
		log.Errorf("Invalid pipecatcall ID: %s", c.Param("id"))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if errHandle := h.pipecatcallHandler.RunnerWebsocketHandle(id, c); errHandle != nil {
		log.Errorf("Could not handle websocket connection. pipecatcall_id: %s, err: %v", id, errHandle)
		c.JSON(http.StatusBadRequest, gin.H{"error": errHandle.Error()})
		return
	}
}

func (h *httpHandler) toolHandle(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "toolHandle",
	})

	id := uuid.FromStringOrNil(c.Param("id"))
	if id == uuid.Nil {
		log.Errorf("Invalid pipecatcall ID: %s", c.Param("id"))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if errHandle := h.pipecatcallHandler.RunnerToolHandle(id, c); errHandle != nil {
		log.Errorf("Could not handle tool request. pipecatcall_id: %s, err: %v", id, errHandle)
		c.JSON(http.StatusBadRequest, gin.H{"error": errHandle.Error()})
		return
	}
}
