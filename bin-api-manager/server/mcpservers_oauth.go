package server

import (
	"fmt"
	"net/url"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// mcpOAuthReturnURL is the client application's landing route the public
// GET /mcpservers/oauth/callback relay redirects the browser to after
// its thin, no-mutation existence check (design
// docs/plans/2026-09-12-mcp-server-oauth-support-design.md §7, §10).
const mcpOAuthReturnURL = "https://admin.voipbin.net/#/resources/mcpservers/oauth-return"

// PostMcpserversOauthStart handles POST /mcpservers/oauth/start
// (authenticated). Starts the vendor OAuth 2.1 authorization-code + PKCE
// flow for the authenticated customer.
func (h *server) PostMcpserversOauthStart(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostMcpserversOauthStart",
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

	var req openapi_server.PostMcpserversOauthStartJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	var mcpServerID *uuid.UUID
	if req.McpServerId != nil {
		id := uuid.UUID(*req.McpServerId)
		mcpServerID = &id
	}

	authorizeURL, linkToken, err := h.serviceHandler.McpOAuthStart(c.Request.Context(), a, string(req.Vendor), mcpServerID)
	if err != nil {
		log.Errorf("Could not start mcp oauth flow. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"authorize_url": authorizeURL,
		"link_token":    linkToken,
	})
}

// GetMcpserversOauthCallback handles GET /mcpservers/oauth/callback
// (PUBLIC, unauthenticated). The vendor's authorization server redirects
// the user's browser here after consent; this is a thin, no-mutation
// redirect relay -- it never exchanges the code or touches McpServer
// (design §7a Layer 1, §10). It always 302-redirects the browser onward
// to the client application's oauth-return route, with the vendor's
// query parameters preserved when the state exists, or a generic
// invalid_state error otherwise.
func (h *server) GetMcpserversOauthCallback(c *gin.Context, params openapi_server.GetMcpserversOauthCallbackParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetMcpserversOauthCallback",
		"request_address": c.ClientIP,
	})

	exists, err := h.serviceHandler.McpOAuthCallback(c.Request.Context(), params.State)
	if err != nil {
		// Fail closed to the same generic error redirect as an
		// invalid/expired state (design §7a's anti-enumeration
		// posture) -- this endpoint has no UI of its own.
		log.Errorf("Could not check mcp oauth callback state. err: %v", err)
		exists = false
	}

	q := url.Values{}
	q.Set("state", params.State)

	if !exists {
		q.Set("error", "invalid_state")
		c.Redirect(302, fmt.Sprintf("%s?%s", mcpOAuthReturnURL, q.Encode()))
		return
	}

	if params.Code != nil && *params.Code != "" {
		q.Set("code", *params.Code)
	}

	c.Redirect(302, fmt.Sprintf("%s?%s", mcpOAuthReturnURL, q.Encode()))
}

// PostMcpserversOauthComplete handles POST /mcpservers/oauth/complete
// (authenticated). Verifies the state belongs to the authenticated
// customer, exchanges the vendor code for tokens, and creates/updates
// the McpServer row.
func (h *server) PostMcpserversOauthComplete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostMcpserversOauthComplete",
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

	var req openapi_server.PostMcpserversOauthCompleteJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.McpOAuthComplete(c.Request.Context(), a, req.State, req.Code)
	if err != nil {
		log.Errorf("Could not complete mcp oauth flow. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
