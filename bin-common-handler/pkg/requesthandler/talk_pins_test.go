package requesthandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	tkpin "monorepo/bin-talk-manager/models/pin"
)

const (
	testPinCustomerID = "550e8400-e29b-41d4-a716-446655440000"
	testPinOwnerID    = "660e8400-e29b-41d4-a716-446655440000"
	testPinChatID     = "770e8400-e29b-41d4-a716-446655440000"
	testPinTargetID   = "880e8400-e29b-41d4-a716-446655440000"
	testPinID         = "990e8400-e29b-41d4-a716-446655440000"

	testPinJSON = `{"id":"` + testPinID + `","customer_id":"` + testPinCustomerID + `","owner_type":"agent","owner_id":"` + testPinOwnerID + `","chat_id":"` + testPinChatID + `","target_type":"message","target_id":"` + testPinTargetID + `","tm_create":"2026-11-02T09:15:30Z"}`
)

func testExpectedPin() *tkpin.Pin {
	tm := time.Date(2026, 11, 2, 9, 15, 30, 0, time.UTC)
	return &tkpin.Pin{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil(testPinID), CustomerID: uuid.FromStringOrNil(testPinCustomerID)},
		Owner:      commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.FromStringOrNil(testPinOwnerID)},
		ChatID:     uuid.FromStringOrNil(testPinChatID),
		TargetType: tkpin.TargetTypeMessage,
		TargetID:   uuid.FromStringOrNil(testPinTargetID),
		TMCreate:   &tm,
	}
}

func Test_TalkV1PinCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}
	ctx := context.Background()

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.talk-manager.request", &sock.Request{
		URI:      "/v1/pins",
		Method:   sock.RequestMethodPost,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"customer_id":"` + testPinCustomerID + `","owner_type":"agent","owner_id":"` + testPinOwnerID + `","target_type":"message","target_id":"` + testPinTargetID + `"}`),
	}).Return(&sock.Response{StatusCode: 200, DataType: ContentTypeJSON, Data: []byte(testPinJSON)}, nil)

	res, err := reqHandler.TalkV1PinCreate(ctx, uuid.FromStringOrNil(testPinCustomerID), "agent", uuid.FromStringOrNil(testPinOwnerID), tkpin.TargetTypeMessage, uuid.FromStringOrNil(testPinTargetID))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(testExpectedPin(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testExpectedPin(), res)
	}
}

func Test_TalkV1PinList(t *testing.T) {
	tests := []struct {
		name string

		targetType tkpin.TargetType
		chatID     uuid.UUID
		pageToken  string
		pageSize   uint64

		expectURI  string
		expectData string
	}{
		{
			name:       "no filters",
			pageSize:   10,
			expectURI:  "/v1/pins?page_token=&page_size=10",
			expectData: `{"customer_id":"` + testPinCustomerID + `","owner_type":"agent","owner_id":"` + testPinOwnerID + `"}`,
		},
		{
			name:       "filters and token",
			targetType: tkpin.TargetTypeMessage,
			chatID:     uuid.FromStringOrNil(testPinChatID),
			pageToken:  "2026-11-02T09:15:30.000000Z",
			pageSize:   5,
			expectURI:  "/v1/pins?page_token=2026-11-02T09%3A15%3A30.000000Z&page_size=5",
			expectData: `{"customer_id":"` + testPinCustomerID + `","owner_type":"agent","owner_id":"` + testPinOwnerID + `","target_type":"message","chat_id":"` + testPinChatID + `"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.talk-manager.request", &sock.Request{
				URI:      tt.expectURI,
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(tt.expectData),
			}).Return(&sock.Response{StatusCode: 200, DataType: ContentTypeJSON, Data: []byte(`[` + testPinJSON + `]`)}, nil)

			res, err := reqHandler.TalkV1PinList(context.Background(), uuid.FromStringOrNil(testPinCustomerID), "agent", uuid.FromStringOrNil(testPinOwnerID), tt.targetType, tt.chatID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual([]*tkpin.Pin{testExpectedPin()}, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testExpectedPin(), res)
			}
		})
	}
}

func Test_TalkV1PinDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.talk-manager.request", &sock.Request{
		URI:      "/v1/pins/" + testPinID,
		Method:   sock.RequestMethodDelete,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"customer_id":"` + testPinCustomerID + `","owner_type":"agent","owner_id":"` + testPinOwnerID + `"}`),
	}).Return(&sock.Response{StatusCode: 200, DataType: ContentTypeJSON, Data: []byte(testPinJSON)}, nil)

	res, err := reqHandler.TalkV1PinDelete(context.Background(), uuid.FromStringOrNil(testPinID), uuid.FromStringOrNil(testPinCustomerID), "agent", uuid.FromStringOrNil(testPinOwnerID))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(testExpectedPin(), res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", testExpectedPin(), res)
	}
}

// A 404 (the pin of another owner, an unknown target) reaches the caller as ErrNotFound, and a typed
// error body is passed through; this is what makes the api answer 404 instead of 500.
func Test_TalkV1Pin_errors(t *testing.T) {
	typed := `{"status":"NOT_FOUND","reason":"PIN_NOT_FOUND","domain":"talk-manager","message":"The pin was not found."}`

	tests := []struct {
		name     string
		response *sock.Response
		sendErr  error
		check    func(error) bool
	}{
		{"bare 404", &sock.Response{StatusCode: 404, DataType: ContentTypeJSON, Data: []byte(`{}`)}, nil, func(e error) bool { return errors.Is(e, ErrNotFound) }},
		{"typed 404", &sock.Response{StatusCode: 404, DataType: cerrors.DataTypeVoipbinError, Data: []byte(typed)}, nil, func(e error) bool {
			var ve *cerrors.VoipbinError
			return errors.As(e, &ve) && ve.Status == cerrors.StatusNotFound
		}},
		{"bad request", &sock.Response{StatusCode: 400, DataType: ContentTypeJSON, Data: []byte(`{}`)}, nil, func(e error) bool { return errors.Is(e, ErrBadRequest) }},
		{"transport error", nil, errors.New("rpc down"), func(e error) bool { return e != nil }},
	}

	calls := map[string]func(r *requestHandler) error{
		"create": func(r *requestHandler) error {
			_, err := r.TalkV1PinCreate(context.Background(), uuid.FromStringOrNil(testPinCustomerID), "agent", uuid.FromStringOrNil(testPinOwnerID), tkpin.TargetTypeChat, uuid.FromStringOrNil(testPinChatID))
			return err
		},
		"list": func(r *requestHandler) error {
			_, err := r.TalkV1PinList(context.Background(), uuid.FromStringOrNil(testPinCustomerID), "agent", uuid.FromStringOrNil(testPinOwnerID), "", uuid.Nil, "", 10)
			return err
		},
		"delete": func(r *requestHandler) error {
			_, err := r.TalkV1PinDelete(context.Background(), uuid.FromStringOrNil(testPinID), uuid.FromStringOrNil(testPinCustomerID), "agent", uuid.FromStringOrNil(testPinOwnerID))
			return err
		},
	}

	for callName, call := range calls {
		for _, tt := range tests {
			t.Run(callName+" "+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockSock := sockhandler.NewMockSockHandler(mc)
				reqHandler := requestHandler{sock: mockSock}
				mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.response, tt.sendErr)

				if err := call(&reqHandler); !tt.check(err) {
					t.Errorf("Wrong match. got: %v", err)
				}
			})
		}
	}
}
