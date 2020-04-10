package arihandler

import (
	"time"
)

// ARIHandler arihandler package interface
type ARIHandler interface {
	ReceiveEventQueue(addr, queue, receiver string)
}

// ARIEvent is the structure for ARI event parse.
type ARIEvent struct {
	eventType   string
	application string
	asteriskID  string
	timestamp   time.Time
	event       interface{}
}
