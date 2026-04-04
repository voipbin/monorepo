package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	rmrag "monorepo/bin-rag-manager/models/rag"
)

// ragGet is the private helper — fetches rag without permission check.
func (h *serviceHandler) ragGet(ctx context.Context, id uuid.UUID) (*rmrag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ragGet",
		"rag_id": id,
	})

	res, err := h.reqHandler.RagV1RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the rag info. err: %v", err)
		return nil, err
	}
	log.WithField("rag", res).Debug("Received result.")

	return res, nil
}

func (h *serviceHandler) RagCreate(ctx context.Context, a *auth.AuthIdentity, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "RagCreate",
		"customer_id": a.CustomerID,
		"name":        name,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	log.Debug("Creating a new rag.")
	tmp, err := h.reqHandler.RagV1RagCreate(ctx, a.CustomerID, name, description, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not create a new rag. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmrag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "RagGet",
		"customer_id": a.CustomerID,
		"rag_id":      id,
	})
	log.Debug("Getting a rag.")

	tmp, err := h.ragGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagGets(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmrag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "RagGets",
		"customer_id": a.CustomerID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting rags.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	filters := map[rmrag.Field]any{
		rmrag.FieldCustomerID: a.CustomerID,
	}
	rags, err := h.reqHandler.RagV1RagGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get rags info. err: %v", err)
		return nil, fmt.Errorf("could not find rags info. err: %v", err)
	}

	res := []*rmrag.WebhookMessage{}
	for _, r := range rags {
		tmp := r.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

func (h *serviceHandler) RagUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "RagUpdate",
		"customer_id": a.CustomerID,
		"rag_id":      id,
	})
	log.Debug("Updating a rag.")

	tmp, err := h.ragGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	updated, err := h.reqHandler.RagV1RagUpdate(ctx, id, fields)
	if err != nil {
		log.Errorf("Could not update rag. err: %v", err)
		return nil, err
	}

	res := updated.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmrag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDelete",
		"customer_id": a.CustomerID,
		"rag_id":      id,
	})
	log.Debug("Deleting a rag.")

	tmp, err := h.ragGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if err := h.reqHandler.RagV1RagDelete(ctx, id); err != nil {
		log.Errorf("Could not delete rag. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagAddSources(ctx context.Context, a *auth.AuthIdentity, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "RagAddSources",
		"customer_id": a.CustomerID,
		"rag_id":      ragID,
	})
	log.Debug("Adding sources to rag.")

	// Verify RAG exists and belongs to this customer
	tmp, err := h.ragGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	r, err := h.reqHandler.RagV1RagAddSources(ctx, ragID, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not add sources. err: %v", err)
		return nil, err
	}

	res := r.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagRemoveSource(ctx context.Context, a *auth.AuthIdentity, ragID, sourceID uuid.UUID) (*rmrag.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "RagRemoveSource",
		"customer_id": a.CustomerID,
		"rag_id":      ragID,
		"source_id":   sourceID,
	})
	log.Debug("Removing source from rag.")

	// Verify RAG exists and belongs to this customer
	tmp, err := h.ragGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	r, err := h.reqHandler.RagV1RagRemoveSource(ctx, ragID, sourceID)
	if err != nil {
		log.Errorf("Could not remove source. err: %v", err)
		return nil, err
	}

	res := r.ConvertWebhookMessage()
	return res, nil
}
