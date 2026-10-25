package listenhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

// listenMocks holds the downstream mocks. No expectation is set unless a case asks for one,
// so a downstream call that the handler must not make fails the test.
type listenMocks struct {
	conv *conversationhandler.MockConversationHandler
	acc  *accounthandler.MockAccountHandler
	msg  *messagehandler.MockMessageHandler
}

func newErrorPathHandler(mc *gomock.Controller) (*listenHandler, *listenMocks) {
	m := &listenMocks{
		conv: conversationhandler.NewMockConversationHandler(mc),
		acc:  accounthandler.NewMockAccountHandler(mc),
		msg:  messagehandler.NewMockMessageHandler(mc),
	}
	h := &listenHandler{
		conversationHandler: m.conv,
		accountHandler:      m.acc,
		messageHandler:      m.msg,
	}
	return h, m
}

func newReq(uri string, method sock.RequestMethod, data string) *sock.Request {
	r := &sock.Request{URI: uri, Method: method, DataType: "application/json"}
	if data != "" {
		r.Data = []byte(data)
	}
	return r
}

const (
	testConvID  = "8d8ab6ae-0074-11ee-80d0-df60c15605d7"
	testAccID   = "792c0222-e4a9-11ec-af5e-679fe5991907"
	testConvURI = "/v1/conversations/" + testConvID
	testAccURI  = "/v1/accounts/" + testAccID
)

