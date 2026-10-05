package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetAiModels handles GET /ai_models.
// It returns the whole engine model catalog in a single page.
func (h *server) GetAiModels(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAiModels",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	tmps, err := h.serviceHandler.AIModelList(c.Request.Context(), a)
	if err != nil {
		log.Errorf("Could not get the AI model list. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	res := GenerateListResponse(tmps, "")
	c.JSON(200, res)
}
