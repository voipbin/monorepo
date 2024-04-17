package billingaccounts

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// billingAccountsPOST handles POST /billing_accounts request.
// It creates a new billing account and returns created billing account.
//	@Summary		Create a new billing account
//	@Description	create a new billing account
//	@Produce		json
//	@Param			call	body		request.BodyCallsPOST	true	"The call detail"
//	@Success		200		{object}	account.Account
//	@Router			/v1.0/billing_accounts [post]
func billingAccountsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingAccountsPOST",
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

	var req request.BodyBillingAccountsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create billing account
	res, err := serviceHandler.BillingAccountCreate(c.Request.Context(), &a, req.Name, req.Detail, req.PaymentType, req.PaymentMethod)
	if err != nil {
		log.Errorf("Could not create a billing account. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// billingaccountsGET handles GET /billingaccounts request.
// It returns list of billing accounts of the given customer.

//	@Summary		Get list of billing accounts
//	@Description	get list of the customer's billing accounts
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyBillingAccountsGET
//	@Router			/v1.0/billing_accounts [get]
func billingaccountsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingaccountsGET",
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

	var requestParam request.ParamBillingAccountsGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("callsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	tmps, err := serviceHandler.BillingAccountGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get billing accounts info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyBillingAccountsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// billingAccountsIDDelete handles DELETE /billing_accounts/<billing_account-id> request.
// It deletes the billing_account.
//	@Summary		delete billing account
//	@Description	Delete billing account of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the billing_account"
//	@Success		200	{object}	account.Account
//	@Router			/v1.0/billing_accounts/{id} [delete]
func billingAccountsIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingAccountsIDDelete",
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
	log.Debug("Executing callsIDDelete.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := serviceHandler.BillingAccountDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// billingAccountsIDGET handles GET /billing_accounts/{id} request.
// It returns detail billing account info.
//	@Summary		Get detail billing account info.
//	@Description	Returns detail billing account info of the given call id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the billing account"
//	@Success		200	{object}	account.Account
//	@Router			/v1.0/billing_accounts/{id} [get]
func billingAccountsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingAccountsIDGET",
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
	log = log.WithField("billing_account_id", id)
	log.Debug("Executing billingAccountsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.BillingAccountGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a billing account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// billingAccountsIDPut handles PUT /billing_accounts/<billing_account-id> request.
// It updates the billing_account.
//	@Summary		Update billing account
//	@Description	Update billing account of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the billing_account"
//	@Success		200	{object}	account.Account
//	@Router			/v1.0/billing_accounts/{id} [put]
func billingAccountsIDPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingAccountsIDPut",
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
	log.Debug("Executing callsIDDelete.")

	var req request.BodyBillingAccountsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the reqeust. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.BillingAccountUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// billingAccountsIDPaymentInfoPut handles PUT /billing_accounts/<billing_account-id>/payment_info request.
// It updates the billing_account.
//	@Summary		Update billing account's payment info
//	@Description	Update billing account's payment info of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the billing_account"
//	@Success		200	{object}	account.Account
//	@Router			/v1.0/billing_accounts/{id}/payment_info [put]
func billingAccountsIDPaymentInfoPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingAccountsIDPaymentInfoPut",
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
	log.Debug("Executing callsIDDelete.")

	var req request.BodyBillingAccountsIDPaymentInfoPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the reqeust. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := serviceHandler.BillingAccountUpdatePaymentInfo(c.Request.Context(), &a, id, req.PaymentType, req.PaymentMethod)
	if err != nil {
		log.Errorf("Could not update the payment. info err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// billingAccountsIDGET handles POST /billing_accounts/{id}/balance_add_force request.
// Adds the given balance to the billing account.
//	@Summary		Adds the given balance to the billing account.
//	@Description	Adds the given balance to the billing account.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the billing account"
//	@Success		200	{object}	account.Account
//	@Router			/v1.0/billing_accounts/{id}/balance_add_force [post]
func billingAccountsIDBalanceAddForcePOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingAccountsIDBalanceAddForcePOST",
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
	log = log.WithField("billing_account_id", id)
	log.Debug("Executing billingAccountsIDBalanceAddForcePOST.")

	var req request.BodyBillingAccountsIDBalanceAddForcePOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.BillingAccountAddBalanceForce(c.Request.Context(), &a, id, req.Balance)
	if err != nil {
		log.Errorf("Could not add the balance to the billing account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// billingAccountsIDGET handles POST /billing_accounts/{id}/balance_add_force request.
// Adds the given balance to the billing account.
//	@Summary		Adds the given balance to the billing account.
//	@Description	Adds the given balance to the billing account.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the billing account"
//	@Success		200	{object}	account.Account
//	@Router			/v1.0/billing_accounts/{id}/balance_add_force [post]
func billingAccountsIDBalanceSubtractForcePOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingAccountsIDBalanceSubtractForcePOST",
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
	log = log.WithField("billing_account_id", id)
	log.Debug("Executing billingAccountsIDBalanceSubtractForcePOST.")

	var req request.BodyBillingAccountsIDBalanceSubtractForcePOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.BillingAccountSubtractBalanceForce(c.Request.Context(), &a, id, req.Balance)
	if err != nil {
		log.Errorf("Could not subtract the balance from the billing account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
