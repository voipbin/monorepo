package database

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // imported for gorm
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/database/models"
)

// Init intializes the database
func Init(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open("mysql", dsn)
	db.LogMode(true)
	if err != nil {
		return nil, err
	}

	models.Migrate(db)

	return db, nil
}

// Inject injects database to gin context
func Inject(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}
