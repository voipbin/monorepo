package transfers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// transfersPOST handles POST /transfers request.
// It starts a transfer the call and returns the result.
// @Summary     Start a transfer
// @Description Transfer the call
// @Produce     json
// @Param       transcribe body     request.BodyTransfersPOST true "Transfer info."
// @Success     200        {object} transfer.Transfer
// @Router      /v1.0/transfers [post]
func transfersPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "transfersPOST",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
	})

	var req request.BodyTransfersPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing transcribesPOST.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// start a transfer
	res, err := serviceHandler.TransferStart(c.Request.Context(), &u, req.TransferType, req.TransfererCallID, req.TransfereeAddresses)
	if err != nil {
		log.Errorf("Could not create a transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
