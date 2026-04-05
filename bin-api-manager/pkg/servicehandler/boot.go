package servicehandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-api-manager/models/auth"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// directResourceMapping maps a direct resource_type to the allowed_resource_types
var directResourceMapping = map[string][]string{
	"ai": {"aicall"},
}

// BootResponse is the typed response for POST /auth/boot.
type BootResponse struct {
	Token        string    `json:"token"`
	Type         string    `json:"type"`
	ResourceType string    `json:"resource_type"`
	ResourceID   uuid.UUID `json:"resource_id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	Expire       string    `json:"expire"`
}

// AuthBoot resolves a direct hash and returns a resource-scoped JWT.
func (h *serviceHandler) AuthBoot(ctx context.Context, directHash string) (*BootResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AuthBoot",
		"direct_hash": truncateHash(directHash),
	})

	// validate hash format
	if !strings.HasPrefix(directHash, dmdirect.DirectPrefix) {
		return nil, fmt.Errorf("invalid direct hash format")
	}

	// resolve hash
	d, err := h.reqHandler.DirectV1DirectGetByHash(ctx, directHash)
	if err != nil {
		log.Infof("Could not get direct by hash. err: %v", err)
		return nil, fmt.Errorf("direct hash not found")
	}
	log.WithField("direct", d).Debugf("Retrieved direct info. direct_id: %s", d.ID)

	// validate customer is active
	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, d.CustomerID)
	if err != nil {
		log.Infof("Could not get customer. err: %v", err)
		return nil, fmt.Errorf("customer not found")
	}
	log.WithField("customer", cu).Debugf("Retrieved customer info. customer_id: %s", cu.ID)

	if cu.Status != cscustomer.StatusActive {
		log.Infof("Customer is not active. status: %s", cu.Status)
		return nil, fmt.Errorf("customer not active")
	}

	// look up allowed resource types
	allowedTypes, ok := directResourceMapping[d.ResourceType]
	if !ok {
		log.Infof("Unsupported direct resource type. resource_type: %s", d.ResourceType)
		return nil, fmt.Errorf("unsupported resource type: %s", d.ResourceType)
	}

	// build direct scope
	scope := &auth.DirectScope{
		CustomerID:           d.CustomerID,
		ResourceType:         d.ResourceType,
		ResourceID:           d.ResourceID,
		AllowedResourceTypes: allowedTypes,
	}

	// generate JWT with boot expiration
	data := map[string]interface{}{
		"type":   "direct",
		"direct": scope,
	}
	token, expire, err := h.authJWTGenerateWithExpiration(data, BootExpiration)
	if err != nil {
		log.Errorf("Could not generate boot JWT. err: %v", err)
		return nil, fmt.Errorf("could not generate token")
	}

	res := &BootResponse{
		Token:        token,
		Type:         "direct",
		ResourceType: d.ResourceType,
		ResourceID:   d.ResourceID,
		CustomerID:   d.CustomerID,
		Expire:       expire,
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
