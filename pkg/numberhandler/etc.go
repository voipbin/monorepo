package numberhandler

import (
	"context"
	"fmt"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// generateTags returns a tags of the given number
func (h *numberHandler) generateTags(ctx context.Context, n *number.Number) []string {
	customerID := fmt.Sprintf("CustomerID_%s", n.CustomerID)
	numberID := fmt.Sprintf("NumberID_%s", n.ID)

	res := []string{
		customerID,
		numberID,
	}

	return res
}
