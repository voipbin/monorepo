package activeflowhandler

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// variableCreate builds and persists the initial variable map for a new activeflow.
//
// Merge order (load-bearing):
//  1. parent inheritance  (maps.Copy of the reference activeflow's variables, if any)
//  2. external injection   (sanitized initialVariables: reserved-dropped, size-validated)
//  3. system-reserved      (voipbin.activeflow.* keys) applied LAST so they cannot be forged
//
// The on-complete depth counter is parsed from the INHERITED map (step 1) BEFORE the external
// injection is applied, so an externally-supplied voipbin.activeflow.complete_count cannot
// influence the depth bound. (It also cannot reach the map: the sanitizer drops every
// voipbin.* key.)
func (h *activeflowHandler) variableCreate(ctx context.Context, af *activeflow.Activeflow, initialVariables map[string]string) (*variable.Variable, error) {
	completeCount := 0
	variables := map[string]string{}
	if af.ReferenceActiveflowID != uuid.Nil {
		if errSet := h.variableSetFromReferenceActiveflow(ctx, variables, af.ReferenceActiveflowID); errSet != nil {
			return nil, errors.Wrapf(errSet, "could not set variable from reference activeflow. reference_activeflow_id: %s", af.ReferenceActiveflowID)
		}

		// Parse the depth counter from the inherited map ONLY, before external injection.
		tmpCompleteCount, err := h.variableParseCompleteCount(variables)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse complete count from variable. reference_activeflow_id: %s", af.ReferenceActiveflowID)
		}
		tmpCompleteCount++

		if tmpCompleteCount >= maxActiveflowCompleteCount {
			return nil, fmt.Errorf("the activeflow has reached the max complete count (%d). activeflow_id: %s", maxActiveflowCompleteCount, af.ID)
		}

		completeCount = tmpCompleteCount
	}

	// external injection (step 2): sanitize then merge. A rejected/empty injection is a no-op
	// (inheritance + reserved are still seeded below); creation is never failed over external vars.
	sanitized := h.sanitizeInitialVariables(initialVariables)
	maps.Copy(variables, sanitized)

	// system-reserved (step 3): applied LAST so external/inherited values cannot forge them.
	variables[variableActiveflowID] = af.ID.String()
	variables[variableActiveflowReferenceType] = string(af.ReferenceType)
	variables[variableActiveflowReferenceID] = af.ReferenceID.String()
	variables[variableActiveflowReferenceActiveflowID] = af.ReferenceActiveflowID.String()
	variables[variableActiveflowFlowID] = af.FlowID.String()
	variables[variableActiveflowCompleteCount] = fmt.Sprintf("%d", completeCount)

	// defense-in-depth merged cap. Unreachable on current paths (derived creation passes nil),
	// but if the final map somehow exceeds the cap, drop the external contribution and keep
	// inheritance + reserved. Never produce a reserved-less map; never fail creation here.
	if variablesByteSize(variables) > mergedVariablesMaxTotalBytes && len(sanitized) > 0 {
		promActiveflowVariableInjectionTotal.WithLabelValues("rejected_merged").Inc()
		variables = map[string]string{}
		if af.ReferenceActiveflowID != uuid.Nil {
			if errSet := h.variableSetFromReferenceActiveflow(ctx, variables, af.ReferenceActiveflowID); errSet != nil {
				return nil, errors.Wrapf(errSet, "could not re-set variable from reference activeflow. reference_activeflow_id: %s", af.ReferenceActiveflowID)
			}
		}
		variables[variableActiveflowID] = af.ID.String()
		variables[variableActiveflowReferenceType] = string(af.ReferenceType)
		variables[variableActiveflowReferenceID] = af.ReferenceID.String()
		variables[variableActiveflowReferenceActiveflowID] = af.ReferenceActiveflowID.String()
		variables[variableActiveflowFlowID] = af.FlowID.String()
		variables[variableActiveflowCompleteCount] = fmt.Sprintf("%d", completeCount)
	}

	res, err := h.variableHandler.Create(ctx, af.ID, variables)
	if err != nil {
		return nil, errors.Wrapf(err, "could not set the variable. activeflow_id: %s", af.ID)
	}

	return res, nil
}

// sanitizeInitialVariables filters externally-supplied variables before they are merged into a
// new activeflow's variable map. It returns the accepted subset (possibly empty). It never
// errors: a rejected injection returns an empty map and is recorded via the metric, so
// activeflow creation is never failed over external variables.
//
// Rules (in order):
//  1. trim each key; drop if empty, or if the key normalizes (lowercase) to the reserved
//     voipbin. prefix. Normalization is comparison-only; the surviving key keeps its original
//     bytes so substitution (${Key}) still matches.
//  2. reject the whole injection if any surviving value exceeds initialVariablesMaxValueBytes.
//  3. reject the whole injection if the surviving key count or total byte size exceeds the caps
//     (counted AFTER reserved/empty drops, so junk keys cannot pad legitimate keys over a cap).
func (h *activeflowHandler) sanitizeInitialVariables(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}

	survivors := make(map[string]string, len(in))
	droppedReserved := false
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(key), variableReservedPrefix) {
			droppedReserved = true
			continue
		}
		survivors[key] = v
	}
	if droppedReserved {
		promActiveflowVariableInjectionTotal.WithLabelValues("dropped_reserved").Inc()
	}

	if len(survivors) == 0 {
		return map[string]string{}
	}

	total := 0
	for k, v := range survivors {
		if len(v) > initialVariablesMaxValueBytes {
			promActiveflowVariableInjectionTotal.WithLabelValues("rejected_value_size").Inc()
			return map[string]string{}
		}
		total += len(k) + len(v)
	}

	if len(survivors) > initialVariablesMaxKeyCount {
		promActiveflowVariableInjectionTotal.WithLabelValues("rejected_count").Inc()
		return map[string]string{}
	}
	if total > initialVariablesMaxTotalBytes {
		promActiveflowVariableInjectionTotal.WithLabelValues("rejected_total").Inc()
		return map[string]string{}
	}

	promActiveflowVariableInjectionTotal.WithLabelValues("accepted").Inc()
	return survivors
}

// variablesByteSize returns the total UTF-8 byte size of all keys and values in the map.
func variablesByteSize(variables map[string]string) int {
	total := 0
	for k, v := range variables {
		total += len(k) + len(v)
	}
	return total
}

func (h *activeflowHandler) variableSetFromReferenceActiveflow(ctx context.Context, variables map[string]string, referenceActiveflowID uuid.UUID) error {
	tmp, err := h.variableHandler.Get(ctx, referenceActiveflowID)
	if err != nil {
		return errors.Wrapf(err, "could not get the variable. reference_activeflow_id: %s", referenceActiveflowID)
	}

	// copy the variables
	maps.Copy(variables, tmp.Variables)

	return nil
}

func (h *activeflowHandler) variableParseCompleteCount(variables map[string]string) (int, error) {
	val, ok := variables[variableActiveflowCompleteCount]
	if !ok {
		return 0, fmt.Errorf("could not find the complete count variable")
	}

	res := 0
	_, err := fmt.Sscanf(val, "%d", &res)
	if err != nil {
		return 0, errors.Wrapf(err, "could not parse the complete count variable. value: %s ", val)
	}

	return res, nil
}
