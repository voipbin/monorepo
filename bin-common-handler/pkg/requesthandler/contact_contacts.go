package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
)

// ContactV1ContactCreate sends a request to contact-manager to create a contact.
func (r *requestHandler) ContactV1ContactCreate(
	ctx context.Context,
	customerID uuid.UUID,
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
) (*cmcontact.Contact, error) {
	uri := "/v1/contacts"

	data := &cmrequest.ContactCreate{
		CustomerID:   customerID,
		FirstName:    firstName,
		LastName:     lastName,
		DisplayName:  displayName,
		Company:      company,
		JobTitle:     jobTitle,
		Source:       source,
		ExternalID:   externalID,
		Notes:        notes,
		PhoneNumbers: phoneNumbers,
		Emails:       emails,
		TagIDs:       tagIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactGet sends a request to contact-manager to get a contact.
func (r *requestHandler) ContactV1ContactGet(ctx context.Context, contactID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s", contactID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts/<contact-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactList sends a request to contact-manager to list contacts.
func (r *requestHandler) ContactV1ContactList(ctx context.Context, pageToken string, pageSize uint64, filters map[cmcontact.Field]any) ([]cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ContactV1ContactUpdate sends a request to contact-manager to update a contact.
func (r *requestHandler) ContactV1ContactUpdate(
	ctx context.Context,
	contactID uuid.UUID,
	firstName *string,
	lastName *string,
	displayName *string,
	company *string,
	jobTitle *string,
	externalID *string,
	notes *string,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s", contactID)

	data := &cmrequest.ContactUpdate{
		FirstName:   firstName,
		LastName:    lastName,
		DisplayName: displayName,
		Company:     company,
		JobTitle:    jobTitle,
		ExternalID:  externalID,
		Notes:       notes,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/contacts/<contact-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactDelete sends a request to contact-manager to delete a contact.
func (r *requestHandler) ContactV1ContactDelete(ctx context.Context, contactID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s", contactID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactLookup sends a request to contact-manager to lookup a contact by phone or email.
func (r *requestHandler) ContactV1ContactLookup(ctx context.Context, customerID uuid.UUID, phoneE164 string, email string) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/lookup?customer_id=%s", customerID)
	if phoneE164 != "" {
		uri = fmt.Sprintf("%s&phone_e164=%s", uri, url.QueryEscape(phoneE164))
	}
	if email != "" {
		uri = fmt.Sprintf("%s&email=%s", uri, url.QueryEscape(email))
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts/lookup", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
