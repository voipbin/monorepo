package raghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/rag"
)

func (h *ragHandler) RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagCreate",
		"customer_id": customerID,
		"name":        name,
	})

	id, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate UUID. err: %v", err)
		return nil, fmt.Errorf("could not generate rag id: %w", err)
	}

	r := &rag.Rag{
		ID:          id,
		CustomerID:  customerID,
		Name:        name,
		Description: description,
	}

	if errCreate := h.dbHandler.RagCreate(ctx, r); errCreate != nil {
		log.Errorf("Could not create rag. err: %v", errCreate)
		return nil, fmt.Errorf("could not create rag: %w", errCreate)
	}
	log.WithField("rag", r).Debugf("Created rag. rag_id: %s", r.ID)

	// Create documents for each source and trigger ingestion
	h.createDocumentsForSources(ctx, customerID, id, storageFileIDs, sourceURLs)

	// Return enriched Rag with Status/Sources populated
	return h.RagGet(ctx, id)
}

func (h *ragHandler) RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "RagGet",
		"id":   id,
	})

	r, err := h.dbHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag. err: %v", err)
		return nil, fmt.Errorf("could not get rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Retrieved rag. rag_id: %s", r.ID)

	docs, err := h.dbHandler.DocumentGetsByRagID(ctx, id)
	if err != nil {
		log.Errorf("Could not get documents for rag. err: %v", err)
		return nil, fmt.Errorf("could not get documents for rag: %w", err)
	}
	log.Debugf("Retrieved %d documents for rag. rag_id: %s", len(docs), id)

	r.Status = computeRagStatus(docs)
	r.Sources = buildSources(docs)

	return r, nil
}

func (h *ragHandler) RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RagList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	rags, err := h.dbHandler.RagList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not list rags. err: %v", err)
		return nil, fmt.Errorf("could not list rags: %w", err)
	}
	log.Debugf("Listed rags. count: %d", len(rags))

	if len(rags) == 0 {
		return rags, nil
	}

	ragIDs := make([]uuid.UUID, len(rags))
	for i, r := range rags {
		ragIDs[i] = r.ID
	}

	docsMap, err := h.dbHandler.DocumentGetsByRagIDs(ctx, ragIDs)
	if err != nil {
		log.Errorf("Could not batch fetch documents for rags. err: %v", err)
		return rags, nil
	}

	for _, r := range rags {
		docs := docsMap[r.ID]
		r.Status = computeRagStatus(docs)
		r.Sources = buildSources(docs)
	}

	return rags, nil
}

func (h *ragHandler) RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "RagUpdate",
		"id":     id,
		"fields": fields,
	})

	if err := h.dbHandler.RagUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update rag. err: %v", err)
		return nil, fmt.Errorf("could not update rag: %w", err)
	}

	r, err := h.dbHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated rag. err: %v", err)
		return nil, fmt.Errorf("could not get updated rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Updated rag. rag_id: %s", r.ID)

	return r, nil
}

func (h *ragHandler) RagDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "RagDelete",
		"id":   id,
	})

	// cascade: soft-delete chunks and documents first
	if err := h.dbHandler.ChunkSoftDeleteByRagID(ctx, id); err != nil {
		log.Errorf("Could not soft delete chunks. err: %v", err)
		return fmt.Errorf("could not delete rag chunks: %w", err)
	}

	if err := h.dbHandler.DocumentDeleteByRagID(ctx, id); err != nil {
		log.Errorf("Could not soft delete documents. err: %v", err)
		return fmt.Errorf("could not delete rag documents: %w", err)
	}

	if err := h.dbHandler.RagDelete(ctx, id); err != nil {
		log.Errorf("Could not delete rag. err: %v", err)
		return fmt.Errorf("could not delete rag: %w", err)
	}
	log.Debugf("Deleted rag. rag_id: %s", id)

	// Background hard-delete of chunks to reclaim storage
	go h.chunkHardDeleteByRagID(id)

	return nil
}

func (h *ragHandler) RagAddSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "RagAddSources",
		"rag_id": ragID,
	})

	r, err := h.dbHandler.RagGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get rag. err: %v", err)
		return nil, fmt.Errorf("could not get rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Retrieved rag for adding sources. rag_id: %s", r.ID)

	h.createDocumentsForSources(ctx, r.CustomerID, r.ID, storageFileIDs, sourceURLs)

	return h.RagGet(ctx, ragID)
}

