package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// indexIncrementalRequest represents the request body for incremental indexing
type indexIncrementalRequest struct {
	Files []string `json:"files"`
}

// processV1RagIndexPost handles POST /v1/rags/index (full re-index)
func (h *listenHandler) processV1RagIndexPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1RagIndexPost")

	// Run indexing in background so the RPC response returns immediately
	go func() {
		if err := h.ragHandler.IndexFull(context.Background()); err != nil {
			log.Errorf("Full indexing failed. err: %v", err)
		}
	}()

	log.Infof("Full re-indexing triggered")
	return jsonResponse(202, map[string]string{"status": "indexing_started"}), nil
}

// processV1RagIndexIncrementalPost handles POST /v1/rags/index/incremental
func (h *listenHandler) processV1RagIndexIncrementalPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1RagIndexIncrementalPost")

	var req indexIncrementalRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Infof("Could not unmarshal request data. err: %v", err)
		return simpleResponse(400), nil
	}

	if len(req.Files) == 0 {
		log.Infof("No files specified for incremental indexing")
		return simpleResponse(400), nil
	}

	// Run indexing in background
	go func() {
		if err := h.ragHandler.IndexIncremental(context.Background(), req.Files); err != nil {
			log.Errorf("Incremental indexing failed. err: %v", err)
		}
	}()

	log.Infof("Incremental re-indexing triggered for %d files", len(req.Files))
	return jsonResponse(202, map[string]string{"status": "indexing_started"}), nil
}

// processV1RagIndexStatusGet handles GET /v1/rags/index/status
func (h *listenHandler) processV1RagIndexStatusGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1RagIndexStatusGet")

	status, err := h.ragHandler.IndexStatus(ctx)
	if err != nil {
		log.Errorf("Could not get index status. err: %v", err)
		return simpleResponse(500), nil
	}
	log.Debugf("Index status retrieved. chunk_count: %d", status.ChunkCount)

	return jsonResponse(200, status), nil
}
