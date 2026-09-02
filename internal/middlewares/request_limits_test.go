package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
)

func TestRequestBodyLimitRejectsKnownOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handlerCalled := false
	router := gin.New()
	router.Use(RequestBodyLimit())
	router.POST("/api/v1/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewReader([]byte("small")))
	request.ContentLength = maxRequestBodyBytes + 1
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if handlerCalled {
		t.Fatal("handler was called for an oversized request")
	}
}

func TestRequestBodyLimitBoundsUnknownLengthBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestBodyLimit())
	router.POST("/api/v1/test", func(c *gin.Context) {
		_, err := io.Copy(io.Discard, c.Request.Body)
		if err != nil {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	body := bytes.Repeat([]byte("x"), maxRequestBodyBytes+1)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewReader(body))
	request.ContentLength = -1
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRequestBodyLimitUsesBoundedUploadAllowance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestBodyLimit())
	router.POST("/api/v1/apps/app-1/files/upload", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/apps/app-1/files/upload", bytes.NewReader([]byte("small")))
	request.ContentLength = maxRequestBodyBytes + 1
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("upload status = %d, want %d", response.Code, http.StatusNoContent)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/apps/app-1/files/upload", bytes.NewReader([]byte("small")))
	request.ContentLength = maxUploadBodyBytes + 1
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized upload status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}
