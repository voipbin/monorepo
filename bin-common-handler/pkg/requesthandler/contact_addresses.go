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

// ContactV1AddressCreate sends a request to contact-manager to add an address to a contact.
func (r *requestHandler) ContactV1AddressCreate(
	ctx context.Context,
	contactID uuid.UUID,
	addrType string,
	target string,
	isPrimary bool,
	name string,
	detail string,
) (*cmcontact.Address, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/addresses", contactID)

	data := &cmrequest.AddressCreate{
		Type:      addrType,
		Target:    target,
		IsPrimary: isPrimary,
		Name:      name,
		Detail:    detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts/<contact-id>/addresses", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Address
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1AddressGet sends a request to contact-manager to list addresses of a contact.
func (r *requestHandler) ContactV1AddressGet(ctx context.Context, contactID uuid.UUID) ([]cmcontact.Address, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/addresses", contactID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts/<contact-id>/addresses", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	type addressesResponse struct {
		Result []cmcontact.Address `json:"result"`
	}
	var res addressesResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res.Result, nil
}

// ContactV1AddressUpdate sends a request to contact-manager to update an address on a contact.
func (r *requestHandler) ContactV1AddressUpdate(
	ctx context.Context,
	contactID uuid.UUID,
	addressID uuid.UUID,
	fields map[string]any,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/addresses/%s", contactID, addressID)

	data := &cmrequest.AddressUpdate{}
	if v, ok := fields["target"]; ok {
		if s, isStr := v.(string); isStr {
			data.Target = &s
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

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/contacts/<contact-id>/addresses/<address-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1AddressDelete sends a request to contact-manager to remove an address from a contact.
func (r *requestHandler) ContactV1AddressDelete(ctx context.Context, contactID uuid.UUID, addressID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/addresses/%s", contactID, addressID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>/addresses/<address-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
