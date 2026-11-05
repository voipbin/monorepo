package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_convertConversationMedias(t *testing.T) {
	fileID := "6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61"
	imageType := openapi_server.ConversationManagerMediaType("image")
	filename := "a.jpg"
	customerID := "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"
	badID := "abc"
	emptyID := ""

	tests := []struct {
		name string

		in []openapi_server.ConversationManagerMedia

		expectRes []cvmedia.Media
	}{
		{
			name:      "nil list is an empty list that is not nil",
			in:        nil,
			expectRes: []cvmedia.Media{},
		},
		{
			name:      "empty list is an empty list that is not nil",
			in:        []openapi_server.ConversationManagerMedia{},
			expectRes: []cvmedia.Media{},
		},
		{
			name: "id type and filename are copied and the customer id is dropped",
			in: []openapi_server.ConversationManagerMedia{
				{Id: &fileID, Type: &imageType, Filename: &filename, CustomerId: &customerID},
			},
			expectRes: []cvmedia.Media{
				{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(fileID)}, Type: cvmedia.TypeImage, Filename: "a.jpg"},
			},
		},
		{
			name: "nil pointers give empty values",
			in:   []openapi_server.ConversationManagerMedia{{}},
			expectRes: []cvmedia.Media{
				{},
			},
		},
		{
			name: "a malformed or empty id becomes the nil uuid",
			in: []openapi_server.ConversationManagerMedia{
				{Id: &badID},
				{Id: &emptyID},
			},
			expectRes: []cvmedia.Media{
				{Identity: commonidentity.Identity{ID: uuid.Nil}},
				{Identity: commonidentity.Identity{ID: uuid.Nil}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := convertConversationMedias(tt.in)
			if res == nil {
				t.Fatalf("the result must not be nil")
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_PostConversationsIdMessages_medias(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f")},
	})
	conversationID := uuid.FromStringOrNil("5950b02c-ed2f-11ec-9093-d3dcc91a72fa")
	fileID := uuid.FromStringOrNil("6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61")

	tests := []struct {
		name string

		path string
		body string

		expectMedias []cvmedia.Media
	}{
		{
			name:         "an id is passed on",
			path:         "/conversations/5950b02c-ed2f-11ec-9093-d3dcc91a72fa/messages",
			body:         `{"text":"hi","medias":[{"id":"6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61","type":"image","filename":"a.jpg","customer_id":"7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"}]}`,
			expectMedias: []cvmedia.Media{{Identity: commonidentity.Identity{ID: fileID}, Type: cvmedia.TypeImage, Filename: "a.jpg"}},
		},
		{
			name:         "an item shaped like the square-talk request has no id and passes on as the nil id",
			path:         "/conversations/5950b02c-ed2f-11ec-9093-d3dcc91a72fa/messages",
			body:         `{"text":"hi","medias":[{"type":"file","file_id":"6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61","name":"a.pdf"}]}`,
			expectMedias: []cvmedia.Media{{Type: cvmedia.TypeFile}},
		},
		{
			name:         "no medias key is an empty list",
			path:         "/conversations/5950b02c-ed2f-11ec-9093-d3dcc91a72fa/messages",
			body:         `{"text":"hi"}`,
			expectMedias: []cvmedia.Media{},
		},
		{
			name:         "null medias is an empty list",
			path:         "/conversations/5950b02c-ed2f-11ec-9093-d3dcc91a72fa/messages",
			body:         `{"text":"hi","medias":null}`,
			expectMedias: []cvmedia.Media{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationMessageSend(req.Context(), agent, conversationID, "hi", tt.expectMedias).Return(&cvmessage.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_PostServiceAgentsConversationsIdMessages_medias(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f")},
	})
	conversationID := uuid.FromStringOrNil("5950b02c-ed2f-11ec-9093-d3dcc91a72fa")
	fileID := uuid.FromStringOrNil("6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61")

	tests := []struct {
		name string

		body string

		expectMedias []cvmedia.Media
	}{
		{
			name:         "an id is passed on",
			body:         `{"text":"hi","medias":[{"id":"6a1c4d63-2d3e-11f1-9b22-1c2d3e4f5a61"}]}`,
			expectMedias: []cvmedia.Media{{Identity: commonidentity.Identity{ID: fileID}}},
		},
		{
			name:         "no medias key is an empty list",
			body:         `{"text":"hi"}`,
			expectMedias: []cvmedia.Media{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", "/service_agents/conversations/5950b02c-ed2f-11ec-9093-d3dcc91a72fa/messages", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentConversationMessageSend(req.Context(), agent, conversationID, "hi", tt.expectMedias).Return(&cvmessage.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
