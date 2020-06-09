package routers

import (
	"time"

	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	_ "gitlab.com/voipbin/bin-manager/api-manager/docs"

	"gitlab.com/voipbin/bin-manager/api-manager/middleware/jwt"
	"gitlab.com/voipbin/bin-manager/api-manager/routers/api"
)

// InitRouter initialize routing information
func InitRouter() *gin.Engine {
	r := gin.New()

	// Add a ginrus middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	//   - RFC3339 with UTC time format.
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// r.StaticFS("/export", http.Dir(export.GetExcelFullPath()))
	// r.StaticFS("/upload/images", http.Dir(upload.GetImageFullPath()))
	// r.StaticFS("/qrcode", http.Dir(qrcode.GetQrCodeFullPath()))

	r.POST("/auth", api.GetAuth)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// r.POST("/upload", api.UploadImage)

	apiv1 := r.Group("/api/v1")
	apiv1.Use(jwt.JWT())
	{
		//获取标签列表
		// apiv1.GET("/tags", v1.GetTags)
		//新建标签
		// apiv1.POST("/tags", v1.AddTag)
		//更新指定标签
		// apiv1.PUT("/tags/:id", v1.EditTag)
		//删除指定标签
		// apiv1.DELETE("/tags/:id", v1.DeleteTag)
		//导出标签
		// r.POST("/tags/export", v1.ExportTag)
		//导入标签
		// r.POST("/tags/import", v1.ImportTag)

		//获取文章列表
		// apiv1.GET("/articles", v1.GetArticles)
		//获取指定文章
		// apiv1.GET("/articles/:id", v1.GetArticle)
		//新建文章
		// apiv1.POST("/articles", v1.AddArticle)
		//更新指定文章
		// apiv1.PUT("/articles/:id", v1.EditArticle)
		//删除指定文章
		// apiv1.DELETE("/articles/:id", v1.DeleteArticle)
		//生成文章海报
		// apiv1.POST("/articles/poster/generate", v1.GenerateArticlePoster)
	}

	return r
}
