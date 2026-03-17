package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	rmquery "monorepo/bin-rag-manager/models/query"
)

// RagQuery queries the RAG documentation system.
func (h *serviceHandler) RagQuery(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, queryText string, topK int) (*rmquery.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "RagQuery",
		"rag_id": ragID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.RagV1RagQuery(ctx, ragID, queryText, topK)
	if err != nil {
		log.Errorf("Could not query RAG. err: %v", err)
		return nil, err
	}
	log.WithField("response", res).Debug("RAG query completed.")

	return res, nil
}
