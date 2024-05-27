package storage_accounts

import (
	_ "monorepo/bin-storage-manager/models/account" // for swag use

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// storageAccountsGet handles GET /storage_accounts request.
// It gets a list of storage accounts with the given info.
//
//	@Summary		Gets a list of storage accounts.
//	@Description	Gets a list of storage accounts
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyStorageAccountsGET
//	@Router			/v1.0/storage_accounts [get]
func storageAccountsGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageAccountsGet",
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

	var req request.ParamStorageAccountsGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get tmpRes
	tmpRes, err := serviceHandler.StorageAccountGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a storage list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmpRes) > 0 {
		nextToken = tmpRes[len(tmpRes)-1].TMCreate
	}
	res := response.BodyStorageAccountsGET{
		Result: tmpRes,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// storageAccountsPost handles POST /storage_accounts request.
// It creates a new storage account with the given info and returns created storage account info.
//
//	@Summary		Create a new storage account and returns detail created storage account info.
//	@Description	Create a new storage account and returns detail created storage account info.
//	@Produce		json
//	@Param			customer	body		request.BodyStorageAccountsPOST	true	"customer info."
//	@Success		200			{object}	account.WebhookMessage
//	@Router			/v1.0/storage_accounts [post]
func storageAccountsPost(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageAccountsPost",
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

	var req request.BodyStorageAccountsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Creating a customer.")

	// create a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.StorageAccountCreate(c.Request.Context(), &a, req.CustomerID)
	if err != nil {
		log.Errorf("Could not create a storage account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// storageAccountsIDGet handles GET /storage_accounts/{id} request.
// It returns detail storage account info.
//
//	@Summary		Returns detail storage account info.
//	@Description	Returns detail storage account info of the given storage account id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the storage account"
//	@Success		200	{object}	account.WebhookMessage
//	@Router			/v1.0/storage_accounts/{id} [get]
func storageAccountsIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageAccountsIDGet",
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
	res, err := serviceHandler.StorageAccountGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// storageAccountsIDDelete handles DELETE /storage_accounts/{id} request.
// It returns detail storage account info.
//
//	@Summary		Returns detail storage account info.
//	@Description	Returns detail storage account info of the given storage account id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the storage account"
//	@Success		200	{object}	account.WebhookMessage
//	@Router			/v1.0/storage_accounts/{id} [put]
func storageAccountsIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "storageAccountsIDDelete",
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

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.StorageAccountDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the storage account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
