package agenthandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
)

func Test_DirectHashRegenerate_errors(t *testing.T) {
	id := uuid.FromStringOrNil("a1a1a1a1-5555-5555-5555-555555555555")
	customerID := uuid.FromStringOrNil("c1c1c1c1-5555-5555-5555-555555555555")
	directID := uuid.FromStringOrNil("d1d1d1d1-5555-5555-5555-555555555555")

	errBoom := fmt.Errorf("boom")
	withDirect := &agent.Agent{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, DirectID: directID}
	withoutDirect := &agent.Agent{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}
	direct := &dmdirect.Direct{Identity: commonidentity.Identity{ID: directID}, Hash: "test-direct-hash"}
	expectFields := map[agent.Field]any{
		agent.FieldDirectID:   directID,
		agent.FieldDirectHash: "test-direct-hash",
	}

	tests := []struct {
		name string

		setup func(ctx context.Context, m *errMocks)

		expectMessage string
		// expectSame is true when the original error is returned without wrapping.
		expectSame bool
	}{
		{
			name: "agent lookup failure",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom)
			},
			expectMessage: "could not get agent",
		},
		{
			name: "regenerate failure for an agent with a direct",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(withDirect, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, directID).Return(nil, errBoom)
			},
			expectMessage: "could not regenerate direct hash",
		},
		{
			name: "create failure for an agent without a direct",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(withoutDirect, nil)
				m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, id).Return(nil, errBoom)
			},
			expectMessage: "could not create direct hash",
		},
		{
			name: "agent update failure",
			setup: func(ctx context.Context, m *errMocks) {
				m.db.EXPECT().AgentGet(ctx, id).Return(withDirect, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, directID).Return(direct, nil)
				m.db.EXPECT().AgentUpdate(ctx, id, expectFields).Return(errBoom)
			},
			expectMessage: "could not update agent",
		},
		{
			name: "lookup after update failure is returned as is",
			setup: func(ctx context.Context, m *errMocks) {
				gomock.InOrder(
					m.db.EXPECT().AgentGet(ctx, id).Return(withDirect, nil),
					m.req.EXPECT().DirectV1DirectRegenerate(ctx, directID).Return(direct, nil),
					m.db.EXPECT().AgentUpdate(ctx, id, expectFields).Return(nil),
					m.db.EXPECT().AgentGet(ctx, id).Return(nil, errBoom),
				)
			},
			expectSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, m)

			res, err := h.DirectHashRegenerate(ctx, id)
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !errors.Is(err, errBoom) {
				t.Errorf("Wrong error chain. expect: %v, got: %v", errBoom, err)
			}
			if tt.expectSame {
				if err != errBoom {
					t.Errorf("Wrong error. expect the original error, got: %v", err)
				}
				return
			}
			if !strings.HasPrefix(err.Error(), tt.expectMessage+": ") {
				t.Errorf("Wrong message. expect prefix: %s, got: %v", tt.expectMessage, err)
			}
		})
	}
}
