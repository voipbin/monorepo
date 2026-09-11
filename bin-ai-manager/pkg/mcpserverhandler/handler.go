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

	if err := h.db.McpServerDelete(ctx, id); err != nil {
		return nil, errors.Wrapf(err, "could not delete mcp server")
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, mcpserver.EventTypeDeleted, res)

	return res, nil
}
