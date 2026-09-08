package servicehandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// directResourceMapping maps a direct resource_type to the allowed_resource_types
var directResourceMapping = map[string][]string{
	dmdirect.ResourceTypeAI:            {"aicall"},
	dmdirect.ResourceTypeAITeam:        {"aicall"},
	dmdirect.ResourceTypeWebchatWidget: {"webchat_session"},
}

// directHashFingerprintDomain separates this MAC's key usage from JWT signing.
const directHashFingerprintDomain = "voipbin/direct-hash-fingerprint/v1\x00"

// directHashFingerprint derives the value stored in DirectScope.HashFingerprint
// and re-checked by AuthBootRefresh.
//
// Keyed with the signing key rather than a bare digest. The underlying direct
// hash is only 48 bits (bin-direct-manager generateHash uses 6 random bytes) in
// a known format, so an unkeyed SHA-256 of it is brute-forceable offline in
// hours. That matters because the token travels further than the hash does: the
// WebSocket paths log the whole *auth.AuthIdentity, so an unkeyed fingerprint
// would let anyone with log access recover the direct hash, and that recovery
// outlives both the token's expiry and the boot session ceiling.
//
// HMAC keeps every property the check needs: deterministic, stable across
// replicas and restarts, and full width (no truncation).
//
// The constant prefix domain-separates this from JWT signing, which uses the
// same key. Nothing here is exploitable without it (HMAC has no length
// extension, and no direct hash can collide with a JWT signing input), but the
// prefix removes the need to make that argument at all.
func (h *serviceHandler) directHashFingerprint(hash string) string {
	mac := hmac.New(sha256.New, h.jwtKey)
	mac.Write([]byte(directHashFingerprintDomain))
	mac.Write([]byte(hash))
	return hex.EncodeToString(mac.Sum(nil))
}

