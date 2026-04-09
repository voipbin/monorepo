package streaminghandler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"
)

func (h *streamingHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender streaming.Gender,
	direction streaming.Direction,
) (*streaming.Streaming, error) {
	id := h.utilHandler.UUIDCreate()
	return h.createWithID(ctx, id, customerID, activeflowID, referenceType, referenceID, language, gender, "", "", direction)
}

// createWithID creates a streaming record with a pre-determined ID, provider, and voiceID.
func (h *streamingHandler) createWithID(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender streaming.Gender,
	provider string,
	voiceID string,
	direction streaming.Direction,
) (*streaming.Streaming, error) {
	h.muStreaming.Lock()
	defer h.muStreaming.Unlock()

	res := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		PodID: h.podID,

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Language:  language,
		Gender:    gender,
		Provider:  provider,
		VoiceID:   voiceID,
		Direction: direction,

		VendorName:   streaming.VendorNameNone,
		VendorLock:   sync.Mutex{},
		VendorConfig: nil,

		CreatedAt: time.Now(),
	}

	_, ok := h.mapStreaming[id]
	if ok {
		return nil, fmt.Errorf("streaming already exists. streaming_id: %s", id)
	}

	h.mapStreaming[id] = res
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingCreated, res)

	// metrics: track creation (vendor is not yet known at this point, use "unknown")
	promStreamingCreatedTotal.WithLabelValues("unknown").Inc()
	promStreamingActive.WithLabelValues("unknown").Inc()
	promStreamingLanguageTotal.WithLabelValues(language, string(gender)).Inc()

	return res, nil
}

// List returns streaming
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

	// metrics: track session end
	vendor := string(tmp.VendorName)
	if vendor == "" {
		vendor = "unknown"
	}
	promStreamingEndedTotal.WithLabelValues(vendor).Inc()
	promStreamingActive.WithLabelValues("unknown").Dec()
	promStreamingDurationSeconds.WithLabelValues(vendor).Observe(time.Since(tmp.CreatedAt).Seconds())
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

func (h *streamingHandler) UpdateConnAst(streamingID uuid.UUID, connAst *websocket.Conn) (*streaming.Streaming, error) {
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

// WaitFinish blocks until all streaming audio has been delivered to Asterisk.
// It waits on the vendor-specific processDone channel (GCP) or ConnAstDone as fallback.
func (h *streamingHandler) WaitFinish(ctx context.Context, id uuid.UUID) error {
	st, err := h.Get(ctx, id)
	if err != nil {
		return err
	}

	st.VendorLock.Lock()
	vc := st.VendorConfig
	vendorName := st.VendorName
	connAstDone := st.ConnAstDone
	st.VendorLock.Unlock()

	if vc == nil {
		return fmt.Errorf("vendor config not initialized for streaming %s", id)
	}

	switch vendorName {
	case streaming.VendorNameGCP:
		cf, ok := vc.(*GCPConfig)
		if !ok {
			return fmt.Errorf("invalid vendor config type for GCP")
		}
		// Snapshot processDone under muStream to avoid data race with
		// waitAndReconnectLocked which may replace cf.processDone.
		cf.muStream.Lock()
		doneCh := cf.processDone
		cf.muStream.Unlock()
		select {
		case <-doneCh:
			return nil
		case <-cf.Ctx.Done():
			return fmt.Errorf("streaming session cancelled before audio completed for %s", id)
		case <-ctx.Done():
			return ctx.Err() // Caller timeout
		}
	default:
		// For non-GCP vendors, wait on ConnAstDone as fallback
		if connAstDone == nil {
			return fmt.Errorf("no completion channel for streaming %s", id)
		}
		select {
		case <-connAstDone:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
