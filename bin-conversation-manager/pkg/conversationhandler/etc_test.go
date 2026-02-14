package conversationhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_NumberGet(t *testing.T) {
	tests := []struct {
		name string

		number string

		responseNumbers []nmnumber.Number
		expectNumber    *nmnumber.Number
	}{
		{
			name:   "normal",
			number: "+1234567890",

			responseNumbers: []nmnumber.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
					},
					Number: "+1234567890",
				},
			},
			expectNumber: &nmnumber.Number{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Number: "+1234567890",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &conversationHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			filters := map[nmnumber.Field]any{
				nmnumber.FieldNumber:  tt.number,
				nmnumber.FieldDeleted: false,
			}

			mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), filters).Return(tt.responseNumbers, nil)

			res, err := h.NumberGet(ctx, tt.number)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if res.ID != tt.expectNumber.ID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.expectNumber.ID, res.ID)
			}
			if res.Number != tt.expectNumber.Number {
				t.Errorf("Wrong Number. expect: %s, got: %s", tt.expectNumber.Number, res.Number)
			}
		})
	}
}

func Test_NumberGet_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &conversationHandler{
		reqHandler: mockReq,
	}
	ctx := context.Background()

	number := "+1234567890"
	filters := map[nmnumber.Field]any{
		nmnumber.FieldNumber:  number,
		nmnumber.FieldDeleted: false,
	}

	mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), filters).Return([]nmnumber.Number{}, nil)

	_, err := h.NumberGet(ctx, number)
	if err == nil {
		t.Errorf("Expected error for not found, got nil")
	}
}
