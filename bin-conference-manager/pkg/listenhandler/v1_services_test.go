package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

func Test_processV1ServicesTypeConferencecallPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseService *commonservice.Service

		expectedActiveflowID  uuid.UUID
		expectedConferenceID  uuid.UUID
		expectedReferenceType conferencecall.ReferenceType
		expectedReferenceID   uuid.UUID
		expectedRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/services/type/conferencecall",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"activeflow_id":"c2a6aa3e-0e44-11f0-92f8-03deb7d17448","conference_id":"43c7671e-c0ab-11ed-a8bc-6f436b081030","reference_type":"call","reference_id":"440e58f4-c0ab-11ed-89d9-7340c26c4034"}`),
			},
			responseService: &commonservice.Service{
				ID: uuid.FromStringOrNil("95cf180c-98c6-11ed-8330-bb119cab4678"),
			},

			expectedActiveflowID:  uuid.FromStringOrNil("c2a6aa3e-0e44-11f0-92f8-03deb7d17448"),
			expectedConferenceID:  uuid.FromStringOrNil("43c7671e-c0ab-11ed-a8bc-6f436b081030"),
			expectedReferenceType: conferencecall.ReferenceTypeCall,
			expectedReferenceID:   uuid.FromStringOrNil("440e58f4-c0ab-11ed-89d9-7340c26c4034"),

			expectedRes: &sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			mockConfcall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				conferenceHandler:     mockConf,
				conferencecallHandler: mockConfcall,
			}

			mockConfcall.EXPECT().ServiceStart(gomock.Any(), tt.expectedActiveflowID, tt.expectedConferenceID, tt.expectedReferenceType, tt.expectedReferenceID.Return(tt.responseService, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
