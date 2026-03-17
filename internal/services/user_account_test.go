package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupUserAccountServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(&entities.User{}))

	db.DB = testDB
}

func seedAccountTestUser(t *testing.T, userID, username, password, role string) *entities.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &entities.User{
		Base:     entities.Base{ID: userID},
		Username: username,
		Email:    username + "@example.com",
		Password: string(hashedPassword),
		Fullname: "Original Name",
		Bio:      "Original bio",
		Role:     role,
	}

	require.NoError(t, db.DB.Create(user).Error)

	return user
}

func TestGetCurrentUserReturnsBio(t *testing.T) {
	setupUserAccountServiceTestDB(t)
	seedAccountTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	user, err := GetCurrentUserProfile("user-1")
	require.NoError(t, err)

	assert.Equal(t, "Original Name", user.Fullname)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Original bio", user.Bio)
}

func TestUpdateCurrentUserProfilePersistsFullnameEmailAndBio(t *testing.T) {
	setupUserAccountServiceTestDB(t)
	seedAccountTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	user, err := UpdateCurrentUserProfile("user-1", "Alice Example", "alice+new@example.com", "Updated bio")
	require.NoError(t, err)

	assert.Equal(t, "Alice Example", user.Fullname)
	assert.Equal(t, "alice+new@example.com", user.Email)
	assert.Equal(t, "Updated bio", user.Bio)

	var persisted entities.User
	require.NoError(t, db.DB.First(&persisted, "id = ?", "user-1").Error)
	assert.Equal(t, "Alice Example", persisted.Fullname)
	assert.Equal(t, "alice+new@example.com", persisted.Email)
	assert.Equal(t, "Updated bio", persisted.Bio)
}

func TestChangeCurrentUserPasswordRejectsWrongCurrentPassword(t *testing.T) {
	setupUserAccountServiceTestDB(t)
	seedAccountTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	err := ChangeCurrentUserPassword("user-1", "wrong-password", "new-secret123")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCurrentPassword)
}

func TestChangeCurrentUserPasswordUpdatesHash(t *testing.T) {
	setupUserAccountServiceTestDB(t)
	seedAccountTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	err := ChangeCurrentUserPassword("user-1", "secret123", "new-secret123")
	require.NoError(t, err)

	var persisted entities.User
	require.NoError(t, db.DB.First(&persisted, "id = ?", "user-1").Error)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(persisted.Password), []byte("new-secret123")))
}

func TestAdminChangeUserPasswordUpdatesHash(t *testing.T) {
	setupUserAccountServiceTestDB(t)
	seedAccountTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	err := ChangeUserPassword("user-1", "admin-reset123")
	require.NoError(t, err)

	var persisted entities.User
	require.NoError(t, db.DB.First(&persisted, "id = ?", "user-1").Error)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(persisted.Password), []byte("admin-reset123")))
}
