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
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmaction"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestFlowsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        models.User
		requestBody request.BodyFlowsPOST
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID:         1,
				Permission: models.UserPermissionAdmin,
			},
			request.BodyFlowsPOST{
				Name:   "test name",
				Detail: "test detail",
				Actions: []models.Action{
					{
						Type: models.ActionTypeAnswer,
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
				c.Set(models.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().FlowCreate(&tt.user, gomock.Any(), tt.requestBody.Name, tt.requestBody.Detail, tt.requestBody.Actions, true).Return(&models.Flow{}, nil)
			req, _ := http.NewRequest("POST", "/v1.0/flows", bytes.NewBuffer(body))
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

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name string
		user models.User
		flow *fmflow.Flow

		expectFlow *models.Flow
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID: 1,
			},
			&fmflow.Flow{
				ID:     uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				Name:   "test name",
				Detail: "test detail",
				UserID: 1,
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
						Type: "answer",
					},
				},
			},
			&models.Flow{
				ID:     uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				Name:   "test name",
				Detail: "test detail",
				UserID: 1,
				Actions: []models.Action{
					{
						ID:   uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
						Type: models.ActionTypeAnswer,
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
				c.Set(models.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().FlowGet(&tt.user, tt.flow.ID).Return(tt.expectFlow, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/flows/%s", tt.flow.ID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestFlowsIDPUT(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name        string
		user        models.User
		flowID      uuid.UUID
		requestBody request.BodyFlowsIDPUT
		expectFlow  *models.Flow
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID:         1,
				Permission: models.UserPermissionAdmin,
			},
			uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
			request.BodyFlowsIDPUT{
				Name:   "test name",
				Detail: "test detail",
				Actions: []models.Action{
					{
						Type: models.ActionTypeAnswer,
					},
				},
			},
			&models.Flow{
				ID:     uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []models.Action{
					{
						Type: models.ActionTypeAnswer,
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
				c.Set(models.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().FlowUpdate(&tt.user, tt.expectFlow).Return(&models.Flow{}, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/flows/"+tt.flowID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestFlowsIDDELETE(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name   string
		user   models.User
		flowID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			models.User{
				ID: 1,
			},
			uuid.FromStringOrNil("d466f900-67cb-11eb-b2ff-1f9adc48f842"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(models.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().FlowDelete(&tt.user, tt.flowID).Return(nil)
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/flows/%s", tt.flowID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
