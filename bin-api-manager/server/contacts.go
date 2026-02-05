package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetContacts(c *gin.Context, params openapi_server.GetContactsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContacts",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent":    a,
		"username": a.Username,
	})

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}

	tmps, err := h.serviceHandler.ContactList(c.Request.Context(), &a, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get contacts info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil { nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z") }
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostContacts(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContacts",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req openapi_server.PostContactsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	firstName := ""
	if req.FirstName != nil {
		firstName = *req.FirstName
	}

	lastName := ""
	if req.LastName != nil {
		lastName = *req.LastName
	}

	displayName := ""
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}

	company := ""
	if req.Company != nil {
		company = *req.Company
	}

	jobTitle := ""
	if req.JobTitle != nil {
		jobTitle = *req.JobTitle
	}

	source := ""
	if req.Source != nil {
		source = string(*req.Source)
	}

	externalID := ""
	if req.ExternalId != nil {
		externalID = *req.ExternalId
	}

	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	phoneNumbers := []cmrequest.PhoneNumberCreate{}
	if req.PhoneNumbers != nil {
		for _, v := range *req.PhoneNumbers {
			pn := cmrequest.PhoneNumberCreate{}
			if v.Number != nil {
				pn.Number = *v.Number
			}
			if v.Type != nil {
				pn.Type = string(*v.Type)
			}
			if v.IsPrimary != nil {
				pn.IsPrimary = *v.IsPrimary
			}
			phoneNumbers = append(phoneNumbers, pn)
		}
	}

	emails := []cmrequest.EmailCreate{}
	if req.Emails != nil {
		for _, v := range *req.Emails {
			e := cmrequest.EmailCreate{}
			if v.Address != nil {
				e.Address = string(*v.Address)
			}
			if v.Type != nil {
				e.Type = string(*v.Type)
			}
			if v.IsPrimary != nil {
				e.IsPrimary = *v.IsPrimary
			}
			emails = append(emails, e)
		}
	}

	tagIDs := []uuid.UUID{}
	if req.TagIds != nil {
		for _, v := range *req.TagIds {
			tagIDs = append(tagIDs, uuid.UUID(v))
		}
	}

	res, err := h.serviceHandler.ContactCreate(
		c.Request.Context(),
		&a,
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
		c.AbortWithStatus(400)
		return
	}

	c.JSON(201, res)
}

func (h *server) GetContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent":    a,
		"username": a.Username,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the contact info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutContactsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent":    a,
		"username": a.Username,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutContactsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactUpdate(
		c.Request.Context(),
		&a,
		target,
		req.FirstName,
		req.LastName,
		req.DisplayName,
		req.Company,
		req.JobTitle,
		req.ExternalId,
		req.Notes,
	)
	if err != nil {
		log.Errorf("Could not update the contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent":    a,
		"username": a.Username,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not delete the contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetContactsLookup(c *gin.Context, params openapi_server.GetContactsLookupParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactsLookup",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent":    a,
		"username": a.Username,
	})

	phoneE164 := ""
	if params.Phone != nil {
		phoneE164 = *params.Phone
	}

	email := ""
	if params.Email != nil {
		email = *params.Email
	}

	if phoneE164 == "" && email == "" {
		log.Error("At least one of phone or email must be provided.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactLookup(c.Request.Context(), &a, phoneE164, email)
	if err != nil {
		log.Infof("Could not lookup the contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdPhoneNumbers(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactsIdPhoneNumbers",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostContactsIdPhoneNumbersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	phoneType := ""
	if req.Type != nil {
		phoneType = string(*req.Type)
	}

	isPrimary := false
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	res, err := h.serviceHandler.ContactPhoneNumberCreate(c.Request.Context(), &a, target, req.Number, "", phoneType, isPrimary)
	if err != nil {
		log.Errorf("Could not add phone number to contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(201, res)
}

func (h *server) DeleteContactsIdPhoneNumbersPhoneNumberId(c *gin.Context, id string, phoneNumberId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsIdPhoneNumbersPhoneNumberId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		c.AbortWithStatus(400)
		return
	}

	phoneNumID := uuid.FromStringOrNil(phoneNumberId)
	if phoneNumID == uuid.Nil {
		log.Error("Could not parse the phone number id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactPhoneNumberDelete(c.Request.Context(), &a, contactID, phoneNumID)
	if err != nil {
		log.Infof("Could not delete phone number from contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdEmails(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactsIdEmails",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostContactsIdEmailsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	emailType := ""
	if req.Type != nil {
		emailType = string(*req.Type)
	}

	isPrimary := false
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	res, err := h.serviceHandler.ContactEmailCreate(c.Request.Context(), &a, target, string(req.Address), emailType, isPrimary)
	if err != nil {
		log.Errorf("Could not add email to contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(201, res)
}

func (h *server) DeleteContactsIdEmailsEmailId(c *gin.Context, id string, emailId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsIdEmailsEmailId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		c.AbortWithStatus(400)
		return
	}

	emlID := uuid.FromStringOrNil(emailId)
	if emlID == uuid.Nil {
		log.Error("Could not parse the email id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactEmailDelete(c.Request.Context(), &a, contactID, emlID)
	if err != nil {
		log.Infof("Could not delete email from contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdTags(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactsIdTags",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostContactsIdTagsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	tagID := uuid.UUID(req.TagId)
	if tagID == uuid.Nil {
		log.Error("Could not parse the tag id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactTagAdd(c.Request.Context(), &a, contactID, tagID)
	if err != nil {
		log.Errorf("Could not add tag to contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(201, res)
}

func (h *server) DeleteContactsIdTagsTagId(c *gin.Context, id string, tagId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsIdTagsTagId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		c.AbortWithStatus(400)
		return
	}

	tID := uuid.FromStringOrNil(tagId)
	if tID == uuid.Nil {
		log.Error("Could not parse the tag id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactTagRemove(c.Request.Context(), &a, contactID, tID)
	if err != nil {
		log.Infof("Could not remove tag from contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
