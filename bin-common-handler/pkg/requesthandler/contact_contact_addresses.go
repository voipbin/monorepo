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

// ContactV1ContactAddressList sends a request to contact-manager to list addresses.
// filters: contact_id (uuid), type (string)
func (r *requestHandler) ContactV1ContactAddressList(
	ctx context.Context,
	customerID uuid.UUID,
	filters map[string]any,
	pageToken string,
	pageSize uint64,
) ([]cmcontact.Address, error) {
	uri := fmt.Sprintf("/v1/contact_addresses?customer_id=%s", customerID)
	if v, ok := filters["contact_id"]; ok {
		if cid, ok2 := v.(uuid.UUID); ok2 && cid != uuid.Nil {
			uri += fmt.Sprintf("&contact_id=%s", cid)
		}
	}
	if v, ok := filters["type"]; ok {
		if t, ok2 := v.(string); ok2 && t != "" {
			uri += fmt.Sprintf("&type=%s", t)
		}
	}
	if pageToken != "" {
		uri += fmt.Sprintf("&page_token=%s", pageToken)
	}
	if pageSize > 0 {
		uri += fmt.Sprintf("&page_size=%d", pageSize)
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contact_addresses", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	type listResponse struct {
		Result []cmcontact.Address `json:"result"`
	}
	var res listResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res.Result, nil
}

// ContactV1ContactAddressCreate sends a request to contact-manager to create an address.
// customerID is the tenant-scoping value, always derived from the authenticated
// identity in bin-api-manager — never client-supplied. When contactID is
// uuid.Nil, contact-manager creates an "unresolved" address (no contact_id).
func (r *requestHandler) ContactV1ContactAddressCreate(
	ctx context.Context,
	customerID uuid.UUID,
	contactID uuid.UUID,
	addrType string,
	target string,
	isPrimary bool,
	name string,
	detail string,
) (*cmcontact.Address, error) {
	uri := "/v1/contact_addresses"

	data := &cmrequest.ContactAddressCreate{
		CustomerID: customerID,
		ContactID:  contactID,
		Type:       addrType,
		Target:     target,
		IsPrimary:  isPrimary,
		Name:       name,
		Detail:     detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contact_addresses", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Address
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactAddressClaim sends a request to contact-manager to claim
// an unresolved address onto a contact.
func (r *requestHandler) ContactV1ContactAddressClaim(
	ctx context.Context,
	customerID uuid.UUID,
	addressID uuid.UUID,
	contactID uuid.UUID,
) (*cmcontact.Address, error) {
	uri := fmt.Sprintf("/v1/contact_addresses/%s/claim?customer_id=%s", addressID, customerID)

	data := &cmrequest.ContactAddressClaim{ContactID: contactID}
	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contact_addresses/<id>/claim", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Address
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactAddressGet sends a request to contact-manager to get a single address.
func (r *requestHandler) ContactV1ContactAddressGet(
	ctx context.Context,
	customerID uuid.UUID,
	addressID uuid.UUID,
) (*cmcontact.Address, error) {
	uri := fmt.Sprintf("/v1/contact_addresses/%s?customer_id=%s", addressID, customerID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contact_addresses/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Address
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactAddressUpdate sends a request to contact-manager to update an address.
func (r *requestHandler) ContactV1ContactAddressUpdate(
	ctx context.Context,
	customerID uuid.UUID,
	contactID uuid.UUID,
	addressID uuid.UUID,
	fields map[string]any,
) (*cmcontact.Address, error) {
	uri := fmt.Sprintf("/v1/contact_addresses/%s?customer_id=%s&contact_id=%s", addressID, customerID, contactID)

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

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/contact_addresses/<id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Address
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactAddressDelete sends a request to contact-manager to delete an address.
func (r *requestHandler) ContactV1ContactAddressDelete(
	ctx context.Context,
	customerID uuid.UUID,
	contactID uuid.UUID,
	addressID uuid.UUID,
) (*cmcontact.Address, error) {
	uri := fmt.Sprintf("/v1/contact_addresses/%s?customer_id=%s&contact_id=%s", addressID, customerID, contactID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contact_addresses/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Address
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
