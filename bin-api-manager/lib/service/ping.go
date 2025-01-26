package service

import "github.com/gin-gonic/gin"

// ping handler
//
//	@Summary		Returns message pong
//	@Description	Used to check the server is alive
//	@Produce		json
//	@Router			/ping [get]
//	@Success		200	"{"message": "pong"}"
//	@BasePath
func GetPing(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
