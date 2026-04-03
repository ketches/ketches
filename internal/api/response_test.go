package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	appcore "github.com/ketches/ketches/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestErrorSanitizesWrappedRecordNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/apps/app-1", nil)

	Error(c, http.StatusNotFound, appcore.WrapError("app not found", gorm.ErrRecordNotFound))

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusText(http.StatusNotFound), resp.Error)
}

func TestErrorSanitizesDatabaseConstraintDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)

	Error(c, http.StatusBadRequest, errors.New("UNIQUE constraint failed: users.email"))

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusText(http.StatusBadRequest), resp.Error)
}

func TestErrorKeepsBusinessValidationMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)

	Error(c, http.StatusBadRequest, errors.New("username is required"))

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "username is required", resp.Error)
}
