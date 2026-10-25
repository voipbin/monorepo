package callhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_DigitsSet_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	afID := uuid.FromStringOrNil("88888888-0000-0000-0000-000000000001")

	tests := []struct {
		name      string
		getErr    error
		setErr    error
		expectErr bool
	}{
		{"success", nil, nil, false},
		{"get call error", dbhandler.ErrNotFound, nil, true},
		{"set variable error", nil, fmt.Errorf("err"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			if tt.getErr != nil {
				m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
			} else {
				c := covBCall(covBCallID, call.StatusProgressing)
				c.ActiveflowID = afID
				m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, afID, map[string]string{variableCallDigits: "123"}).Return(tt.setErr)
			}

			err := h.DigitsSet(ctx, id, "123")
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}
