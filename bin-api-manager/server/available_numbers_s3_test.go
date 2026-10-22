package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"

	"go.uber.org/mock/gomock"
)

func Test_GetAvailableNumbers_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/available_numbers?page_size=20&country_code=US&type=normal", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "missing_country_code", method: "GET", path: "/available_numbers?page_size=20&type=normal", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT"},
		{name: "service_error", method: "GET", path: "/available_numbers?page_size=20&country_code=US&type=normal", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AvailableNumberList(gomock.Any(), a, uint64(20), "US", "normal").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}
