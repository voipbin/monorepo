package pipecatcallhandler

import (
	"context"
	"fmt"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"net"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *pipecatcallHandler) SessionCreate(
	pc *pipecatcall.Pipecatcall,
	asteriskStreamingID uuid.UUID,
	asteriskConn net.Conn,
	llmKey string,
) (*pipecatcall.Session, error) {

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

		RunnerWebsocketChan: make(chan *pipecatcall.SessionFrame, defaultRunnerWebsocketChanBufferSize),

		AsteriskStreamingID: asteriskStreamingID,
		AsteriskConn:        asteriskConn,

		LLMKey: llmKey,
	}

	h.muPipecatcallSession.Lock()
	defer h.muPipecatcallSession.Unlock()

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
	log.Debugf("Stopping pipecatcall session. pipecatcall_id: %s", id)

	pc, err := h.SessionGet(id)
	if err != nil {
		log.Errorf("Could not get pipecatcall session: %v", err)
		return
	}

	if pc.AsteriskConn != nil {
		if errClose := pc.AsteriskConn.Close(); errClose != nil {
			log.Errorf("Could not close the asterisk connection. err: %v", errClose)
		} else {
			log.Infof("Closed the asterisk connection.")
		}
	}

	h.sessionDelete(pc.ID)
	if errStop := h.pythonRunner.Stop(context.Background(), id); errStop != nil {
		log.Errorf("Could not stop the pipecatcall in python runner. err: %v", errStop)
		return
	}

	log.Debugf("Stopped pipecatcall session. pipecatcall_id: %s", id)
}
