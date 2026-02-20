package streaminghandler

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	websocketSubprotocol = "media" // chan_websocket subprotocol
)

// websocketConnect dials the Asterisk chan_websocket endpoint and waits for the
// MEDIA_START text message that signals the channel is ready.
func websocketConnect(ctx context.Context, mediaURI string) (*websocket.Conn, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "websocketConnect",
		"media_uri": mediaURI,
	})

	dialer := websocket.Dialer{
		Subprotocols: []string{websocketSubprotocol},
	}

	conn, _, err := dialer.DialContext(ctx, mediaURI, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not dial WebSocket. media_uri: %s", mediaURI)
	}

	// Set a read deadline for the MEDIA_START handshake
	if errDeadline := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); errDeadline != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(errDeadline, "could not set read deadline")
	}

	// Read the MEDIA_START text message from Asterisk
	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(err, "could not read MEDIA_START message")
	}

	// Clear the deadline for subsequent reads
	if errDeadline := conn.SetReadDeadline(time.Time{}); errDeadline != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(errDeadline, "could not clear read deadline")
	}
	if msgType != websocket.TextMessage {
		_ = conn.Close()
		return nil, errors.Errorf("expected text message for MEDIA_START, got type %d", msgType)
	}
	log.Debugf("Received MEDIA_START message: %s", string(msg))

	return conn, nil
}

