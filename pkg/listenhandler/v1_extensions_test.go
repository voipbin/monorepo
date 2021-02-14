package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/requesthandler"
)

func TestProcessV1ExtensionsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDomain := domainhandler.NewMockDomainHandler(mc)
	mockExtension := extensionhandler.NewMockExtensionHandler(mc)

	h := &listenHandler{
		rabbitSock:       mockSock,
		reqHandler:       mockReq,
		domainHandler:    mockDomain,
		extensionHandler: mockExtension,
	}

	type test struct {
		name      string
		extension *models.Extension
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&models.Extension{
				ID:     uuid.FromStringOrNil("3f4bc63e-6ebf-11eb-b7de-df47266bf559"),
				UserID: 1,

				DomainID: uuid.FromStringOrNil("42dd6424-6ebf-11eb-8630-6b91b6089dc4"),

				EndpointID: "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",
				AORID:      "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",
				AuthID:     "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",

				Extension: "45eb6bac-6ebf-11eb-bcf3-3b9157826d22",
				Password:  "4b1f7a6e-6ebf-11eb-a47e-5351700cd612",
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "domain_id": "42dd6424-6ebf-11eb-8630-6b91b6089dc4", "extension": "45eb6bac-6ebf-11eb-bcf3-3b9157826d22", "password": "4b1f7a6e-6ebf-11eb-a47e-5351700cd612"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3f4bc63e-6ebf-11eb-b7de-df47266bf559","user_id":1,"domain_id":"42dd6424-6ebf-11eb-8630-6b91b6089dc4","endpoint_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","aor_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","auth_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","extension":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22","password":"4b1f7a6e-6ebf-11eb-a47e-5351700cd612","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockExtension.EXPECT().CreateExtension(gomock.Any(), tt.extension.UserID, tt.extension.DomainID, tt.extension.Extension, tt.extension.Password).Return(tt.extension, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
