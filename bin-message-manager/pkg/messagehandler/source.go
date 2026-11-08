package messagehandler

import (
	"context"
	"regexp"
	"strings"

	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// sourceRawPattern is what the raw text of a source may contain: a leading plus, ASCII digits and the usual
// separators. NormalizeTarget strips every other character, so without this check a value with letters in it
// (for example "+Acme1234567") would be normalized into a number.
var sourceRawPattern = regexp.MustCompile(`^\+[0-9 ().\-]+$`)

// prepareSource checks the format of the source of a message without any call to another service. It works on a copy
// and never changes the address of the caller.
//
// A source is missing when it is nil or its target is blank, whatever its type is (the api-manager sends an empty
// address when the client leaves the source out); the result is then nil and the caller falls back to the default of
// the customer. A source that is given must be a phone number in E.164 format: an empty type is read as tel, the raw
// text is checked, normalized and validated. Anything else is SOURCE_INVALID.
func prepareSource(source *commonaddress.Address) (*commonaddress.Address, error) {
	if source == nil {
		return nil, nil
	}

	target := strings.TrimSpace(source.Target)
	if target == "" {
		return nil, nil
	}

	invalid := cerrors.InvalidArgument(commonoutline.ServiceNameMessageManager, "SOURCE_INVALID", "The source must be a phone number in E.164 format.")

	if source.Type != commonaddress.TypeNone && source.Type != commonaddress.TypeTel {
		return nil, invalid
	}
	if !sourceRawPattern.MatchString(target) {
		return nil, invalid
	}

	normalized, err := commonaddress.NormalizeTarget(commonaddress.TypeTel, target)
	if err != nil {
		return nil, invalid
	}
	if err := commonaddress.ValidateTarget(commonaddress.TypeTel, normalized); err != nil {
		return nil, invalid
	}

	res := *source
	res.Type = commonaddress.TypeTel
	res.Target = normalized

	return &res, nil
}

// resolveSource returns the source that the message is sent from.
//
// prepared is the result of prepareSource. When it is nil the default outgoing source number of the customer is used;
// when it is given it must be a number of the customer and it is never replaced by the default. In both cases the
// number must be a normal, active, not deleted number of the customer, which is checked against number-manager every
// time because the stored default can be stale (released, deactivated or changed after it was set).
//
// A failed lookup is an error and never a pass or a fallback.
func (h *messageHandler) resolveSource(ctx context.Context, customerID uuid.UUID, prepared *commonaddress.Address) (*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "resolveSource",
		"customer_id": customerID,
	})

	if prepared != nil {
		filters := map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: customerID,
			nmnumber.FieldNumber:     prepared.Target,
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}
		num, err := h.lookupNumber(ctx, filters)
		if err != nil {
			return nil, err
		}
		if num == nil {
			log.Info("The source is not a number of the customer.")
			return nil, cerrors.PermissionDenied(commonoutline.ServiceNameMessageManager, "SOURCE_NOT_OWNED", "The source is not a number of the customer.")
		}

		res := *prepared
		return &res, nil
	}

	cfgs, err := h.reqHandler.CallV1OutboundConfigList(ctx, customerID, 1, "")
	if err != nil {
		log.Errorf("Could not get the outbound config. err: %v", err)
		return nil, cerrors.Internal(commonoutline.ServiceNameMessageManager, "SOURCE_LOOKUP_FAILED", "The source could not be verified.").Wrap(err)
	}
	if !hasDefaultSource(cfgs) {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameMessageManager, "SOURCE_REQUIRED", "The source is required because the customer has no default outgoing source number.")
	}

	filters := map[nmnumber.Field]any{
		nmnumber.FieldCustomerID: customerID,
		nmnumber.FieldID:         cfgs[0].DefaultOutgoingSourceNumberID,
		nmnumber.FieldType:       nmnumber.TypeNormal,
		nmnumber.FieldStatus:     nmnumber.StatusActive,
		nmnumber.FieldDeleted:    false,
	}
	num, err := h.lookupNumber(ctx, filters)
	if err != nil {
		return nil, err
	}
	if num == nil {
		log.Warnf("The default outgoing source number is not valid. number_id: %s", cfgs[0].DefaultOutgoingSourceNumberID)
		return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameMessageManager, "DEFAULT_SOURCE_INVALID", "The default outgoing source number of the outbound config is not an active number of the customer. Check the outbound config.")
	}

	return &commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: num.Number,
	}, nil
}

// hasDefaultSource tells whether the outbound config rows hold a default outgoing source number. The table has one
// row for each customer; the nil id means that the fallback is disabled.
func hasDefaultSource(cfgs []cmoutboundconfig.OutboundConfig) bool {
	return len(cfgs) > 0 && cfgs[0].DefaultOutgoingSourceNumberID != uuid.Nil
}

// lookupNumber returns the first number that matches the filters, or nil when there is none.
func (h *messageHandler) lookupNumber(ctx context.Context, filters map[nmnumber.Field]any) (*nmnumber.Number, error) {
	nums, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
	if err != nil {
		logrus.WithField("func", "lookupNumber").Errorf("Could not list the numbers. err: %v", err)
		return nil, cerrors.Internal(commonoutline.ServiceNameMessageManager, "SOURCE_LOOKUP_FAILED", "The source could not be verified.").Wrap(err)
	}
	if len(nums) == 0 {
		return nil, nil
	}

	return &nums[0], nil
}
