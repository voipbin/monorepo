package mcpserverhandler

import (
	stderrors "errors"

	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// Create creates a new McpServer record. The URL is SSRF-validated before
// persisting (design §7); the secret, if non-empty, is encrypted before
// persisting (design §6).
func (h *mcpServerHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	url string,
	status mcpserver.Status,
	authType mcpserver.AuthType,
	apiKeyHeader string,
	secret string,
) (*mcpserver.McpServer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Create",
	})

	if err := ValidateURL(url); err != nil {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_URL", err.Error()).Wrap(err)
	}

	if status == "" {
		status = mcpserver.StatusActive
	}
	if !status.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_STATUS", "invalid status: "+string(status))
	}
	if !authType.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_AUTH_TYPE", "invalid auth_type: "+string(authType))
	}
	// A create has no prior row to transition from, so unlike Update this is a
	// plain value rejection: `oauth` is a state the OAuth flow produces, not
	// an input. Creating a server already claiming it would advertise a bearer
	// token that does not exist.
	if authType == mcpserver.AuthTypeOAuth {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameAIManager,
			"INVALID_MCP_SERVER_AUTH_TYPE",
			"auth_type \"oauth\" is set by completing the OAuth authorization flow, not directly",
		)
	}

	m := &mcpserver.McpServer{
		Identity: identity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: customerID,
		},

		Name:   name,
		Detail: detail,

		URL:    url,
		Status: status,

		AuthType:     authType,
		APIKeyHeader: apiKeyHeader,
	}

	if secret != "" {
		ciphertext, nonce, version, err := h.crypto.Encrypt(secret)
		if err != nil {
			return nil, errors.Wrap(err, "could not encrypt secret")
		}
		m.SecretCiphertext = ciphertext
		m.SecretNonce = nonce
		m.KeyVersion = version
	}

	if err := h.db.McpServerCreate(ctx, m); err != nil {
		return nil, errors.Wrapf(err, "could not create mcp server")
	}

	res, err := h.db.McpServerGet(ctx, m.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created mcp server")
	}
	log.WithField("mcp_server", res).Debugf("Created mcp server. mcp_server_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, mcpserver.EventTypeCreated, res)

	return res, nil
}

// Get returns the McpServer.
func (h *mcpServerHandler) Get(ctx context.Context, id uuid.UUID) (*mcpserver.McpServer, error) {
	res, err := h.db.McpServerGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"MCP_SERVER_NOT_FOUND",
				"The MCP server was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not get mcp server")
	}

	return res, nil
}

// getLive returns the McpServer only if it exists AND has not been
// soft-deleted, mapping both cases to the same NotFound the REST layer already
// translates.
//
// Separate from Get on purpose. Get must keep returning soft-deleted rows: the
// REST read of a deleted server answers 200 by design (asserted by the
// validator), and Delete reads the row back after deleting it. Write paths need
// the opposite, so they call this instead. Gating Get itself, or
// dbhandler.McpServerGet, would break those consumers.
func (h *mcpServerHandler) getLive(ctx context.Context, id uuid.UUID) (*mcpserver.McpServer, error) {
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if res.TMDelete != nil {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameAIManager,
			"MCP_SERVER_NOT_FOUND",
			"The MCP server was not found.",
		)
	}

	return res, nil
}

// List returns a list of McpServers.
func (h *mcpServerHandler) List(ctx context.Context, size uint64, token string, filters map[mcpserver.Field]any) ([]*mcpserver.McpServer, error) {
	res, err := h.db.McpServerList(ctx, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not list mcp servers")
	}

	return res, nil
}

