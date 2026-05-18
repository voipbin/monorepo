package service

import (
	"errors"
	"net/http"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// RequestBodyDelegatePOST is the request body for POST /auth/delegate.
type RequestBodyDelegatePOST struct {
	CustomerID string `json:"customer_id" binding:"required"`
	Reason     string `json:"reason"      binding:"required"`
}

// PostDelegate handles POST /auth/delegate.
// Issues a short-lived delegate JWT granting customer-admin access to a specific customer.
// Requires PermissionProjectSuperAdmin.
func PostDelegate(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostDelegate",
		"request_address": c.ClientIP(),
	})

	var req RequestBodyDelegatePOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind request body. err: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	targetCustomerID, err := uuid.FromString(req.CustomerID)
	if err != nil {
		log.Warnf("Invalid customer_id. err: %v", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

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

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := sh.AuthDelegate(c.Request.Context(), identity, targetCustomerID, req.Reason)
	if err != nil {
		log.Infof("AuthDelegate failed. err: %v", err)
		switch {
		case errors.Is(err, serviceerrors.ErrPermissionDenied):
			c.AbortWithStatus(http.StatusForbidden)
		case errors.Is(err, serviceerrors.ErrNotFound):
			c.AbortWithStatus(http.StatusNotFound)
		case errors.Is(err, serviceerrors.ErrInvalidArgument):
			c.AbortWithStatus(http.StatusUnprocessableEntity)
		default:
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	log.Infof("Delegate token issued. customer_id: %s", targetCustomerID)
	c.JSON(http.StatusOK, res)
}
