package arihandler

import (
	"database/sql"

	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arievent"
	db "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
	svchandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/svchandler"
)

// ARIHandler arihandler package interface
type ARIHandler interface {
	Connect()
	Run()
}

type ariHandler struct {
	rabbitQueueARIEvent string
	rabbitAddr          string

	rabbitSock rabbitmq.Rabbit

	evtHandler arievent.EventHandler
	reqHandler requesthandler.RequestHandler
	svcHandler svchandler.SVCHandler
	db         db.DBHandler
}

// NewARIHandler creates ARIHandler interface
func NewARIHandler(sqlDB *sql.DB, rabbitAddr, rabbitQueueARIEvent string) ARIHandler {
	handler := &ariHandler{}

	handler.db = db.NewHandler(sqlDB)

	handler.rabbitAddr = rabbitAddr
	handler.rabbitQueueARIEvent = rabbitQueueARIEvent

	return handler
}

// Connect connects to rabbitmq
func (h *ariHandler) Connect() {
	// sock connect
	h.rabbitSock = rabbitmq.NewRabbit(h.rabbitAddr)
	h.rabbitSock.Connect()

	// new handlers
	h.reqHandler = requesthandler.NewRequestHandler(h.rabbitSock)
	h.svcHandler = svchandler.NewSvcHandler(h.reqHandler, h.db)
	h.evtHandler = arievent.NewEventHandler(h.rabbitSock, h.db, h.reqHandler, h.svcHandler)
}

// Run runs the arihandler
func (h *ariHandler) Run() {
	err := h.evtHandler.HandleARIEvent(h.rabbitQueueARIEvent, "call-manager")
	if err != nil {
		log.Errorf("Could not handle the ari event. err: %v", err)
	}
}
