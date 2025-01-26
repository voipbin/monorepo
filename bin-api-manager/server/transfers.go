package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"
	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// transfersPOST handles POST /transfers request.
// It starts a transfer the call and returns the result.
//
//	@Summary		Start a transfer
//	@Description	Transfer the call
//	@Produce		json
//	@Param			transcribe	body		request.BodyTransfersPOST	true	"Transfer info."
//	@Success		200			{object}	transfer.Transfer
//	@Router			/v1.0/transfers [post]
func (h *server) PostTransfers(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostTransfers",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req openapi_server.PostTransfersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	transfererCallID := uuid.FromStringOrNil(req.TransfererCallId)
	transfereeAddresses := []commonaddress.Address{}
	for _, v := range req.TransfereeAddresses {
		transfereeAddresses = append(transfereeAddresses, ConvertCommonAddress(v))
	}

	res, err := h.serviceHandler.TransferStart(c.Request.Context(), &a, tmtransfer.Type(req.TransferType), transfererCallID, transfereeAddresses)
	if err != nil {
		log.Errorf("Could not create a transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
