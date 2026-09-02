package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		&entities.RecycleBinDeletionClaim{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Env{},
		&entities.App{},
		&entities.AppFavorite{},
		&entities.Notification{},
		&entities.OperationLog{},
	))

	db.DB = testDB
}

func setupFullUserRecycleServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	originalConfig := app.Config
	t.Cleanup(func() {
		db.DB = originalDB
		app.Config = originalConfig
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	app.Config.SecretEncryptionKey = "user-recycle-test-key-32-bytes-minimum"
	db.DB = testDB
	require.NoError(t, db.Migrate())
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

func TestPermanentlyDeleteUserCleansPersonalAndAuditReferences(t *testing.T) {
	setupUserRecycleServiceTestDB(t)

	userID := "user-record-cleanup"
	projectID := "project-record-cleanup"
	seedUserOwnedProject(t, userID, projectID)
	require.NoError(t, db.DB.Create(&entities.AppFavorite{
		ID: "favorite-record-cleanup", UserID: userID, EnvID: "env-1", AppID: "app-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Notification{
		Base: entities.Base{ID: "notification-recipient-cleanup"}, RecipientID: userID,
		Category: "info", EventType: "test", Title: "Recipient notification", Status: "pending",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Notification{
		Base: entities.Base{ID: "notification-sender-cleanup"}, RecipientID: "other-user", SenderID: userID,
		Category: "info", EventType: "test", Title: "Sender notification", Status: "pending",
	}).Error)
	logUserID := userID
	require.NoError(t, db.DB.Create(&entities.OperationLog{
		Base: entities.Base{ID: "operation-log-record-cleanup"}, UserID: &logUserID,
		Username: "deleted-user", Action: "test", ResourceType: "user", Status: "success", StatusCode: 200,
	}).Error)
	require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error)
	require.NoError(t, db.DB.Delete(&entities.User{}, "id = ?", userID).Error)

	require.NoError(t, PermanentlyDeleteUser(userID))

	for _, model := range []any{&entities.AppFavorite{}} {
		var count int64
		require.NoError(t, db.DB.Unscoped().Model(model).Count(&count).Error)
		assert.Zero(t, count)
	}
	var recipientCount int64
	require.NoError(t, db.DB.Unscoped().Model(&entities.Notification{}).
		Where("recipient_id = ?", userID).Count(&recipientCount).Error)
	assert.Zero(t, recipientCount)
	var senderNotification entities.Notification
	require.NoError(t, db.DB.Unscoped().First(&senderNotification, "id = ?", "notification-sender-cleanup").Error)
	assert.Empty(t, senderNotification.SenderID)
	var operationLog entities.OperationLog
	require.NoError(t, db.DB.Unscoped().First(&operationLog, "id = ?", "operation-log-record-cleanup").Error)
	assert.Nil(t, operationLog.UserID)
}

func TestPermanentlyDeleteUserCleansOwnedProjectNamespaces(t *testing.T) {
	setupFullUserRecycleServiceTestDB(t)

	deletedNamespace := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodDelete && request.URL.Path == "/api/v1/namespaces/work" {
			deletedNamespace = true
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, request)
	}))
	defer server.Close()

	userID := "user-namespace-cleanup"
	projectID := "project-namespace-cleanup"
	seedUserOwnedProject(t, userID, projectID)
	clusterID := registerAppVolumeTestCluster(t, server.URL)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-namespace-cleanup"},
		Slug:             "prod",
		Name:             "Production",
		ProjectID:        projectID,
		ClusterID:        clusterID,
		ClusterNamespace: "work",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base: entities.Base{ID: "app-namespace-cleanup"}, Slug: "active-app", Name: "Active App",
		EnvID: "env-namespace-cleanup", ContainerImage: "nginx:latest",
	}).Error)
	require.NoError(t, DeleteUser(userID))

	require.NoError(t, PermanentlyDeleteUser(userID))
	assert.True(t, deletedNamespace)
	var appCount int64
	require.NoError(t, db.DB.Unscoped().Model(&entities.App{}).
		Where("id = ?", "app-namespace-cleanup").Count(&appCount).Error)
	assert.Zero(t, appCount)
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
