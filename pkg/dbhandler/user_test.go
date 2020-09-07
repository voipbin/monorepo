package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/cachehandler"
)

func TestUserCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name       string
		user       *user.User
		expectUser *user.User
	}

	tests := []test{
		{
			"test normal",
			&user.User{
				ID:           1,
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
			&user.User{
				ID:           1,
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().UserSet(gomock.Any(), gomock.Any())
			if err := h.UserCreate(context.Background(), tt.user); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().UserGet(gomock.Any(), tt.user.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().UserSet(gomock.Any(), gomock.Any())
			res, err := h.UserGet(context.Background(), tt.user.ID)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectUser, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectUser, res)
			}
		})
	}
}
