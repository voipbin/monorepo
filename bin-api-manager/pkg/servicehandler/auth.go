package servicehandler

import (
	"context"
	"errors"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/dgrijalva/jwt-go"
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
		"agent": a,
	}

	res, err := h.AuthJWTGenerate(data)
	if err != nil {
		log.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return res, nil
}

func (h *serviceHandler) AuthJWTGenerate(data map[string]interface{}) (string, error) {
	log := logrus.WithField("func", "JWTGenerate")
	log.Debugf("Generating the token. data: %v", data)

	claims := jwt.MapClaims{}
	for k, v := range data {
		claims[k] = v
	}

	// token is valid for 7 days
	claims["expire"] = h.utilHandler.TimeGetCurTimeAdd(TokenExpiration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	res, err := token.SignedString(h.jwtKey)
	if err != nil {
		return "", err
	}

	return res, nil
}

func (h *serviceHandler) AuthJWTParse(ctx context.Context, tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// don't forget to validate the alg is what you expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
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

func (h *serviceHandler) AuthAccesskeyParse(ctx context.Context, accesskey string) (map[string]interface{}, error) {

	ak, err := h.AccesskeyRawGetByToken(ctx, accesskey)
	if err != nil {
		return nil, err
	}

	curTime := h.utilHandler.TimeGetCurTime()
	if ak.TMExpire < curTime {
		return nil, errors.New("token expired")
	}

	// generate dummy agent with
	dummyAgent := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         ak.ID,
			CustomerID: ak.CustomerID,
		},
		Name:   ak.Name,
		Detail: ak.Detail,

		Permission: amagent.PermissionCustomerAdmin,
	}

	res := map[string]interface{}{
		"agent": dummyAgent,
	}

	return res, nil
}
