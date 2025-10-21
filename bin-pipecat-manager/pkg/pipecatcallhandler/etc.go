package pipecatcallhandler

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"net"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
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
