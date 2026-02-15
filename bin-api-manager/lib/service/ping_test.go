package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetPing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetPing(c)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got: %v", w.Code)
	}

	expectedBody := `{"message":"pong"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %v, got: %v", expectedBody, w.Body.String())
	}
}
