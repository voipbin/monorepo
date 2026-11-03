package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tkmessage "monorepo/bin-talk-manager/models/message"
)

func Test_GetServiceAgentsTalkMentions(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")},
	})

	msg := &tkmessage.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
			CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		ChatID:   uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
		Type:     tkmessage.TypeNormal,
		Text:     "hi",
		TMCreate: timePtr("2020-09-20T03:23:20.995000Z"),
	}

	tests := []struct {
		name string

		reqQuery string

		responseMessages []*tkmessage.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name:     "the next page token is the last message's creation time",
			reqQuery: "/service_agents/talk_mentions?page_token=2021-01-01T00:00:00.000000Z&page_size=10",

			responseMessages: []*tkmessage.WebhookMessage{msg},

			expectPageToken: "2021-01-01T00:00:00.000000Z",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"93d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chat_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","type":"normal","text":"hi","medias":null,"metadata":{"reactions":null},"tm_create":"2020-09-20T03:23:20.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:20.995000Z"}`,
		},
		{
			name:     "defaults to 100",
			reqQuery: "/service_agents/talk_mentions",

			responseMessages: []*tkmessage.WebhookMessage{},

			expectPageToken: "",
			expectPageSize:  100,
			expectRes:       `{"result":[],"next_page_token":""}`,
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentTalkMentionList(req.Context(), agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseMessages, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_GetServiceAgentsTalkMentions_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/service_agents/talk_mentions", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().ServiceAgentTalkMentionList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}
