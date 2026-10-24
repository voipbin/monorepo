package listenhandler

import (
	"errors"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/accounthandler"
)

func Test_processV1AccountsIDPaddlePortalSessionPost(t *testing.T) {

	accountID := uuid.FromStringOrNil("922907b6-0942-11ee-960e-f31d2cc10daa")

	type test struct {
		name    string
		request *sock.Request

		expectCall  bool
		expectID    uuid.UUID
		responseURL string
		responseErr error

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/accounts/922907b6-0942-11ee-960e-f31d2cc10daa/paddle_portal_session",
				Method: sock.RequestMethodPost,
			},

			expectCall:  true,
			expectID:    accountID,
			responseURL: "https://portal.example.com/session/abc",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"url":"https://portal.example.com/session/abc"}`),
			},
		},
		{
			name: "nil uuid",
			request: &sock.Request{
				URI:    "/v1/accounts/00000000-0000-0000-0000-000000000000/paddle_portal_session",
				Method: sock.RequestMethodPost,
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "handler error",
			request: &sock.Request{
				URI:    "/v1/accounts/922907b6-0942-11ee-960e-f31d2cc10daa/paddle_portal_session",
				Method: sock.RequestMethodPost,
			},

			expectCall:  true,
			expectID:    accountID,
			responseErr: errors.New("portal session failure"),

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "unsupported method",
			request: &sock.Request{
				URI:    "/v1/accounts/922907b6-0942-11ee-960e-f31d2cc10daa/paddle_portal_session",
				Method: sock.RequestMethodGet,
			},

			expectRes: &sock.Response{StatusCode: 404},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			if tt.expectCall {
				mockAccount.EXPECT().PaddleCreatePortalSession(gomock.Any(), tt.expectID).Return(tt.responseURL, tt.responseErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
