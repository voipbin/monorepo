package listenhandler

import (
	"errors"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/pkg/dbhandler"
	"monorepo/bin-outdial-manager/pkg/outdialhandler"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
)

// The tests in this file characterize the current behavior of processRequest:
// a request that cannot be decoded, or a business handler that returns a
// generic error, is answered with 400 (the legacy default branch), while
// dbhandler.ErrNotFound and a typed *cerrors.VoipbinError are mapped through
// errorResponse.

const (
	testOutdialID       = "9f4a6b1c-b9c8-11ef-8a01-0a1b2c3d4e5f"
	testOutdialTargetID = "a05b7c2d-b9c8-11ef-9b12-1b2c3d4e5f60"
)

// errorCase is one endpoint with a request that decodes correctly and the
// expectation on the business handler it reaches.
type errorCase struct {
	name string

	request *sock.Request
	// malformed is a request of the same endpoint whose body cannot be decoded.
	// It is nil when the endpoint has no body to decode.
	malformed *sock.Request

	// expect registers the call the endpoint makes and makes it return err.
	expect func(mockOutdial *outdialhandler.MockOutdialHandler, mockTarget *outdialtargethandler.MockOutdialTargetHandler, err error)
}

func errorCases() []errorCase {
	outdialURI := "/v1/outdials/" + testOutdialID
	targetURI := "/v1/outdialtargets/" + testOutdialTargetID

	return []errorCase{
		{
			name:      "outdials post",
			request:   &sock.Request{URI: "/v1/outdials", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"name":"n"}`)},
			malformed: &sock.Request{URI: "/v1/outdials", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(o *outdialhandler.MockOutdialHandler, _ *outdialtargethandler.MockOutdialTargetHandler, err error) {
				o.EXPECT().Create(gomock.Any(), uuid.Nil, uuid.Nil, "n", "", "").Return(nil, err)
			},
		},
		{
			name:    "outdials get",
			request: &sock.Request{URI: "/v1/outdials?page_size=5&page_token=t", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{}`)},
			expect: func(o *outdialhandler.MockOutdialHandler, _ *outdialtargethandler.MockOutdialTargetHandler, err error) {
				o.EXPECT().List(gomock.Any(), "t", uint64(5), gomock.Any()).Return(nil, err)
			},
		},
		{
			name:    "outdials id get",
			request: &sock.Request{URI: outdialURI, Method: sock.RequestMethodGet},
			expect: func(o *outdialhandler.MockOutdialHandler, _ *outdialtargethandler.MockOutdialTargetHandler, err error) {
				o.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testOutdialID)).Return(nil, err)
			},
		},
		{
			name:    "outdials id delete",
			request: &sock.Request{URI: outdialURI, Method: sock.RequestMethodDelete},
			expect: func(o *outdialhandler.MockOutdialHandler, _ *outdialtargethandler.MockOutdialTargetHandler, err error) {
				o.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testOutdialID)).Return(nil, err)
			},
		},
		{
			name:      "outdials id put",
			request:   &sock.Request{URI: outdialURI, Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"name":"n","detail":"d"}`)},
			malformed: &sock.Request{URI: outdialURI, Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(o *outdialhandler.MockOutdialHandler, _ *outdialtargethandler.MockOutdialTargetHandler, err error) {
				o.EXPECT().UpdateBasicInfo(gomock.Any(), uuid.FromStringOrNil(testOutdialID), "n", "d").Return(nil, err)
			},
		},
		{
			name:      "outdials id campaign_id put",
			request:   &sock.Request{URI: outdialURI + "/campaign_id", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"campaign_id":"b16c8d3e-b9c8-11ef-8c23-2c3d4e5f6071"}`)},
			malformed: &sock.Request{URI: outdialURI + "/campaign_id", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(o *outdialhandler.MockOutdialHandler, _ *outdialtargethandler.MockOutdialTargetHandler, err error) {
				o.EXPECT().UpdateCampaignID(gomock.Any(), uuid.FromStringOrNil(testOutdialID), uuid.FromStringOrNil("b16c8d3e-b9c8-11ef-8c23-2c3d4e5f6071")).Return(nil, err)
			},
		},
		{
			name:      "outdials id data put",
			request:   &sock.Request{URI: outdialURI + "/data", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"data":"x"}`)},
			malformed: &sock.Request{URI: outdialURI + "/data", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(o *outdialhandler.MockOutdialHandler, _ *outdialtargethandler.MockOutdialTargetHandler, err error) {
				o.EXPECT().UpdateData(gomock.Any(), uuid.FromStringOrNil(testOutdialID), "x").Return(nil, err)
			},
		},
		{
			name:      "outdials id available get",
			request:   &sock.Request{URI: outdialURI + "/available", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{}`)},
			malformed: &sock.Request{URI: outdialURI + "/available", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(_ *outdialhandler.MockOutdialHandler, ot *outdialtargethandler.MockOutdialTargetHandler, err error) {
				ot.EXPECT().GetAvailable(gomock.Any(), uuid.FromStringOrNil(testOutdialID), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
			},
		},
		{
			name:      "outdials id targets post",
			request:   &sock.Request{URI: outdialURI + "/targets", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"name":"n"}`)},
			malformed: &sock.Request{URI: outdialURI + "/targets", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(_ *outdialhandler.MockOutdialHandler, ot *outdialtargethandler.MockOutdialTargetHandler, err error) {
				ot.EXPECT().Create(gomock.Any(), uuid.FromStringOrNil(testOutdialID), "n", "", "", nil, nil, nil, nil, nil).Return(nil, err)
			},
		},
		{
			name:    "outdials id targets get",
			request: &sock.Request{URI: outdialURI + "/targets?page_size=3&page_token=t", Method: sock.RequestMethodGet},
			expect: func(_ *outdialhandler.MockOutdialHandler, ot *outdialtargethandler.MockOutdialTargetHandler, err error) {
				ot.EXPECT().GetsByOutdialID(gomock.Any(), uuid.FromStringOrNil(testOutdialID), "t", uint64(3)).Return(nil, err)
			},
		},
		{
			name:    "outdialtargets id get",
			request: &sock.Request{URI: targetURI, Method: sock.RequestMethodGet},
			expect: func(_ *outdialhandler.MockOutdialHandler, ot *outdialtargethandler.MockOutdialTargetHandler, err error) {
				ot.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testOutdialTargetID)).Return(nil, err)
			},
		},
		{
			name:    "outdialtargets id delete",
			request: &sock.Request{URI: targetURI, Method: sock.RequestMethodDelete},
			expect: func(_ *outdialhandler.MockOutdialHandler, ot *outdialtargethandler.MockOutdialTargetHandler, err error) {
				ot.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testOutdialTargetID)).Return(nil, err)
			},
		},
		{
			name:      "outdialtargets id progressing post",
			request:   &sock.Request{URI: targetURI + "/progressing", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"destination_index":2}`)},
			malformed: &sock.Request{URI: targetURI + "/progressing", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(_ *outdialhandler.MockOutdialHandler, ot *outdialtargethandler.MockOutdialTargetHandler, err error) {
				ot.EXPECT().UpdateProgressing(gomock.Any(), uuid.FromStringOrNil(testOutdialTargetID), 2).Return(nil, err)
			},
		},
		{
			name:      "outdialtargets id status put",
			request:   &sock.Request{URI: targetURI + "/status", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"status":"done"}`)},
			malformed: &sock.Request{URI: targetURI + "/status", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{broken`)},
			expect: func(_ *outdialhandler.MockOutdialHandler, ot *outdialtargethandler.MockOutdialTargetHandler, err error) {
				ot.EXPECT().UpdateStatus(gomock.Any(), uuid.FromStringOrNil(testOutdialTargetID), gomock.Any()).Return(nil, err)
			},
		},
	}
}

