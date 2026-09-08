package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// AuthBootRefresh reissues a direct token without changing which resource it is
// bound to, so a conversation survives the boot token's 4 hour expiry.
//
// It exists because the widgets refresh 5 minutes before expiry by calling
// POST /auth/boot again, which mints a *new* AllowedResourceID while the client
// keeps talking about the old resource. That kills a live conversation at
// ~3h55m. Refreshing through this endpoint carries the assignment forward.
//
// Unlike POST /auth/boot this endpoint is authenticated, which is what makes it
// safe: the assignment is copied from the presented token, never read from the
// request body, so a caller cannot name a resource it does not already hold.
//
// The checks below are ordered cheap-first (claim-only before RPC) and every
// failure after the identity check returns the same 401, so the response does
// not distinguish "record deleted" from "customer frozen". See design
// docs/plans/2026-09-08-direct-token-resource-binding-design.md §3.6.
func (h *serviceHandler) AuthBootRefresh(ctx context.Context, a *auth.AuthIdentity) (*BootResponse, error) {
	log := logrus.WithField("func", "AuthBootRefresh")

	// 1. identity type. authProtected admits agent/accesskey/delegate, and
	// DirectScope is nil for all of them -- without this the field reads below
	// would panic into a 500.
	if a == nil || !a.IsDirect() || a.DirectScope == nil {
		return nil, fmt.Errorf("%w: refresh is only for direct tokens", serviceerrors.ErrPermissionDenied)
	}
	scope := a.DirectScope
	log = log.WithField("direct_id", scope.DirectID)

	// 2. the token must already carry a full binding. A token minted before
	// this feature deserializes to zero values; reissuing it would launder
	// uuid.Nil into a token that looks current. Minting a replacement id here
	// would silently orphan the visitor's in-flight conversation, which is the
	// exact failure this endpoint exists to prevent -- so reject instead.
	if scope.AllowedResourceID == uuid.Nil || scope.DirectID == uuid.Nil || scope.HashFingerprint == "" {
		log.Info("Direct token predates resource binding. Rejecting refresh.")
		return nil, fmt.Errorf("%w: token is not resource bound", serviceerrors.ErrAuthenticationRequired)
	}

	// 3. absolute ceiling, copied verbatim on every refresh so it cannot be
	// walked forward.
	bootExpire, err := h.utilHandler.TimeParseWithError(scope.BootExpire)
	if err != nil {
		log.Infof("Could not parse boot expire. Rejecting refresh. err: %v", err)
		return nil, fmt.Errorf("%w: malformed boot expire", serviceerrors.ErrAuthenticationRequired)
	}
	// Read "now" through the util handler like every other time access in this
	// package, so the near-ceiling case stays drivable from a test.
	now, err := h.utilHandler.TimeParseWithError(h.utilHandler.TimeGetCurTime())
	if err != nil {
		log.Errorf("Could not parse the current time. err: %v", err)
		return nil, fmt.Errorf("%w: clock unavailable", serviceerrors.ErrInternal)
	}
	remaining := bootExpire.Sub(now)
	if remaining <= 0 {
		log.Info("Boot session lifetime exhausted. Rejecting refresh.")
		return nil, fmt.Errorf("%w: boot session expired", serviceerrors.ErrAuthenticationRequired)
	}

	// 4. the direct record must still exist. Deletion is a hard delete and the
	// lookup is uncached, so this is observed immediately. Fail closed on an
	// RPC error too -- do not reuse EnforceAccountStatus's fail-open branch.
	d, err := h.reqHandler.DirectV1DirectGet(ctx, scope.DirectID)
	if err != nil {
		log.Infof("Could not get direct record. Rejecting refresh. err: %v", err)
		return nil, fmt.Errorf("%w: direct not available", serviceerrors.ErrAuthenticationRequired)
	}

	// 5. the hash must not have been regenerated. Regeneration is how an
	// operator revokes a leaked public link, and it rotates only the hash --
	// a lookup keyed on customer or resource would not notice.
	if h.directHashFingerprint(d.Hash) != scope.HashFingerprint {
		log.Info("Direct hash was regenerated. Rejecting refresh.")
		return nil, fmt.Errorf("%w: direct hash rotated", serviceerrors.ErrAuthenticationRequired)
	}

	// 6. same customer check AuthBoot performs, fail closed on both branches.
	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, d.CustomerID)
	if err != nil {
		log.Infof("Could not get customer. Rejecting refresh. err: %v", err)
		return nil, fmt.Errorf("%w: customer not available", serviceerrors.ErrAuthenticationRequired)
	}
	if cu.Status != cscustomer.StatusActive {
		log.Infof("Customer is not active. Rejecting refresh. status: %s", cu.Status)
		return nil, fmt.Errorf("%w: customer not active", serviceerrors.ErrAuthenticationRequired)
	}

	allowedTypes, ok := directResourceMapping[d.ResourceType]
	if !ok {
		log.Infof("Unsupported direct resource type. resource_type: %s", d.ResourceType)
		return nil, fmt.Errorf("%w: unsupported resource type", serviceerrors.ErrAuthenticationRequired)
	}

	// 7. copy the whole scope, then re-derive the record-backed fields from the
	// record we just fetched rather than trusting the claim. Enumerating fields
	// here instead of copying wholesale is how CustomerID, DirectID,
	// HashFingerprint or ScopeVersion get silently dropped, which breaks the
	// second refresh or the WebSocket customer check.
	next := *scope
	next.CustomerID = d.CustomerID
	next.ResourceType = d.ResourceType
	next.ResourceID = d.ResourceID
	next.AllowedResourceTypes = allowedTypes

	// The JWT helper computes expiry from a duration, so the ceiling has to be
	// applied here; an absolute expire computed outside would be discarded.
	duration := BootExpiration
	if remaining < duration {
		duration = remaining
	}

	data := map[string]interface{}{
		"type":   "direct",
		"direct": &next,
	}
	token, expire, err := h.authJWTGenerateWithExpiration(data, duration)
	if err != nil {
		log.Errorf("Could not generate refreshed JWT. err: %v", err)
		return nil, fmt.Errorf("%w: token generation failed", serviceerrors.ErrInternal)
	}

	// ResourceData is deliberately omitted: it costs a widget RPC per call and
	// the client already holds it from the original boot. Every other field is
	// present because cachedBoot consumers read them.
	res := &BootResponse{
		Token:        token,
		Type:         "direct",
		ResourceType: next.ResourceType,
		ResourceID:   next.ResourceID,
		CustomerID:   next.CustomerID,
		Expire:       expire,

		AllowedResourceID: next.AllowedResourceID,
		ScopeVersion:      next.ScopeVersion,
	}

	log.WithField("expire", expire).Debug("Refreshed direct boot token.")
	return res, nil
}
