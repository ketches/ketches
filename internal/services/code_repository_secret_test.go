package services

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCodeRepositoryServiceTestDB(t *testing.T) {
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
	require.NoError(t, testDB.AutoMigrate(&entities.Project{}, &entities.CodeRepository{}))

	db.DB = testDB
	app.Config.SecretEncryptionKey = "test-master-key"
}

func seedCodeRepositoryProject(t *testing.T) {
	t.Helper()
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo",
		Name: "Demo",
	}).Error)
}

func TestToCodeRepositoryResponse_OmitsGitPassword(t *testing.T) {
	resp := ToCodeRepositoryResponse(&entities.CodeRepository{
		Base:        entities.Base{ID: "repo-1"},
		ProjectID:   "project-1",
		Name:        "demo-repo",
		Slug:        "demo-repo",
		GitRepoURL:  "https://example.com/demo.git",
		GitUsername: "demo",
		GitPassword: "enc:v1:opaque",
	})

	assert.Empty(t, resp.GitPassword)
	assert.True(t, resp.HasGitPassword)
}

func TestUpdateCodeRepositoryClearsGitPassword(t *testing.T) {
	setupCodeRepositoryServiceTestDB(t)
	seedCodeRepositoryProject(t)

	encryptedPassword, err := secrets.EncryptString("old-secret")
	require.NoError(t, err)
	require.NoError(t, db.DB.Create(&entities.CodeRepository{
		Base:        entities.Base{ID: "repo-2"},
		ProjectID:   "project-1",
		Name:        "demo-repo-2",
		Slug:        "demo-repo-2",
		GitRepoURL:  "https://example.com/demo.git",
		GitUsername: "demo",
		GitPassword: encryptedPassword,
	}).Error)

	clearPassword := true
	_, err = UpdateCodeRepository("repo-2", &models.UpdateCodeRepositoryRequest{
		ClearGitPassword: &clearPassword,
	})
	require.NoError(t, err)

	var stored entities.CodeRepository
	require.NoError(t, db.DB.First(&stored, "id = ?", "repo-2").Error)
	assert.Empty(t, stored.GitPassword)
}

func TestCreateCodeRepository_StoresEncryptedGitPassword(t *testing.T) {
	setupCodeRepositoryServiceTestDB(t)
	seedCodeRepositoryProject(t)

	repo, err := CreateCodeRepository("project-1", &models.CreateCodeRepositoryRequest{
		Name:        "demo-repo",
		Slug:        "demo-repo",
		GitRepoURL:  "https://example.com/demo.git",
		GitUsername: "demo",
		GitPassword: "plain-secret",
	})
	require.NoError(t, err)

	var stored entities.CodeRepository
	require.NoError(t, db.DB.First(&stored, "id = ?", repo.ID).Error)
	assert.NotEqual(t, "plain-secret", stored.GitPassword)
	assert.True(t, strings.HasPrefix(stored.GitPassword, "enc:v1:"))
}

func TestUpdateCodeRepository_StoresEncryptedGitPassword(t *testing.T) {
	setupCodeRepositoryServiceTestDB(t)
	seedCodeRepositoryProject(t)

	require.NoError(t, db.DB.Create(&entities.CodeRepository{
		Base:        entities.Base{ID: "repo-1"},
		ProjectID:   "project-1",
		Name:        "demo-repo",
		Slug:        "demo-repo",
		GitRepoURL:  "https://example.com/demo.git",
		GitUsername: "demo",
		GitPassword: "old-secret",
	}).Error)

	_, err := UpdateCodeRepository("repo-1", &models.UpdateCodeRepositoryRequest{
		GitPassword: "next-secret",
	})
	require.NoError(t, err)

	var stored entities.CodeRepository
	require.NoError(t, db.DB.First(&stored, "id = ?", "repo-1").Error)
	assert.NotEqual(t, "next-secret", stored.GitPassword)
	assert.True(t, strings.HasPrefix(stored.GitPassword, "enc:v1:"))
}

func TestGetCodeRepositoryMigratesLegacyPlaintextGitPassword(t *testing.T) {
	setupCodeRepositoryServiceTestDB(t)
	seedCodeRepositoryProject(t)

	require.NoError(t, db.DB.Create(&entities.CodeRepository{
		Base:        entities.Base{ID: "repo-legacy"},
		ProjectID:   "project-1",
		Name:        "legacy-repo",
		Slug:        "legacy-repo",
		GitRepoURL:  "https://example.com/demo.git",
		GitUsername: "demo",
		GitPassword: "legacy-plaintext-password",
	}).Error)

	repo, err := GetCodeRepository("repo-legacy")

	require.NoError(t, err)
	require.NotNil(t, repo)
	assert.True(t, strings.HasPrefix(repo.GitPassword, "enc:v1:"))

	var stored entities.CodeRepository
	require.NoError(t, db.DB.First(&stored, "id = ?", "repo-legacy").Error)
	assert.True(t, strings.HasPrefix(stored.GitPassword, "enc:v1:"))

	decrypted, err := secrets.DecryptString(stored.GitPassword)
	require.NoError(t, err)
	assert.Equal(t, "legacy-plaintext-password", decrypted)
}
