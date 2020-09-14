package calls

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/servicehandler"
)

func setupServer(app *gin.Engine) {
	ApplyRoutes(&app.RouterGroup)
}

func TestCallsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name string
		user user.User
		req  RequestBodyCallsPOST
		flow *flow.Flow
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			RequestBodyCallsPOST{
				Source: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "source@test.voipbin.net",
				},
				Destination: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "destination@test.voipbin.net",
				},
				Actions: []action.Action{},
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("044cf45a-f3a3-11ea-963d-1fc4372fcff8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", "/calls", bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().FlowCreate(&tt.user, uuid.Nil, "temp", "tmp outbound flow", tt.req.Actions, false).Return(tt.flow, nil)
			mockSvc.EXPECT().CallCreate(&tt.user, tt.flow.ID, tt.req.Source, tt.req.Destination).Return(nil, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}
