package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRecycleBinSecurityDB(t *testing.T) {
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
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Env{},
		&entities.App{},
		&entities.CodeRepository{},
	))

	db.DB = testDB
}

func seedRecycleBinProject(t *testing.T, projectID, ownerID string) {
	t.Helper()
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: projectID},
		Slug: projectID,
		Name: projectID,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          projectID + "-member-" + ownerID,
		ProjectID:   projectID,
		UserID:      ownerID,
		ProjectRole: app.ProjectRoleOwner,
	}).Error)
}

func TestRecycleBinPermanentDeleteRequiresOwnerAndSoftDelete(t *testing.T) {
	setupRecycleBinSecurityDB(t)
	seedRecycleBinProject(t, "project-owned", "owner-1")
	seedRecycleBinProject(t, "project-other", "owner-2")

	require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", "project-other").Error)
	require.ErrorIs(t, PermanentlyDeleteProject("project-other"), ErrRecycleBinAccessDenied)
	err := BatchPermanentlyDeleteProjects([]string{"project-other"}, RecycleBinActor{UserID: "owner-1", Role: app.UserRoleUser})
	require.ErrorIs(t, err, ErrRecycleBinAccessDenied)

	var otherProject entities.Project
	require.NoError(t, db.DB.Unscoped().First(&otherProject, "id = ?", "project-other").Error)
	require.True(t, otherProject.DeletedAt.Valid)

	err = BatchPermanentlyDeleteProjects([]string{"project-owned"}, RecycleBinActor{UserID: "owner-1", Role: app.UserRoleUser})
	require.ErrorIs(t, err, ErrRecycleBinResourceActive)

	require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", "project-owned").Error)
	require.NoError(t, BatchPermanentlyDeleteProjects([]string{"project-owned"}, RecycleBinActor{UserID: "owner-1", Role: app.UserRoleUser}))
	require.ErrorIs(t, db.DB.Unscoped().First(&otherProject, "id = ?", "project-owned").Error, gorm.ErrRecordNotFound)
}

func TestRecycleBinBatchValidationIsAtomic(t *testing.T) {
	setupRecycleBinSecurityDB(t)
	seedRecycleBinProject(t, "project-deleted", "owner-1")
	seedRecycleBinProject(t, "project-active", "owner-1")
	require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", "project-deleted").Error)

	err := BatchPermanentlyDeleteProjects(
		[]string{"project-deleted", "project-active"},
		RecycleBinActor{UserID: "owner-1", Role: app.UserRoleUser},
	)
	require.ErrorIs(t, err, ErrRecycleBinResourceActive)

	var deletedProject entities.Project
	require.NoError(t, db.DB.Unscoped().First(&deletedProject, "id = ?", "project-deleted").Error)
	require.True(t, deletedProject.DeletedAt.Valid, "the valid ID must not be partially deleted")
}

func TestRecycleBinResourceChecksCoverEnvironmentAppAndRepository(t *testing.T) {
	setupRecycleBinSecurityDB(t)
	seedRecycleBinProject(t, "project-owned", "owner-1")
	seedRecycleBinProject(t, "project-other", "owner-2")

	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-owned"},
		Slug:      "env-owned",
		Name:      "env-owned",
		ProjectID: "project-owned",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-other"},
		Slug:      "env-other",
		Name:      "env-other",
		ProjectID: "project-other",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-owned"},
		Slug:           "app-owned",
		Name:           "app-owned",
		EnvID:          "env-owned",
		ContainerImage: "busybox",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-other"},
		Slug:           "app-other",
		Name:           "app-other",
		EnvID:          "env-other",
		ContainerImage: "busybox",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.CodeRepository{
		Base:       entities.Base{ID: "repo-owned"},
		ProjectID:  "project-owned",
		Name:       "repo-owned",
		Slug:       "repo-owned",
		GitRepoURL: "https://example.com/owned.git",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.CodeRepository{
		Base:       entities.Base{ID: "repo-other"},
		ProjectID:  "project-other",
		Name:       "repo-other",
		Slug:       "repo-other",
		GitRepoURL: "https://example.com/other.git",
	}).Error)

	owner := RecycleBinActor{UserID: "owner-1", Role: app.UserRoleUser}
	require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", "env-other").Error)
	require.ErrorIs(t, BatchPermanentlyDeleteEnvs([]string{"env-other"}, owner), ErrRecycleBinAccessDenied)
	require.ErrorIs(t, BatchPermanentlyDeleteEnvs([]string{"env-owned"}, owner), ErrRecycleBinResourceActive)

	require.NoError(t, db.DB.Delete(&entities.App{}, "id = ?", "app-other").Error)
	require.ErrorIs(t, BatchPermanentlyDeleteApps([]string{"app-other"}, owner), ErrRecycleBinAccessDenied)
	require.ErrorIs(t, BatchPermanentlyDeleteApps([]string{"app-owned"}, owner), ErrRecycleBinResourceActive)

	require.NoError(t, db.DB.Delete(&entities.CodeRepository{}, "id = ?", "repo-other").Error)
	require.ErrorIs(t, BatchPermanentlyDeleteCodeRepositories([]string{"repo-other"}, owner), ErrRecycleBinAccessDenied)
	require.ErrorIs(t, BatchPermanentlyDeleteCodeRepositories([]string{"repo-owned"}, owner), ErrRecycleBinResourceActive)

	var activeApp entities.App
	require.NoError(t, db.DB.First(&activeApp, "id = ?", "app-owned").Error)
	var activeRepo entities.CodeRepository
	require.NoError(t, db.DB.First(&activeRepo, "id = ?", "repo-owned").Error)
}
