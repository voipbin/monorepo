package models

import "github.com/jinzhu/gorm"

// Migrate automigrates models using ORM
func Migrate(db *gorm.DB) {
	// db.AutoMigrate(&User{})

	// db.Model()
}
