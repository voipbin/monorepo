package activeflowhandler

import (
	"context"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

// Test_variableCreate_initialVariablesMerge exercises real (non-nil) initialVariables to cover
// the external-injection merge, reserved-key override, and merged-cap drop branches.
func Test_variableCreate_initialVariablesMerge(t *testing.T) {
	tests := []struct {
		name string

		activeflow       *activeflow.Activeflow
		initialVariables map[string]string

		responseReferenceActiveflowVariable *variable.Variable

		// validate inspects the final map passed to variableHandler.Create.
		validate func(t *testing.T, got map[string]string)
	}{
		{
			name: "external injection merges into parent inheritance and external wins over inherited non-reserved key",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1000000-0000-11f0-8000-000000000001"),
				},
				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("a1000000-0000-11f0-8000-000000000002"),
				ReferenceActiveflowID: uuid.FromStringOrNil("a1000000-0000-11f0-8000-000000000003"),
				FlowID:                uuid.FromStringOrNil("a1000000-0000-11f0-8000-000000000004"),
			},
			initialVariables: map[string]string{
				"shared": "ext_val",
				"key_e":  "ev",
			},
			responseReferenceActiveflowVariable: &variable.Variable{
				Variables: map[string]string{
					"shared":                        "parent_val",
					"key_p":                         "pv",
					variableActiveflowCompleteCount: "1",
				},
			},
			validate: func(t *testing.T, got map[string]string) {
				if got["shared"] != "ext_val" {
					t.Errorf("external should win over inherited 'shared'. got %q", got["shared"])
				}
				if got["key_p"] != "pv" {
					t.Errorf("inherited key_p missing. got %q", got["key_p"])
				}
				if got["key_e"] != "ev" {
					t.Errorf("external key_e missing. got %q", got["key_e"])
				}
				if got[variableActiveflowID] != "a1000000-0000-11f0-8000-000000000001" {
					t.Errorf("reserved activeflow id wrong. got %q", got[variableActiveflowID])
				}
				if got[variableActiveflowCompleteCount] != "2" {
					t.Errorf("complete count should be 2. got %q", got[variableActiveflowCompleteCount])
				}
			},
		},
		{
			name: "externally injected voipbin.* key is dropped and system-reserved value wins",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a2000000-0000-11f0-8000-000000000001"),
				},
				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("a2000000-0000-11f0-8000-000000000002"),
				ReferenceActiveflowID: uuid.FromStringOrNil("a2000000-0000-11f0-8000-000000000003"),
				FlowID:                uuid.FromStringOrNil("a2000000-0000-11f0-8000-000000000004"),
			},
			initialVariables: map[string]string{
				variableActiveflowFlowID: "forged-flow-id",
				"normal":                 "n",
			},
			responseReferenceActiveflowVariable: &variable.Variable{
				Variables: map[string]string{
					variableActiveflowCompleteCount: "0",
				},
			},
			validate: func(t *testing.T, got map[string]string) {
				if got[variableActiveflowFlowID] != "a2000000-0000-11f0-8000-000000000004" {
					t.Errorf("reserved flow_id must win over forged external. got %q", got[variableActiveflowFlowID])
				}
				if got["normal"] != "n" {
					t.Errorf("non-reserved external key 'normal' missing. got %q", got["normal"])
				}
			},
		},
		{
			name: "merged-cap drops external contribution but keeps reserved keys",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a3000000-0000-11f0-8000-000000000001"),
				},
				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("a3000000-0000-11f0-8000-000000000002"),
				ReferenceActiveflowID: uuid.FromStringOrNil("a3000000-0000-11f0-8000-000000000003"),
				FlowID:                uuid.FromStringOrNil("a3000000-0000-11f0-8000-000000000004"),
			},
			// external stays within the per-injection sanitize caps (value < 32KB, total < 64KB),
			// but the inherited map is large enough that the FINAL merged map exceeds
			// mergedVariablesMaxTotalBytes (256KB), triggering the rejected_merged branch.
			initialVariables: map[string]string{
				"ext_key": strings.Repeat("e", 10*1024),
			},
			responseReferenceActiveflowVariable: &variable.Variable{
				Variables: map[string]string{
					"inherited_big":                 strings.Repeat("p", 260*1024),
					variableActiveflowCompleteCount: "0",
				},
			},
			validate: func(t *testing.T, got map[string]string) {
				if _, ok := got["ext_key"]; ok {
					t.Errorf("external key must be dropped when merged map exceeds cap")
				}
				if got["inherited_big"] == "" {
					t.Errorf("inherited key must be preserved after merged-cap drop")
				}
				if got[variableActiveflowID] != "a3000000-0000-11f0-8000-000000000001" {
					t.Errorf("reserved activeflow id must be present after merged-cap drop. got %q", got[variableActiveflowID])
				}
				if got[variableActiveflowFlowID] != "a3000000-0000-11f0-8000-000000000004" {
					t.Errorf("reserved flow_id must be present after merged-cap drop. got %q", got[variableActiveflowFlowID])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				variableHandler: mockVar,
			}
			ctx := context.Background()

			// merged-cap branch re-reads the reference activeflow, so allow Get to be called
			// once or twice with the same response.
			mockVar.EXPECT().
				Get(ctx, tt.activeflow.ReferenceActiveflowID).
				Return(tt.responseReferenceActiveflowVariable, nil).
				AnyTimes()

			responseVariable := &variable.Variable{ID: tt.activeflow.ID}
			var captured map[string]string
			mockVar.EXPECT().
				Create(ctx, tt.activeflow.ID, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ uuid.UUID, v map[string]string) (*variable.Variable, error) {
					captured = v
					return responseVariable, nil
				})

			_, err := h.variableCreate(ctx, tt.activeflow, tt.initialVariables)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.validate(t, captured)
		})
	}
}

// Test_sanitizeInitialVariables_totalCap covers the rejected_total path: every individual value
// is <= 32KB but the SUM of keys+values exceeds initialVariablesMaxTotalBytes (64KB).
func Test_sanitizeInitialVariables_totalCap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &activeflowHandler{
		db:              dbhandler.NewMockDBHandler(mc),
		reqHandler:      requesthandler.NewMockRequestHandler(mc),
		variableHandler: variablehandler.NewMockVariableHandler(mc),
	}

	// 3 values of 30KB each = 90KB total; each value (30KB) is within the 32KB single-value cap,
	// but the total exceeds the 64KB total cap -> whole injection rejected -> empty map.
	in := map[string]string{
		"k0": strings.Repeat("a", 30*1024),
		"k1": strings.Repeat("b", 30*1024),
		"k2": strings.Repeat("c", 30*1024),
	}

	got := h.sanitizeInitialVariables(in)
	if len(got) != 0 {
		t.Errorf("expected empty map (rejected_total), got %d keys", len(got))
	}
}
