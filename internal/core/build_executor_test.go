package core

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func TestBuildDockerConfigJSONDecryptsEncryptedPassword(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedPassword, err := secrets.EncryptString("super-secret")
	require.NoError(t, err)

	data, err := buildDockerConfigJSON(&entities.ContainerRegistry{
		Provider: entities.RegistryProviderGHCR,
		Endpoint: "ghcr.io",
		Username: "demo",
		Password: encryptedPassword,
	})

	require.NoError(t, err)

	var decoded map[string]map[string]map[string]string
	require.NoError(t, json.Unmarshal(data, &decoded))
	auth := decoded["auths"]["ghcr.io"]["auth"]
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("demo:super-secret")), auth)
}

func TestResolveCodeRepositoryGitPasswordDecryptsEncryptedPassword(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedPassword, err := secrets.EncryptString("git-secret")
	require.NoError(t, err)

	password, err := resolveCodeRepositoryGitPassword(&entities.CodeRepository{
		GitUsername: "demo",
		GitPassword: encryptedPassword,
	})

	require.NoError(t, err)
	assert.Equal(t, "git-secret", password)
}
