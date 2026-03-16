package services

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserRecycleServiceTestDB(t *testing.T) {
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
	))

	db.DB = testDB
}

func seedUserOwnedProject(t *testing.T, userID, projectID string) {
	t.Helper()

	user := entities.User{
		Base:     entities.Base{ID: userID},
		Username: "user-" + userID,
		Email:    userID + "@example.com",
		Password: "password",
		Role:     app.UserRoleUser,
	}
	project := entities.Project{
		Base: entities.Base{ID: projectID},
		Slug: "project-" + projectID,
		Name: "Project " + projectID,
	}
	member := entities.ProjectMember{
		ID:          "member-" + userID + "-" + projectID,
		ProjectID:   projectID,
		UserID:      userID,
		ProjectRole: app.ProjectRoleOwner,
	}

	require.NoError(t, db.DB.Create(&user).Error)
	require.NoError(t, db.DB.Create(&project).Error)
	require.NoError(t, db.DB.Create(&member).Error)
}

func TestDeleteUserSoftDeletesOwnedProjects(t *testing.T) {
	setupUserRecycleServiceTestDB(t)

	userID := "user-delete"
	projectID := "project-delete"
	seedUserOwnedProject(t, userID, projectID)

	err := DeleteUser(userID)
	require.NoError(t, err)

	var deletedUser entities.User
	require.NoError(t, db.DB.Unscoped().First(&deletedUser, "id = ?", userID).Error)
	assert.True(t, deletedUser.DeletedAt.Valid)

	var deletedProject entities.Project
	require.NoError(t, db.DB.Unscoped().First(&deletedProject, "id = ?", projectID).Error)
	assert.True(t, deletedProject.DeletedAt.Valid)
}

func TestPermanentlyDeleteUserDeletesOwnedProjects(t *testing.T) {
	setupUserRecycleServiceTestDB(t)

	userID := "user-permanent"
	projectID := "project-permanent"
	seedUserOwnedProject(t, userID, projectID)

	require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error)
	require.NoError(t, db.DB.Delete(&entities.User{}, "id = ?", userID).Error)

	err := PermanentlyDeleteUser(userID)
	require.NoError(t, err)

	var deletedUser entities.User
	err = db.DB.Unscoped().First(&deletedUser, "id = ?", userID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var deletedProject entities.Project
	err = db.DB.Unscoped().First(&deletedProject, "id = ?", projectID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var memberCount int64
	require.NoError(t, db.DB.Model(&entities.ProjectMember{}).
		Where("user_id = ? OR project_id = ?", userID, projectID).
		Count(&memberCount).Error)
	assert.Equal(t, int64(0), memberCount)
}

func TestPermanentlyDeleteUserRejectsActiveUser(t *testing.T) {
	setupUserRecycleServiceTestDB(t)

	userID := "user-active-hard-delete"
	projectID := "project-active-hard-delete"
	seedUserOwnedProject(t, userID, projectID)

	err := PermanentlyDeleteUser(userID)
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot permanently delete active user")

	var user entities.User
	require.NoError(t, db.DB.First(&user, "id = ?", userID).Error)

	var project entities.Project
	require.NoError(t, db.DB.First(&project, "id = ?", projectID).Error)
}

func TestRestoreUserAlsoRestoresOwnedProjects(t *testing.T) {
	setupUserRecycleServiceTestDB(t)

	userID := "user-restore"
	projectID := "project-restore"
	seedUserOwnedProject(t, userID, projectID)

	require.NoError(t, DeleteUser(userID))
	require.NoError(t, RestoreUser(userID))

	var restoredUser entities.User
	require.NoError(t, db.DB.Unscoped().First(&restoredUser, "id = ?", userID).Error)
	assert.False(t, restoredUser.DeletedAt.Valid)

	var restoredProject entities.Project
	require.NoError(t, db.DB.Unscoped().First(&restoredProject, "id = ?", projectID).Error)
	assert.False(t, restoredProject.DeletedAt.Valid)
}

func TestRestoreUserRejectsActiveUser(t *testing.T) {
	setupUserRecycleServiceTestDB(t)

	userID := "user-active-restore"
	projectID := "project-active-restore"
	seedUserOwnedProject(t, userID, projectID)

	err := RestoreUser(userID)
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot restore active user")

	var user entities.User
	require.NoError(t, db.DB.First(&user, "id = ?", userID).Error)

	var project entities.Project
	require.NoError(t, db.DB.First(&project, "id = ?", projectID).Error)
}

func TestRecycleBinUsersListAndBatchActions(t *testing.T) {
	setupUserRecycleServiceTestDB(t)

	deletedUserID := "user-rb-deleted"
	deletedProjectID := "project-rb-deleted"
	activeUserID := "user-rb-active"
	activeProjectID := "project-rb-active"

	seedUserOwnedProject(t, deletedUserID, deletedProjectID)
	seedUserOwnedProject(t, activeUserID, activeProjectID)

	require.NoError(t, DeleteUser(deletedUserID))

	total, deletedUsers, err := ListDeletedUsers(1, 10, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, deletedUsers, 1)
	assert.Equal(t, deletedUserID, deletedUsers[0].ID)

	require.NoError(t, BatchRestoreUsers([]string{deletedUserID}))

	var restoredUser entities.User
	require.NoError(t, db.DB.Unscoped().First(&restoredUser, "id = ?", deletedUserID).Error)
	assert.False(t, restoredUser.DeletedAt.Valid)

	var restoredProject entities.Project
	require.NoError(t, db.DB.Unscoped().First(&restoredProject, "id = ?", deletedProjectID).Error)
	assert.False(t, restoredProject.DeletedAt.Valid)

	require.NoError(t, DeleteUser(deletedUserID))
	require.NoError(t, BatchPermanentlyDeleteUsers([]string{deletedUserID}))

	var deletedUser entities.User
	err = db.DB.Unscoped().First(&deletedUser, "id = ?", deletedUserID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var deletedProject entities.Project
	err = db.DB.Unscoped().First(&deletedProject, "id = ?", deletedProjectID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var activeUser entities.User
	require.NoError(t, db.DB.First(&activeUser, "id = ?", activeUserID).Error)

	var activeProject entities.Project
	require.NoError(t, db.DB.First(&activeProject, "id = ?", activeProjectID).Error)
}
