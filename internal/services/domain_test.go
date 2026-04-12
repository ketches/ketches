package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDomainTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Cluster{}, &entities.Project{}, &entities.Env{}, &entities.Domain{}))
	db.DB = testDB
}

func TestCreateClusterDomain(t *testing.T) {
	setupDomainTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base: entities.Base{ID: "cluster-1"},
		Slug: "cluster-1",
		Name: "Cluster 1",
	}).Error)

	item, err := CreateClusterDomain("cluster-1", &models.CreateDomainRequest{
		Name:        "Primary",
		Domain:      "*.Example.COM",
		Description: "primary domain",
	})
	require.NoError(t, err)
	assert.Equal(t, "Primary", item.Name)
	assert.Equal(t, "*.example.com", item.Domain)
	assert.Equal(t, "cluster", item.Scope)
	assert.Equal(t, "cluster-1", item.ClusterID)
}

func TestCreateEnvDomain(t *testing.T) {
	setupDomainTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base: entities.Base{ID: "cluster-1"},
		Slug: "cluster-1",
		Name: "Cluster 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-1"},
		Slug:      "env-1",
		Name:      "Env 1",
		ProjectID: "project-1",
		ClusterID: "cluster-1",
	}).Error)

	item, err := CreateEnvDomain("env-1", &models.CreateDomainRequest{
		Name:   "Env Primary",
		Domain: "*.env.example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, item.EnvID)
	assert.Equal(t, "env", item.Scope)
	assert.Equal(t, "env-1", *item.EnvID)
	assert.Equal(t, "cluster-1", item.ClusterID)
}
