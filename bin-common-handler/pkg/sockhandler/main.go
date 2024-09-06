package sockhandler

import "monorepo/bin-common-handler/models/sock"

// CbMsgConsume is func prototype for message read callback.
type CbMsgConsume func(*sock.Event) error

// CbMsgRPC is func prototype for RPC callback
type CbMsgRPC func(*sock.Request) (*sock.Response, error)

type SockHandler interface {
}
