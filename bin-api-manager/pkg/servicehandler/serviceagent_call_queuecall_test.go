package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	qmqueue "monorepo/bin-queue-manager/models/queue"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentCallQueuecallGet(t *testing.T) {
	var (
		customerID      = uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9")
		otherCustomerID = uuid.FromStringOrNil("6e27823d-3b9f-11ef-8a51-f30f1e2ce1e9")
		agentID         = uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a")
		otherAgentID    = uuid.FromStringOrNil("7f38934e-3b9f-11ef-98ac-db226570f09a")
		callID          = uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c")
		masterCallID    = uuid.FromStringOrNil("8996f42a-3f9f-11ef-9c18-0b629aa668a1")
		queuecallID     = uuid.FromStringOrNil("3c8e1f20-7a4b-4d6e-9b1a-5f2c8d0e7a31")
		queueID         = uuid.FromStringOrNil("7d2a9f10-3b5c-4e8a-a6d1-1c4e7b9f2a05")
		tmCreate        = time.Date(2026, 11, 2, 9, 10, 0, 0, time.UTC)
		tmDelete        = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: agentID, CustomerID: customerID},
		Permission: amagent.PermissionCustomerAgent,
	})

	call := func(owner, customer, master uuid.UUID) *cmcall.Call {
		return &cmcall.Call{
			Identity:     commonidentity.Identity{ID: callID, CustomerID: customer},
			OwnerID:      owner,
			MasterCallID: master,
		}
	}
	queuecall := func(customer, serviceAgent uuid.UUID, status qmqueuecall.Status) *qmqueuecall.Queuecall {
		return &qmqueuecall.Queuecall{
			Identity:       commonidentity.Identity{ID: queuecallID, CustomerID: customer},
			QueueID:        queueID,
			Status:         status,
			ServiceAgentID: serviceAgent,
			TMCreate:       &tmCreate,
			// fields that must never reach the response
			ReferenceID: masterCallID,
			GroupcallID: uuid.FromStringOrNil("11111111-3b9f-11ef-98ac-db226570f09a"),
			TagIDs:      []uuid.UUID{uuid.FromStringOrNil("22222222-3b9f-11ef-98ac-db226570f09a")},
		}
	}
	queue := func(customer uuid.UUID, deleted *time.Time) *qmqueue.Queue {
		return &qmqueue.Queue{
			Identity: commonidentity.Identity{ID: queueID, CustomerID: customer},
			Name:     "Support",
			TMDelete: deleted,
		}
	}
	summary := func(status string, name string) *openapi_server.QueueManagerQueuecallSummary {
		id := queuecallID.String()
		qid := queueID.String()
		st := openapi_server.QueueManagerQueuecallStatus(status)
		return &openapi_server.QueueManagerQueuecallSummary{
			Id: &id, QueueId: &qid, QueueName: &name, Status: &st, TmCreate: &tmCreate,
		}
	}
	notFound := requesthandler.ErrNotFound
	errRPC := errors.New("rpc failed")

	type lookup struct {
		referenceID uuid.UUID
		res         *qmqueuecall.Queuecall
		err         error
	}

	tests := []struct {
		name string

		agent *auth.AuthIdentity

		responseCall    *cmcall.Call
		responseCallErr error
		lookups         []lookup // expected QueueV1QueuecallGetByReferenceID calls, in order
		queueGet        bool     // whether QueueV1QueueGet is expected
		responseQueue   *qmqueue.Queue
		responseQueueEr error

		expectRes *openapi_server.QueueManagerQueuecallSummary
		expectErr error // matched with errors.Is; nil means success
	}{
		{
			name:         "found by master_call_id, service status",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:      true,
			responseQueue: queue(customerID, nil),
			expectRes:     summary("service", "Support"),
		},
		{
			name:         "connecting status is reported as is",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, agentID, qmqueuecall.StatusConnecting), nil},
			},
			queueGet:      true,
			responseQueue: queue(customerID, nil),
			expectRes:     summary("connecting", "Support"),
		},
		{
			name:         "found by call id after master_call_id is not found (agent joined the queue directly)",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, nil, notFound},
				{callID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:      true,
			responseQueue: queue(customerID, nil),
			expectRes:     summary("service", "Support"),
		},
		{
			name:         "empty master_call_id looks up only the call id",
			agent:        agent,
			responseCall: call(agentID, customerID, uuid.Nil),
			lookups: []lookup{
				{callID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:      true,
			responseQueue: queue(customerID, nil),
			expectRes:     summary("service", "Support"),
		},
		{
			name:         "empty master_call_id and no queuecall of the call id is not found",
			agent:        agent,
			responseCall: call(agentID, customerID, uuid.Nil),
			lookups: []lookup{
				{callID, nil, notFound},
			},
			expectErr: serviceerrors.ErrNotFound,
		},
		{
			name:         "both candidates not found",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, nil, notFound},
				{callID, nil, notFound},
			},
			expectErr: serviceerrors.ErrNotFound,
		},
		{
			name:         "typed not-found error also falls through to the next candidate",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, nil, cerrors.NotFound("queue-manager", "QUEUECALL_NOT_FOUND", "x")},
				{callID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:      true,
			responseQueue: queue(customerID, nil),
			expectRes:     summary("service", "Support"),
		},
		{
			name:         "other call owner is denied before any queue lookup",
			agent:        agent,
			responseCall: call(otherAgentID, customerID, masterCallID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "empty call owner is denied (fail-closed)",
			agent:        agent,
			responseCall: call(uuid.Nil, customerID, masterCallID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:         "call of another customer is denied",
			agent:        agent,
			responseCall: call(agentID, otherCustomerID, masterCallID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:            "call lookup failure is returned",
			agent:           agent,
			responseCallErr: errRPC,
			expectErr:       errRPC,
		},
		{
			name:            "call not found is passed through as not found",
			agent:           agent,
			responseCallErr: notFound,
			expectErr:       requesthandler.ErrNotFound,
		},
		{
			name:         "queuecall handled by another agent is not found",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, otherAgentID, qmqueuecall.StatusService), nil},
			},
			expectErr: serviceerrors.ErrNotFound,
		},
		{
			name:         "queuecall without a service agent is not found",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, uuid.Nil, qmqueuecall.StatusWaiting), nil},
			},
			expectErr: serviceerrors.ErrNotFound,
		},
		{
			name:         "queuecall of another customer is not found",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(otherCustomerID, agentID, qmqueuecall.StatusService), nil},
			},
			expectErr: serviceerrors.ErrNotFound,
		},
		{
			name:         "queue of another customer is not found",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:      true,
			responseQueue: queue(otherCustomerID, nil),
			expectErr:     serviceerrors.ErrNotFound,
		},
		{
			name:         "queue not found",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:        true,
			responseQueueEr: notFound,
			expectErr:       serviceerrors.ErrNotFound,
		},
		{
			name:         "deleted queue still returns its name",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:      true,
			responseQueue: queue(customerID, &tmDelete),
			expectRes:     summary("service", "Support"),
		},
		{
			name:         "queuecall lookup failure other than not-found stops the search",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, nil, errRPC},
			},
			expectErr: errRPC,
		},
		{
			name:         "queue lookup failure other than not-found is returned",
			agent:        agent,
			responseCall: call(agentID, customerID, masterCallID),
			lookups: []lookup{
				{masterCallID, queuecall(customerID, agentID, qmqueuecall.StatusService), nil},
			},
			queueGet:        true,
			responseQueueEr: errRPC,
			expectErr:       errRPC,
		},
		{
			name: "agent with an empty id never matches an empty call owner",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity:   commonidentity.Identity{ID: uuid.Nil, CustomerID: customerID},
				Permission: amagent.PermissionCustomerAgent,
			}),
			responseCall: call(uuid.Nil, customerID, masterCallID),
			expectErr:    serviceerrors.ErrPermissionDenied,
		},
		{
			name:      "not an agent identity",
			agent:     auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{}),
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.responseCall != nil || tt.responseCallErr != nil {
				mockReq.EXPECT().CallV1CallGet(ctx, callID).Return(tt.responseCall, tt.responseCallErr)
			}
			prev := (*gomock.Call)(nil)
			for _, l := range tt.lookups {
				c := mockReq.EXPECT().QueueV1QueuecallGetByReferenceID(ctx, l.referenceID).Return(l.res, l.err)
				if prev != nil {
					c.After(prev)
				}
				prev = c
			}
			if tt.queueGet {
				mockReq.EXPECT().QueueV1QueueGet(ctx, queueID).Return(tt.responseQueue, tt.responseQueueEr)
			}

			res, err := h.ServiceAgentCallQueuecallGet(ctx, tt.agent, callID)
			if tt.expectErr == nil {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				if tt.expectErr == errRPC && errors.Is(err, serviceerrors.ErrNotFound) {
					t.Errorf("Wrong error. a backend failure must not become not found: %v", err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil result on error, got: %v", res)
				}
				return
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
