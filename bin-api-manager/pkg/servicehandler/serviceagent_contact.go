package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentContactCreate sends a request to contact-manager
// to create a contact for the service agent's customer.
func (h *serviceHandler) ServiceAgentContactCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	firstName string,
	lastName string,
	displayName string,
	company string,
	jobTitle string,
	source string,
	externalID string,
	notes string,
	addresses []cmrequest.AddressCreate,
	tagIDs []uuid.UUID,
) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactCreate",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1ContactCreate(
		ctx,
		a.CustomerID,
		firstName,
		lastName,
		displayName,
		company,
		jobTitle,
		source,
		externalID,
		notes,
		addresses,
		tagIDs,
	)
	if err != nil {
		log.Errorf("Could not create a contact. err: %v", err)
		return nil, err
	}
	log.WithField("contact", tmp).Debug("Created contact.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactGet sends a request to contact-manager
// to get a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactGet(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactGet",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactList sends a request to contact-manager
// to get a list of contacts for the service agent's customer.
func (h *serviceHandler) ServiceAgentContactList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":    "ServiceAgentContactList",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	typedFilters, err := h.convertContactFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	tmps, err := h.contactList(ctx, size, token, typedFilters)
	if err != nil {
		log.Infof("Could not get contacts info. err: %v", err)
		return nil, err
	}

	res := []*cmcontact.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// ServiceAgentContactUpdate sends a request to contact-manager
// to update a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	firstName *string,
	lastName *string,
	displayName *string,
	company *string,
	jobTitle *string,
	externalID *string,
	notes *string,
) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactUpdate",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1ContactUpdate(
		ctx,
		contactID,
		firstName,
		lastName,
		displayName,
		company,
		jobTitle,
		externalID,
		notes,
	)
	if err != nil {
		log.Infof("Could not update the contact info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactDelete sends a request to contact-manager
// to delete a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactDelete(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactDelete",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1ContactDelete(ctx, contactID)
	if err != nil {
		log.Infof("Could not delete the contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactLookup sends a request to contact-manager
// to lookup a contact by phone or email for the service agent.
func (h *serviceHandler) ServiceAgentContactLookup(ctx context.Context, a *auth.AuthIdentity, phoneE164 string, email string) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactLookup",
		"customer_id": a.CustomerID,
		"phone_e164":  phoneE164,
		"email":       email,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1ContactLookup(ctx, a.CustomerID, phoneE164, email)
	if err != nil {
		log.Infof("Could not lookup the contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactAddressCreate sends a request to contact-manager
// to add an address to a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactAddressCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	addrType string,
	target string,
	isPrimary bool,
) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactAddressCreate",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if _, err := h.reqHandler.ContactV1AddressCreate(ctx, contactID, addrType, target, isPrimary); err != nil {
		log.Infof("Could not add address to contact. err: %v", err)
		return nil, err
	}

	// Re-fetch the contact to return updated state
	updated, err := h.reqHandler.ContactV1ContactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get updated contact. err: %v", err)
		return nil, err
	}

	res := updated.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactAddressUpdate sends a request to contact-manager
// to update an address on a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactAddressUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	contactID uuid.UUID,
	addressID uuid.UUID,
	fields map[string]any,
) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactAddressUpdate",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"address_id":  addressID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1AddressUpdate(ctx, contactID, addressID, fields)
	if err != nil {
		log.Infof("Could not update address on contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactAddressDelete sends a request to contact-manager
// to remove an address from a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactAddressDelete(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addressID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactAddressDelete",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"address_id":  addressID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1AddressDelete(ctx, contactID, addressID)
	if err != nil {
		log.Infof("Could not delete address from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactTagAdd sends a request to contact-manager
// to add a tag to a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactTagAdd(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactTagAdd",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"tag_id":      tagID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1TagAdd(ctx, contactID, tagID)
	if err != nil {
		log.Infof("Could not add tag to contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactTagRemove sends a request to contact-manager
// to remove a tag from a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactTagRemove(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactTagRemove",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"tag_id":      tagID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ContactV1TagRemove(ctx, contactID, tagID)
	if err != nil {
		log.Infof("Could not remove tag from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
