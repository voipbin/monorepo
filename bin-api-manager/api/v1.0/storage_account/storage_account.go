package storage_accounts

import (
	_ "monorepo/bin-storage-manager/models/account" // for swag use

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// storageAccountGet handles GET /storage_account request.
// It returns detail storage account info.
//
//	@Summary		Returns detail storage account info.
//	@Description	Returns detail storage account info of the given customer.
//	@Produce		json
//	@Success		200	{object}	account.WebhookMessage
//	@Router			/v1.0/storage_account [get]
func storageAccountGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageAccountGet",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing storageAccountsIDGet.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.StorageAccountGetByCustomerID(c.Request.Context(), &a)
	if err != nil {
		log.Errorf("Could not get a storage account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