func newErrorListenHandler(mc *gomock.Controller) (*listenHandler, *outdialhandler.MockOutdialHandler, *outdialtargethandler.MockOutdialTargetHandler) {
	mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
	mockTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

	h := &listenHandler{
		sockHandler:          sockhandler.NewMockSockHandler(mc),
		outdialHandler:       mockOutdial,
		outdialTargetHandler: mockTarget,
	}

	return h, mockOutdial, mockTarget
}

func Test_processRequest_handlerError(t *testing.T) {
	typedNotFound := cerrors.NotFound("outdial-manager", "TEST_NOT_FOUND", "not found for test")

	tests := []struct {
		name string

		err        error
		expectCode int
	}{
		{"generic error is answered with 400", errors.New("boom"), http.StatusBadRequest},
		{"ErrNotFound is answered with 404", dbhandler.ErrNotFound, http.StatusNotFound},
		{"typed not found keeps its own code", typedNotFound, http.StatusNotFound},
	}

	for _, ec := range errorCases() {
		for _, tt := range tests {
			t.Run(ec.name+" "+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				h, mockOutdial, mockTarget := newErrorListenHandler(mc)
				ec.expect(mockOutdial, mockTarget, tt.err)

				res, err := h.processRequest(ec.request)
				if err != nil {
					t.Fatalf("processRequest must not return an error, got: %v", err)
				}
				if res == nil || res.StatusCode != tt.expectCode {
					t.Errorf("wrong status code. expect: %d, got: %+v", tt.expectCode, res)
				}
			})
		}
	}
}

func Test_processRequest_malformedBody(t *testing.T) {
	for _, ec := range errorCases() {
		if ec.malformed == nil {
			continue
		}

		t.Run(ec.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the business handlers have no EXPECT: any call fails the test
			h, _, _ := newErrorListenHandler(mc)

			res, err := h.processRequest(ec.malformed)
			if err != nil {
				t.Fatalf("processRequest must not return an error, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("wrong status code. expect: 400, got: %+v", res)
			}
		})
	}
}

func Test_v1OutdialsGet_malformedFilters(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"filters are not json", `{broken`},
		{"a filter value has the wrong type", `{"customer_id":"not-a-uuid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no List expectation: the handler must not be reached
			h, _, _ := newErrorListenHandler(mc)

			res, err := h.processRequest(&sock.Request{
				URI:      "/v1/outdials?page_size=5",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(tt.data),
			})
			if err != nil {
				t.Fatalf("processRequest must not return an error, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("wrong status code. expect: 400, got: %+v", res)
			}
		})
	}
}
