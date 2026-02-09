package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	rmrag "monorepo/bin-rag-manager/pkg/raghandler"

	"github.com/sirupsen/logrus"
)

// RagQuery queries the RAG documentation system.
func (h *serviceHandler) RagQuery(ctx context.Context, a *amagent.Agent, query string, docTypes []string, topK int) (*rmrag.QueryResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "RagQuery",
		"query": query,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.RagV1RagQuery(ctx, query, docTypes, topK)
	if err != nil {
		log.Errorf("Could not query RAG. err: %v", err)
		return nil, err
	}
	log.WithField("response", res).Debug("RAG query completed.")

	return res, nil
}
