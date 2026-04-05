package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetMessages(c *gin.Context, params openapi_server.GetMessagesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetMessages",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("agent", a)

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

	tmps, err := h.serviceHandler.MessageList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get messages list. err: %v", err)
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

func (h *server) PostMessages(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostMessages",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostMessagesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	source := ConvertCommonAddress(req.Source)
	destinations := []commonaddress.Address{}
	for _, v := range req.Destinations {
		destinations = append(destinations, ConvertCommonAddress(v))
	}

	res, err := h.serviceHandler.MessageSend(c.Request.Context(), a, &source, destinations, req.Text)
	if err != nil {
		log.Errorf("Could not send the message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteMessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteMessagesId",
		"request_address": c.ClientIP,
		"message_id":      id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.MessageDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete a message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetMessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetMessagesId",
		"request_address": c.ClientIP,
		"message_id":      id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.MessageGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get an message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
