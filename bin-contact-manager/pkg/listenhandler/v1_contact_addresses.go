package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// processV1ContactAddressesGet handles GET /v1/contact_addresses?... request
// Supports optional filters: customer_id (required for scoping), contact_id, type
// Returns {"result": [...addresses...], "next_page_token": "..."}
func (h *listenHandler) processV1ContactAddressesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1ContactAddressesGet")
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse URI. err: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))
	if customerID == uuid.Nil {
		log.Error("Missing or invalid customer_id.")
		return simpleResponse(400), nil
	}

	filters := map[string]any{}
	if v := u.Query().Get("contact_id"); v != "" {
		cid := uuid.FromStringOrNil(v)
		if cid != uuid.Nil {
			filters["contact_id"] = cid
		}
	}
	if v := u.Query().Get("type"); v != "" {
		filters["type"] = v
	}

	pageToken := u.Query().Get("page_token")
	pageSize := uint64(20)

	tmp, err := h.addressHandler.ListAddresses(ctx, customerID, filters, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not list addresses. err: %v", err)
		return simpleResponse(500), nil
	}

	type listResponse struct {
		Result []contact.Address `json:"result"`
	}

	data, err := json.Marshal(listResponse{Result: tmp})
	if err != nil {
		log.Debugf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1ContactAddressesPost handles POST /v1/contact_addresses request
// Body: {contact_id, type, target, is_primary}
// Returns the created address (201)
func (h *listenHandler) processV1ContactAddressesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1ContactAddressesPost")
	log.WithField("request", m).Debug("Received request.")

	var reqData request.ContactAddressCreate
	if err := json.Unmarshal(m.Data, &reqData); err != nil {
		log.Errorf("Could not unmarshal the request. err: %v", err)
		return simpleResponse(400), nil
	}

	if reqData.ContactID == uuid.Nil {
		log.Error("Missing contact_id.")
		return simpleResponse(400), nil
	}

	tmp, err := h.contactHandler.AddAddress(ctx, reqData.ContactID, &contact.Address{
		Type:      reqData.Type,
		Target:    reqData.Target,
		IsPrimary: reqData.IsPrimary,
	})
	if err != nil {
		log.Errorf("Could not add address. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 201,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1ContactAddressesIDGet handles GET /v1/contact_addresses/{id} request
func (h *listenHandler) processV1ContactAddressesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactAddressesIDGet",
		"address_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	// customer_id is passed via query param for tenant scoping
	u, _ := url.Parse(m.URI)
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))
	if customerID == uuid.Nil {
		log.Error("Missing or invalid customer_id.")
		return simpleResponse(400), nil
	}

	tmp, err := h.addressHandler.GetAddress(ctx, customerID, id)
	if err != nil {
		log.Errorf("Could not get address. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1ContactAddressesIDPut handles PUT /v1/contact_addresses/{id} request
// Body: {target?, is_primary?}
func (h *listenHandler) processV1ContactAddressesIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactAddressesIDPut",
		"address_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.AddressUpdate
	if err := json.Unmarshal(m.Data, &reqData); err != nil {
		log.Errorf("Could not unmarshal the request. err: %v", err)
		return simpleResponse(400), nil
	}

	// Build contact_id from URI query (needed for UpdateAddress)
	u, _ := url.Parse(m.URI)
	contactID := uuid.FromStringOrNil(u.Query().Get("contact_id"))

	fields := map[string]any{}
	if reqData.Target != nil {
		fields["target"] = *reqData.Target
	}
	if reqData.IsPrimary != nil {
		fields["is_primary"] = *reqData.IsPrimary
	}

	tmp, err := h.contactHandler.UpdateAddress(ctx, contactID, id, fields)
	if err != nil {
		log.Errorf("Could not update address. err: %v", err)
		return simpleResponse(500), nil
	}

	// Return updated address
	var updated *contact.Address
	for i := range tmp.Addresses {
		if tmp.Addresses[i].ID == id {
			updated = &tmp.Addresses[i]
			break
		}
	}
	if updated == nil {
		// fetch directly via contactHandler
		u2, _ := url.Parse(m.URI)
		customerID := uuid.FromStringOrNil(u2.Query().Get("customer_id"))
		updated, err = h.addressHandler.GetAddress(ctx, customerID, id)
		if err != nil {
			return simpleResponse(500), nil
		}
	}

	data, err := json.Marshal(updated)
	if err != nil {
		log.Debugf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1ContactAddressesIDDelete handles DELETE /v1/contact_addresses/{id} request
func (h *listenHandler) processV1ContactAddressesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactAddressesIDDelete",
		"address_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	u, _ := url.Parse(m.URI)
	contactID := uuid.FromStringOrNil(u.Query().Get("contact_id"))

	tmp, err := h.contactHandler.RemoveAddress(ctx, contactID, id)
	if err != nil {
		log.Errorf("Could not delete address. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
