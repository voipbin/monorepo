package listenhandler

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/pkg/dbhandler"
	"monorepo/bin-tag-manager/pkg/taghandler"
)

func Test_processRequest_handlerError(t *testing.T) {
	tagID := uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a")

	tests := []struct {
		name string

		request   *sock.Request
		getErr    error
		expectGet bool

		expectStatus int
	}{
		{
			name: "typed_error_keeps_its_status",
			request: &sock.Request{
				URI:    "/v1/tags/" + tagID.String(),
				Method: sock.RequestMethodGet,
			},
			getErr:       cerrors.NotFound(commonoutline.ServiceNameTagManager, "TAG_NOT_FOUND", "x"),
			expectGet:    true,
			expectStatus: 404,
		},
		{
			name: "db_not_found_is_404",
			request: &sock.Request{
				URI:    "/v1/tags/" + tagID.String(),
				Method: sock.RequestMethodGet,
			},
			getErr:       fmt.Errorf("lookup failed: %w", dbhandler.ErrNotFound),
			expectGet:    true,
			expectStatus: 404,
		},
		{
			name: "unclassified_error_is_400",
			request: &sock.Request{
				URI:    "/v1/tags/" + tagID.String(),
				Method: sock.RequestMethodGet,
			},
			getErr:       fmt.Errorf("unclassified"),
			expectGet:    true,
			expectStatus: 400,
		},
		{
			name: "list_uri_with_invalid_escape_is_400",
			request: &sock.Request{
				URI:    "/%zz/v1/tags?a=b",
				Method: sock.RequestMethodGet,
			},
			expectGet:    false,
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)
			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			if tt.expectGet {
				mockTag.EXPECT().Get(gomock.Any(), tagID).Return(nil, tt.getErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("processRequest should not return an error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_processV1TagsGet_badFilters(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "body_is_not_json",
			data: []byte("{not json"),
		},
		{
			name: "filter_value_cannot_be_converted",
			data: []byte(`{"customer_id":"not-a-uuid"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)
			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			// the tag handler mock has no expectation: any List call fails the test
			res, err := h.processRequest(&sock.Request{
				URI:      "/v1/tags?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     json.RawMessage(tt.data),
			})
			if err != nil {
				t.Fatalf("processRequest should not return an error, got: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("Wrong status code. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}

func Test_Run_consumeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)
	h := NewListenHandler(mockSock, mockTag)

	consumed := make(chan struct{})
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "tag-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
			defer close(consumed)
			return fmt.Errorf("consume failed")
		})

	if err := h.Run("test-queue", "test-delay"); err != nil {
		t.Fatalf("Run should not return the consume error, got: %v", err)
	}

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		t.Fatalf("ConsumeRPC was not called in time")
	}
}
