package serviceauth

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager/models"
)

// Auth structure
type Auth struct {
	Username string
	Password string
}

// GetUser returns User info if the credential is correct
func (a *Auth) GetUser() *models.User {

	auth := models.Auth{
		Username: a.Username,
		Password: a.Password,
	}

	return auth.GetUser()
}

// Login validates auth credential and return the generated jwt token
func (a *Auth) Login() (string, error) {
	// check credential is valid
	user := a.GetUser()
	if user == nil {
		return "", fmt.Errorf("no user found")
	}

	serialized := user.Serialize()
	token, err := middleware.GenerateToken(serialized)
	if err != nil {
		logrus.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return token, nil
}
