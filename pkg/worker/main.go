package worker

import (
	"database/sql"

	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler"
	callhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// Worker arihandler package interface
type Worker interface {
	Connect()
	Run()
}

type worker struct {
	rabbitQueueARIEvent string
	rabbitAddr          string

	rabbitSock rabbitmq.Rabbit

	ariHandler  arihandler.ARIHandler
	reqHandler  requesthandler.RequestHandler
	callHandler callhandler.CallHandler
	db          dbhandler.DBHandler
}

// NewWorker creates worker interface
func NewWorker(sqlDB *sql.DB, rabbitAddr, rabbitQueueARIEvent string) Worker {
	db := dbhandler.NewHandler(sqlDB)

	handler := &worker{
		db:                  db,
		rabbitAddr:          rabbitAddr,
		rabbitQueueARIEvent: rabbitQueueARIEvent,
	}

	return handler
}

// Connect connects to rabbitmq
func (h *worker) Connect() {
	// sock connect
	h.rabbitSock = rabbitmq.NewRabbit(h.rabbitAddr)
	h.rabbitSock.Connect()

	// new handlers
	h.reqHandler = requesthandler.NewRequestHandler(h.rabbitSock)
	h.callHandler = callhandler.NewSvcHandler(h.reqHandler, h.db)
	h.ariHandler = arihandler.NewARIHandler(h.rabbitSock, h.db, h.reqHandler, h.callHandler)
}

// Run runs the arihandler
func (h *worker) Run() {

	err := h.ariHandler.Run(h.rabbitQueueARIEvent, "call-manager")
	if err != nil {
		log.Errorf("Could not handle the ari event. err: %v", err)
	}
}
