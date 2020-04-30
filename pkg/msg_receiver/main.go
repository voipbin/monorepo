package msgreceiver

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	db "gitlab.com/voipbin/bin-manager/flow-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow"
	flowhandler "gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow_handler"
	rabbitmq "gitlab.com/voipbin/bin-manager/flow-manager/pkg/rabbitmq"
)

// MsgReceiver type
type MsgReceiver interface {
	Run() error
}

type msgReceiver struct {
	rabbitAddr string
	db         db.DBHandler
	sock       rabbitmq.Rabbit

	queueName    string // queue name for message receive
	consumerName string // consumer name for message receive

	flowHandler flowhandler.FlowHandler
}

var (
	metricsNamespace = "flow_manager"

	promRequestURITotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "request_type_total",
			Help:      "Total number of received request types",
		},
		[]string{"type", "method"},
	)
)

func init() {
	prometheus.MustRegister(
		promRequestURITotal,
	)
}

// NewMsgReceiver creates and returns MsgReceiver interface
func NewMsgReceiver(rabbitAddr string, db db.DBHandler, queueName, consumerName string) MsgReceiver {
	flowHandler := flowhandler.NewFlowHandler(db)

	res := &msgReceiver{
		rabbitAddr: rabbitAddr,
		db:         db,

		queueName:    queueName,
		consumerName: consumerName,

		flowHandler: flowHandler,
	}

	return res
}

// Run runs the message receiver.
func (r *msgReceiver) Run() error {
	r.sock = rabbitmq.NewRabbit(r.rabbitAddr)
	r.sock.Connect()

	// declare receive queue
	if err := r.sock.DeclareQueue(r.queueName, true, false, false, false); err != nil {
		return err
	}

	// register message processor
	r.sock.ConsumeMessage(r.queueName, r.consumerName, r.process)
	return nil
}

func (r *msgReceiver) process(req *rabbitmq.Request) (*rabbitmq.Response, error) {
	log.WithFields(log.Fields{
		"uri":       req.URI,
		"method":    req.Method,
		"data_type": req.DataType,
		"data":      req.Data,
	}).Infof("Request received.")
	promRequestURITotal.WithLabelValues(req.URI, string(req.Method)).Inc()

	if m, _ := regexp.MatchString("/v1/flows/.*/actions/.*", req.URI); m == true {
		switch req.Method {
		case rabbitmq.RequestMethodGet:
			return r.v1FlowsIDActionsIDGet(req)
		}
	}

	if m, _ := regexp.MatchString("/v1/flows/*", req.URI); m == true {
		switch req.Method {
		case rabbitmq.RequestMethodGet:
			return r.v1FlowsIDGet(req)
		}
	}

	if m, _ := regexp.MatchString("/v1/flows", req.URI); m == true {
		switch req.Method {
		case rabbitmq.RequestMethodPost:
			return r.v1FlowsPost(req)
		}
	}

	// return 404
	res := &rabbitmq.Response{
		StatusCode: 404,
	}

	return res, nil
}

func (r *msgReceiver) v1FlowsIDGet(req *rabbitmq.Request) (*rabbitmq.Response, error) {

	return nil, nil
}

func (r *msgReceiver) v1FlowsPost(req *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	flow := &flow.Flow{}
	if err := json.Unmarshal([]byte(req.Data), flow); err != nil {
		return nil, err
	}

	resFlow, err := r.flowHandler.FlowCreate(ctx, flow)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resFlow)
	if err != nil {
		return nil, err
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       string(data),
	}

	return res, nil
}

// handlerFlowsIDActionsIDGet handles
// /v1/flows/{id}/actions/{id} GET
func (r *msgReceiver) v1FlowsIDActionsIDGet(req *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/actions/ab1f7732-8a74-11ea-98f6-9b02a042df6a"
	tmpVals := strings.Split(req.URI, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])
	actionID := uuid.FromStringOrNil(tmpVals[5])
	revision := flow.FlowRevisionLatest

	resAction, err := r.flowHandler.ActionGet(ctx, flowID, revision, actionID)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resAction)
	if err != nil {
		return nil, err
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       string(data),
	}

	return res, nil
}
