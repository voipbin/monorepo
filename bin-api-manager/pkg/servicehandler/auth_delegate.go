package servicehandler

import (
	"context"
	"fmt"
	"unicode"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	metricDelegateIssued = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_delegate_token_issued_total",
		Help: "Total number of delegate tokens issued.",
	}, []string{"issued_by_agent_id"})
)

// DelegateResponse is the response for POST /auth/delegate.
type DelegateResponse struct {
	Token      string    `json:"token"`
	CustomerID uuid.UUID `json:"customer_id"`
	Expire     string    `json:"expire"`
}

// AuthDelegate issues a short-lived JWT granting PermissionCustomerAdmin-equivalent
// access scoped to targetCustomerID. Only PermissionProjectSuperAdmin agents may call this.
func (h *serviceHandler) AuthDelegate(ctx context.Context, a *auth.AuthIdentity, targetCustomerID uuid.UUID, reason string) (*DelegateResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "AuthDelegate",
		"target_customer_id": targetCustomerID,
	})

	// Block recursive delegation
	if a.IsDelegate() {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"denial_reason": "recursive_delegation",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: recursive delegation not permitted", serviceerrors.ErrPermissionDenied)
	}

	// Verify caller has PermissionProjectSuperAdmin
	if !a.HasPermission(amagent.PermissionProjectSuperAdmin) {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           a.AgentID(),
			"denial_reason": "not_superadmin",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: PermissionProjectSuperAdmin required", serviceerrors.ErrPermissionDenied)
	}

	agentID := a.AgentID()

	// Validate reason
	if err := validateDelegateReason(reason); err != nil {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           agentID,
			"denial_reason": "invalid_input",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: %v", serviceerrors.ErrInvalidArgument, err)
	}

	// Verify target customer exists and is not deleted
	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, targetCustomerID)
	if err != nil {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           agentID,
			"denial_reason": "customer_not_found",
		}).Warn("Delegate token request denied")
		return nil, fmt.Errorf("%w: target customer not found", serviceerrors.ErrNotFound)
	}
	if cu.Status == cscustomer.StatusDeleted {
		log.WithFields(logrus.Fields{
			"audit":         true,
			"event":         "delegate_token_denied",
			"sub":           agentID,
			"denial_reason": "customer_deleted",
		}).Warn("Delegate token request denied — customer is deleted")
		return nil, fmt.Errorf("%w: target customer not found", serviceerrors.ErrNotFound)
	}

	// Generate jti
	jti, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate jti. err: %v", err)
		return nil, fmt.Errorf("%w: jti generation failed", serviceerrors.ErrInternal)
	}

	// Generate JWT
	data := map[string]interface{}{
		"type":        string(auth.TypeDelegate),
		"sub":         agentID.String(),
		"aud":         "voipbin-api",
		"jti":         jti.String(),
		"customer_id": targetCustomerID.String(),
	}
	token, expire, err := h.authJWTGenerateWithExpiration(data, DelegateExpiration)
	if err != nil {
		log.Errorf("Could not generate delegate JWT. err: %v", err)
		return nil, fmt.Errorf("%w: token generation failed", serviceerrors.ErrInternal)
	}

	// Write audit log
	log.WithFields(logrus.Fields{
		"audit":              true,
		"event":              "delegate_token_issued",
		"jti":                jti.String(),
		"sub":                agentID,
		"target_customer_id": targetCustomerID,
		"reason":             reason,
		"expire":             expire,
	}).Info("Delegate token issued")

	// Emit metric
	metricDelegateIssued.WithLabelValues(agentID.String()).Inc()

	return &DelegateResponse{
		Token:      token,
		CustomerID: targetCustomerID,
		Expire:     expire,
	}, nil
}

// validateDelegateReason enforces reason field constraints: 10–200 printable chars, no control chars.
func validateDelegateReason(reason string) error {
	if len(reason) < 10 {
		return fmt.Errorf("reason must be at least 10 characters")
	}
	if len(reason) > 200 {
		return fmt.Errorf("reason must be at most 200 characters")
	}
	for _, r := range reason {
		if unicode.IsControl(r) {
			return fmt.Errorf("reason must not contain control characters")
		}
	}
	return nil
}
