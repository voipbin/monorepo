package util

import (
	"github.com/gin-gonic/gin"
	"github.com/unknwon/com"

	"gitlab.com/voipbin/bin-manager/api-manager/pkg/setting"
)

// GetPage get page parameters
func GetPage(c *gin.Context) int {
	result := 0
	page := com.StrTo(c.Query("page")).MustInt()
	if page > 0 {
		result = (page - 1) * setting.AppSetting.PageSize
	}

	return result
}
