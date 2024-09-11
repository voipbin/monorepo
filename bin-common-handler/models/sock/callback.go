package sock

// CbMsgConsume is func prototype for message read callback.
type CbMsgConsume func(*Event) error

// CbMsgRPC is func prototype for RPC callback
type CbMsgRPC func(*Request) (*Response, error)
