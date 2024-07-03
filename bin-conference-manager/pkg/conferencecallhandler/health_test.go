package conferencecallhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_HealthCheck(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		retryCount int

		responseConferencecall *conferencecall.Conferencecall
		responseCurTimeAdd     string
		responseCall           *cmcall.Call
		responseConference     *conference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("ad03211e-94d2-11ed-8e1d-2f0082408655"),
			0,

			&conferencecall.Conferencecall{
				ID:          uuid.FromStringOrNil("ad03211e-94d2-11ed-8e1d-2f0082408655"),
				Status:      conferencecall.StatusJoined,
				ReferenceID: uuid.FromStringOrNil("ae1a2bec-94d2-11ed-913d-73ee1991cfa1"),
				TMCreate:    "2023-01-03 21:35:02.809",
			},
			"2023-01-03 21:35:02.809",
			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ae1a2bec-94d2-11ed-913d-73ee1991cfa1"),
				},
				ConfbridgeID: uuid.FromStringOrNil("ae654a6e-94d2-11ed-b9ca-5b472f10031a"),
				Status:       cmcall.StatusProgressing,
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("ae4358e6-94d2-11ed-ab3a-c37c216ef696"),
				ConfbridgeID: uuid.FromStringOrNil("ae654a6e-94d2-11ed-b9ca-5b472f10031a"),
				Status:       conference.StatusProgressing,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, nil)
			mockUtil.EXPECT().TimeGetCurTimeAdd(-maxConferencecallDuration).Return(tt.responseCurTimeAdd)
			mockReq.EXPECT().CallV1CallGet(ctx, tt.responseConferencecall.ReferenceID).Return(tt.responseCall, nil)
			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.responseConferencecall.ConferenceID).Return(tt.responseConference, nil)

			mockReq.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, tt.id, 0, defaultHealthCheckDelay)

			h.HealthCheck(ctx, tt.id, tt.retryCount)

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_HealthCheck_error(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		retryCount int

		responseConferencecall      *conferencecall.Conferencecall
		responseConferencecallError error
		responseCurTimeAdd          string
		responseCall                *cmcall.Call
		responseCallError           error
		responseConference          *conference.Conference
		responseconferenceError     error

		expectRetryCount int
	}{
		{
			name: "exceeded max retry count",

			id:         uuid.FromStringOrNil("8e1fd114-94d4-11ed-87da-43e1e2fa43d6"),
			retryCount: defaultHealthCheckRetryMax + 1,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:          uuid.FromStringOrNil("8e1fd114-94d4-11ed-87da-43e1e2fa43d6"),
				Status:      conferencecall.StatusJoined,
				ReferenceID: uuid.FromStringOrNil("8e75c934-94d4-11ed-9de9-7fce898af73a"),
				TMCreate:    "2023-01-03 21:35:02.809",
			},
			responseCurTimeAdd: "2023-01-03 21:35:02.809",
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8e75c934-94d4-11ed-9de9-7fce898af73a"),
				},
				ConfbridgeID: uuid.FromStringOrNil("8ea3058e-94d4-11ed-b42c-b7dc2e29c819"),
				Status:       cmcall.StatusProgressing,
			},
			responseConference: &conference.Conference{
				ID:           uuid.FromStringOrNil("8ecae996-94d4-11ed-aff9-577bc62739a2"),
				ConfbridgeID: uuid.FromStringOrNil("8ea3058e-94d4-11ed-b42c-b7dc2e29c819"),
			},
		},
		{
			name: "conferencecall get failed",

			id:         uuid.FromStringOrNil("37609c36-94d5-11ed-ba73-3fc1b31f642f"),
			retryCount: 0,

			responseConferencecallError: fmt.Errorf(""),
			expectRetryCount:            1,
		},
		{
			name: "conferencecall status is leaved",

			id:         uuid.FromStringOrNil("c01de0ec-94d5-11ed-b353-77ec1e03ac6a"),
			retryCount: 0,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:       uuid.FromStringOrNil("c01de0ec-94d5-11ed-b353-77ec1e03ac6a"),
				Status:   conferencecall.StatusLeaved,
				TMCreate: "2023-01-03 21:35:02.809",
			},
			responseCurTimeAdd: "2023-01-03 21:35:02.809",
		},
		{
			name: "call get failed",

			id:         uuid.FromStringOrNil("c04eba0a-94d5-11ed-94b1-875536510dcb"),
			retryCount: 0,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:          uuid.FromStringOrNil("c04eba0a-94d5-11ed-94b1-875536510dcb"),
				Status:      conferencecall.StatusJoined,
				ReferenceID: uuid.FromStringOrNil("c07ee874-94d5-11ed-8da1-6fef0f91be9b"),
				TMCreate:    "2023-01-03 21:35:02.809",
			},
			responseCurTimeAdd: "2023-01-03 21:35:02.809",
			responseCallError:  fmt.Errorf(""),

			expectRetryCount: 1,
		},
		{
			name: "call has invalid status",

			id:         uuid.FromStringOrNil("8582538a-94d7-11ed-8719-0fd363bed4df"),
			retryCount: 0,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:          uuid.FromStringOrNil("8582538a-94d7-11ed-8719-0fd363bed4df"),
				Status:      conferencecall.StatusJoined,
				ReferenceID: uuid.FromStringOrNil("85a99b98-94d7-11ed-87b3-4f8bf6bfbd39"),
				TMCreate:    "2023-01-03 21:35:02.809",
			},
			responseCurTimeAdd: "2023-01-03 21:35:02.809",
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("85a99b98-94d7-11ed-87b3-4f8bf6bfbd39"),
				},
				Status: cmcall.StatusRinging,
			},

			expectRetryCount: 1,
		},
		{
			name: "conference get failed",

			id:         uuid.FromStringOrNil("a5481a34-94d6-11ed-add0-7ba046e6403b"),
			retryCount: 0,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("a5481a34-94d6-11ed-add0-7ba046e6403b"),
				Status:       conferencecall.StatusJoined,
				ReferenceID:  uuid.FromStringOrNil("a5724d7c-94d6-11ed-bc34-5f290587207e"),
				ConferenceID: uuid.FromStringOrNil("a599133a-94d6-11ed-9556-9fe210b5e9df"),
				TMCreate:     "2023-01-03 21:35:02.809",
			},
			responseCurTimeAdd: "2023-01-03 21:35:02.809",
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a5724d7c-94d6-11ed-bc34-5f290587207e"),
				},
			},
			responseConferencecallError: fmt.Errorf(""),

			expectRetryCount: 1,
		},
		{
			name: "conference has invalid status",

			id:         uuid.FromStringOrNil("196a6c64-94d7-11ed-8672-87a16e3986ed"),
			retryCount: 0,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("196a6c64-94d7-11ed-8672-87a16e3986ed"),
				Status:       conferencecall.StatusJoined,
				ReferenceID:  uuid.FromStringOrNil("198fbabe-94d7-11ed-b69a-5336a8f18455"),
				ConferenceID: uuid.FromStringOrNil("19bc2734-94d7-11ed-9373-37b511c36f27"),
				TMCreate:     "2023-01-03 21:35:02.809",
			},
			responseCurTimeAdd: "2023-01-03 21:35:02.809",
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("198fbabe-94d7-11ed-b69a-5336a8f18455"),
				},
				ConfbridgeID: uuid.FromStringOrNil("19ec879e-94d7-11ed-957c-5b56c6cdd831"),
			},
			responseConference: &conference.Conference{
				ID:           uuid.FromStringOrNil("19bc2734-94d7-11ed-9373-37b511c36f27"),
				ConfbridgeID: uuid.FromStringOrNil("1a346b04-94d7-11ed-8435-db5573ae1660"),
				Status:       conference.StatusTerminated,
			},

			expectRetryCount: 1,
		},
		{
			name: "conferencecall timed out",

			id:         uuid.FromStringOrNil("40f411a8-1e26-44b5-b335-ac2c4e00276f"),
			retryCount: 0,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("40f411a8-1e26-44b5-b335-ac2c4e00276f"),
				Status:       conferencecall.StatusJoined,
				ReferenceID:  uuid.FromStringOrNil("3f3059d5-6a74-4e6b-a225-964ee1d315b8"),
				ConferenceID: uuid.FromStringOrNil("bfb771f8-c594-45f2-bca7-7dd06e431031"),
				TMCreate:     "2023-01-03 21:35:02.809",
			},
			responseCurTimeAdd: "2023-01-01 21:35:02.809",
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3f3059d5-6a74-4e6b-a225-964ee1d315b8"),
				},
				ConfbridgeID: uuid.FromStringOrNil("a772d8cd-1fdb-4553-ae35-19b0f9f499cc"),
			},
			responseConference: &conference.Conference{
				ID:           uuid.FromStringOrNil("bfb771f8-c594-45f2-bca7-7dd06e431031"),
				ConfbridgeID: uuid.FromStringOrNil("a772d8cd-1fdb-4553-ae35-19b0f9f499cc"),
				Status:       conference.StatusTerminated,
			},

			expectRetryCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockConference := conferencehandler.NewMockConferenceHandler(mc)

			h := conferencecallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,

				conferenceHandler: mockConference,
			}

			ctx := context.Background()

			if tt.retryCount > defaultHealthCheckRetryMax {
				// terminate
				mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, nil)
				mockConference.EXPECT().Get(ctx, tt.responseConferencecall.ConferenceID).Return(tt.responseConference, nil)
				mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.id, conferencecall.StatusLeaving).Return(nil)
				mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, nil)
				mockNotify.EXPECT().PublishEvent(ctx, gomock.Any(), gomock.Any())
				mockReq.EXPECT().CallV1ConfbridgeCallKick(ctx, tt.responseConference.ConfbridgeID, tt.responseConferencecall.ReferenceID).Return(nil)
			} else {
				mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, tt.responseConferencecallError)
				if tt.responseConferencecallError != nil {
					mockReq.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, tt.id, tt.expectRetryCount, defaultHealthCheckDelay)
				} else {
					mockUtil.EXPECT().TimeGetCurTimeAdd(-maxConferencecallDuration).Return(tt.responseCurTimeAdd)
					if tt.responseConferencecall.TMCreate < tt.responseCurTimeAdd {
						mockReq.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, tt.id, tt.expectRetryCount, defaultHealthCheckDelay)
					} else {
						if tt.responseConferencecall.Status != conferencecall.StatusLeaved {
							mockReq.EXPECT().CallV1CallGet(ctx, tt.responseConferencecall.ReferenceID).Return(tt.responseCall, tt.responseCallError)
							if tt.responseCallError != nil {
								mockReq.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, tt.id, tt.expectRetryCount, defaultHealthCheckDelay)
							} else {
								if tt.responseCall.Status != cmcall.StatusProgressing {
									mockReq.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, tt.id, tt.expectRetryCount, defaultHealthCheckDelay)
								} else {
									mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.responseConferencecall.ConferenceID).Return(tt.responseConference, tt.responseconferenceError)
									if tt.responseConference.ConfbridgeID != tt.responseCall.ConfbridgeID {
										mockReq.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, tt.id, tt.expectRetryCount, defaultHealthCheckDelay)
									}
								}
							}
						}
					}
				}
			}

			h.HealthCheck(ctx, tt.id, tt.retryCount)

			time.Sleep(time.Millisecond * 100)
		})
	}
}
