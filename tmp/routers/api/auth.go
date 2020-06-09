package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/pkg/app"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/e"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/util"
	"gitlab.com/voipbin/bin-manager/api-manager/service/auth_service"
)

type auth struct {
	Username string `valid:"Required; MaxSize(50)"`
	Password string `valid:"Required; MaxSize(50)"`
}

// @Summary Get Auth
// @Produce  json
// @Param username query string true "userName"
// @Param password query string true "password"
// @Success 200 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /auth [get]
func GetAuth(c *gin.Context) {
	appG := app.Gin{C: c}

	type RequestBody struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var body RequestBody
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	c.MustGet()
	// valid := validation.Validation{}

	// username := c.PostForm("username")
	// password := c.PostForm("password")
	logrus.Debugf("Request auth request. username: %s, password: %d", body.Username, len(body.Password))

	// a := auth{Username: username, Password: password}
	// ok, _ := valid.Valid(&a)

	// if !ok {
	// 	app.MarkErrors(valid.Errors)
	// 	appG.Response(http.StatusBadRequest, e.INVALID_PARAMS, nil)
	// 	return
	// }

	authService := auth_service.Auth{Username: body.Username, Password: body.Password}
	isExist, err := authService.Check()
	if err != nil {
		appG.Response(http.StatusInternalServerError, e.ERROR_AUTH_CHECK_TOKEN_FAIL, nil)
		return
	}

	if !isExist {
		appG.Response(http.StatusUnauthorized, e.ERROR_AUTH, nil)
		return
	}

	token, err := util.GenerateToken(body.Username, body.Password)
	if err != nil {
		appG.Response(http.StatusInternalServerError, e.ERROR_AUTH_TOKEN, nil)
		return
	}

	appG.Response(http.StatusOK, e.SUCCESS, map[string]string{
		"token": token,
	})
}
