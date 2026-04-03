package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAppGroupHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.Cluster{},
		&entities.Env{},
		&entities.App{},
		&entities.AppGroup{},
		&entities.AppGroupMember{},
	))

	db.DB = testDB
}

func TestListSpecificGroupedAppsUsesStandardPaginationRequest(t *testing.T) {
	setupAppGroupHandlerTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-1",
		Name:       "Cluster 1",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "env-1",
		Name:             "Env 1",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "ns-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGroup{
		ID:    "group-1",
		EnvID: "env-1",
		Name:  "Group 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:         entities.Base{ID: "app-1"},
		Slug:         "app-1",
		Name:         "App 1",
		EnvID:        "env-1",
		AppType:      "Deployment",
		ContainerImage: "nginx:latest",
		DeployStatus: "undeployed",
		Replicas:     1,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGroupMember{
		ID:      "member-1",
		GroupID: "group-1",
		AppID:   "app-1",
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/app-groups/:groupID/apps", ListSpecificGroupedApps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-groups/group-1/apps?page=0&page_size=500", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Pagination struct {
				Page     int `json:"page"`
				PageSize int `json:"page_size"`
			} `json:"pagination"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Data.Pagination.Page)
	assert.Equal(t, 100, resp.Data.Pagination.PageSize)
}
