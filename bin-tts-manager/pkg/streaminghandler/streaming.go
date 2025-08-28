package streaminghandler

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"
)

func (h *streamingHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender streaming.Gender,
	direction streaming.Direction,
) (*streaming.Streaming, error) {
	h.muStreaming.Lock()
	defer h.muStreaming.Unlock()

	id := h.utilHandler.UUIDCreate()
	res := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		PodID: h.podID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Language:  language,
		Gender:    gender,
		Direction: direction,

		VendorName:   streaming.VendorNameNone,
		VendorLock:   sync.Mutex{},
		VendorConfig: nil,
	}

	_, ok := h.mapStreaming[id]
	if ok {
		return nil, fmt.Errorf("streaming already exists. streaming_id: %s", id)
	}

	h.mapStreaming[id] = res
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingCreated, res)

	return res, nil
}

// Gets returns streaming
func (h *streamingHandler) Get(ctx context.Context, streamingID uuid.UUID) (*streaming.Streaming, error) {
	h.muStreaming.Lock()
	defer h.muStreaming.Unlock()

	res, ok := h.mapStreaming[streamingID]
	if !ok {
		return nil, fmt.Errorf("streaming not found. streaming_id: %s", streamingID)
	}

	return res, nil
}

func (h *streamingHandler) Delete(ctx context.Context, streamingID uuid.UUID) {
	h.muStreaming.Lock()
	defer h.muStreaming.Unlock()

	tmp, ok := h.mapStreaming[streamingID]
	if !ok {
		return
	}

	delete(h.mapStreaming, streamingID)
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingDeleted, tmp)
}

func (h *streamingHandler) UpdateMessageID(ctx context.Context, streamingID uuid.UUID, messageID uuid.UUID) (*streaming.Streaming, error) {
	h.muStreaming.Lock()
	defer h.muStreaming.Unlock()

	res, ok := h.mapStreaming[streamingID]
	if !ok {
		return nil, fmt.Errorf("streaming not found. streaming_id: %s", streamingID)
	}

	res.MessageID = messageID
	return res, nil
}

func (h *streamingHandler) UpdateConnAst(streamingID uuid.UUID, connAst net.Conn) (*streaming.Streaming, error) {
	h.muStreaming.Lock()
	defer h.muStreaming.Unlock()

	res, ok := h.mapStreaming[streamingID]
	if !ok {
		return nil, fmt.Errorf("streaming not found. streaming_id: %s", streamingID)
	}

	res.ConnAst = connAst
	return res, nil
}

func (h *streamingHandler) SetVendorInfo(st *streaming.Streaming, venderName streaming.VendorName, vendorConfig any) {
	st.VendorLock.Lock()
	defer st.VendorLock.Unlock()

	st.VendorName = venderName
	st.VendorConfig = vendorConfig
}
