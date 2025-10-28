package pipecatcallhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"net"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func (h *pipecatcallHandler) setRunnerInfo(
	pc *pipecatcall.Pipecatcall,
	listener net.Listener,
	server *http.Server,
) {

	pc.RunnerListener = listener
	pc.RunnerPort = listener.Addr().(*net.TCPAddr).Port
	pc.RunnerServer = server
}

func (h *pipecatcallHandler) setRunnerWebsocket(pc *pipecatcall.Pipecatcall, ws *websocket.Conn) {
	pc.RunnerWebsocket = ws
}

func (h *pipecatcallHandler) setAsteriskInfo(pc *pipecatcall.Pipecatcall, streamingID uuid.UUID, conn net.Conn) {
	pc.AsteriskConn = conn
	pc.AsteriskStreamingID = streamingID
}

func (h *pipecatcallHandler) SendMessage(ctx context.Context, id uuid.UUID, messageID string, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	pc, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall info")
	}

	tmpID := h.utilHandler.UUIDCreate()
	res := message.Message{
		Identity: commonidentity.Identity{
			ID:         tmpID,
			CustomerID: pc.CustomerID,
		},

		PipecatcallID:            pc.ID,
		PipecatcallReferenceType: pc.ReferenceType,
		PipecatcallReferenceID:   pc.ReferenceID,

		Text: messageText,
	}
	h.notifyHandler.PublishEvent(ctx, message.EventTypeBotTranscription, res)

	if errSend := h.pipecatframeHandler.SendRTVIText(pc, messageID, messageText, runImmediately, audioResponse); errSend != nil {
		return nil, errors.Wrapf(errSend, "could not send the message to pipecatcall")
	}

	return &res, nil
}
