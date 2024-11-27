package servicehandler

import (
	"errors"
	"fmt"
	"monorepo/bin-api-manager/lib/common"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

func (h *serviceHandler) JWTGenerate(data map[string]interface{}) (string, error) {
	log := logrus.WithField("func", "JWTGenerate")
	log.Debugf("Generating the token. data: %v", data)

	claims := jwt.MapClaims{}
	for k, v := range data {
		claims[k] = v
	}

	// token is valid for 7 days
	claims["expire"] = h.utilHandler.TimeGetCurTimeAdd(common.TokenExpiration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	res, err := token.SignedString(h.jwtKey)
	if err != nil {
		return "", err
	}

	return res, nil
}

func (h *serviceHandler) JWTParse(tokenString string) (common.JSON, error) {
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
