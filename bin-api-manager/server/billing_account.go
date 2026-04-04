package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	bmaccount "monorepo/bin-billing-manager/models/account"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetBillingAccount(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBillingAccount",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("auth", a)

	res, err := h.serviceHandler.BillingAccountSelfGet(c.Request.Context(), a)
	if err != nil {
		log.Infof("Could not get the billing account info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutBillingAccount(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutBillingAccount",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("auth", a)

	var req openapi_server.PutBillingAccountJSONBody
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

	res, err := h.serviceHandler.BillingAccountSelfUpdateBasicInfo(c.Request.Context(), a, name, detail)
	if err != nil {
		log.Errorf("Could not update. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutBillingAccountPaymentInfo(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutBillingAccountPaymentInfo",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("auth", a)

	var req openapi_server.PutBillingAccountPaymentInfoJSONBody
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

	res, err := h.serviceHandler.BillingAccountSelfUpdatePaymentInfo(c.Request.Context(), a, paymentType, paymentMethod)
	if err != nil {
		log.Errorf("Could not update payment info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostBillingAccountPaddlePortalSession(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostBillingAccountPaddlePortalSession",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("auth", a)

	url, err := h.serviceHandler.BillingAccountSelfCreatePaddlePortalSession(c.Request.Context(), a)
	if err != nil {
		log.Infof("Could not create portal session. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, gin.H{"url": url})
}
