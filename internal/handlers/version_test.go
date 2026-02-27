package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/handlers"
)

func TestGetVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/version", handlers.GetVersion)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data struct {
			Version   string `json:"version"`
			BuildTime string `json:"build_time"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.Version == "" {
		t.Error("expected non-empty version")
	}
	if resp.Data.BuildTime == "" {
		t.Error("expected non-empty build_time")
	}
}
