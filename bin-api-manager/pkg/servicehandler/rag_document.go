package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	rmdocument "monorepo/bin-rag-manager/models/document"
)

// ragDocumentGet is the private helper — fetches document without permission check.
func (h *serviceHandler) ragDocumentGet(ctx context.Context, id uuid.UUID) (*rmdocument.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ragDocumentGet",
		"document_id": id,
	})

	res, err := h.reqHandler.RagV1DocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the rag document info. err: %v", err)
		return nil, err
	}
	log.WithField("document", res).Debug("Received result.")

	return res, nil
}

func (h *serviceHandler) RagDocumentGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdocument.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDocumentGet",
		"customer_id": a.CustomerID,
		"document_id": id,
	})
	log.Debug("Getting a rag document.")

	tmp, err := h.ragDocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag document info. err: %v", err)
		return nil, fmt.Errorf("could not find rag document info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagDocumentGets(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, size uint64, token string) ([]*rmdocument.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDocumentGets",
		"customer_id": a.CustomerID,
		"rag_id":      ragID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting rag documents.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	filters := map[rmdocument.Field]any{
		rmdocument.FieldCustomerID: a.CustomerID,
	}
	if ragID != uuid.Nil {
		filters[rmdocument.FieldRagID] = ragID
	}

	docs, err := h.reqHandler.RagV1DocumentGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get rag documents info. err: %v", err)
		return nil, fmt.Errorf("could not find rag documents info. err: %v", err)
	}

	res := []*rmdocument.WebhookMessage{}
	for _, d := range docs {
		tmp := d.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}
