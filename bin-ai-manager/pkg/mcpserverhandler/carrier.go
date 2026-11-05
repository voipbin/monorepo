package mcpserverhandler

import (
	"strings"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-ai-manager/models/mcpserver"
)

// checkCarrierURL refuses a url on the reserved carrier host that is not a url
// of the carrier table, so a typo does not become a silent call failure. The
// message lists the table's urls and never repeats the input. A url on any
// other host, ".invalid" ones included, is not this check's concern.
func (h *mcpServerHandler) checkCarrierURL(rawURL string) error {
	if !mcpserver.IsReservedHost(rawURL) {
		return nil
	}
	if _, ok := h.carrierTable.InternalCarrier(rawURL); ok {
		return nil
	}

	message := "no carrier integration is available"
	if allowed := h.carrierTable.AllowedURLs(); len(allowed) > 0 {
		message = "url must be one of: " + strings.Join(allowed, ", ")
	}
	return cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_URL", message)
}

// checkCarrierSecret checks a secret that arrives with a carrier url against
// the format that carrier needs. It checks nothing when the request carries no
// secret, when the auth type is not one the adapter reads (bearer, api_key),
// or when the url is not a carrier url. The message is fixed.
func (h *mcpServerHandler) checkCarrierSecret(rawURL string, authType mcpserver.AuthType, secret string) error {
	if secret == "" || (authType != mcpserver.AuthTypeBearer && authType != mcpserver.AuthTypeAPIKey) {
		return nil
	}
	entry, ok := h.carrierTable.InternalCarrier(rawURL)
	if !ok {
		return nil
	}
	if _, err := mcpserver.ParseCredential(entry.Credential, secret); err != nil {
		return cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_SECRET", "the secret does not match the format required by this carrier url")
	}
	return nil
}
