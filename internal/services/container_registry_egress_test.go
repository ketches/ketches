package services

import (
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/require"
)

func TestResolveRegistryHostRejectsInsecureAndPrivateEndpoints(t *testing.T) {
	for _, endpoint := range []string{
		"http://registry.example.com",
		"https://127.0.0.1:5000",
		"https://169.254.169.254",
		"https://metadata.google.internal",
	} {
		_, _, err := resolveRegistryHost(string(entities.RegistryProviderCustom), endpoint)
		require.Error(t, err, endpoint)
	}
}
