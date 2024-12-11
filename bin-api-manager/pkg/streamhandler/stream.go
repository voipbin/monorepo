package streamhandler

import (
	"context"
	"fmt"
	"monorepo/bin-api-manager/models/stream"
	"net"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
)

func (h *streamHandler) Get(id uuid.UUID) (*stream.Stream, error) {
	h.streamLock.Lock()
	defer h.streamLock.Unlock()

	res, ok := h.streamData[id.String()]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

func (h *streamHandler) SetAudiosock(id uuid.UUID, sock net.Conn) (*stream.Stream, error) {
	h.streamLock.Lock()
	defer h.streamLock.Unlock()

	res, ok := h.streamData[id.String()]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	res.ConnAsterisk = sock

	return res, nil
}

func (h *streamHandler) SetExternalMedia(id uuid.UUID, em *cmexternalmedia.ExternalMedia) (*stream.Stream, error) {
	h.streamLock.Lock()
	defer h.streamLock.Unlock()

	res, ok := h.streamData[id.String()]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	res.ExternalMedia = em

	return res, nil
}

func (h *streamHandler) Create(ctx context.Context, id uuid.UUID, connWebsock *websocket.Conn, encapsulation stream.Encapsulation) (*stream.Stream, error) {
	h.streamLock.Lock()
	defer h.streamLock.Unlock()

	res := &stream.Stream{
		ID:            id,
		ConnWebsocket: connWebsock,
		Encapsulation: encapsulation,
	}

	h.streamData[id.String()] = res

	return res, nil
}

func (h *streamHandler) Terminate(id uuid.UUID) {
	h.streamLock.Lock()
	defer h.streamLock.Unlock()

	st, ok := h.streamData[id.String()]
	if !ok {
		return
	}

	if st.ConnAsterisk != nil {
		st.ConnAsterisk.Close()
		st.ConnAsterisk = nil
	}
	if st.ConnWebsocket != nil {
		st.ConnWebsocket.Close()
		st.ConnWebsocket = nil
	}

	delete(h.streamData, id.String())
}
