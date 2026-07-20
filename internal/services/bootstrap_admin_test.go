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

func setupBootstrapAdminTestDB(t *testing.T) {
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
	require.NoError(t, testDB.AutoMigrate(&entities.User{}))

	db.DB = testDB
}

func TestEnsureBootstrapAdminRequiresPasswordWhenCreatingAdmin(t *testing.T) {
	setupBootstrapAdminTestDB(t)

	app.Config.BootstrapAdminUsername = ""
	app.Config.BootstrapAdminPassword = ""

	err := EnsureBootstrapAdmin()
	require.ErrorIs(t, err, ErrBootstrapAdminPasswordNotConfigured)

	var count int64
	require.NoError(t, db.DB.Model(&entities.User{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestEnsureBootstrapAdminCreatesConfiguredAdmin(t *testing.T) {
	setupBootstrapAdminTestDB(t)

	app.Config.BootstrapAdminUsername = "bootstrap-admin"
	app.Config.BootstrapAdminPassword = "bootstrap-password-123"

	require.NoError(t, EnsureBootstrapAdmin())

	var user entities.User
	require.NoError(t, db.DB.First(&user, "username = ?", "bootstrap-admin").Error)
	assert.Equal(t, app.UserRoleAdmin, user.Role)
	assert.Equal(t, "Bootstrap Admin", user.Fullname)
	assert.Equal(t, "bootstrap-admin@local.ketches", user.Email)
	assert.True(t, user.MustChangePassword)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("bootstrap-password-123")))
}

func TestEnsureBootstrapAdminRequiresPasswordWhenCustomUsernameProvided(t *testing.T) {
	setupBootstrapAdminTestDB(t)

	app.Config.BootstrapAdminUsername = "root-admin"
	app.Config.BootstrapAdminPassword = ""

	require.ErrorIs(t, EnsureBootstrapAdmin(), ErrBootstrapAdminPasswordNotConfigured)
}

func TestEnsureBootstrapAdminDoesNotCreateDuplicateAdmin(t *testing.T) {
	setupBootstrapAdminTestDB(t)

	app.Config.BootstrapAdminUsername = "bootstrap-admin"
	app.Config.BootstrapAdminPassword = ""

	require.NoError(t, db.DB.Create(&entities.User{
		Base:     entities.Base{ID: "admin-1"},
		Username: "existing-admin",
		Email:    "existing-admin@example.com",
		Password: "hashed-password",
		Role:     app.UserRoleAdmin,
	}).Error)

	require.NoError(t, EnsureBootstrapAdmin())

	var count int64
	require.NoError(t, db.DB.Model(&entities.User{}).Where("role = ?", app.UserRoleAdmin).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
