package conversationhandler

import (
	"context"
	"fmt"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/pkg/errors"
)

func (h *conversationHandler) NumberGet(ctx context.Context, number string) (*nmnumber.Number, error) {
	filters := map[nmnumber.Field]any{
		nmnumber.FieldNumber:  number,
		nmnumber.FieldDeleted: false,
	}

	tmps, err := h.reqHandler.NumberV1NumberGets(ctx, "", 1, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get number info. number: %s", number)
	}

	if len(tmps) == 0 {
		return nil, fmt.Errorf("number not found")
	}

	res := tmps[0]
	return &res, nil
}
