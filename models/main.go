package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // imported for gorm
	"golang.org/x/crypto/bcrypt"
)

var db *gorm.DB

// MaxTimeStamp const string for maximum timestamp.
const MaxTimeStamp = "9999-12-31 23:59:59.999999"

// Setup initiates model database
func Setup(dsn string) error {
	var err error
	db, err = gorm.Open("mysql", dsn)
	db.LogMode(true)
	if err != nil {
		return err
	}

	return nil
}

// GenerateHash generates hash from auth
func GenerateHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}
