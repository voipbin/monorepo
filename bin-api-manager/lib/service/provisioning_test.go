package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func setupProvisioningServer(app *gin.Engine) {
	provisioning := app.Group("/provisioning")
	provisioning.GET("/extension", GetProvisioningExtension)
}

func Test_provisioningExtensionGET_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token := strings.Repeat("ab", 32) // 64 lowercase hex chars
	responseXML := []byte(`<?xml version="1.0" encoding="UTF-8"?><config xmlns="http://www.linphone.org/xsds/lpconfig.xsd"></config>`)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	setupProvisioningServer(r)

	req, _ := http.NewRequest("GET", "/provisioning/extension?token="+token, nil)
	mockSvc.EXPECT().ExtensionProvisioningXMLGet(req.Context(), token).Return(responseXML, nil)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/xml; charset=utf-8" {
		t.Errorf("Wrong match. expect: %s, got: %s", "application/xml; charset=utf-8", contentType)
	}

	if w.Body.String() != string(responseXML) {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", responseXML, w.Body)
	}
}

func Test_provisioningExtensionGET_BadTokenFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		reqQuery string
	}{
		{"missing token", "/provisioning/extension"},
		{"empty token", "/provisioning/extension?token="},
		{"too short", "/provisioning/extension?token=abc123"},
		{"too long", "/provisioning/extension?token=" + strings.Repeat("a", 65)},
		{"uppercase hex", "/provisioning/extension?token=" + strings.Repeat("A", 64)},
		{"non hex", "/provisioning/extension?token=" + strings.Repeat("g", 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// The servicehandler must never be consulted for a malformed
			// token; no EXPECT is registered so any call fails the test.
			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
			})
			setupProvisioningServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
			if w.Body.Len() != 0 {
				t.Errorf("Wrong match. expect empty body, got: %s", w.Body)
			}
		})
	}
}

func Test_provisioningExtensionGET_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token := strings.Repeat("cd", 32)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set(common.OBJServiceHandler, mockSvc)
	})
	setupProvisioningServer(r)

	req, _ := http.NewRequest("GET", "/provisioning/extension?token="+token, nil)
	mockSvc.EXPECT().ExtensionProvisioningXMLGet(req.Context(), token).Return(nil, errors.New("token not found"))

	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}

	// Enumeration resistance: the failure body must be identical (empty) to
	// the bad-format case so callers cannot distinguish failure modes.
	if w.Body.Len() != 0 {
		t.Errorf("Wrong match. expect empty body, got: %s", w.Body)
	}
}
