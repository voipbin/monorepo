package streamhandler

import (
	"context"
	"monorepo/bin-api-manager/models/stream"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// handleStreamFromWebsocket handles the stream data
// websocket client -> api-manager -> asterisk
func (h *streamHandler) handleStreamFromWebsocket(ctx context.Context, st *stream.Stream) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "handleStreamFromWebsocket",
		"stream": st,
	})
	defer h.Terminate(st.ID)

	for {
		if ctx.Err() != nil {
			log.Debugf("The context is over. Exiting the process.")
			return
		}

		if st.ConnWebsocket == nil {
			// connection is closed
			return
		}

		// read the message from the websocket
		t, m, err := st.ConnWebsocket.ReadMessage()
		if err != nil {
			log.Infof("Could not read the message correctly. err: %v", err)
			return
		}

		if t != websocket.BinaryMessage {
			// wrong message type
			continue
		}

		if st.ConnAsterisk == nil {
			// asterisk connection is not ready.
			continue
		}

		d, err := h.ConvertFromWebsocket(st, m)
		if err != nil {
			// could not convert the stream data from the websocket
			continue
		}

		_, err = st.ConnAsterisk.Write(d)
		if err != nil {
			log.Errorf("Could not send the data. err: %v", err)
			return
		}
	}
}

// handleStreamFromAsterisk handles the stream data
// asterisk -> api-manager -> websocket client
func (h *streamHandler) handleStreamFromAsterisk(st *stream.Stream) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "handleStreamFromAsterisk",
		"stream": st,
	})
	defer h.Terminate(st.ID)

	sequence := uint16(0)
	timestamp := uint32(0)
	ssrc := uint32(0)

	for {
		m, err := audiosocket.NextMessage(st.ConnAsterisk)
		if err != nil {
			log.Errorf("Could not receive audiosock data. err: %v", err)
			return
		}

		switch {
		case m.Kind() == audiosocket.KindError:
			log.Debugf("Received error. err: %d", m.ErrorCode())
			continue

		case m.Kind() != audiosocket.KindSlin:
			log.Debugf("Ignoring non-slin message")
			continue

		case m.ContentLength() < 1:
			log.Debugf("No content")
			continue
		}

		d, s, t, err := h.ConvertFromAsterisk(st, m, sequence, timestamp, ssrc)
		if err != nil {
			// could not convert the stream data from the websocket
			continue
		}
		sequence = s
		timestamp = t

		if errWrite := st.ConnWebsocket.WriteMessage(websocket.BinaryMessage, d); errWrite != nil {
			log.Debugf("Could not write the message. err: %v", errWrite)
			return
		}
	}
}
