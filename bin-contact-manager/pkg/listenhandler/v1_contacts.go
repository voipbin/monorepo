package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// processV1ContactsGet handles GET /v1/contacts?... request
func (h *listenHandler) processV1ContactsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1ContactsGet",
		"size":  pageSize,
		"token": pageToken,
	})
	log.WithField("request", req).Debug("Received request.")

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(req.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[contact.FieldStruct, contact.Field](contact.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.contactHandler.List(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get contacts info. err:%v", err)
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

// processV1ContactsPost handles POST /v1/contacts request
func (h *listenHandler) processV1ContactsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1ContactsPost",
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.ContactCreate
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id": reqData.CustomerID,
	})
	log.WithField("request", reqData).Debug("Creating a contact.")

	// Build contact from request
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			CustomerID: reqData.CustomerID,
		},
		FirstName:   reqData.FirstName,
		LastName:    reqData.LastName,
		DisplayName: reqData.DisplayName,
		Company:     reqData.Company,
		JobTitle:    reqData.JobTitle,
		Source:      reqData.Source,
		ExternalID:  reqData.ExternalID,
		Notes:       reqData.Notes,
		TagIDs:      reqData.TagIDs,
	}

	// Convert phone numbers
	for _, p := range reqData.PhoneNumbers {
		c.PhoneNumbers = append(c.PhoneNumbers, contact.PhoneNumber{
			Number:     p.Number,
			NumberE164: p.NumberE164,
			Type:       p.Type,
			IsPrimary:  p.IsPrimary,
		})
	}

	// Convert emails
	for _, e := range reqData.Emails {
		c.Emails = append(c.Emails, contact.Email{
			Address:   e.Address,
			Type:      e.Type,
			IsPrimary: e.IsPrimary,
		})
	}

	tmp, err := h.contactHandler.Create(ctx, c)
	if err != nil {
		log.Errorf("Could not create a contact. err: %v", err)
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

// processV1ContactsIDGet handles GET /v1/contacts/{id} request
func (h *listenHandler) processV1ContactsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsIDGet",
		"contact_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.contactHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get a contact info. err: %v", err)
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

// processV1ContactsIDPut handles PUT /v1/contacts/{id} request
func (h *listenHandler) processV1ContactsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsIDPut",
		"contact_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.ContactUpdate
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Updating the contact.")

	// Build fields map from request
	fields := make(map[contact.Field]any)
	if reqData.FirstName != nil {
		fields[contact.FieldFirstName] = *reqData.FirstName
	}
	if reqData.LastName != nil {
		fields[contact.FieldLastName] = *reqData.LastName
	}
	if reqData.DisplayName != nil {
		fields[contact.FieldDisplayName] = *reqData.DisplayName
	}
	if reqData.Company != nil {
		fields[contact.FieldCompany] = *reqData.Company
	}
	if reqData.JobTitle != nil {
		fields[contact.FieldJobTitle] = *reqData.JobTitle
	}
	if reqData.ExternalID != nil {
		fields[contact.FieldExternalID] = *reqData.ExternalID
	}
	if reqData.Notes != nil {
		fields[contact.FieldNotes] = *reqData.Notes
	}

	tmp, err := h.contactHandler.Update(ctx, id, fields)
	if err != nil {
		log.Errorf("Could not update the contact info. err: %v", err)
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

// processV1ContactsIDDelete handles DELETE /v1/contacts/{id} request
func (h *listenHandler) processV1ContactsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsIDDelete",
		"contact_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.contactHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the contact info. err: %v", err)
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