func (h *ragHandler) RagRemoveSource(ctx context.Context, ragID, sourceID uuid.UUID) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "RagRemoveSource",
		"rag_id":    ragID,
		"source_id": sourceID,
	})

	// Verify document exists and belongs to this RAG
	doc, err := h.dbHandler.DocumentGet(ctx, sourceID)
	if err != nil {
		log.Errorf("Could not get document. err: %v", err)
		return nil, fmt.Errorf("could not get source: %w", err)
	}
	log.WithField("document", doc).Debugf("Retrieved document. document_id: %s", doc.ID)

	if doc.RagID != ragID {
		log.Errorf("Document does not belong to this rag. doc.rag_id: %s, rag_id: %s", doc.RagID, ragID)
		return nil, fmt.Errorf("source does not belong to this rag")
	}

	// Cascade: soft-delete chunks, then delete document
	if err := h.dbHandler.ChunkSoftDeleteByDocumentID(ctx, sourceID); err != nil {
		log.Errorf("Could not soft delete chunks. err: %v", err)
		return nil, fmt.Errorf("could not delete source chunks: %w", err)
	}

	if err := h.dbHandler.DocumentDelete(ctx, sourceID); err != nil {
		log.Errorf("Could not delete document. err: %v", err)
		return nil, fmt.Errorf("could not delete source: %w", err)
	}
	log.Debugf("Deleted source. source_id: %s", sourceID)

	// Background hard-delete of chunks to reclaim storage
	go h.chunkHardDeleteByDocumentID(sourceID)

	return h.RagGet(ctx, ragID)
}

// chunkHardDeleteByDocumentID hard-deletes all chunks for a document in the background.
// Uses context.Background() because the request context may be cancelled after the response is sent.
// If the hard-delete fails, chunks remain soft-deleted (invisible to search) — safe degradation.
func (h *ragHandler) chunkHardDeleteByDocumentID(documentID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chunkHardDeleteByDocumentID",
		"document_id": documentID,
	})

	if err := h.dbHandler.ChunkDeleteByDocumentID(context.Background(), documentID); err != nil {
		log.Errorf("Could not hard delete chunks for document. err: %v", err)
		return
	}
	log.Debugf("Hard deleted chunks for document. document_id: %s", documentID)
}

// chunkHardDeleteByRagID hard-deletes all chunks for a rag in the background.
// Uses context.Background() because the request context may be cancelled after the response is sent.
// If the hard-delete fails, chunks remain soft-deleted (invisible to search) — safe degradation.
func (h *ragHandler) chunkHardDeleteByRagID(ragID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "chunkHardDeleteByRagID",
		"rag_id": ragID,
	})

	if err := h.dbHandler.ChunkDeleteByRagID(context.Background(), ragID); err != nil {
		log.Errorf("Could not hard delete chunks for rag. err: %v", err)
		return
	}
	log.Debugf("Hard deleted chunks for rag. rag_id: %s", ragID)
}

// createDocumentsForSources creates documents for each file ID and URL, then triggers ingestion.
// Uses request ctx for DB writes; ingestion goroutines use context.Background().
func (h *ragHandler) createDocumentsForSources(ctx context.Context, customerID, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "createDocumentsForSources",
		"rag_id": ragID,
	})
	for _, fileID := range storageFileIDs {
		doc, err := h.documentCreateInternal(ctx, customerID, ragID, fileID, "")
		if err != nil {
			log.Errorf("Could not create document for file_id %s: %v", fileID, err)
			continue
		}
		go h.documentIngest(doc)
	}

	for _, u := range sourceURLs {
		doc, err := h.documentCreateInternal(ctx, customerID, ragID, uuid.Nil, u)
		if err != nil {
			log.Errorf("Could not create document for url %s: %v", u, err)
			continue
		}
		go h.documentIngest(doc)
	}
}

// computeRagStatus derives RAG status from its documents.
// Priority: processing > error-only > ready.
// If any doc is pending/processing, the RAG is "processing".
// If all docs are terminal (ready or error) and at least one is ready, the RAG is "ready"
// (individual source errors are visible in the sources list).
// If all docs are error, the RAG is "error".
func computeRagStatus(docs []*document.Document) document.Status {
	if len(docs) == 0 {
		return document.StatusPending
	}

	hasReady := false
	hasError := false
	hasInProgress := false

	for _, d := range docs {
		switch d.Status {
		case document.StatusPending, document.StatusProcessing:
			hasInProgress = true
		case document.StatusReady:
			hasReady = true
		case document.StatusError:
			hasError = true
		}
	}

	if hasInProgress {
		return document.StatusProcessing
	}
	if hasReady {
		return document.StatusReady
	}
	if hasError {
		return document.StatusError
	}
	return document.StatusPending
}

// buildSources converts documents to Source structs for the RAG response.
func buildSources(docs []*document.Document) []rag.Source {
	sources := make([]rag.Source, 0, len(docs))
	for _, d := range docs {
		s := rag.Source{
			Status:        d.Status,
			StatusMessage: d.StatusMessage,
		}
		s.ID = d.ID
		s.CustomerID = d.CustomerID
		if d.StorageFileID != uuid.Nil {
			fileID := d.StorageFileID
			s.StorageFileID = &fileID
		}
		if d.SourceURL != "" {
			s.SourceURL = d.SourceURL
		}
		sources = append(sources, s)
	}
	return sources
}
