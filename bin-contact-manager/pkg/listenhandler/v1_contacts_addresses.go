package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// processV1ContactsAddressesGet handles GET /v1/contacts/{id}/addresses request
// Returns {"result": [...addresses...]}
func (h *listenHandler) processV1ContactsAddressesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsAddressesGet",
		"contact_id": contactID,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.contactHandler.Get(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	addresses := tmp.Addresses
	if addresses == nil {
		addresses = []contact.Address{}
	}

	type addressesResponse struct {
		Result []contact.Address `json:"result"`
	}

	data, err := json.Marshal(addressesResponse{Result: addresses})
	if err != nil {
		log.Debugf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ContactsAddressesPost handles POST /v1/contacts/{id}/addresses request
// Body: AddressCreate{Type, Target, IsPrimary}
// Returns updated Contact
func (h *listenHandler) processV1ContactsAddressesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsAddressesPost",
		"contact_id": contactID,
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.AddressCreate
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Adding address.")

	address := &contact.Address{
		Address: commonaddress.Address{
			Type:   commonaddress.Type(reqData.Type),
			Target: reqData.Target,
			Name:   reqData.Name,
			Detail: reqData.Detail,
		},
		IsPrimary: reqData.IsPrimary,
	}

	tmp, err := h.contactHandler.AddAddress(ctx, contactID, address)
	if err != nil {
		log.Errorf("Could not add address. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 201,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ContactsAddressesIDPut handles PUT /v1/contacts/{id}/addresses/{address_id} request
// Body: AddressUpdate{Target*, IsPrimary*}
// Returns updated Contact
func (h *listenHandler) processV1ContactsAddressesIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	addressID := uuid.FromStringOrNil(uriItems[5])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsAddressesIDPut",
		"contact_id": contactID,
		"address_id": addressID,
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.AddressUpdate
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Updating address.")

	// Build fields map from request
	fields := make(map[string]any)
	if reqData.Target != nil {
		fields["target"] = *reqData.Target
	}
	if reqData.Name != nil {
		fields["name"] = *reqData.Name
	}
	if reqData.Detail != nil {
		fields["detail"] = *reqData.Detail
	}
	if reqData.IsPrimary != nil {
		fields["is_primary"] = *reqData.IsPrimary
	}

	tmp, err := h.contactHandler.UpdateAddress(ctx, contactID, addressID, fields)
	if err != nil {
		log.Errorf("Could not update address. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ContactsAddressesIDDelete handles DELETE /v1/contacts/{id}/addresses/{address_id} request
// Returns updated Contact
func (h *listenHandler) processV1ContactsAddressesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	addressID := uuid.FromStringOrNil(uriItems[5])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsAddressesIDDelete",
		"contact_id": contactID,
		"address_id": addressID,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.contactHandler.RemoveAddress(ctx, contactID, addressID)
	if err != nil {
		log.Errorf("Could not remove address. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
