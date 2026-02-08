package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentContactCreate sends a request to contact-manager
// to create a contact for the service agent's customer.
func (h *serviceHandler) ServiceAgentContactCreate(
	ctx context.Context,
	a *amagent.Agent,
	firstName string,
	lastName string,
	displayName string,
	company string,
	jobTitle string,
	source string,
	externalID string,
	notes string,
	phoneNumbers []cmrequest.PhoneNumberCreate,
	emails []cmrequest.EmailCreate,
	tagIDs []uuid.UUID,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactCreate",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
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
		phoneNumbers,
		emails,
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
func (h *serviceHandler) ServiceAgentContactGet(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
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
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactList sends a request to contact-manager
// to get a list of contacts for the service agent's customer.
func (h *serviceHandler) ServiceAgentContactList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error) {
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
		return nil, fmt.Errorf("agent has no permission")
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
	a *amagent.Agent,
	contactID uuid.UUID,
	firstName *string,
	lastName *string,
	displayName *string,
	company *string,
	jobTitle *string,
	externalID *string,
	notes *string,
) (*cmcontact.WebhookMessage, error) {
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
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ServiceAgentContactDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
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
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ServiceAgentContactLookup(ctx context.Context, a *amagent.Agent, phoneE164 string, email string) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactLookup",
		"customer_id": a.CustomerID,
		"phone_e164":  phoneE164,
		"email":       email,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1ContactLookup(ctx, a.CustomerID, phoneE164, email)
	if err != nil {
		log.Infof("Could not lookup the contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactPhoneNumberCreate sends a request to contact-manager
// to add a phone number to a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactPhoneNumberCreate(
	ctx context.Context,
	a *amagent.Agent,
	contactID uuid.UUID,
	number string,
	numberE164 string,
	phoneType string,
	isPrimary bool,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactPhoneNumberCreate",
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
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1PhoneNumberCreate(ctx, contactID, number, numberE164, phoneType, isPrimary)
	if err != nil {
		log.Infof("Could not add phone number to contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactPhoneNumberUpdate sends a request to contact-manager
// to update a phone number on a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactPhoneNumberUpdate(
	ctx context.Context,
	a *amagent.Agent,
	contactID uuid.UUID,
	phoneNumberID uuid.UUID,
	fields map[string]any,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ServiceAgentContactPhoneNumberUpdate",
		"customer_id":     a.CustomerID,
		"contact_id":      contactID,
		"phone_number_id": phoneNumberID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1PhoneNumberUpdate(ctx, contactID, phoneNumberID, fields)
	if err != nil {
		log.Infof("Could not update phone number on contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactPhoneNumberDelete sends a request to contact-manager
// to remove a phone number from a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactPhoneNumberDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ServiceAgentContactPhoneNumberDelete",
		"customer_id":     a.CustomerID,
		"contact_id":      contactID,
		"phone_number_id": phoneNumberID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1PhoneNumberDelete(ctx, contactID, phoneNumberID)
	if err != nil {
		log.Infof("Could not delete phone number from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactEmailCreate sends a request to contact-manager
// to add an email to a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactEmailCreate(
	ctx context.Context,
	a *amagent.Agent,
	contactID uuid.UUID,
	address string,
	emailType string,
	isPrimary bool,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactEmailCreate",
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
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1EmailCreate(ctx, contactID, address, emailType, isPrimary)
	if err != nil {
		log.Infof("Could not add email to contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactEmailUpdate sends a request to contact-manager
// to update an email on a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactEmailUpdate(
	ctx context.Context,
	a *amagent.Agent,
	contactID uuid.UUID,
	emailID uuid.UUID,
	fields map[string]any,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactEmailUpdate",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"email_id":    emailID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1EmailUpdate(ctx, contactID, emailID, fields)
	if err != nil {
		log.Infof("Could not update email on contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactEmailDelete sends a request to contact-manager
// to remove an email from a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactEmailDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentContactEmailDelete",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"email_id":    emailID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1EmailDelete(ctx, contactID, emailID)
	if err != nil {
		log.Infof("Could not delete email from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentContactTagAdd sends a request to contact-manager
// to add a tag to a contact for the service agent.
func (h *serviceHandler) ServiceAgentContactTagAdd(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
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
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ServiceAgentContactTagRemove(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
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
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ContactV1TagRemove(ctx, contactID, tagID)
	if err != nil {
		log.Infof("Could not remove tag from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
