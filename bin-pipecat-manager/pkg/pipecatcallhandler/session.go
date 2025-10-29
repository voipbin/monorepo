package pipecatcallhandler

import (
	"context"
	"fmt"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"net"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func (h *pipecatcallHandler) SessionCreate(
	pc *pipecatcall.Pipecatcall,
	asteriskStreamingID uuid.UUID,
	asteriskConn net.Conn,
) (*pipecatcall.Session, error) {
	h.muPipecatcallSession.Lock()
	defer h.muPipecatcallSession.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	res := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         pc.ID,
			CustomerID: pc.CustomerID,
		},

		PipecatcallReferenceType: pc.ReferenceType,
		PipecatcallReferenceID:   pc.ReferenceID,

		Ctx:    ctx,
		Cancel: cancel,

		RunnerWebsocketChan: make(chan *pipecatframe.Frame, defaultRunnerWebsocketChanBufferSize),

		AsteriskStreamingID: asteriskStreamingID,
		AsteriskConn:        asteriskConn,
	}

	_, ok := h.mapPipecatcallSession[res.ID]
	if ok {
		return nil, fmt.Errorf("session already exists. session_id: %s", res.ID)
	}

	h.mapPipecatcallSession[res.ID] = res
	return res, nil
}

func (h *pipecatcallHandler) SessionGet(id uuid.UUID) (*pipecatcall.Session, error) {
	h.muPipecatcallSession.Lock()
	defer h.muPipecatcallSession.Unlock()

	res, ok := h.mapPipecatcallSession[id]
	if !ok {
		return nil, fmt.Errorf("could not find session. session_id: %s", id)
	}

	return res, nil
}

func (h *pipecatcallHandler) SessionsetRunnerInfo(
	ps *pipecatcall.Session,
	listener net.Listener,
	server *http.Server,
) {
	ps.RunnerListener = listener
	ps.RunnerPort = listener.Addr().(*net.TCPAddr).Port
	ps.RunnerServer = server
}

func (h *pipecatcallHandler) SessionsetRunnerWebsocket(pc *pipecatcall.Session, ws *websocket.Conn) {
	pc.RunnerWebsocket = ws
}

func (h *pipecatcallHandler) SessionsetAsteriskInfo(pc *pipecatcall.Session, streamingID uuid.UUID, conn net.Conn) {
	pc.AsteriskConn = conn
	pc.AsteriskStreamingID = streamingID
}

func (h *pipecatcallHandler) sessionDelete(id uuid.UUID) {
	h.muPipecatcallSession.Lock()
	defer h.muPipecatcallSession.Unlock()

	delete(h.mapPipecatcallSession, id)
}

func (h *pipecatcallHandler) SessionStop(id uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "SessionStop",
		"pipecatcall_id": id,
	})

	pc, err := h.SessionGet(id)
	if err != nil {
		log.Errorf("Could not get pipecatcall session: %v", err)
		return
	}

	if pc.RunnerWebsocket != nil {
		if errClose := pc.RunnerWebsocket.Close(); errClose != nil {
			log.Errorf("Could not close the pipecat runner websocket. err: %v", errClose)
		} else {
			log.Infof("Closed the pipecat runner websocket.")
		}
	}

	if pc.RunnerServer != nil {
		if errClose := pc.RunnerServer.Close(); errClose != nil {
			log.Errorf("Could not close the pipecat runner server. err: %v", errClose)
		} else {
			log.Infof("Closed the pipecat runner server.")
		}
	}

	if pc.RunnerListener != nil {
		if errClose := pc.RunnerListener.Close(); errClose != nil {
			log.Errorf("Could not close the pipecat runner listener. err: %v", errClose)
		} else {
			log.Infof("Closed the pipecat runner listener.")
		}
	}

	if pc.AsteriskConn != nil {
		if errClose := pc.AsteriskConn.Close(); errClose != nil {
			log.Errorf("Could not close the asterisk connection. err: %v", errClose)
		} else {
			log.Infof("Closed the asterisk connection.")
		}
	}

	h.sessionDelete(pc.ID)
}
