package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/requesthandler"
)

func TestProcessV1DomainsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDomain := domainhandler.NewMockDomainHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		reqHandler:    mockReq,
		domainHandler: mockDomain,
	}

	type test struct {
		name      string
		domain    *models.Domain
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&models.Domain{
				ID:         uuid.FromStringOrNil("1744ccb4-6e13-11eb-b08d-bb42431b2fb3"),
				UserID:     1,
				DomainName: "0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net",
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/domains",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "domain_name": "0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1744ccb4-6e13-11eb-b08d-bb42431b2fb3","user_id":1,"name":"","detail":"","domain_name":"0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDomain.EXPECT().CreateDomain(gomock.Any(), tt.domain.UserID, tt.domain.DomainName).Return(tt.domain, nil)
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
