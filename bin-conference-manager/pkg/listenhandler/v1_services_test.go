package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/service"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
)

func Test_processV1ServicesTypeConferencecallPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseService *service.Service

		expectConferenceID  uuid.UUID
		expectReferenceType conferencecall.ReferenceType
		expectReferenceID   uuid.UUID
		expectRes           *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/services/type/conferencecall",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"conference_id":"43c7671e-c0ab-11ed-a8bc-6f436b081030","reference_type":"call","reference_id":"440e58f4-c0ab-11ed-89d9-7340c26c4034"}`),
			},
			responseService: &service.Service{
				ID: uuid.FromStringOrNil("95cf180c-98c6-11ed-8330-bb119cab4678"),
			},

			expectConferenceID:  uuid.FromStringOrNil("43c7671e-c0ab-11ed-a8bc-6f436b081030"),
			expectReferenceType: conferencecall.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("440e58f4-c0ab-11ed-89d9-7340c26c4034"),

			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"95cf180c-98c6-11ed-8330-bb119cab4678","type":"","push_actions":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			mockConfcall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				rabbitSock:            mockSock,
				conferenceHandler:     mockConf,
				conferencecallHandler: mockConfcall,
			}

			mockConfcall.EXPECT().ServiceStart(gomock.Any(), tt.expectConferenceID, tt.expectReferenceType, tt.expectReferenceID).Return(tt.responseService, nil)
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
