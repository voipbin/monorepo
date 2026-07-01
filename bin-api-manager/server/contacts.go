package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (h *server) GetContacts(c *gin.Context, params openapi_server.GetContactsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContacts",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth":     a,
		"username": a.AgentUsername(),
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

	tmps, err := h.serviceHandler.ContactList(c.Request.Context(), a, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get contacts info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostContacts(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContacts",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	var req openapi_server.PostContactsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
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

	addresses := []cmrequest.AddressCreate{}
	if req.Addresses != nil {
		for _, v := range *req.Addresses {
			if v.Type == nil || *v.Type == "" {
				log.Error("Missing required field: type in addresses")
				abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ADDRESS_TYPE", "Each address must have a type."))
				return
			}
			addrType := string(*v.Type)
			if addrType != "tel" && addrType != "email" {
				log.Errorf("Invalid address type: %s", addrType)
				abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ADDRESS_TYPE", "Address type must be 'tel' or 'email'."))
				return
			}
			if v.Target == nil || *v.Target == "" {
				log.Error("Missing required field: target in addresses")
				abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ADDRESS_TARGET", "Each address must have a target."))
				return
			}
			addr := cmrequest.AddressCreate{
				Type:   addrType,
				Target: *v.Target,
			}
			if v.IsPrimary != nil {
				addr.IsPrimary = *v.IsPrimary
			}
			if v.Name != nil {
				addr.Name = *v.Name
			}
			if v.Detail != nil {
				addr.Detail = *v.Detail
			}
			addresses = append(addresses, addr)
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
		a,
		firstName,
		lastName,
		displayName,
		company,
		jobTitle,
		source,
		externalID,
		notes,
		addresses,
		tagIDs,
	)
	if err != nil {
		log.Errorf("Could not create a contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(201, res)
}

func (h *server) GetContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactsId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth":     a,
		"username": a.AgentUsername(),
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ContactGet(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not get the contact info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutContactsId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth":     a,
		"username": a.AgentUsername(),
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutContactsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ContactUpdate(
		c.Request.Context(),
		a,
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
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth":     a,
		"username": a.AgentUsername(),
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ContactDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not delete the contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetContactsLookup(c *gin.Context, params openapi_server.GetContactsLookupParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactsLookup",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth":     a,
		"username": a.AgentUsername(),
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
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "At least one of phone or email must be provided."))
		return
	}

	res, err := h.serviceHandler.ContactLookup(c.Request.Context(), a, phoneE164, email)
	if err != nil {
		log.Infof("Could not lookup the contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdAddresses(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactsIdAddresses",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.UUID(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PostContactsIdAddressesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	isPrimary := false
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}

	res, err := h.serviceHandler.ContactAddressCreate(c.Request.Context(), a, target, string(req.Type), req.Target, isPrimary, name, detail)
	if err != nil {
		log.Errorf("Could not add address to contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(201, res)
}

func (h *server) PutContactsIdAddressesAddressId(c *gin.Context, id openapi_types.UUID, addressId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutContactsIdAddressesAddressId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	contactID := uuid.UUID(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	addrID := uuid.UUID(addressId)
	if addrID == uuid.Nil {
		log.Error("Could not parse the address id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided address_id is not a valid UUID."))
		return
	}

	var req openapi_server.PutContactsIdAddressesAddressIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	// Build fields map from request
	fields := make(map[string]any)
	if req.Target != nil {
		fields["target"] = *req.Target
	}
	if req.IsPrimary != nil {
		fields["is_primary"] = *req.IsPrimary
	}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Detail != nil {
		fields["detail"] = *req.Detail
	}

	res, err := h.serviceHandler.ContactAddressUpdate(c.Request.Context(), a, contactID, addrID, fields)
	if err != nil {
		log.Errorf("Could not update address on contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteContactsIdAddressesAddressId(c *gin.Context, id openapi_types.UUID, addressId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsIdAddressesAddressId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	contactID := uuid.UUID(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	addrID := uuid.UUID(addressId)
	if addrID == uuid.Nil {
		log.Error("Could not parse the address id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided address_id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ContactAddressDelete(c.Request.Context(), a, contactID, addrID)
	if err != nil {
		log.Infof("Could not delete address from contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdTags(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactsIdTags",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PostContactsIdTagsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	tagID := uuid.UUID(req.TagId)
	if tagID == uuid.Nil {
		log.Error("Could not parse the tag id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided tag_id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ContactTagAdd(c.Request.Context(), a, contactID, tagID)
	if err != nil {
		log.Errorf("Could not add tag to contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(201, res)
}

func (h *server) DeleteContactsIdTagsTagId(c *gin.Context, id string, tagId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsIdTagsTagId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Could not parse the contact id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	tID := uuid.FromStringOrNil(tagId)
	if tID == uuid.Nil {
		log.Error("Could not parse the tag id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided tag_id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ContactTagRemove(c.Request.Context(), a, contactID, tID)
	if err != nil {
		log.Infof("Could not remove tag from contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
