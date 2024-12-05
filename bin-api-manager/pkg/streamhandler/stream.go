package streamhandler

import (
	"context"
	"fmt"
	"monorepo/bin-api-manager/models/stream"
	"net"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

func (h *streamHandler) GetByMediaID(mediaID uuid.UUID) (*stream.Stream, error) {
	h.streamLock.Lock()
	defer h.streamLock.Unlock()

	res, ok := h.streamData[mediaID.String()]
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

	res.ConnAusiosocket = sock

	return res, nil
}

func (h *streamHandler) Create(ctx context.Context, mediaID uuid.UUID, connWebsock *websocket.Conn) (*stream.Stream, error) {
	res := &stream.Stream{
		ConnWebsocket: connWebsock,
	}

	h.streamLock.Lock()
	defer h.streamLock.Unlock()
	h.streamData[mediaID.String()] = res

	return res, nil
}

func (h *streamHandler) Terminate(mediaID uuid.UUID) {
	h.streamLock.Lock()
	defer h.streamLock.Unlock()

	st, ok := h.streamData[mediaID.String()]
	if !ok {
		return
	}

	st.ConnAusiosocket.Close()
	st.ConnWebsocket.Close()

	delete(h.streamData, mediaID.String())
}

func (h *streamHandler) getMediaID(c net.Conn) (uuid.UUID, error) {
	m, err := audiosocket.NextMessage(c)
	if err != nil {
		return uuid.Nil, err
	}

	if m.Kind() != audiosocket.KindID {
		return uuid.Nil, fmt.Errorf("invalid message type %d getting CallID", m.Kind())
	}

	return uuid.FromBytes(m.Payload())
}
