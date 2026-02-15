package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	bmaccount "monorepo/bin-billing-manager/models/account"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetBillingAccountsIdAllowances(c *gin.Context, id string, params openapi_server.GetBillingAccountsIdAllowancesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingAccountsIdAllowances",
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

	pageSize := uint64(10)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.BillingAccountAllowancesGet(c.Request.Context(), &a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get allowances. err: %v", err)
		c.AbortWithStatus(400)
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

func (h *server) GetBillingAccountsIdAllowance(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingAccountsIdAllowance",
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

	res, err := h.serviceHandler.BillingAccountAllowanceGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get current allowance. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetBillingAccountsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingAccountsId",
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

	res, err := h.serviceHandler.BillingAccountGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a billing account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutBillingAccountsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutBillingAccountsId",
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

	var req openapi_server.PutBillingAccountsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}

	res, err := h.serviceHandler.BillingAccountUpdateBasicInfo(c.Request.Context(), &a, target, name, detail)
	if err != nil {
		log.Errorf("Could not update. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutBillingAccountsIdPaymentInfo(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutBillingAccountsIdPaymentInfo",
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

	var req openapi_server.PutBillingAccountsIdPaymentInfoJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	paymentType := bmaccount.PaymentTypeNone
	if req.PaymentType != nil {
		paymentType = bmaccount.PaymentType(*req.PaymentType)
	}

	paymentMethod := bmaccount.PaymentMethodNone
	if req.PaymentMethod != nil {
		paymentMethod = bmaccount.PaymentMethod(*req.PaymentMethod)
	}

	res, err := h.serviceHandler.BillingAccountUpdatePaymentInfo(c.Request.Context(), &a, target, paymentType, paymentMethod)
	if err != nil {
		log.Errorf("Could not update. info err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostBillingAccountsIdBalanceAddForce(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostBillingAccountsIdBalanceAddForce",
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

	var req openapi_server.PostBillingAccountsIdBalanceAddForceJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	balance := float32(0)
	if req.Balance != nil {
		balance = *req.Balance
	}
	if balance < 0 {
		log.Error("Invalid balance.")
		c.AbortWithStatus(400)
	}

	res, err := h.serviceHandler.BillingAccountAddBalanceForce(c.Request.Context(), &a, target, balance)
	if err != nil {
		log.Errorf("Could not add the balance to the billing account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostBillingAccountsIdBalanceSubtractForce(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostBillingAccountsIdBalanceSubtractForce",
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

	var req openapi_server.PostBillingAccountsIdBalanceSubtractForceJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	balance := float32(0)
	if req.Balance != nil {
		balance = *req.Balance
	}
	if balance < 0 {
		log.Error("Invalid balance.")
		c.AbortWithStatus(400)
	}

	res, err := h.serviceHandler.BillingAccountSubtractBalanceForce(c.Request.Context(), &a, target, balance)
	if err != nil {
		log.Errorf("Could not subtract the balance from the billing account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
