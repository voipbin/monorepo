package outboundconfighandler

import (
	"fmt"
	"strings"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

// validateUpdateRequest validates the fields in an UpdateRequest.
// It returns an error if DestinationWhitelist or Codecs contain invalid values.
func (h *outboundConfigHandler) validateUpdateRequest(req *outboundconfig.UpdateRequest) error {
	if req == nil {
		return nil
	}
	if req.DestinationWhitelist != nil {
		if err := validateWhitelist(*req.DestinationWhitelist); err != nil {
			return err
		}
	}
	if req.Codecs != nil {
		if !outboundconfig.ValidateCodecs(*req.Codecs) {
			return fmt.Errorf("codecs must be empty or comma-separated alphanumeric tokens (max 255 chars, no special chars)")
		}
	}
	return nil
}

// validateWhitelist checks that all entries are valid, unique ISO 3166 alpha-2 country codes.
func validateWhitelist(entries []string) error {
	seen := make(map[string]struct{})
	for _, e := range entries {
		normalized := strings.ToLower(strings.TrimSpace(e))
		if normalized == "" {
			return fmt.Errorf("destination_whitelist entry cannot be empty")
		}
		if _, ok := outboundconfig.ISOCountryCodes[normalized]; !ok {
			return fmt.Errorf("invalid ISO 3166 alpha-2 country code: %q", e)
		}
		if _, dup := seen[normalized]; dup {
			return fmt.Errorf("duplicate country code after normalization: %q", normalized)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}