// processV1ContactsLookupGet handles GET /v1/contacts/lookup?... request
func (h *listenHandler) processV1ContactsLookupGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	customerIDStr := u.Query().Get("customer_id")
	phoneE164 := u.Query().Get("phone_e164")
	email := u.Query().Get("email")

	log := logrus.WithFields(logrus.Fields{
		"func":        "processV1ContactsLookupGet",
		"customer_id": customerIDStr,
		"phone_e164":  phoneE164,
		"email":       email,
	})
	log.WithField("request", req).Debug("Received request.")

	customerID := uuid.FromStringOrNil(customerIDStr)
	if customerID == uuid.Nil {
		log.Errorf("Invalid customer_id")
		return simpleResponse(400), nil
	}

	var tmp *contact.Contact
	if phoneE164 != "" {
		tmp, err = h.contactHandler.LookupByPhone(ctx, customerID, phoneE164)
	} else if email != "" {
		tmp, err = h.contactHandler.LookupByEmail(ctx, customerID, email)
	} else {
		log.Errorf("Either phone_e164 or email must be provided")
		return simpleResponse(400), nil
	}

	if err != nil {
		log.Errorf("Could not lookup contact. err: %v", err)
		return simpleResponse(404), nil
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

// processV1ContactsPhoneNumbersPost handles POST /v1/contacts/{id}/phone-numbers request
func (h *listenHandler) processV1ContactsPhoneNumbersPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsPhoneNumbersPost",
		"contact_id": contactID,
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.PhoneNumberCreate
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Adding phone number.")

	phoneNumber := &contact.PhoneNumber{
		Number:     reqData.Number,
		NumberE164: reqData.NumberE164,
		Type:       reqData.Type,
		IsPrimary:  reqData.IsPrimary,
	}

	tmp, err := h.contactHandler.AddPhoneNumber(ctx, contactID, phoneNumber)
	if err != nil {
		log.Errorf("Could not add phone number. err: %v", err)
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

// processV1ContactsPhoneNumbersIDDelete handles DELETE /v1/contacts/{id}/phone-numbers/{phone_id} request
func (h *listenHandler) processV1ContactsPhoneNumbersIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	phoneID := uuid.FromStringOrNil(uriItems[5])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsPhoneNumbersIDDelete",
		"contact_id": contactID,
		"phone_id":   phoneID,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.contactHandler.RemovePhoneNumber(ctx, contactID, phoneID)
	if err != nil {
		log.Errorf("Could not remove phone number. err: %v", err)
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

// processV1ContactsEmailsPost handles POST /v1/contacts/{id}/emails request
func (h *listenHandler) processV1ContactsEmailsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsEmailsPost",
		"contact_id": contactID,
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.EmailCreate
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Adding email.")

	email := &contact.Email{
		Address:   reqData.Address,
		Type:      reqData.Type,
		IsPrimary: reqData.IsPrimary,
	}

	tmp, err := h.contactHandler.AddEmail(ctx, contactID, email)
	if err != nil {
		log.Errorf("Could not add email. err: %v", err)
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

// processV1ContactsEmailsIDDelete handles DELETE /v1/contacts/{id}/emails/{email_id} request
func (h *listenHandler) processV1ContactsEmailsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	emailID := uuid.FromStringOrNil(uriItems[5])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsEmailsIDDelete",
		"contact_id": contactID,
		"email_id":   emailID,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.contactHandler.RemoveEmail(ctx, contactID, emailID)
	if err != nil {
		log.Errorf("Could not remove email. err: %v", err)
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

// processV1ContactsTagsPost handles POST /v1/contacts/{id}/tags request
func (h *listenHandler) processV1ContactsTagsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsTagsPost",
		"contact_id": contactID,
	})
	log.WithField("request", m).Debug("Received request.")

	var reqData request.TagAssignment
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", reqData).Debug("Adding tag.")

	tmp, err := h.contactHandler.AddTag(ctx, contactID, reqData.TagID)
	if err != nil {
		log.Errorf("Could not add tag. err: %v", err)
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

// processV1ContactsTagsIDDelete handles DELETE /v1/contacts/{id}/tags/{tag_id} request
func (h *listenHandler) processV1ContactsTagsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}

	contactID := uuid.FromStringOrNil(uriItems[3])
	tagID := uuid.FromStringOrNil(uriItems[5])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1ContactsTagsIDDelete",
		"contact_id": contactID,
		"tag_id":     tagID,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.contactHandler.RemoveTag(ctx, contactID, tagID)
	if err != nil {
		log.Errorf("Could not remove tag. err: %v", err)
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
