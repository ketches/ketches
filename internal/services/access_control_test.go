package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAccessControlTestDB(t *testing.T) {
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
		&entities.User{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Cluster{},
		&entities.Env{},
	))

	db.DB = testDB
}

func TestEnsureProjectAccess(t *testing.T) {
	setupAccessControlTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          "member-1",
		ProjectID:   "project-1",
		UserID:      "user-1",
		ProjectRole: app.ProjectRoleDeveloper,
	}).Error)

	assert.NoError(t, EnsureProjectAccess("admin-1", app.UserRoleAdmin, ""))
	assert.ErrorIs(t, EnsureProjectAccess("user-1", app.UserRoleUser, ""), ErrProjectScopeRequired)
	assert.ErrorIs(t, EnsureProjectAccess("user-2", app.UserRoleUser, "project-1"), ErrProjectAccessDenied)
	assert.NoError(t, EnsureProjectAccess("user-1", app.UserRoleUser, "project-1"))
}

func TestEnsureClusterProjectAccess(t *testing.T) {
	setupAccessControlTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          "member-1",
		ProjectID:   "project-1",
		UserID:      "user-1",
		ProjectRole: app.ProjectRoleDeveloper,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-1",
		Name:       "Cluster 1",
		Enabled:    true,
		KubeConfig: "enc:v1:test",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-2"},
		Slug:       "cluster-2",
		Name:       "Cluster 2",
		Enabled:    true,
		KubeConfig: "enc:v1:test",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-1"},
		Slug:      "env-1",
		Name:      "Env 1",
		ProjectID: "project-1",
		ClusterID: "cluster-1",
	}).Error)

	assert.NoError(t, EnsureClusterProjectAccess("admin-1", app.UserRoleAdmin, "", "cluster-2"))
	assert.ErrorIs(t, EnsureClusterProjectAccess("user-1", app.UserRoleUser, "", "cluster-1"), ErrProjectScopeRequired)
	assert.ErrorIs(t, EnsureClusterProjectAccess("user-2", app.UserRoleUser, "project-1", "cluster-1"), ErrProjectAccessDenied)
	assert.ErrorIs(t, EnsureClusterProjectAccess("user-1", app.UserRoleUser, "project-1", "cluster-2"), ErrClusterProjectDenied)
	assert.NoError(t, EnsureClusterProjectAccess("user-1", app.UserRoleUser, "project-1", "cluster-1"))
}

func TestListProjectClustersSimple(t *testing.T) {
	setupAccessControlTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:             entities.Base{ID: "cluster-1"},
		Slug:             "cluster-1",
		Name:             "Cluster 1",
		KubeConfig:       "enc:v1:test",
		ConnectionStatus: "ready",
		Enabled:          true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-1"},
		Slug:      "env-1",
		Name:      "Env 1",
		ProjectID: "project-1",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-2"},
		Slug:      "env-2",
		Name:      "Env 2",
		ProjectID: "project-1",
		ClusterID: "cluster-1",
	}).Error)

	clusters, err := ListProjectClustersSimple("project-1")
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.Equal(t, "cluster-1", clusters[0].ID)
}
