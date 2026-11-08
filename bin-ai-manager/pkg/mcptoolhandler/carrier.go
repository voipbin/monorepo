package mcptoolhandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

// CarrierHandler runs the tool calls of the built-in carrier adapters. It is
// defined here, where it is used, so that the adapter package does not import
// this one; the process wiring connects them.
type CarrierHandler interface {
	// Tools returns the tools of carrier, and false when no adapter is
	// registered for it.
	Tools(carrier string) ([]mcpserver.CarrierToolDef, bool)

	// Call runs one tool call. It returns the text the model sees and whether
	// that text is an error result; err is set only for a fault of the
	// service.
	Call(ctx context.Context, call mcpserver.CarrierCall) (string, bool, error)
}

var (
	errCarrierUnavailable = errors.New("carrier integration is not available")
	errSecretDecrypt      = errors.New("could not decrypt the saved secret")
)

// carrierEntry returns the adapter's table entry for a server on the reserved
// host, and false when the url is not a url of the table or no adapter handler
// is wired. A server on the reserved host never takes the network path
// whatever this returns.
func (h *mcpToolHandler) carrierEntry(m *mcpserver.McpServer) (mcpserver.CarrierEntry, bool) {
	if h.carriers == nil {
		return mcpserver.CarrierEntry{}, false
	}
	return h.carrierTable.InternalCarrier(m.URL)
}

// listCarrierTools returns the tools of the adapter a reserved-host server
// names. The messages quote neither the url nor anything of the record.
func (h *mcpToolHandler) listCarrierTools(m *mcpserver.McpServer) ([]McpTool, error) {
	entry, ok := h.carrierEntry(m)
	if !ok {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: %w", errCarrierUnavailable)
	}
	defs, ok := h.carriers.Tools(entry.Carrier)
	if !ok {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: %w", errCarrierUnavailable)
	}

	tools := make([]McpTool, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, McpTool{Name: d.Name, Description: d.Description, InputSchema: d.InputSchema})
	}
	return tools, nil
}

// callCarrierTool runs a tool call on the adapter a reserved-host server
// names. When the adapter cannot be reached the model is told so in a fixed
// text and the call is an error result, not a fault.
func (h *mcpToolHandler) callCarrierTool(ctx context.Context, m *mcpserver.McpServer, toolName, argumentsJSON string) (string, bool, error) {
	entry, ok := h.carrierEntry(m)
	if !ok {
		logrus.WithField("mcp_server_id", m.ID).Warnf("A carrier integration is not available for this mcp server.")
		return mcpserver.IntegrationUnavailableText, true, nil
	}

	call := mcpserver.CarrierCall{
		CustomerID:    m.CustomerID,
		ServerID:      m.ID,
		Carrier:       entry.Carrier,
		Credential:    entry.Credential,
		ToolName:      toolName,
		ArgumentsJSON: argumentsJSON,
		Secret:        h.carrierSecret(m),
	}
	return h.carriers.Call(ctx, call)
}

// carrierSecret returns the function the adapter calls, once, to read the
// saved secret. Only the bearer and api_key types are read; the secret is
// decrypted here, with this handler's keys, so no key material leaves it.
func (h *mcpToolHandler) carrierSecret(m *mcpserver.McpServer) func() (string, bool, error) {
	return func() (string, bool, error) {
		if (m.AuthType != mcpserver.AuthTypeBearer && m.AuthType != mcpserver.AuthTypeAPIKey) || len(m.SecretCiphertext) == 0 {
			return "", false, nil
		}
		secret, err := mcpserverhandler.DecryptSecret(h.crypto, m.SecretCiphertext, m.SecretNonce, m.KeyVersion)
		if err != nil {
			return "", false, errSecretDecrypt
		}
		return secret, true, nil
	}
}
