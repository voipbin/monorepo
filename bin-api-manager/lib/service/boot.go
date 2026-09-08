package service

import (
	stderrors "errors"
	"net/http"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestBodyBootPOST is request body for POST /boot
type RequestBodyBootPOST struct {
	DirectHash string `json:"direct_hash" binding:"required"`
}

// PostBoot handles POST /boot request.
// It resolves a direct hash and returns a resource-scoped JWT.
func PostBoot(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostBoot",
		"request_address": c.ClientIP(),
	})

	var req RequestBodyBootPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	directHashLog := req.DirectHash
	if len(directHashLog) > 12 {
		directHashLog = directHashLog[:12] + "..."
	}
	log = log.WithFields(logrus.Fields{
		"direct_hash": directHashLog,
	})
	log.Debugf("Processing boot request.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.AuthBoot(c.Request.Context(), req.DirectHash)
	if err != nil {
		log.Infof("Boot failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debug("Boot successful.")
	c.JSON(200, res)
}

// PostBootRefresh handles POST /auth/boot/refresh request.
// It reissues a direct token that keeps the same allowed resource, so a live
// conversation survives the boot token's expiry.
//
// Unlike PostBoot this does not collapse every failure into 400. The client
// distinguishes them: 403 means "wrong identity type, stop", 401 means "this
// token can no longer be refreshed, boot again". Collapsing them would leave
// the widget with no way to tell a permanent failure from a recoverable one.
func PostBootRefresh(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostBootRefresh",
		"request_address": c.ClientIP(),
	})

	// Identity extraction mirrors auth_delegate.go: never c.MustGet, which
	// would turn a missing or mistyped identity into a panic.
	tmp, exists := c.Get("auth_identity")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	identity, ok := tmp.(*auth.AuthIdentity)
	if !ok || identity == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.AuthBootRefresh(c.Request.Context(), identity)
	if err != nil {
		log.Infof("Boot refresh failed. err: %v", err)
		if stderrors.Is(err, serviceerrors.ErrPermissionDenied) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	log.Debug("Boot refresh successful.")
	c.JSON(http.StatusOK, res)
}
