package models

import (
	"gitlab.com/voipbin/bin-manager/api-manager/lib/common"
)

// MaxTimeStamp const string for maximum timestamp.
const MaxTimeStamp = "9999-12-31 23:59:59.999999"

// User data model
type User struct {
	// gorm.Model
	ID           uint64 `gorm:"primary_key"`
	Username     string
	PasswordHash string

	TMCreate string `gorm:"column:tm_create"`
	TMUpdate string `gorm:"column:tm_update"`
	TMDelete string `gorm:"column:tm_delete"`
}

// Serialize serializes user data
func (u *User) Serialize() common.JSON {
	return common.JSON{
		"id":       u.ID,
		"username": u.Username,
	}
}

// Read reads the user info
func (u *User) Read(m common.JSON) {
	u.ID = uint64(m["id"].(float64))
	u.Username = m["username"].(string)
}
