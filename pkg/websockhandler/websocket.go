package websockhandler

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmqsubhandler"
)

// Run creates a new websocket and starts socket message listen.
func (h *websockHandler) Run(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Run",
		"agent": a,
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
	go h.runWebsock(newCtx, newCancel, a, ws, sock)
	go h.runZMQSub(newCtx, newCancel, ws, sock)

	<-newCtx.Done()
	sock.Terminate()

	return nil
}
