package mcpserverhandler

//go:generate mockgen -package mcpserverhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// McpServerHandler provides CRUD operations for customer-registered
// McpServer resources.
//
// Update's secret parameter is *string per design §10's PUT semantics: nil
// means "leave the existing encrypted secret untouched", a non-nil pointer
// to "" means "explicitly clear the secret", and a non-nil pointer to a
// value means "re-encrypt and replace".
type McpServerHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		url string,
		status mcpserver.Status,
		authType mcpserver.AuthType,
		apiKeyHeader string,
		secret string,
	) (*mcpserver.McpServer, error)
	Get(ctx context.Context, id uuid.UUID) (*mcpserver.McpServer, error)
	List(ctx context.Context, size uint64, token string, filters map[mcpserver.Field]any) ([]*mcpserver.McpServer, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		url string,
		status mcpserver.Status,
		authType mcpserver.AuthType,
		apiKeyHeader string,
		secret *string,
	) (*mcpserver.McpServer, error)
	Delete(ctx context.Context, id uuid.UUID) (*mcpserver.McpServer, error)
}

type mcpServerHandler struct {
	utilHandler   utilhandler.UtilHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	crypto        *SecretCrypto
}

// NewMcpServerHandler creates a new McpServerHandler. cryptoKeys is the raw
// MCP_SECRET_ENCRYPTION_KEYS config value (design §6/§13); a malformed
// value is a boot-time failure, surfaced by the caller via internal/config's
// Validate(), so this constructor returns an error rather than panicking or
// silently degrading.
func NewMcpServerHandler(
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	cryptoKeys string,
) (McpServerHandler, error) {
	crypto, err := NewSecretCrypto(cryptoKeys)
	if err != nil {
		return nil, err
	}

	return &mcpServerHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		notifyHandler: notifyHandler,
		db:            db,
		crypto:        crypto,
	}, nil
}
