package models

import (
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/lib/common"
)

// User data model
type User struct {
	// gorm.Model
	ID           uint64 `gorm:"primary_key" json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`

	TMCreate string `gorm:"column:tm_create" json:"tm_create"`
	TMUpdate string `gorm:"column:tm_update" json:"tm_update"`
	TMDelete string `gorm:"column:tm_delete" json:"tm_delete"`
}

// Serialize serializes user data
func (u *User) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"id":       u.ID,
		"username": u.Username,
	}
}

// Read reads the user info
func (u *User) Read(m map[string]interface{}) {
	u.ID = uint64(m["id"].(float64))
	u.Username = m["username"].(string)
}

// UserExists returns true if the given user is exists
func UserExists(username string) bool {
	var exists User
	if err := db.Where("username = ?", username).First(&exists).Error; err == nil {
		logrus.Debugf("The given user is already exsits. username: %s", username)
		return true
	}

	return false
}

// UserCreate creates a user
func UserCreate(u *User) (*User, error) {
	log := logrus.WithFields(logrus.Fields{
		"Username": u.Username,
	})
	log.Debug("Creating a new user.")

	// create user
	user := User{
		Username:     u.Username,
		PasswordHash: u.PasswordHash,

		TMCreate: common.GetCurTime(),
		TMUpdate: common.GetCurTime(),
		TMDelete: MaxTimeStamp,
	}

	db.NewRecord(user)
	db.Create(&user)

	return &user, nil
}

// UserGetByUsername return user info by username
func UserGetByUsername(username string) (*User, error) {
	// get user
	var user User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// UserGet return user info by id
func UserGet(id int64) (*User, error) {
	// get user
	var user User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
