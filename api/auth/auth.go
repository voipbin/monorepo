package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/voipbin/bin-manager/api-manager/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager/service/serviceauth"
	"gitlab.com/voipbin/bin-manager/api-manager/service/serviceuser"
	// "gitlab.com/voipbin/bin-manager/api-manager/models"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	auth.POST("/register", middleware.Authorized, register)
	auth.POST("/login", login)
}

func register(c *gin.Context) {
	// db := c.MustGet("db").(*gorm.DB)
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

	// create an user
	user, err := serviceuser.UserCreate(body.Username, body.Password)
	if err != nil {
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, user)
}

func login(c *gin.Context) {
	logrus.Debug("Login.")

	type RequestBody struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	var body RequestBody
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	auth := serviceauth.Auth{
		Username: body.Username,
		Password: body.Password,
	}

	token, err := auth.Login()
	if err != nil {
		logrus.Debugf("Login failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.SetCookie("token", token, 60*60*24*7, "/", "", false, true)
	c.JSON(200, map[string]interface{}{
		"username": auth.Username,
		"token":    token,
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
