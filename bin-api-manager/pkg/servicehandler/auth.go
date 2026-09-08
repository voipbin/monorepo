package servicehandler

import (
	"context"
	"errors"
	"fmt"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// AuthLogin generate jwt token of an customer
func (h *serviceHandler) AuthLogin(ctx context.Context, username string, password string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AuthLogin",
		"username": username,
		"password": len(password),
	})

	// agent login
	a, err := h.reqHandler.AgentV1Login(ctx, 30000, username, password)
	if err != nil {
		log.Warningf("Could not login the agent. err: %v", err)
		return "", err
	}
	log.WithField("agent", a).Debugf("Found agent info. agent_id: %s, customer_id: %s", a.ID, a.CustomerID)

	// Refuse to mint a token for an account that the v1 gate would reject
	// anyway (VOIP-1491). Without this an expired-account holder can keep
	// stamping fresh 7-day JWTs and simply wait for the gate's customer
	// lookup to fail open during a customer-manager/RabbitMQ wobble, at
	// which point all 414 v1 routes open up. Closing the issuing window is
	// what removes that amplification, not the gate alone.
	//
	// ORDER MATTERS: this runs AFTER AgentV1Login has verified the password.
	// Checking account status first would let an unauthenticated caller tell
	// "no such user" apart from "expired user" and enumerate usernames --
	// the same reason AuthPasswordForgot swallows every error.
	//
	// The predicate is a DENY-LIST on purpose. AuthBoot's allow-list
	// (status != active) must NOT be copied here:
	//   - 'initial' is a normal user inside the 72h post-signup verification
	//     window (VOIP-1490); blocking it would break onboarding.
	//   - 'frozen' users must still be able to log in, because the frozen
	//     self-recovery UX calls DELETE /auth/unregister, which sits behind
	//     authentication -- blocking login there would delete a shipped
	//     recovery path.
	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, a.CustomerID)
	if err != nil {
		// Fail CLOSED. The gate stays fail-open so existing traffic survives
		// an outage; if login also failed open, the outage window would double
		// as a credential-issuing window and hand back exactly the
		// amplification this check removes. The cost is explicit and accepted:
		// while customer-manager is down nobody gets a new session.
		//
		// This is not an expiry determination, so it must NOT be reported as
		// one -- it stays an opaque credential-shaped failure to the caller.
		log.Errorf("Could not get the customer info. err: %v", err)
		return "", fmt.Errorf("could not get the customer info: %w", err)
	}

	switch cu.Status {
	case cscustomer.StatusExpired:
		log.Infof("The customer is expired. customer_id: %s", cu.ID)
		return "", fmt.Errorf("%w: customer_id: %s", serviceerrors.ErrAccountExpired, cu.ID)

	case cscustomer.StatusDeleted:
		// Reachable because the login filter is AgentGetByUsername's
		// deleted=false, which only helps when the customer_deleted cascade
		// actually soft-deleted the agent -- and that cascade is known to miss
		// customers (VOIP-1395). A missed customer keeps a live agent and can
		// still log in today.
		log.Infof("The customer is deleted. customer_id: %s", cu.ID)
		return "", fmt.Errorf("%w: customer_id: %s", serviceerrors.ErrAccountDeleted, cu.ID)
	}

	data := map[string]interface{}{
		"type":  "agent",
		"agent": a,
	}

	res, err := h.AuthJWTGenerate(data)
	if err != nil {
		log.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("%w: could not create a jwt token", err)
	}

	return res, nil
}

func (h *serviceHandler) AuthJWTGenerate(data map[string]interface{}) (string, error) {
	log := logrus.WithField("func", "JWTGenerate")
	log.Debugf("Generating the token. data: %v", data)

	token, _, err := h.authJWTGenerateWithExpiration(data, TokenExpiration)
	return token, err
}

func (h *serviceHandler) AuthJWTParse(ctx context.Context, tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// don't forget to validate the alg is what you expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", serviceerrors.ErrInvalidArgument, token.Header["alg"])
		}

		return h.jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	res := token.Claims.(jwt.MapClaims)

	curTime := h.utilHandler.TimeGetCurTime()
	if res["expire"].(string) < curTime {
		return nil, errors.New("token expired")
	}

	return res, nil
}

// AuthPasswordForgot requests a password reset token and sends the reset email.
// Always returns nil to prevent username enumeration.
func (h *serviceHandler) AuthPasswordForgot(ctx context.Context, username string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AuthPasswordForgot",
		"username": username,
	})
	log.Debug("Processing password forgot request.")

	if err := h.reqHandler.AgentV1PasswordForgot(ctx, 30000, username); err != nil {
		log.Infof("Could not process password forgot. err: %v", err)
	}

	return nil
}

// AuthPasswordReset validates the token and updates the password.
func (h *serviceHandler) AuthPasswordReset(ctx context.Context, token string, password string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "AuthPasswordReset",
	})
	log.Debug("Processing password reset request.")

	if err := h.reqHandler.AgentV1PasswordReset(ctx, 30000, token, password); err != nil {
		log.Errorf("Could not reset password. err: %v", err)
		return fmt.Errorf("%w: password reset failed", serviceerrors.ErrInternal)
	}

	return nil
}
