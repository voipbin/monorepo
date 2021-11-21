package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
)

// AuthLogin generate jwt token of an user
func (h *serviceHandler) AuthLogin(username, password string) (string, error) {
	ctx := context.Background()

	u, err := h.reqHandler.UMV1UserLogin(ctx, username, password)
	if err != nil {
		logrus.Warningf("Could not find userinfo. username: %s", username)
		return "", err
	}

	// if !checkHash(password, u.PasswordHash) {
	// 	logrus.Warningf("The password does not match. username: %s", username)
	// 	return "", fmt.Errorf("password does not match")
	// }

	serialized := u.Serialize()
	token, err := middleware.GenerateToken(serialized)
	if err != nil {
		logrus.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return token, nil
}

// checkHash returns true if the given hashstring is correct
func checkHash(password, hashString string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(password)); err != nil {
		return false
	}

	return true
}

// GenerateHash generates hash from auth
func generateHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}
