package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
)

// ContactV1EmailCreate sends a request to contact-manager to add an email to a contact.
func (r *requestHandler) ContactV1EmailCreate(
	ctx context.Context,
	contactID uuid.UUID,
	address string,
	emailType string,
	isPrimary bool,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/emails", contactID)

	data := &cmrequest.EmailCreate{
		Address:   address,
		Type:      emailType,
		IsPrimary: isPrimary,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts/<contact-id>/emails", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1EmailUpdate sends a request to contact-manager to update an email on a contact.
func (r *requestHandler) ContactV1EmailUpdate(
	ctx context.Context,
	contactID uuid.UUID,
	emailID uuid.UUID,
	fields map[string]any,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/emails/%s", contactID, emailID)

	data := &cmrequest.EmailUpdate{}
	if v, ok := fields["address"]; ok {
		if s, isStr := v.(string); isStr {
			data.Address = &s
		}
	}
	if v, ok := fields["type"]; ok {
		if s, isStr := v.(string); isStr {
			data.Type = &s
		}
	}
	if v, ok := fields["is_primary"]; ok {
		if b, isBool := v.(bool); isBool {
			data.IsPrimary = &b
		}
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/contacts/<contact-id>/emails/<email-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1EmailDelete sends a request to contact-manager to remove an email from a contact.
func (r *requestHandler) ContactV1EmailDelete(ctx context.Context, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/emails/%s", contactID, emailID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>/emails/<email-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
