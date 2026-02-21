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

// ContactV1PhoneNumberCreate sends a request to contact-manager to add a phone number to a contact.
func (r *requestHandler) ContactV1PhoneNumberCreate(
	ctx context.Context,
	contactID uuid.UUID,
	number string,
	numberE164 string,
	phoneType string,
	isPrimary bool,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/phone-numbers", contactID)

	data := &cmrequest.PhoneNumberCreate{
		Number:     number,
		NumberE164: numberE164,
		Type:       phoneType,
		IsPrimary:  isPrimary,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts/<contact-id>/phone-numbers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1PhoneNumberUpdate sends a request to contact-manager to update a phone number on a contact.
func (r *requestHandler) ContactV1PhoneNumberUpdate(
	ctx context.Context,
	contactID uuid.UUID,
	phoneNumberID uuid.UUID,
	fields map[string]any,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/phone-numbers/%s", contactID, phoneNumberID)

	data := &cmrequest.PhoneNumberUpdate{}
	if v, ok := fields["number"]; ok {
		if s, isStr := v.(string); isStr {
			data.Number = &s
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

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/contacts/<contact-id>/phone-numbers/<phone-number-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1PhoneNumberDelete sends a request to contact-manager to remove a phone number from a contact.
func (r *requestHandler) ContactV1PhoneNumberDelete(ctx context.Context, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/phone-numbers/%s", contactID, phoneNumberID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>/phone-numbers/<phone-number-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
