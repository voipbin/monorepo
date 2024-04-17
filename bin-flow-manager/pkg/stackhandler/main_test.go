package stackhandler

import (
	"sort"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/stack"
)

func getItemByIndex(stackMap map[uuid.UUID]*stack.Stack, idx int) *stack.Stack {

	// sort stackMap
	var tmpSort []string
	for stackID := range stackMap {
		tmpSort = append(tmpSort, stackID.String())
	}
	sort.Strings(tmpSort)

	id := uuid.FromStringOrNil(tmpSort[idx])
	res, ok := stackMap[id]
	if !ok {
		return nil
	}

	return res
}
