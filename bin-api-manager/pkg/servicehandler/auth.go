package servicehandler

import (
	"context"
	"errors"
	"fmt"
	"monorepo/bin-api-manager/pkg/serviceerrors"

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
