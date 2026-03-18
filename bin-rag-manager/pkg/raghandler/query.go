package raghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/models/query"
)

func (h *ragHandler) QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "QueryRag",
		"rag_id": ragID,
		"top_k":  topK,
	})

	if topK <= 0 {
		topK = 5
	}

	// embed the query text
	embedding, err := h.embedder.EmbedText(ctx, queryText)
	if err != nil {
		log.Errorf("Could not embed query. err: %v", err)
		return nil, fmt.Errorf("could not embed query: %w", err)
	}

	// vector similarity search
	chunks, scores, err := h.dbHandler.ChunkSearchByRagID(ctx, ragID, embedding, topK)
	if err != nil {
		log.Errorf("Could not search chunks. err: %v", err)
		return nil, fmt.Errorf("could not search chunks: %w", err)
	}
	log.Debugf("Found %d matching chunks for rag_id: %s", len(chunks), ragID)

	// build sources from chunks + scores
	sources := make([]query.Source, len(chunks))
	for i, c := range chunks {
		// look up document name
		docName := ""
		doc, errDoc := h.dbHandler.DocumentGet(ctx, c.DocumentID)
		if errDoc == nil {
			docName = doc.Name
		}

		sources[i] = query.Source{
			DocumentID:     c.DocumentID,
			DocumentName:   docName,
			SectionTitle:   c.SectionTitle,
			RelevanceScore: scores[i],
		}
	}

	return &query.Response{
		Sources: sources,
	}, nil
}
