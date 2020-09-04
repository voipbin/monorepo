package serviceuser

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/models"
)

// UserCreate creates a user
func UserCreate(username, password string) (*models.User, error) {
	log := logrus.WithFields(logrus.Fields{
		"Username": username,
	})
	log.Debug("Creating a new user.")

	// check existence
	if models.UserExists(username) == true {
		log.Info("User is already existing.")
		return nil, fmt.Errorf("user already exist")
	}

	// generate hash password
	hashPassword, err := models.GenerateHash(password)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return nil, err
	}

	// create user
	user := models.User{
		Username:     username,
		PasswordHash: hashPassword,
	}
	res, err := models.UserCreate(&user)
	if err != nil {
		log.Errorf("Could not create a new user. err: %v", err)
		return nil, err
	}

	return res, nil
}
