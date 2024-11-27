package servicehandler

import (
	"monorepo/bin-api-manager/lib/common"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_JWTGenerate(t *testing.T) {

	tests := []struct {
		name string

		data map[string]interface{}

		responseCurTime string

		expectRes common.JSON
	}{
		{
			name: "normal",

			data: map[string]interface{}{
				"key1": "val1",
				"key2": "val2",
			},

			responseCurTime: "2023-11-19 09:29:11.763331118",
			expectRes: common.JSON{
				"key1":   "val1",
				"key2":   "val2",
				"expire": "2023-11-19 09:29:11.763331118",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}

			mockUtil.EXPECT().TimeGetCurTimeAdd(common.TokenExpiration).Return(tt.responseCurTime)
			token, err := h.JWTGenerate(tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			res, err := h.JWTParse(token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
