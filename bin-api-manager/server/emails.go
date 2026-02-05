package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostEmails(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostEmails",
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

	var req openapi_server.PostEmailsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	destinations := []commonaddress.Address{}
	for _, v := range req.Destinations {
		destinations = append(destinations, ConvertCommonAddress(v))
	}

	attachments := []ememail.Attachment{}
	for _, v := range req.Attachments {
		attachments = append(attachments, ConvertEmailMamagerEmailAttachment(v))
	}

	res, err := h.serviceHandler.EmailSend(c.Request.Context(), &a, destinations, req.Subject, req.Content, attachments)
	if err != nil {
		log.Errorf("Could not send an email. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetEmails(c *gin.Context, params openapi_server.GetEmailsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetEmails",
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

	tmps, err := h.serviceHandler.EmailList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get email list. err: %v", err)
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

func (h *server) GetEmailsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetEmailsId",
		"request_address": c.ClientIP,
		"target_id":       id,
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

	res, err := h.serviceHandler.EmailGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get email. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteEmailsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteEmailsId",
		"request_address": c.ClientIP,
		"target_id":       id,
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

	res, err := h.serviceHandler.EmailDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the item. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
