package raghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/models/document"
)

func (h *ragHandler) DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType document.DocType, sourceURL string, storageFileID uuid.UUID) (*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DocumentCreate",
		"customer_id": customerID,
		"rag_id":      ragID,
		"name":        name,
		"doc_type":    docType,
	})

	id, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate UUID. err: %v", err)
		return nil, fmt.Errorf("could not generate document id: %w", err)
	}

	d := &document.Document{
		ID:            id,
		CustomerID:    customerID,
		RagID:         ragID,
		Name:          name,
		DocType:       docType,
		SourceURL:     sourceURL,
		StorageFileID: storageFileID,
		Status:        document.StatusPending,
	}

	if errCreate := h.dbHandler.DocumentCreate(ctx, d); errCreate != nil {
		log.Errorf("Could not create document. err: %v", errCreate)
		return nil, fmt.Errorf("could not create document: %w", errCreate)
	}
	log.WithField("document", d).Debugf("Created document. document_id: %s", d.ID)

	// TODO: trigger async ingestion goroutine (Phase 2b — chunking + embedding pipeline)

	return d, nil
}

func (h *ragHandler) DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "DocumentGet",
		"id":   id,
	})

	d, err := h.dbHandler.DocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get document. err: %v", err)
		return nil, fmt.Errorf("could not get document: %w", err)
	}
	log.WithField("document", d).Debugf("Retrieved document. document_id: %s", d.ID)

	return d, nil
}

func (h *ragHandler) DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DocumentList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	docs, err := h.dbHandler.DocumentList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not list documents. err: %v", err)
		return nil, fmt.Errorf("could not list documents: %w", err)
	}
	log.Debugf("Listed documents. count: %d", len(docs))

	return docs, nil
}

func (h *ragHandler) DocumentDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "DocumentDelete",
		"id":   id,
	})

	// cascade: soft-delete chunks first
	if err := h.dbHandler.ChunkSoftDeleteByDocumentID(ctx, id); err != nil {
		log.Errorf("Could not soft delete chunks. err: %v", err)
		return fmt.Errorf("could not delete document chunks: %w", err)
	}

	if err := h.dbHandler.DocumentDelete(ctx, id); err != nil {
		log.Errorf("Could not delete document. err: %v", err)
		return fmt.Errorf("could not delete document: %w", err)
	}
	log.Debugf("Deleted document. document_id: %s", id)

	return nil
}
