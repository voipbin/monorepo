package websockhandler

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// receiveBinaryFromWebsock receives the binary byte from the websock
func (h *websockHandler) receiveBinaryFromWebsock(ctx context.Context, ws *websocket.Conn) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "receiveBinaryFromWebsock",
	})

	if ctx.Err() != nil {
		log.Infof("The context is canceled. Exiting the receiving loop. err: %v", ctx.Err())
		return nil, ctx.Err()
	}

	// read the message from the websocket
	t, m, err := ws.ReadMessage()
	if err != nil {
		log.Infof("Could not read the message correctly. err: %v", err)
		return nil, err
	}

	if t != websocket.BinaryMessage {
		// wrong message type
		return nil, fmt.Errorf("wrong message type")
	}

	return m, nil
}

// receiveTextFromWebsock receives the binary byte from the websock
func (h *websockHandler) receiveTextFromWebsock(ctx context.Context, ws *websocket.Conn) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "receiveTextFromWebsock",
	})

	if ctx.Err() != nil {
		log.Infof("The context is canceled. Exiting the receiving loop. err: %v", ctx.Err())
		return nil, ctx.Err()
	}

	// read the message from the websocket
	t, m, err := ws.ReadMessage()
	if err != nil {
		log.Infof("Could not read the message correctly. err: %v", err)
		return nil, err
	}

	if t != websocket.TextMessage {
		// wrong message type
		return nil, fmt.Errorf("wrong message type")
	}

	return m, nil
}