// Update updates the McpServer. Every field below follows the same PUT
// pointer semantics established for secret in design §10 MN2 and extended
// to the other six fields in
// docs/plans/2026-09-12-mcp-server-put-partial-update-design.md: nil
// means "leave the existing value untouched", a non-nil pointer
// (including one pointing at the zero value, e.g. auth_type: "" or
// url: "") means "set to exactly this value, validate it as normal".
// secret additionally treats a non-nil pointer to "" as "explicitly
// clear" (distinct from "set to empty string" for the other string
// fields, since an empty secret is a real clear operation, not a
// validatable value) -- this asymmetry is intentional and pre-existing,
// not something this change alters.
func (h *mcpServerHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	detail *string,
	url *string,
	status *mcpserver.Status,
	authType *mcpserver.AuthType,
	apiKeyHeader *string,
	secret *string,
) (*mcpserver.McpServer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Update",
	})

	// Existence gate, placed ahead of BOTH the validation block below and the
	// `len(fields) == 0` short-circuit further down. Without it a `PUT {}` on a
	// deleted server still returns 200 with the row, and a malformed-URL PUT on
	// a deleted server returns 400 instead of 404 -- validation would otherwise
	// decide the status code for a resource that no longer exists.
	//
	// This gate lives here and NOT inside h.Get or dbhandler.McpServerGet: Get's
	// other consumers require the opposite behaviour (the REST read of a
	// soft-deleted server must keep answering 200, and Delete reads the row back
	// after deleting it).
	live, err := h.getLive(ctx, id)
	if err != nil {
		return nil, err
	}

	if url != nil {
		if err := ValidateURL(*url); err != nil {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_URL", err.Error()).Wrap(err)
		}
	}
	if status != nil && !status.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_STATUS", "invalid status: "+string(*status))
	}
	if authType != nil && !authType.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_AUTH_TYPE", "invalid auth_type: "+string(*authType))
	}
	// Validate the TRANSITION, not the value: `oauth` is reachable only by
	// completing the OAuth flow, which is what writes the tokens. Accepting a
	// direct move INTO oauth would produce a server whose auth_type promises a
	// bearer token it has no way to obtain.
	//
	// A server that already completed OAuth must stay editable. square-admin
	// re-submits auth_type unchanged on every save (it builds the PUT body
	// from form state, not from a diff), so rejecting the VALUE would 400
	// every rename, URL change, and enable/disable of a connected server --
	// the same permanent-freeze shape as an unremovable whitelist id. Compare
	// against the stored row and reject only an actual transition.
	if authType != nil && *authType == mcpserver.AuthTypeOAuth && live.AuthType != mcpserver.AuthTypeOAuth {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameAIManager,
			"INVALID_MCP_SERVER_AUTH_TYPE",
			"auth_type \"oauth\" is set by completing the OAuth authorization flow, not directly",
		)
	}

	// An OAuth-connected server's URL is the vendor's own fixed MCP
	// endpoint (e.g. GitHub's), not something the customer chose --
	// completing the OAuth flow is what wrote it. Letting it be edited
	// would send the stored vendor access/refresh tokens to whatever URL
	// the customer (or an attacker with console access) puts here next,
	// which is a credential-exfiltration path, not a configuration
	// mistake. Reject only an actual CHANGE, not a re-submit of the
	// current value: square-admin re-submits every field on every save,
	// so rejecting an unchanged url would 400 every rename or
	// enable/disable of a connected server, the same permanent-freeze
	// shape auth_type's re-submit exemption above already avoids.
	if url != nil && *url != live.URL && live.AuthType == mcpserver.AuthTypeOAuth {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameAIManager,
			"MCP_SERVER_OAUTH_URL_IMMUTABLE",
			"url cannot be changed while auth_type is oauth; disconnect (move auth_type out of oauth) first",
		)
	}

	fields := map[mcpserver.Field]any{}
	if name != nil {
		fields[mcpserver.FieldName] = *name
	}
	if detail != nil {
		fields[mcpserver.FieldDetail] = *detail
	}
	if url != nil {
		fields[mcpserver.FieldURL] = *url
	}
	if status != nil {
		fields[mcpserver.FieldStatus] = *status
	}
	if authType != nil {
		fields[mcpserver.FieldAuthType] = *authType

		// Leaving oauth is a legitimate downgrade to a static credential, but
		// the vendor tokens must not survive it. Keeping them would leave
		// decryptable GitHub/Linear material in a row that no longer claims to
		// be an OAuth connection, with no customer gesture left that erases it:
		// the UI offers no way back into oauth (mcpservers_detail.js renders the
		// type as a fixed badge once connected), and re-running the OAuth flow
		// overwrites rather than reveals. It would also make has_secret lie --
		// it is derived from secret_ciphertext OR access_token_ciphertext, so a
		// bearer row carrying a stale access token reports a stored credential
		// that buildAuthHeader cannot use, and every tool call fails with no
		// way for the customer to see why.
		//
		// This mirrors the token-column zeroing in McpServerDelete, for the
		// same reason. The column sets are not identical: Delete also clears
		// the secret envelope and retains oauth_vendor, while this path leaves
		// the secret envelope alone (the same request may be supplying a new
		// secret) and does clear oauth_vendor, which is documented as set only
		// while auth_type is oauth. access_token_expires_at is retained by
		// both as audit metadata; it is read only on the oauth path, which a
		// downgraded row no longer takes.
		if *authType != mcpserver.AuthTypeOAuth && live.AuthType == mcpserver.AuthTypeOAuth {
			fields[mcpserver.FieldOAuthVendor] = ""
			fields[mcpserver.FieldAccessTokenCiphertext] = []byte(nil)
			fields[mcpserver.FieldAccessTokenNonce] = []byte(nil)
			fields[mcpserver.FieldRefreshTokenCiphertext] = []byte(nil)
			fields[mcpserver.FieldRefreshTokenNonce] = []byte(nil)
		}
	}
	if apiKeyHeader != nil {
		fields[mcpserver.FieldAPIKeyHeader] = *apiKeyHeader
	}

	if secret != nil {
		if *secret == "" {
			fields[mcpserver.FieldSecretCiphertext] = []byte(nil)
			fields[mcpserver.FieldSecretNonce] = []byte(nil)
			fields[mcpserver.FieldKeyVersion] = 0
		} else {
			ciphertext, nonce, version, err := h.crypto.Encrypt(*secret)
			if err != nil {
				return nil, errors.Wrap(err, "could not encrypt secret")
			}
			fields[mcpserver.FieldSecretCiphertext] = ciphertext
			fields[mcpserver.FieldSecretNonce] = nonce
			fields[mcpserver.FieldKeyVersion] = version
		}
	}

	if len(fields) == 0 {
		// A PUT with every field omitted (or only unchanged/invalid-nil
		// pointers) is a client no-op, not a server error. Returning the
		// current row (instead of erroring or issuing a zero-column
		// UPDATE, which some SQL builders reject) matches List/Get's
		// "always return current state" contract and avoids a special
		// "PATCH-with-nothing-to-patch" error class nothing else in this
		// API returns. Reuses Get's existing ErrNotFound -> cerrors.NotFound
		// mapping rather than duplicating it.
		return h.Get(ctx, id)
	}

	if err := h.db.McpServerUpdate(ctx, id, fields); err != nil {
		return nil, errors.Wrapf(err, "could not update mcp server")
	}

	res, err := h.db.McpServerGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated mcp server")
	}
	log.WithField("mcp_server", res).Debugf("Updated mcp server. mcp_server_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, mcpserver.EventTypeUpdated, res)

	return res, nil
}

// Delete soft-deletes the McpServer.
func (h *mcpServerHandler) Delete(ctx context.Context, id uuid.UUID) (*mcpserver.McpServer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
	})

	// A repeat DELETE reaches no live row. Swallow that rather than 404ing:
	// DELETE is idempotent by contract, and GET on a soft-deleted server still
	// answers 200, so 404ing only the SECOND delete while GET keeps succeeding
	// would be incoherent. The read-back below still returns the row; the only
	// difference is that the deleted event is not published twice.
	alreadyDeleted := false
	if err := h.db.McpServerDelete(ctx, id); err != nil {
		if !stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, errors.Wrapf(err, "could not delete mcp server")
		}
		alreadyDeleted = true
	}

	res, err := h.db.McpServerGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"MCP_SERVER_NOT_FOUND",
				"The MCP server was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not get deleted mcp server")
	}
	log.WithField("mcp_server", res).Debugf("Deleted mcp server. mcp_server_id: %s", res.ID)
	if !alreadyDeleted {
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, mcpserver.EventTypeDeleted, res)
	}

	return res, nil
}
