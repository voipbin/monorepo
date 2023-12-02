package websockhandler

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmqsubhandler"
)

// Run creates a new websocket and starts socket message listen.
func (h *websockHandler) Run(ctx context.Context, w http.ResponseWriter, r *http.Request, agentID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Run",
		"agent_id": agentID,
	})

	// create a websock
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not create websocket. err: %v", err)
		return err
	}
	defer ws.Close()
	log.Debugf("Created a new websocket correctly.")

	// create a new subscriber sock
	sock, err := zmqsubhandler.NewZMQSubHandler()
	if err != nil {
		log.Errorf("Could not create a new zmq subscirber handler. err: %v", err)
		return err
	}
	defer sock.Terminate()
	log.Debugf("Created a new subscribe socket correctly.")

	newCtx, newCancel := context.WithCancel(context.Background())
	go h.runWebsock(newCtx, newCancel, agentID, ws, sock)
	go h.runZMQSub(newCtx, newCancel, ws, sock)

	<-newCtx.Done()

	return nil
}