// BootResponse is the typed response for POST /auth/boot.
type BootResponse struct {
	Token        string    `json:"token"`
	Type         string    `json:"type"`
	ResourceType string    `json:"resource_type"`
	ResourceID   uuid.UUID `json:"resource_id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	Expire       string    `json:"expire"`

	// AllowedResourceID and ScopeVersion mirror the identically named
	// DirectScope claims. Both MUST be read from the scope struct rather than
	// recomputed, or the client's view and the token's view can diverge -- the
	// client gates its id-mismatch reboot on ScopeVersion, so a response that
	// under-reports it leaves that check asleep exactly when it is needed.
	AllowedResourceID uuid.UUID `json:"allowed_resource_id"`
	ScopeVersion      int       `json:"scope_version"`

	// ResourceData is a resource-type-scoped envelope for additional,
	// publicly-safe data about the boot-scoped resource. Each entry is a
	// named, self-documenting key (see design doc
	// docs/plans/2026-07-20-auth-boot-public-display-config-design.md
	// §3.1/§9) -- do not add bare/generic keys. Currently the only
	// populated key is "public_display_config" (see
	// resourceDisplayConfigFetchers). Populated best-effort: a fetch
	// failure never fails the boot request itself. nil/omitted entirely
	// when no fetcher is registered for ResourceType, or when the
	// fetcher for this resource returned nothing.
	ResourceData map[string]interface{} `json:"resource_data,omitempty"`
}

// resourceDisplayConfigFetchers maps a direct resource_type to a function
// that resolves its public_display_config payload for the resource_data
// envelope. Add an entry here when a new resource type needs to expose
// safe, anonymous-visitor-facing display data through /auth/boot. Every
// fetcher MUST extract a specifically-vetted SUB-FIELD of the resource's
// ConvertWebhookMessage()-shaped external DTO (e.g. .ThemeConfig here) --
// never the raw internal model struct, and never the entire converted
// DTO un-narrowed (WebhookMessage-shaped structs are NOT automatically
// safe for anonymous exposure; e.g. webchat's WebhookMessage still
// carries SessionFlowID/MessageFlowID, which must never reach this
// envelope). Every fetcher MUST also return a true nil interface{} (not
// a nil-but-typed pointer) when there is no data to report, so the
// data != nil check below correctly skips allocating the envelope.
var resourceDisplayConfigFetchers = map[string]func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error){
	dmdirect.ResourceTypeWebchatWidget: func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error) {
		w, err := h.reqHandler.WebchatV1WidgetGet(ctx, resourceID)
		if err != nil {
			return nil, err
		}
		tc := w.ConvertWebhookMessage().ThemeConfig
		if tc == nil {
			// Explicit nil interface, not a nil *ThemeConfig boxed into
			// interface{} -- a boxed typed-nil pointer is a non-nil
			// interface value and the data != nil check below would
			// otherwise allocate an envelope with a null entry instead
			// of omitting resource_data entirely.
			return nil, nil
		}
		return tc, nil
	},
}

// AuthBoot resolves a direct hash and returns a resource-scoped JWT.
func (h *serviceHandler) AuthBoot(ctx context.Context, directHash string) (*BootResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AuthBoot",
		"direct_hash": truncateHash(directHash),
	})

	// validate hash format
	if !strings.HasPrefix(directHash, dmdirect.DirectPrefix) {
		return nil, fmt.Errorf("%w: invalid direct hash format", serviceerrors.ErrInvalidArgument)
	}

	// resolve hash
	d, err := h.reqHandler.DirectV1DirectGetByHash(ctx, directHash)
	if err != nil {
		log.Infof("Could not get direct by hash. err: %v", err)
		return nil, fmt.Errorf("%w: direct hash not found", serviceerrors.ErrNotFound)
	}
	log.WithField("direct", d).Debugf("Retrieved direct info. direct_id: %s", d.ID)

	// validate customer is active
	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, d.CustomerID)
	if err != nil {
		log.Infof("Could not get customer. err: %v", err)
		return nil, fmt.Errorf("%w: customer not found", serviceerrors.ErrNotFound)
	}
	log.WithField("customer_id", cu.ID).Debugf("Retrieved customer info. customer_id: %s", cu.ID)

	if cu.Status != cscustomer.StatusActive {
		log.Infof("Customer is not active. status: %s", cu.Status)
		return nil, fmt.Errorf("%w: customer not active", serviceerrors.ErrStateInvalid)
	}

	// look up allowed resource types
	allowedTypes, ok := directResourceMapping[d.ResourceType]
	if !ok {
		log.Infof("Unsupported direct resource type. resource_type: %s", d.ResourceType)
		return nil, fmt.Errorf("%w: unsupported resource type: %s", serviceerrors.ErrInvalidArgument, d.ResourceType)
	}

	// build direct scope
	scope := &auth.DirectScope{
		CustomerID:           d.CustomerID,
		ResourceType:         d.ResourceType,
		ResourceID:           d.ResourceID,
		AllowedResourceTypes: allowedTypes,

		AllowedResourceID: h.utilHandler.UUIDCreate(),
		DirectID:          d.ID,
		HashFingerprint:   h.directHashFingerprint(d.Hash),
		BootExpire:        h.utilHandler.TimeGetCurTimeAdd(BootSessionMaxLifetime),
		ScopeVersion:      DirectScopeVersionCurrent,
	}

	// generate JWT with boot expiration
	data := map[string]interface{}{
		"type":   "direct",
		"direct": scope,
	}
	token, expire, err := h.authJWTGenerateWithExpiration(data, BootExpiration)
	if err != nil {
		log.Errorf("Could not generate boot JWT. err: %v", err)
		return nil, fmt.Errorf("%w: token generation failed", serviceerrors.ErrInternal)
	}

	res := &BootResponse{
		Token:        token,
		Type:         "direct",
		ResourceType: d.ResourceType,
		ResourceID:   d.ResourceID,
		CustomerID:   d.CustomerID,
		Expire:       expire,

		// Read from scope, never recomputed. See the BootResponse doc comment.
		AllowedResourceID: scope.AllowedResourceID,
		ScopeVersion:      scope.ScopeVersion,
	}

	if fetcher, ok := resourceDisplayConfigFetchers[d.ResourceType]; ok {
		data, ferr := fetcher(ctx, h, d.ResourceID)
		if ferr != nil {
			log.Infof("Could not fetch public display config. resource_type: %s, err: %v", d.ResourceType, ferr)
		} else if data != nil {
			// Only allocate + populate the envelope when there is
			// something real to put in it -- an empty non-nil map
			// would still serialize as "resource_data": {}, which is
			// different from (and worse than) omitting the key
			// entirely.
			res.ResourceData = map[string]interface{}{"public_display_config": data}
		}
	}

	return res, nil
}

// authJWTGenerateWithExpiration generates a JWT with the specified expiration duration.
// It returns the signed token string and the expiration timestamp.
func (h *serviceHandler) authJWTGenerateWithExpiration(data map[string]interface{}, expiration time.Duration) (string, string, error) {
	expire := h.utilHandler.TimeGetCurTimeAdd(expiration)

	claims := jwt.MapClaims{}
	for k, v := range data {
		claims[k] = v
	}
	claims["expire"] = expire

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	res, err := token.SignedString(h.jwtKey)
	if err != nil {
		return "", "", err
	}

	return res, expire, nil
}

// truncateHash returns a masked version of the hash for safe logging.
func truncateHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12] + "..."
}
