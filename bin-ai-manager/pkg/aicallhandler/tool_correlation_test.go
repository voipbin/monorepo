package aicallhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	tmcorrelation "monorepo/bin-timeline-manager/models/correlation"
)

func Test_toolHandleGetCorrelation(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		mockSetup func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall)

		expectRes *messageContent
	}{
		{
			name: "own session - has activeflow and resources",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000001"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000001"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000001"),
			},
			tool: &message.ToolCall{
				ID:   "tool-1",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), a.ActiveflowID).Return(&tmcorrelation.Correlation{
					ResourceID:    a.ActiveflowID,
					ResourceFound: true,
					ActiveflowID:  a.ActiveflowID,
					Resources: []*tmcorrelation.PublisherGroup{
						{
							Publisher: "call-manager",
							Resources: []*tmcorrelation.CorrelatedResource{
								{
									ID:         uuid.FromStringOrNil("44444444-0000-4000-8000-000000000001"),
									DataType:   "call",
									EventTypes: []string{"call_created", "call_hangup"},
								},
							},
						},
						{
							Publisher: "transcribe-manager",
							Resources: []*tmcorrelation.CorrelatedResource{
								{
									ID:         uuid.FromStringOrNil("77777777-0000-4000-8000-000000000001"),
									DataType:   "transcribe",
									EventTypes: []string{},
								},
							},
						},
					},
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), a.ActiveflowID).Return(&fmactiveflow.Activeflow{
					Identity: commonidentity.Identity{
						ID:         a.ActiveflowID,
						CustomerID: a.CustomerID,
					},
				}, nil)
			},

			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-1",
				ResourceType: "correlation",
				ResourceID:   "33333333-0000-4000-8000-000000000001",
				Message:      "Activeflow 33333333-0000-4000-8000-000000000001 is linked to:\n- call-manager: 1 resource(s)\n  - call 44444444-0000-4000-8000-000000000001 (events: call_created, call_hangup)\n- transcribe-manager: 1 resource(s)\n  - resource 77777777-0000-4000-8000-000000000001\n",
			},
		},
		{
			name: "supplied owned id - ownership confirmed",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000002"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000002"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000002"),
			},
			tool: &message.ToolCall{
				ID:   "tool-2",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{"resource_id":"55555555-0000-4000-8000-000000000002"}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				targetID := uuid.FromStringOrNil("55555555-0000-4000-8000-000000000002")
				afID := uuid.FromStringOrNil("66666666-0000-4000-8000-000000000002")
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), targetID).Return(&tmcorrelation.Correlation{
					ResourceID:    targetID,
					ResourceFound: true,
					ActiveflowID:  afID,
					Resources:     []*tmcorrelation.PublisherGroup{},
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(&fmactiveflow.Activeflow{
					Identity: commonidentity.Identity{
						ID:         afID,
						CustomerID: a.CustomerID,
					},
				}, nil)
			},

			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-2",
				ResourceType: "correlation",
				ResourceID:   "66666666-0000-4000-8000-000000000002",
				Message:      "Activeflow 66666666-0000-4000-8000-000000000002 is linked to:\n- (no correlated resources)\n",
			},
		},
		{
			name: "cross-customer id blocked - masked as not found",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000003"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000003"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000003"),
			},
			tool: &message.ToolCall{
				ID:   "tool-3",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{"resource_id":"55555555-0000-4000-8000-000000000003"}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				targetID := uuid.FromStringOrNil("55555555-0000-4000-8000-000000000003")
				afID := uuid.FromStringOrNil("66666666-0000-4000-8000-000000000003")
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), targetID).Return(&tmcorrelation.Correlation{
					ResourceID:    targetID,
					ResourceFound: true,
					ActiveflowID:  afID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(&fmactiveflow.Activeflow{
					Identity: commonidentity.Identity{
						ID:         afID,
						CustomerID: uuid.FromStringOrNil("99999999-0000-4000-8000-000000000003"),
					},
				}, nil)
			},

			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-3",
				ResourceType: "correlation",
				ResourceID:   "55555555-0000-4000-8000-000000000003",
				Message:      msgCorrelationNotFound,
			},
		},
		{
			name: "resource not found",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000004"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000004"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000004"),
			},
			tool: &message.ToolCall{
				ID:   "tool-4",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), a.ActiveflowID).Return(&tmcorrelation.Correlation{
					ResourceID:    a.ActiveflowID,
					ResourceFound: false,
				}, nil)
			},

			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-4",
				ResourceType: "correlation",
				ResourceID:   "33333333-0000-4000-8000-000000000004",
				Message:      msgCorrelationNotFound,
			},
		},
		{
			name: "own session no activeflow - discloses unlinked state",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000005"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000005"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000005"),
			},
			tool: &message.ToolCall{
				ID:   "tool-5",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), a.ActiveflowID).Return(&tmcorrelation.Correlation{
					ResourceID:    a.ActiveflowID,
					ResourceFound: true,
					ActiveflowID:  uuid.Nil,
				}, nil)
			},

			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-5",
				ResourceType: "correlation",
				ResourceID:   "33333333-0000-4000-8000-000000000005",
				Message:      "This resource exists but is not linked to any activeflow.",
			},
		},
		{
			name: "foreign id no activeflow - masked as not found",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000006"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000006"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000006"),
			},
			tool: &message.ToolCall{
				ID:   "tool-6",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{"resource_id":"55555555-0000-4000-8000-000000000006"}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				targetID := uuid.FromStringOrNil("55555555-0000-4000-8000-000000000006")
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), targetID).Return(&tmcorrelation.Correlation{
					ResourceID:    targetID,
					ResourceFound: true,
					ActiveflowID:  uuid.Nil,
				}, nil)
			},

			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-6",
				ResourceType: "correlation",
				ResourceID:   "55555555-0000-4000-8000-000000000006",
				Message:      msgCorrelationNotFound,
			},
		},
		{
			name: "ownership lookup error - masked as not found",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000007"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000007"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000007"),
			},
			tool: &message.ToolCall{
				ID:   "tool-7",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{"resource_id":"55555555-0000-4000-8000-000000000007"}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				targetID := uuid.FromStringOrNil("55555555-0000-4000-8000-000000000007")
				afID := uuid.FromStringOrNil("66666666-0000-4000-8000-000000000007")
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), targetID).Return(&tmcorrelation.Correlation{
					ResourceID:    targetID,
					ResourceFound: true,
					ActiveflowID:  afID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(nil, fmt.Errorf("rpc error"))
			},

			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-7",
				ResourceType: "correlation",
				ResourceID:   "55555555-0000-4000-8000-000000000007",
				Message:      msgCorrelationNotFound,
			},
		},
		{
			name: "invalid resource_id",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000008"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000008"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000008"),
			},
			tool: &message.ToolCall{
				ID:   "tool-8",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{"resource_id":"not-a-uuid"}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				// no RPC expected
			},

			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-8",
				Message:    "invalid resource_id",
			},
		},
		{
			name: "correlation lookup error",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-000000000009"),
					CustomerID: uuid.FromStringOrNil("22222222-0000-4000-8000-000000000009"),
				},
				ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-000000000009"),
			},
			tool: &message.ToolCall{
				ID:   "tool-9",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCorrelation,
					Arguments: `{}`,
				},
			},

			mockSetup: func(mockReq *requesthandler.MockRequestHandler, a *aicall.AIcall) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), a.ActiveflowID).Return(nil, fmt.Errorf("rpc error"))
			},

			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-9",
				Message:    "correlation lookup failed",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &aicallHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			if tt.mockSetup != nil {
				tt.mockSetup(mockReq, tt.aicall)
			}

			res := h.toolHandleGetCorrelation(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_toolHandleGetCorrelation_maskingInvariant locks the core security
// property: for one fixed foreign resource_id, the three "cannot see this"
// scenarios (genuinely absent, exists+cross-customer, exists+ownership-lookup
// error) MUST produce byte-identical results. If they ever diverge, the tool
// becomes a cross-customer existence oracle. This guards against a future
// regression that the per-path string assertions above would not catch.
func Test_toolHandleGetCorrelation_maskingInvariant(t *testing.T) {
	sessionCustomerID := uuid.FromStringOrNil("22222222-0000-4000-8000-0000000000aa")
	foreignResourceID := uuid.FromStringOrNil("55555555-0000-4000-8000-0000000000aa")
	foreignActiveflowID := uuid.FromStringOrNil("66666666-0000-4000-8000-0000000000aa")
	foreignCustomerID := uuid.FromStringOrNil("99999999-0000-4000-8000-0000000000aa")

	newAicall := func() *aicall.AIcall {
		return &aicall.AIcall{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("11111111-0000-4000-8000-0000000000aa"),
				CustomerID: sessionCustomerID,
			},
			ActiveflowID: uuid.FromStringOrNil("33333333-0000-4000-8000-0000000000aa"),
		}
	}
	newTool := func() *message.ToolCall {
		return &message.ToolCall{
			ID:   "tool-mask",
			Type: message.ToolTypeFunction,
			Function: message.FunctionCall{
				Name:      message.FunctionCallNameGetCorrelation,
				Arguments: `{"resource_id":"55555555-0000-4000-8000-0000000000aa"}`,
			},
		}
	}

	scenarios := map[string]func(mockReq *requesthandler.MockRequestHandler){
		"genuinely absent": func(mockReq *requesthandler.MockRequestHandler) {
			mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), foreignResourceID).Return(&tmcorrelation.Correlation{
				ResourceID:    foreignResourceID,
				ResourceFound: false,
			}, nil)
		},
		"exists cross-customer": func(mockReq *requesthandler.MockRequestHandler) {
			mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), foreignResourceID).Return(&tmcorrelation.Correlation{
				ResourceID:    foreignResourceID,
				ResourceFound: true,
				ActiveflowID:  foreignActiveflowID,
			}, nil)
			mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), foreignActiveflowID).Return(&fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         foreignActiveflowID,
					CustomerID: foreignCustomerID,
				},
			}, nil)
		},
		"exists ownership-lookup error": func(mockReq *requesthandler.MockRequestHandler) {
			mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), foreignResourceID).Return(&tmcorrelation.Correlation{
				ResourceID:    foreignResourceID,
				ResourceFound: true,
				ActiveflowID:  foreignActiveflowID,
			}, nil)
			mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), foreignActiveflowID).Return(nil, fmt.Errorf("rpc error"))
		},
		"exists no activeflow": func(mockReq *requesthandler.MockRequestHandler) {
			mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), foreignResourceID).Return(&tmcorrelation.Correlation{
				ResourceID:    foreignResourceID,
				ResourceFound: true,
				ActiveflowID:  uuid.Nil,
			}, nil)
		},
	}

	var results []*messageContent
	for name, setup := range scenarios {
		mc := gomock.NewController(t)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		setup(mockReq)

		h := &aicallHandler{reqHandler: mockReq}
		res := h.toolHandleGetCorrelation(context.Background(), newAicall(), newTool())
		mc.Finish()

		// Sanity: every masked path must report the canonical not-found message.
		if res.Message != msgCorrelationNotFound {
			t.Errorf("scenario %q: expected masking message %q, got %q", name, msgCorrelationNotFound, res.Message)
		}
		results = append(results, res)
	}

	// The invariant: all masked results must be byte-identical to each other.
	for i := 1; i < len(results); i++ {
		if !reflect.DeepEqual(results[0], results[i]) {
			t.Errorf("masking invariant violated: results differ\nresult[0]: %#v\nresult[%d]: %#v", results[0], i, results[i])
		}
	}
}

