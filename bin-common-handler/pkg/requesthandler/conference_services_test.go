package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	cfservice "monorepo/bin-conference-manager/models/service"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_ConferenceV1ServiceTypeConferencecallStart(t *testing.T) {

	tests := []struct {
		name string

		conferenceID  uuid.UUID
		referenceType cfconferencecall.ReferenceType
		referenceID   uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfservice.Service
	}{
		{
			"normal",

			uuid.FromStringOrNil("ef5341ba-ab71-11ed-8b32-b3ea2332246a"),
			cfconferencecall.ReferenceTypeCall,
			uuid.FromStringOrNil("ef7fa3e0-ab71-11ed-9a00-3f98e88afb4e"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"efa863ca-ab71-11ed-a65f-0730598fc7d9"}`),
			},

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/services/type/conferencecall",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"conference_id":"ef5341ba-ab71-11ed-8b32-b3ea2332246a","reference_type":"call","reference_id":"ef7fa3e0-ab71-11ed-9a00-3f98e88afb4e"}`),
			},
			&cfservice.Service{
				ID: uuid.FromStringOrNil("efa863ca-ab71-11ed-a65f-0730598fc7d9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			cf, err := reqHandler.ConferenceV1ServiceTypeConferencecallStart(ctx, tt.conferenceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}
