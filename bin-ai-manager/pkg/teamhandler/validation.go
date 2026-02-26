package teamhandler

import (
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/models/tool"
)

// validateTeam checks all validation rules for team structure.
func validateTeam(startMemberID uuid.UUID, members []team.Member) error {
	// Rule 10: Members list must not be empty
	if len(members) == 0 {
		return fmt.Errorf("members list must not be empty")
	}

	// Rule 1: StartMemberID must not be uuid.Nil
	if startMemberID == uuid.Nil {
		return fmt.Errorf("start_member_id must not be empty")
	}

	// Build member ID set for lookups
	memberIDs := make(map[uuid.UUID]bool)
	for _, m := range members {
		// Rule 4: Member.ID must not be uuid.Nil
		if m.ID == uuid.Nil {
			return fmt.Errorf("member id must not be empty")
		}

		// Rule 3: Member IDs must be unique
		if memberIDs[m.ID] {
			return fmt.Errorf("duplicate member id: %s", m.ID)
		}
		memberIDs[m.ID] = true
	}

	// Rule 2: StartMemberID must exist in Members
	if !memberIDs[startMemberID] {
		return fmt.Errorf("start_member_id %s not found in members", startMemberID)
	}

	// Build reserved tool names set
	reservedNames := make(map[string]bool)
	for _, tn := range tool.AllToolNames {
		reservedNames[string(tn)] = true
	}

	for _, m := range members {
		// Rule 11: Member.Name must not be empty
		if m.Name == "" {
			return fmt.Errorf("member name must not be empty for member %s", m.ID)
		}

		// Rule 5: Member.AIID must not be uuid.Nil
		if m.AIID == uuid.Nil {
			return fmt.Errorf("member ai_id must not be empty for member %s", m.ID)
		}

		// Check transitions
		fnNames := make(map[string]bool)
		for _, t := range m.Transitions {
			// FunctionName must not be empty
			if t.FunctionName == "" {
				return fmt.Errorf("transition function_name must not be empty for member %s", m.ID)
			}

			// Rule 8: FunctionName must not collide with reserved tool names
			if reservedNames[t.FunctionName] {
				return fmt.Errorf("transition function_name %q collides with reserved tool name for member %s", t.FunctionName, m.ID)
			}

			// Rule 9: FunctionName must be unique within a member
			if fnNames[t.FunctionName] {
				return fmt.Errorf("duplicate transition function_name %q for member %s", t.FunctionName, m.ID)
			}
			fnNames[t.FunctionName] = true

			// Rule 7: NextMemberID must reference an existing member
			if !memberIDs[t.NextMemberID] {
				return fmt.Errorf("transition next_member_id %s not found in members for member %s", t.NextMemberID, m.ID)
			}
		}
	}

	return nil
}
