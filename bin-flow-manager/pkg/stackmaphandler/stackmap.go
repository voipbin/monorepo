package stackmaphandler

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

func (h *stackHandler) Create(actions []action.Action) map[uuid.UUID]*stack.Stack {
	tmp := h.CreateStack(stack.IDMain, actions, stack.IDEmpty, action.IDEmpty)
	res := map[uuid.UUID]*stack.Stack{
		stack.IDMain: tmp,
	}

	return res
}
