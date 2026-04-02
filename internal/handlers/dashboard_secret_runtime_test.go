package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/require"
)

func TestExecutePrometheusRequestDecryptsBearerToken(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedToken, err := secrets.EncryptString("real-token")
	require.NoError(t, err)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer real-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"bad token"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	_, err = executePrometheusRequest(server.URL, &entities.ClusterIntegration{
		Endpoint:      server.URL,
		Token:         encryptedToken,
		SkipTLSVerify: true,
		ServiceName:   "",
		Username:      "",
		Password:      "",
	})
	require.NoError(t, err)
}

func TestExecutePrometheusRequestDecryptsBasicAuthPassword(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedPassword, err := secrets.EncryptString("real-password")
	require.NoError(t, err)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "demo" || password != "real-password" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"bad auth"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	_, err = executePrometheusRequest(server.URL, &entities.ClusterIntegration{
		Endpoint:      server.URL,
		Username:      "demo",
		Password:      encryptedPassword,
		SkipTLSVerify: true,
		ServiceName:   "",
	})
	require.NoError(t, err)
}
