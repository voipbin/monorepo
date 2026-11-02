package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmcontactstar "monorepo/bin-contact-manager/models/contactstar"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// contactStarURI builds uri with the optional page_size and page_token query.
func contactStarURI(uri string, size uint64, token string) string {
	u := url.Values{}
	if size > 0 {
		u.Set("page_size", fmt.Sprintf("%d", size))
	}
	if token != "" {
		u.Set("page_token", token)
	}
	if len(u) > 0 {
		uri += "?" + u.Encode()
	}

	return uri
}

// contactStarTokenFormat is the page token layout of the list APIs (UTC, microseconds).
const contactStarTokenFormat = "2006-01-02T15:04:05.000000Z"

// ContactV1ContactStar stars the contact for the agent in contact-manager. It is idempotent.
func (r *requestHandler) ContactV1ContactStar(ctx context.Context, customerID, agentID, contactID uuid.UUID) (*cmcontactstar.ContactStar, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/stars/%s", contactID, agentID)

	m, err := json.Marshal(cmrequest.V1DataContactsIDStars{CustomerID: customerID})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts/<id>/stars/<agent-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontactstar.ContactStar
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactUnstar removes the agent's star on the contact in contact-manager. It is idempotent.
func (r *requestHandler) ContactV1ContactUnstar(ctx context.Context, customerID, agentID, contactID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/contacts/%s/stars/%s", contactID, agentID)

	m, err := json.Marshal(cmrequest.V1DataContactsIDStars{CustomerID: customerID})
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<id>/stars/<agent-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	// a 404 (the contact is not the customer's) must reach the caller
	return parseResponse(tmp, nil)
}

// ContactV1ContactStarredList lists the contacts the agent has starred. The order and the next
// page token follow ContactV1ContactList: the contact's tm_create of the last returned item
// (empty when the result is empty).
func (r *requestHandler) ContactV1ContactStarredList(ctx context.Context, customerID, agentID uuid.UUID, size uint64, token string) ([]cmcontact.Contact, string, error) {
	uri := contactStarURI("/v1/contacts/starred", size, token)

	m, err := json.Marshal(cmrequest.V1DataStarsGet{CustomerID: customerID, AgentID: agentID})
	if err != nil {
		return nil, "", err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts/starred", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, "", err
	}

	var res []cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, "", errParse
	}

	nextToken := ""
	if len(res) > 0 && res[len(res)-1].TMCreate != nil {
		nextToken = res[len(res)-1].TMCreate.UTC().Format(contactStarTokenFormat)
	}

	return res, nextToken, nil
}

// ContactV1ContactStarList lists the agent's star rows, newest star first. The next page token is
// the tm_create of the last returned star (empty when the result is empty).
func (r *requestHandler) ContactV1ContactStarList(ctx context.Context, customerID, agentID uuid.UUID, size uint64, token string) ([]*cmcontactstar.ContactStar, string, error) {
	uri := contactStarURI("/v1/contact_stars", size, token)

	m, err := json.Marshal(cmrequest.V1DataStarsGet{CustomerID: customerID, AgentID: agentID})
	if err != nil {
		return nil, "", err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contact_stars", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, "", err
	}

	var res []*cmcontactstar.ContactStar
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, "", errParse
	}

	nextToken := ""
	if len(res) > 0 && res[len(res)-1].TMCreate != nil {
		nextToken = res[len(res)-1].TMCreate.UTC().Format(contactStarTokenFormat)
	}

	return res, nextToken, nil
}
