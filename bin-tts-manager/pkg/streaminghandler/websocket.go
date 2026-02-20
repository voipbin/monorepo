package streaminghandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	websocketWriteDelay  = 20 * time.Millisecond // 20ms pacing between frames
	websocketSubprotocol = "media"               // chan_websocket subprotocol
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

// websocketWrite fragments and sends raw audio data over a WebSocket connection
// as binary frames with 20ms pacing. frameSize is the number of bytes per 20ms
// frame for the channel's audio format (e.g., 160 for ulaw, 320 for slin, 640 for slin16).
func websocketWrite(ctx context.Context, conn *websocket.Conn, data []byte, frameSize int) error {
	if len(data) == 0 {
		return nil
	}
	if frameSize <= 0 {
		return fmt.Errorf("frameSize must be positive, got %d", frameSize)
	}

	ticker := time.NewTicker(websocketWriteDelay)
	defer ticker.Stop()

	offset := 0
	payloadLen := len(data)

	for offset < payloadLen {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fragmentLen := min(frameSize, payloadLen-offset)
		fragment := data[offset : offset+fragmentLen]

		if err := conn.WriteMessage(websocket.BinaryMessage, fragment); err != nil {
			return errors.Wrapf(err, "failed to write WebSocket binary frame")
		}

		offset += fragmentLen

		// Don't pace after the last fragment
		if offset >= payloadLen {
			break
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// runWebSocketRead reads from the WebSocket connection to handle ping/pong and
// close frames. Without a read loop, gorilla/websocket won't acknowledge pings.
// Closes doneCh when the connection is closed or encounters an error, signalling
// vendor handlers to tear down their sessions.
func runWebSocketRead(conn *websocket.Conn, doneCh chan struct{}) {
	log := logrus.WithField("func", "runWebSocketRead")

	defer close(doneCh)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Debugf("WebSocket closed normally: %v", err)
			} else {
				log.Errorf("WebSocket read error: %v", err)
			}
			return
		}
	}
}
