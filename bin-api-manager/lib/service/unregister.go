package service

import (
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestBodyUnregisterPOST is request body for POST /auth/unregister
type RequestBodyUnregisterPOST struct {
	Password           string `json:"password"`
	ConfirmationPhrase string `json:"confirmation_phrase"`
	Immediate          bool   `json:"immediate"`
}

// PostAuthUnregister handles POST /auth/unregister request.
// Schedules account deletion (freeze) with password or confirmation validation.
func PostAuthUnregister(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthUnregister",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("auth_identity")
	if !exists {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	a, ok := tmp.(*auth.AuthIdentity)
	if !ok {
		log.Errorf("Could not assert auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("auth_identity", a)

	var req RequestBodyUnregisterPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// Validate: exactly one of password or confirmation_phrase must be provided
	hasPassword := req.Password != ""
	hasConfirmation := req.ConfirmationPhrase != ""
	if hasPassword == hasConfirmation {
		log.Warnf("Exactly one of password or confirmation_phrase must be provided.")
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// Validate credentials
	if hasPassword {
		// Validate password by attempting login
		if _, err := serviceHandler.AuthLogin(c.Request.Context(), a.AgentUsername(), req.Password); err != nil {
			log.Debugf("Password validation failed. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	} else {
		// Validate confirmation phrase
		if req.ConfirmationPhrase != "DELETE" {
			log.Warnf("Invalid confirmation phrase.")
			c.AbortWithStatus(400)
			return
		}
	}

	var (
		res *cscustomer.WebhookMessage
		err error
	)
	if req.Immediate {
		res, err = serviceHandler.CustomerSelfFreezeAndDelete(c.Request.Context(), a)
		if err != nil {
			log.Errorf("Could not freeze and delete the customer. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	} else {
		res, err = serviceHandler.CustomerSelfFreeze(c.Request.Context(), a)
		if err != nil {
			log.Errorf("Could not freeze the customer. err: %v", err)
			c.AbortWithStatus(400)
			return
		}
	}

	c.JSON(200, res)
}

// DeleteAuthUnregister handles DELETE /auth/unregister request.
// Cancels account deletion (recovery).
func DeleteAuthUnregister(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteAuthUnregister",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("auth_identity")
	if !exists {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	a, ok := tmp.(*auth.AuthIdentity)
	if !ok {
		log.Errorf("Could not assert auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("auth_identity", a)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := serviceHandler.CustomerSelfRecover(c.Request.Context(), a)
	if err != nil {
		log.Errorf("Could not recover the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
