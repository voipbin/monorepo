package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// PostServiceAgentsTransfers starts a transfer of a call the calling agent owns.
// Thin by design: parse the body, one service-handler call. Only the address type and target are taken from the
// request, the service handler builds the address that is passed on.
func (h *server) PostServiceAgentsTransfers(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsTransfers",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostServiceAgentsTransfersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	transfereeAddresses := make([]commonaddress.Address, 0, len(req.TransfereeAddresses))
	for _, v := range req.TransfereeAddresses {
		transfereeAddresses = append(transfereeAddresses, commonaddress.Address{
			Type:   commonaddress.Type(v.Type),
			Target: v.Target,
		})
	}

	res, err := h.serviceHandler.ServiceAgentTransferStart(c.Request.Context(), a, tmtransfer.Type(req.TransferType), uuid.FromStringOrNil(req.TransfererCallId.String()), transfereeAddresses)
	if err != nil {
		log.Errorf("Could not start the transfer. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
