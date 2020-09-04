package models

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Auth struct
type Auth struct {
	Username string
	Password string
}

// GetUser returns user if the given confidencial is valid
func (a *Auth) GetUser() *User {
	// get user
	var user User
	if err := db.Where("username = ?", a.Username).First(&user).Error; err != nil {
		return nil
	}

	// verify password
	if a.checkHash(user.PasswordHash) != true {
		logrus.Debugf("Could not pass the hash verification. username: %s", a.Username)
		return nil
	}

	return &user
}

// checkHash returns true if the given hashstring is correct
func (a *Auth) checkHash(hashString string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(a.Password)); err != nil {
		return false
	}

	return true
}
