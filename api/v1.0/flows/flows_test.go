package flows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/servicehandler"
)

func setupServer(app *gin.Engine) {
	ApplyRoutes(&app.RouterGroup)
}

func TestFlowsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        user.User
		requestBody request.BodyFlowsPOST
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},
			request.BodyFlowsPOST{
				Name:   "test name",
				Detail: "test detail",
				Actions: []action.Action{
					action.Action{
						Type: action.TypeAnswer,
					},
				},
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
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().FlowCreate(&tt.user, gomock.Any(), tt.requestBody.Name, tt.requestBody.Detail, tt.requestBody.Actions, true).Return(&flow.Flow{}, nil)
			req, _ := http.NewRequest("POST", "/flows", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestFlowsIDGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name string
		user user.User
		flow *fmflow.Flow

		expectFlow *flow.Flow
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			&fmflow.Flow{
				ID:     uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				Name:   "test name",
				Detail: "test detail",
				UserID: 1,
				Actions: []action.Action{
					action.Action{
						ID:   uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
						Type: action.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				Name:   "test name",
				Detail: "test detail",
				UserID: 1,
				Actions: []action.Action{
					action.Action{
						ID:   uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
						Type: action.TypeAnswer,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("requestHandler", mockReq)
				c.Set(api.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().FlowGet(&tt.user, tt.flow.ID).Return(tt.expectFlow, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/flows/%s", tt.flow.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
