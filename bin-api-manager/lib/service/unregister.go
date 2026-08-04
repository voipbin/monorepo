package service

import (
	"errors"
	"net/http"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonrequesthandler "monorepo/bin-common-handler/pkg/requesthandler"
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

// abortWithMappedStatus maps a servicehandler/downstream error to an HTTP
// status and aborts the request. It follows the same idiom as PostDelegate's
// inline switch (auth_delegate.go) -- errors.Is against known sentinels,
// c.AbortWithStatus with no body -- rather than sharing its exact case list;
// the two functions' downstream calls surface different error shapes, so the
// case lists differ. See the reachability note below before assuming every
// branch fires in practice.
//
// Reachability, as of this writing: only two cases are actually reachable
// through this file's four call sites (AuthLogin; CustomerSelfFreeze;
// CustomerSelfFreezeAndDelete; CustomerSelfRecover) --
// serviceerrors.ErrPermissionDenied and serviceerrors.ErrDirectAccessNotSupported,
// which CustomerSelfFreeze/FreezeAndDelete/Recover construct and return from
// a local permission check before any RPC call. Every other branch
// (ErrAuthenticationRequired, ErrNotFound, ErrInvalidArgument, and all three
// commonrequesthandler.* cases) is currently unreachable from these four call
// sites: AuthLogin's RPC (bin-agent-manager's processV1Login) and
// CustomerSelfFreeze*/Recover's RPCs (bin-customer-manager's
// processV1CustomersIDFreezePost/FreezeAndDeletePost/RecoverPost, and the
// processV1CustomersIDGet they call internally) all collapse every failure --
// wrong password, not-found, internal error, everything -- to a bare HTTP 400
// on the wire, which surfaces here as commonrequesthandler.ErrBadRequest,
// mapped below to 400. That collapse is a separate, pre-existing gap in
// those other services, out of scope for this fix.
//
// The unreachable branches are kept anyway so this function is already
// correct once that upstream gap is fixed, and ErrBadRequest maps to 400
// (not 422) to honestly reflect "this RPC failed for an unspecified reason"
// rather than reinterpreting a generic upstream failure as something more
// specific than it is.
func abortWithMappedStatus(c *gin.Context, err error) {
	switch {
	case errors.Is(err, serviceerrors.ErrAuthenticationRequired),
		errors.Is(err, commonrequesthandler.ErrUnauthorized):
		c.AbortWithStatus(http.StatusUnauthorized)
	case errors.Is(err, serviceerrors.ErrPermissionDenied),
		errors.Is(err, serviceerrors.ErrDirectAccessNotSupported),
		errors.Is(err, commonrequesthandler.ErrForbidden):
		c.AbortWithStatus(http.StatusForbidden)
	case errors.Is(err, serviceerrors.ErrNotFound),
		errors.Is(err, commonrequesthandler.ErrNotFound):
		c.AbortWithStatus(http.StatusNotFound)
	case errors.Is(err, serviceerrors.ErrInvalidArgument):
		c.AbortWithStatus(http.StatusUnprocessableEntity)
	case errors.Is(err, commonrequesthandler.ErrBadRequest):
		c.AbortWithStatus(http.StatusBadRequest)
	default:
		c.AbortWithStatus(http.StatusInternalServerError)
	}
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
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	a, ok := tmp.(*auth.AuthIdentity)
	if !ok {
		log.Errorf("Could not assert auth identity.")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	log = log.WithField("auth_identity", a)

	var req RequestBodyUnregisterPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Validate: exactly one of password or confirmation_phrase must be provided
	hasPassword := req.Password != ""
	hasConfirmation := req.ConfirmationPhrase != ""
	if hasPassword == hasConfirmation {
		log.Warnf("Exactly one of password or confirmation_phrase must be provided.")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// Validate credentials
	if hasPassword {
		// Validate password by attempting login
		if _, err := serviceHandler.AuthLogin(c.Request.Context(), a.AgentUsername(), req.Password); err != nil {
			log.Debugf("Password validation failed. err: %v", err)
			abortWithMappedStatus(c, err)
			return
		}
	} else {
		// Validate confirmation phrase
		if req.ConfirmationPhrase != "DELETE" {
			log.Warnf("Invalid confirmation phrase.")
			c.AbortWithStatus(http.StatusBadRequest)
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
			abortWithMappedStatus(c, err)
			return
		}
	} else {
		res, err = serviceHandler.CustomerSelfFreeze(c.Request.Context(), a)
		if err != nil {
			log.Errorf("Could not freeze the customer. err: %v", err)
			abortWithMappedStatus(c, err)
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
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	a, ok := tmp.(*auth.AuthIdentity)
	if !ok {
		log.Errorf("Could not assert auth identity.")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	log = log.WithField("auth_identity", a)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := serviceHandler.CustomerSelfRecover(c.Request.Context(), a)
	if err != nil {
		log.Errorf("Could not recover the customer. err: %v", err)
		abortWithMappedStatus(c, err)
		return
	}

	c.JSON(200, res)
}