// Test_resolveCorrelation locks the two-tier error contract at the helper level:
//   - the four "cannot see" paths (absent / cross-customer / ownership-lookup-fail /
//     foreign-no-activeflow) return ErrCorrelationNotAccessible (caller masks).
//   - the three granted paths (own-with-activeflow / supplied-owned / own-no-activeflow)
//     return nil error.
//   - the timeline-RPC failure returns a NON-sentinel error (caller reports an honest
//     tool failure, not a mask). The negative stderrors.Is assertion guards against a
//     future change that would silently mask infra errors and re-open an existence oracle.
func Test_resolveCorrelation(t *testing.T) {
	callerCustomerID := uuid.FromStringOrNil("22222222-0000-4000-8000-0000000000b1")
	otherCustomerID := uuid.FromStringOrNil("99999999-0000-4000-8000-0000000000b1")
	resourceID := uuid.FromStringOrNil("55555555-0000-4000-8000-0000000000b1")
	activeflowID := uuid.FromStringOrNil("66666666-0000-4000-8000-0000000000b1")

	tests := []struct {
		name string

		ownSession bool

		mockSetup func(mockReq *requesthandler.MockRequestHandler)

		// expectSentinel: err must be ErrCorrelationNotAccessible.
		// expectNilErr:   err must be nil.
		// neither set:    err must be non-nil AND not the sentinel (transient).
		expectSentinel bool
		expectNilErr   bool
	}{
		{
			name:       "granted - own session with activeflow",
			ownSession: true,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(&tmcorrelation.Correlation{
					ResourceID:    resourceID,
					ResourceFound: true,
					ActiveflowID:  activeflowID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), activeflowID).Return(&fmactiveflow.Activeflow{
					Identity: commonidentity.Identity{ID: activeflowID, CustomerID: callerCustomerID},
				}, nil)
			},
			expectNilErr: true,
		},
		{
			name:       "granted - supplied owned id",
			ownSession: false,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(&tmcorrelation.Correlation{
					ResourceID:    resourceID,
					ResourceFound: true,
					ActiveflowID:  activeflowID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), activeflowID).Return(&fmactiveflow.Activeflow{
					Identity: commonidentity.Identity{ID: activeflowID, CustomerID: callerCustomerID},
				}, nil)
			},
			expectNilErr: true,
		},
		{
			name:       "granted - own session no activeflow",
			ownSession: true,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(&tmcorrelation.Correlation{
					ResourceID:    resourceID,
					ResourceFound: true,
					ActiveflowID:  uuid.Nil,
				}, nil)
			},
			expectNilErr: true,
		},
		{
			name:       "not accessible - resource absent",
			ownSession: false,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(&tmcorrelation.Correlation{
					ResourceID:    resourceID,
					ResourceFound: false,
				}, nil)
			},
			expectSentinel: true,
		},
		{
			name:       "not accessible - cross customer",
			ownSession: false,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(&tmcorrelation.Correlation{
					ResourceID:    resourceID,
					ResourceFound: true,
					ActiveflowID:  activeflowID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), activeflowID).Return(&fmactiveflow.Activeflow{
					Identity: commonidentity.Identity{ID: activeflowID, CustomerID: otherCustomerID},
				}, nil)
			},
			expectSentinel: true,
		},
		{
			name:       "not accessible - ownership lookup failure",
			ownSession: false,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(&tmcorrelation.Correlation{
					ResourceID:    resourceID,
					ResourceFound: true,
					ActiveflowID:  activeflowID,
				}, nil)
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), activeflowID).Return(nil, fmt.Errorf("rpc error"))
			},
			expectSentinel: true,
		},
		{
			name:       "not accessible - foreign id no activeflow",
			ownSession: false,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(&tmcorrelation.Correlation{
					ResourceID:    resourceID,
					ResourceFound: true,
					ActiveflowID:  uuid.Nil,
				}, nil)
			},
			expectSentinel: true,
		},
		{
			name:       "transient - timeline rpc failure",
			ownSession: false,
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().TimelineV1CorrelationGet(gomock.Any(), resourceID).Return(nil, fmt.Errorf("rpc error"))
			},
			// neither expectSentinel nor expectNilErr: must be a non-sentinel error.
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.mockSetup != nil {
				tt.mockSetup(mockReq)
			}

			_, err := h.resolveCorrelation(ctx, callerCustomerID, resourceID, tt.ownSession)

			switch {
			case tt.expectNilErr:
				if err != nil {
					t.Errorf("expected nil error, got: %v", err)
				}
			case tt.expectSentinel:
				if !stderrors.Is(err, ErrCorrelationNotAccessible) {
					t.Errorf("expected ErrCorrelationNotAccessible, got: %v", err)
				}
			default:
				// transient: non-nil, but NOT the sentinel.
				if err == nil {
					t.Errorf("expected a non-nil transient error, got nil")
				}
				if stderrors.Is(err, ErrCorrelationNotAccessible) {
					t.Errorf("transient error must NOT match ErrCorrelationNotAccessible, got: %v", err)
				}
			}
		})
	}
}

// Test_correlationResourceLabel locks the OQ5 label derivation: event-type
// prefix via first-underscore cut, with the neutral fallback for empty
// event types. The envelope data_type is always "application/json" and must
// never be used as the label.
func Test_correlationResourceLabel(t *testing.T) {
	tests := []struct {
		name       string
		eventTypes []string
		want       string
	}{
		{"call event", []string{"call_created", "call_hangup"}, "call"},
		{"aicall multi-segment event", []string{"aicall_status_progressing"}, "aicall"},
		{"conferencecall event", []string{"conferencecall_joined"}, "conferencecall"},
		{"empty event types fallback", []string{}, "resource"},
		{"nil event types fallback", nil, "resource"},
		{"no underscore passes through", []string{"weird"}, "weird"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := correlationResourceLabel(tt.eventTypes); got != tt.want {
				t.Errorf("correlationResourceLabel(%v) = %q, want %q", tt.eventTypes, got, tt.want)
			}
		})
	}
}
