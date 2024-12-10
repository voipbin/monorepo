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

	for {
		if ctx.Err() != nil {
			log.Debugf("The context is over. Exiting the process.")
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

		if st.ConnAusiosocket == nil {
			// audiosocket is not ready.
			continue
		}

		d, err := h.ConvertFromWebsocket(st, m)
		if err != nil {
			// could not convert the stream data from the websocket
			continue
		}

		_, err = st.ConnAusiosocket.Write(d)
		if err != nil {
			log.Errorf("Could not send the data. err: %v", err)
			return
		}
	}
}

// handleStreamFromAudiosocket handles the stream data
// asterisk -> api-manager -> websocket client
func (h *streamHandler) handleStreamFromAudiosocket(st *stream.Stream) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "handleStreamFromAudiosocket",
		"stream": st,
	})

	for {
		m, err := audiosocket.NextMessage(st.ConnAusiosocket)
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

		d, err := h.ConvertFromAudiosocket(st, m)
		if err != nil {
			// could not convert the stream data from the websocket
			continue
		}

		if errWrite := st.ConnWebsocket.WriteMessage(websocket.BinaryMessage, d); errWrite != nil {
			log.Debugf("Could not write the message. err: %v", errWrite)
			return
		}

	}
}
