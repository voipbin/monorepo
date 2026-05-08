package outboundconfighandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	nmnumber "monorepo/bin-number-manager/models/number"
)

func (h *outboundConfigHandler) validateUpdateRequest(ctx context.Context, customerID uuid.UUID, req *outboundconfig.UpdateRequest) error {
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
			return fmt.Errorf("codecs must be empty or a comma-separated list of codec names. Each name must be alphanumeric and may contain internal hyphens (e.g. AMR-WB). Max 255 chars")
		}
	}
	if req.DefaultOutgoingSourceNumberID != nil && *req.DefaultOutgoingSourceNumberID != uuid.Nil {
		filters := map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: customerID,
			nmnumber.FieldID:         *req.DefaultOutgoingSourceNumberID,
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}
		nums, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
		if err != nil {
			return fmt.Errorf("could not validate default_outgoing_source_number_id: %w", err)
		}
		if len(nums) == 0 {
			return fmt.Errorf("default_outgoing_source_number_id %s is not a valid normal active number for customer %s", req.DefaultOutgoingSourceNumberID, customerID)
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
