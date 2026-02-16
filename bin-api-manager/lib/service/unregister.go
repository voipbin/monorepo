package service

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestBodyUnregisterPOST is request body for POST /auth/unregister
type RequestBodyUnregisterPOST struct {
	Password           string `json:"password"`
	ConfirmationPhrase string `json:"confirmation_phrase"`
}

// PostAuthUnregister handles POST /auth/unregister request.
// Schedules account deletion (freeze) with password or confirmation validation.
func PostAuthUnregister(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthUnregister",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

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
		if _, err := serviceHandler.AuthLogin(c.Request.Context(), a.Username, req.Password); err != nil {
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

	res, err := serviceHandler.CustomerSelfFreeze(c.Request.Context(), &a)
	if err != nil {
		log.Errorf("Could not freeze the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
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

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithField("agent", a)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := serviceHandler.CustomerSelfRecover(c.Request.Context(), &a)
	if err != nil {
		log.Errorf("Could not recover the customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
