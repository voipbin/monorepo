package servicehandler

import (
	"context"
	"fmt"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcontact "monorepo/bin-contact-manager/models/contact"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// contactGet validates the contact's ownership and returns the contact info.
func (h *serviceHandler) contactGet(ctx context.Context, id uuid.UUID) (*cmcontact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "contactGet",
		"contact_id": id,
	})

	// send request
	res, err := h.reqHandler.ContactV1ContactGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}
	log.WithField("contact", res).Debug("Received result.")

	return res, nil
}

// ContactCreate sends a request to contact-manager
// to create a contact.
// it returns created contact info if it succeeds.
func (h *serviceHandler) ContactCreate(
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
		"func":        "ContactCreate",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	log.Debug("Creating a new contact.")
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
	log.WithField("contact", tmp).Debug("Received result.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactGet sends a request to contact-manager
// to get a contact.
func (h *serviceHandler) ContactGet(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactGet",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactList sends a request to contact-manager
// to get a list of contacts.
// it returns list of contacts if it succeeds.
func (h *serviceHandler) ContactList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ContactList",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// Convert string filters to typed filters
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

	// create result
	res := []*cmcontact.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// contactList sends a request to contact-manager
// to get a list of contacts.
// it returns list of contacts if it succeeds.
func (h *serviceHandler) contactList(ctx context.Context, size uint64, token string, filters map[cmcontact.Field]any) ([]cmcontact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "contactList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	res, err := h.reqHandler.ContactV1ContactList(ctx, token, size, filters)
	if err != nil {
		log.Infof("Could not get contacts info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ContactUpdate sends a request to contact-manager
// to update the contact info.
func (h *serviceHandler) ContactUpdate(
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
		"func":        "ContactUpdate",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
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

// ContactDelete sends a request to contact-manager
// to delete the contact.
func (h *serviceHandler) ContactDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactDelete",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1ContactDelete(ctx, contactID)
	if err != nil {
		log.Infof("Could not delete the contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactLookup sends a request to contact-manager
// to lookup a contact by phone number or email.
func (h *serviceHandler) ContactLookup(ctx context.Context, a *amagent.Agent, phoneE164 string, email string) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactLookup",
		"customer_id": a.CustomerID,
		"phone_e164":  phoneE164,
		"email":       email,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1ContactLookup(ctx, a.CustomerID, phoneE164, email)
	if err != nil {
		log.Infof("Could not lookup the contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactPhoneNumberCreate sends a request to contact-manager
// to add a phone number to a contact.
func (h *serviceHandler) ContactPhoneNumberCreate(
	ctx context.Context,
	a *amagent.Agent,
	contactID uuid.UUID,
	number string,
	numberE164 string,
	phoneType string,
	isPrimary bool,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactPhoneNumberCreate",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1PhoneNumberCreate(ctx, contactID, number, numberE164, phoneType, isPrimary)
	if err != nil {
		log.Infof("Could not add phone number to contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactPhoneNumberDelete sends a request to contact-manager
// to remove a phone number from a contact.
func (h *serviceHandler) ContactPhoneNumberDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ContactPhoneNumberDelete",
		"customer_id":     a.CustomerID,
		"contact_id":      contactID,
		"phone_number_id": phoneNumberID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1PhoneNumberDelete(ctx, contactID, phoneNumberID)
	if err != nil {
		log.Infof("Could not delete phone number from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactEmailCreate sends a request to contact-manager
// to add an email to a contact.
func (h *serviceHandler) ContactEmailCreate(
	ctx context.Context,
	a *amagent.Agent,
	contactID uuid.UUID,
	address string,
	emailType string,
	isPrimary bool,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactEmailCreate",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1EmailCreate(ctx, contactID, address, emailType, isPrimary)
	if err != nil {
		log.Infof("Could not add email to contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactEmailDelete sends a request to contact-manager
// to remove an email from a contact.
func (h *serviceHandler) ContactEmailDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactEmailDelete",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"email_id":    emailID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1EmailDelete(ctx, contactID, emailID)
	if err != nil {
		log.Infof("Could not delete email from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactTagAdd sends a request to contact-manager
// to add a tag to a contact.
func (h *serviceHandler) ContactTagAdd(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactTagAdd",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"tag_id":      tagID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1TagAdd(ctx, contactID, tagID)
	if err != nil {
		log.Infof("Could not add tag to contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ContactTagRemove sends a request to contact-manager
// to remove a tag from a contact.
func (h *serviceHandler) ContactTagRemove(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactTagRemove",
		"customer_id": a.CustomerID,
		"contact_id":  contactID,
		"tag_id":      tagID,
	})

	ct, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ct.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.ContactV1TagRemove(ctx, contactID, tagID)
	if err != nil {
		log.Infof("Could not remove tag from contact. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// convertContactFilters converts map[string]string to map[cmcontact.Field]any
func (h *serviceHandler) convertContactFilters(filters map[string]string) (map[cmcontact.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, cmcontact.Contact{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[cmcontact.Field]any, len(typed))
	for k, v := range typed {
		result[cmcontact.Field(k)] = v
	}

	return result, nil
}
