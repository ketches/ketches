package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAppSecretHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/app-secret-handler-test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.AppEnvVar{}, &entities.AppConfigFile{}))
	db.DB = testDB

	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.SecretEncryptionKey = "app-secret-handler-test-key"

	encryptedEnvValue, err := secrets.EncryptString("secret-env-value")
	require.NoError(t, err)
	encryptedContent, err := secrets.EncryptString("secret-file-content")
	require.NoError(t, err)
	require.NoError(t, db.DB.Create([]entities.AppEnvVar{
		{ID: "env-public", AppID: "app-1", Key: "PUBLIC_URL", Value: "https://example.com"},
		{ID: "env-secret", AppID: "app-1", Key: "API_TOKEN", Value: encryptedEnvValue, IsSecret: true},
	}).Error)
	require.NoError(t, db.DB.Create([]entities.AppConfigFile{
		{ID: "config-public", AppID: "app-1", Slug: "app.conf", MountPath: "/etc/app.conf", Content: "debug=false"},
		{ID: "config-secret", AppID: "app-1", Slug: "credentials", MountPath: "/etc/credentials", Content: encryptedContent, IsSecret: true},
	}).Error)
}

func TestAppConfigurationRoutesPassProjectRoleToFailClosedServiceBoundary(t *testing.T) {
	setupAppSecretHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		role   *string
		reveal bool
	}{
		{name: "owner", role: stringPtr(app.ProjectRoleOwner), reveal: true},
		{name: "developer", role: stringPtr(app.ProjectRoleDeveloper), reveal: true},
		{name: "viewer", role: stringPtr(app.ProjectRoleViewer)},
		{name: "missing role", role: nil},
		{name: "unknown role", role: stringPtr("auditor")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, endpoint := range []struct {
				route          string
				path           string
				handler        gin.HandlerFunc
				expectedSecret string
				decode         func([]byte) ([]string, error)
			}{
				{
					route:          "/apps/:appID/env-vars",
					path:           "/apps/app-1/env-vars",
					handler:        ListAppEnvVars,
					expectedSecret: "secret-env-value",
					decode: func(body []byte) ([]string, error) {
						var response struct {
							Data []models.AppEnvVarResponse `json:"data"`
						}
						if err := json.Unmarshal(body, &response); err != nil {
							return nil, err
						}
						values := make([]string, 0, len(response.Data))
						for _, item := range response.Data {
							values = append(values, item.Value)
						}
						return values, nil
					},
				},
				{
					route:          "/apps/:appID/config-files",
					path:           "/apps/app-1/config-files",
					handler:        ListAppConfigFiles,
					expectedSecret: "secret-file-content",
					decode: func(body []byte) ([]string, error) {
						var response struct {
							Data []models.AppConfigFileResponse `json:"data"`
						}
						if err := json.Unmarshal(body, &response); err != nil {
							return nil, err
						}
						values := make([]string, 0, len(response.Data))
						for _, item := range response.Data {
							values = append(values, item.Content)
						}
						return values, nil
					},
				},
			} {
				router := gin.New()
				middleware := func(c *gin.Context) {
					if tt.role != nil {
						api.SetProjectRole(c, *tt.role)
					}
					c.Next()
				}
				router.GET(endpoint.route, middleware, endpoint.handler)

				request := httptest.NewRequest(http.MethodGet, endpoint.path, nil)
				writer := httptest.NewRecorder()
				router.ServeHTTP(writer, request)
				require.Equal(t, http.StatusOK, writer.Code)
				values, err := endpoint.decode(writer.Body.Bytes())
				require.NoError(t, err)
				if tt.reveal {
					assert.Contains(t, values, endpoint.expectedSecret, endpoint.path)
				} else {
					for _, value := range values {
						assert.Empty(t, value, endpoint.path)
					}
				}
			}
		})
	}
}

func stringPtr(value string) *string {
	return &value
}