// Test_listenHandler_inputValidation covers request rejection before any downstream handler is called.
func Test_listenHandler_inputValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		call func(h *listenHandler) (*sock.Response, error)

		expectStatus int
		expectErr    bool
	}{
		// malformed JSON body -> 400 response
		{"conversations post bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsPost(ctx, newReq("/v1/conversations", sock.RequestMethodPost, `{`))
		}, 400, false},
		{"conversations get_or_create bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsGetOrCreateBySelfAndPeerPost(ctx, newReq("/v1/conversations/get_or_create_by_self_and_peer", sock.RequestMethodPost, `{`))
		}, 400, false},
		{"conversations create_and_execute_flow bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsCreateAndExecuteFlowPost(ctx, newReq("/v1/conversations/create_and_execute_flow", sock.RequestMethodPost, `{`))
		}, 400, false},
		{"conversations metadata put bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDMetadataPut(ctx, newReq(testConvURI+"/metadata", sock.RequestMethodPut, `{`))
		}, 400, false},
		{"accounts post bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsPost(ctx, newReq("/v1/accounts", sock.RequestMethodPost, `{`))
		}, 400, false},
		{"messages post bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesPost(ctx, newReq("/v1/messages", sock.RequestMethodPost, `{`))
		}, 400, false},
		{"messages create post bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesCreatePost(ctx, newReq("/v1/messages/create", sock.RequestMethodPost, `{`))
		}, 400, false},
		{"hooks get bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1HooksGet(ctx, newReq("/v1/hooks", sock.RequestMethodGet, `{`))
		}, 400, false},
		{"hooks get bad received uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1HooksGet(ctx, newReq("/v1/hooks", sock.RequestMethodGet, `{"received_uri":"%zz"}`))
		}, 400, false},
		{"hooks post bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1HooksPost(ctx, newReq("/v1/hooks", sock.RequestMethodPost, `{`))
		}, 400, false},

		// filter that cannot be converted -> 400 response
		{"conversations get bad filter", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsGet(ctx, newReq("/v1/conversations?page_size=10", sock.RequestMethodGet, `{"customer_id":"not-a-uuid"}`))
		}, 400, false},
		{"accounts get bad filter", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsGet(ctx, newReq("/v1/accounts?page_size=10", sock.RequestMethodGet, `{"customer_id":"not-a-uuid"}`))
		}, 400, false},
		{"messages get bad filter", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesGet(ctx, newReq("/v1/messages?page_size=10", sock.RequestMethodGet, `{"customer_id":"not-a-uuid"}`))
		}, 400, false},

		// URI without the id segment -> 400 response (unreachable through the router, so called directly)
		{"conversations id get short uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDGet(ctx, newReq("/v1/conversations", sock.RequestMethodGet, ""))
		}, 400, false},
		{"conversations id put short uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDPut(ctx, newReq("/v1/conversations", sock.RequestMethodPut, `{"name":"x"}`))
		}, 400, false},
		{"conversations metadata put short uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDMetadataPut(ctx, newReq("/v1/conversations/x", sock.RequestMethodPut, `{}`))
		}, 400, false},
		{"accounts id get short uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDGet(ctx, newReq("/v1/accounts", sock.RequestMethodGet, ""))
		}, 400, false},
		{"accounts id put short uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDPut(ctx, newReq("/v1/accounts", sock.RequestMethodPut, `{"name":"x"}`))
		}, 400, false},
		{"accounts id delete short uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDDelete(ctx, newReq("/v1/accounts", sock.RequestMethodDelete, ""))
		}, 400, false},

		// malformed body or uri where the handler returns the error itself
		{"conversations get bad uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsGet(ctx, newReq("/v1/conversations/%zz", sock.RequestMethodGet, ""))
		}, 0, true},
		{"conversations get bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsGet(ctx, newReq("/v1/conversations", sock.RequestMethodGet, `{`))
		}, 0, true},
		{"accounts get bad uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsGet(ctx, newReq("/v1/accounts/%zz", sock.RequestMethodGet, "{}"))
		}, 0, true},
		{"accounts get bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsGet(ctx, newReq("/v1/accounts", sock.RequestMethodGet, `{`))
		}, 0, true},
		{"messages get bad uri", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesGet(ctx, newReq("/v1/messages/%zz", sock.RequestMethodGet, "{}"))
		}, 0, true},
		{"messages get bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesGet(ctx, newReq("/v1/messages", sock.RequestMethodGet, `{`))
		}, 0, true},
		{"conversations id put bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDPut(ctx, newReq(testConvURI, sock.RequestMethodPut, `{`))
		}, 0, true},
		{"conversations id put bad field value", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDPut(ctx, newReq(testConvURI, sock.RequestMethodPut, `{"owner_id":"not-a-uuid"}`))
		}, 0, true},
		{"accounts id put bad json", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDPut(ctx, newReq(testAccURI, sock.RequestMethodPut, `{`))
		}, 0, true},

		// body without any updatable field -> 200 and no update is requested
		{"conversations id put no allowed field", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDPut(ctx, newReq(testConvURI, sock.RequestMethodPut, `{"customer_id":"792c0222-e4a9-11ec-af5e-679fe5991907"}`))
		}, 200, false},
		{"accounts id put no allowed field", func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDPut(ctx, newReq(testAccURI, sock.RequestMethodPut, `{"customer_id":"792c0222-e4a9-11ec-af5e-679fe5991907"}`))
		}, 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newErrorPathHandler(mc)

			res, err := tt.call(h)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got response: %+v", res)
				}
				if res != nil {
					t.Errorf("expected nil response with the error, got: %+v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

// Test_listenHandler_downstreamError covers the mapping of a downstream handler error.
func Test_listenHandler_downstreamError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		setup func(m *listenMocks)
		call  func(h *listenHandler) (*sock.Response, error)

		expectStatus int
		expectErr    bool
	}{
		// errors that errorResponse maps (not found -> 404)
		{"conversations get", func(m *listenMocks) {
			m.conv.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsGet(ctx, newReq("/v1/conversations?page_size=10", sock.RequestMethodGet, ""))
		}, 404, false},
		{"conversations post", func(m *listenMocks) {
			m.conv.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsPost(ctx, newReq("/v1/conversations", sock.RequestMethodPost, `{}`))
		}, 404, false},
		{"conversations id get", func(m *listenMocks) {
			m.conv.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testConvID)).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDGet(ctx, newReq(testConvURI, sock.RequestMethodGet, ""))
		}, 404, false},
		{"conversations get_or_create", func(m *listenMocks) {
			m.conv.EXPECT().GetOrCreateBySelfAndPeer(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsGetOrCreateBySelfAndPeerPost(ctx, newReq("/v1/conversations/get_or_create_by_self_and_peer", sock.RequestMethodPost, `{}`))
		}, 404, false},
		{"conversations create_and_execute_flow", func(m *listenMocks) {
			m.conv.EXPECT().CreateAndExecuteFlow(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsCreateAndExecuteFlowPost(ctx, newReq("/v1/conversations/create_and_execute_flow", sock.RequestMethodPost, `{}`))
		}, 404, false},
		{"conversations metadata put", func(m *listenMocks) {
			m.conv.EXPECT().UpdateMetadata(gomock.Any(), uuid.FromStringOrNil(testConvID), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1ConversationsIDMetadataPut(ctx, newReq(testConvURI+"/metadata", sock.RequestMethodPut, `{}`))
		}, 404, false},
		{"accounts get", func(m *listenMocks) {
			m.acc.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsGet(ctx, newReq("/v1/accounts?page_size=10", sock.RequestMethodGet, "{}"))
		}, 404, false},
		{"accounts id get", func(m *listenMocks) {
			m.acc.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testAccID)).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDGet(ctx, newReq(testAccURI, sock.RequestMethodGet, ""))
		}, 404, false},
		{"accounts id put", func(m *listenMocks) {
			m.acc.EXPECT().Update(gomock.Any(), uuid.FromStringOrNil(testAccID), map[account.Field]any{account.FieldName: "x"}).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDPut(ctx, newReq(testAccURI, sock.RequestMethodPut, `{"name":"x"}`))
		}, 404, false},
		{"accounts id delete", func(m *listenMocks) {
			m.acc.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testAccID)).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsIDDelete(ctx, newReq(testAccURI, sock.RequestMethodDelete, ""))
		}, 404, false},
		{"messages post", func(m *listenMocks) {
			m.conv.EXPECT().MessageSend(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesPost(ctx, newReq("/v1/messages", sock.RequestMethodPost, `{}`))
		}, 404, false},
		{"messages create post", func(m *listenMocks) {
			m.msg.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesCreatePost(ctx, newReq("/v1/messages/create", sock.RequestMethodPost, `{}`))
		}, 404, false},

		// errors that are handed to the caller as-is
		{"accounts post", func(m *listenMocks) {
			m.acc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1AccountsPost(ctx, newReq("/v1/accounts", sock.RequestMethodPost, `{}`))
		}, 0, true},
		{"messages get", func(m *listenMocks) {
			m.msg.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
		}, func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesGet(ctx, newReq("/v1/messages?page_size=10", sock.RequestMethodGet, "{}"))
		}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrorPathHandler(mc)
			tt.setup(m)

			res, err := tt.call(h)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got response: %+v", res)
				}
				if res != nil {
					t.Errorf("expected nil response with the error, got: %+v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}
