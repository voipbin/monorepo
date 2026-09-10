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

// Update updates the McpServer. secret follows design §10's PUT pointer
// semantics: nil leaves the existing encrypted secret untouched, a pointer
// to "" clears it, and a pointer to a non-empty value re-encrypts and
// replaces it.
func (h *mcpServerHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	url string,
	status mcpserver.Status,
	authType mcpserver.AuthType,
	apiKeyHeader string,
	secret *string,
) (*mcpserver.McpServer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Update",
	})

	if err := ValidateURL(url); err != nil {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_URL", err.Error()).Wrap(err)
	}
	if !status.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_STATUS", "invalid status: "+string(status))
	}
	if !authType.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_AUTH_TYPE", "invalid auth_type: "+string(authType))
	}

	fields := map[mcpserver.Field]any{
		mcpserver.FieldName:         name,
		mcpserver.FieldDetail:       detail,
		mcpserver.FieldURL:          url,
		mcpserver.FieldStatus:       status,
		mcpserver.FieldAuthType:     authType,
		mcpserver.FieldAPIKeyHeader: apiKeyHeader,
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
