package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/api-manager/lib/common"
	"gitlab.com/voipbin/bin-manager/api-manager/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/database/models"
	"golang.org/x/crypto/bcrypt"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	auth.POST("/register", middleware.Authorized, register)
	auth.POST("/login", login)
}

func register(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	logrus.Debug("registering a new user.")

	type RequestBody struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	var body RequestBody
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	// check existancy
	var exists models.User
	if err := db.Where("username = ?", body.Username).First(&exists).Error; err == nil {
		logrus.Debugf("The given user is already exsits. username: %s", body.Username)
		c.AbortWithStatus(409)
		return
	}

	hash, err := hash(body.Password)
	if err != nil {
		c.AbortWithStatus(500)
		return
	}

	// create user
	user := models.User{
		Username:     body.Username,
		PasswordHash: hash,

		TMCreate: common.GetCurTime(),
		TMUpdate: common.GetCurTime(),
		TMDelete: models.MaxTimeStamp,
	}

	db.NewRecord(user)
	db.Create(&user)

	serialized := user.Serialize()
	token, err := middleware.GenerateToken(serialized)
	if err != nil {
		logrus.Errorf("Could not generate token. err: %v", err)
		c.AbortWithStatus(500)
		return
	}

	c.SetCookie("token", token, 60*60*24*7, "/", "", false, true)

	c.JSON(200, common.JSON{
		"user":  user.Serialize(),
		"token": token,
	})
}

func login(c *gin.Context) {
	logrus.Debug("Login.")

	db := c.MustGet("db").(*gorm.DB)
	type RequestBody struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	var body RequestBody
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	// get user
	var user models.User
	if err := db.Where("username = ?", body.Username).First(&user).Error; err != nil {
		c.AbortWithStatus(404)
		return
	}

	// verify password
	if checkHash(body.Password, user.PasswordHash) != true {
		c.AbortWithStatus(401)
		return
	}

	serialized := user.Serialize()
	token, err := middleware.GenerateToken(serialized)
	if err != nil {
		logrus.Errorf("Could not create a jwt token. err: %v", err)
		c.AbortWithStatus(500)
		return
	}

	c.SetCookie("token", token, 60*60*24*7, "/", "", false, true)
	c.JSON(200, common.JSON{
		"user":  user.Serialize(),
		"token": token,
	})

}

func checkHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}
