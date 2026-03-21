package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBuilderFiles_SendsConfiguredOpenAICompatibleRequestAndParsesResponse(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload struct {
			Model          string                `json:"model"`
			Messages       []BuilderAgentMessage `json:"messages"`
			ResponseFormat struct {
				Type       string `json:"type"`
				JSONSchema struct {
					Name   string `json:"name"`
					Strict bool   `json:"strict"`
					Schema struct {
						Type       string         `json:"type"`
						Required   []string       `json:"required"`
						Properties map[string]any `json:"properties"`
					} `json:"schema"`
				} `json:"json_schema"`
			} `json:"response_format"`
		}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "builder-model", payload.Model)
		require.Len(t, payload.Messages, 2)
		assert.Equal(t, "system", payload.Messages[0].Role)
		assert.Equal(t, "You are a builder.", payload.Messages[0].Content)
		assert.Equal(t, "user", payload.Messages[1].Role)
		assert.Equal(t, "Create the initial files.", payload.Messages[1].Content)
		assert.Equal(t, "json_schema", payload.ResponseFormat.Type)
		assert.Equal(t, "builder_agent_result", payload.ResponseFormat.JSONSchema.Name)
		assert.True(t, payload.ResponseFormat.JSONSchema.Strict)
		assert.Equal(t, "object", payload.ResponseFormat.JSONSchema.Schema.Type)
		assert.Contains(t, payload.ResponseFormat.JSONSchema.Schema.Required, "assistant_message")
		assert.Contains(t, payload.ResponseFormat.JSONSchema.Schema.Required, "files")
		assert.Contains(t, payload.ResponseFormat.JSONSchema.Schema.Properties, "assistant_message")
		assert.Contains(t, payload.ResponseFormat.JSONSchema.Schema.Properties, "files")

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": `{"assistant_message":"Created initial files.","files":[{"path":"cmd/api/main.go","content":"package main\n"}]}`,
					},
				},
			},
		}))
	}))
	defer server.Close()

	app.Config = newBuilderAgentRegistryTestConfig(t, server.URL)

	result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{
		{Role: "system", Content: "You are a builder."},
		{Role: "user", Content: "Create the initial files."},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Created initial files.", result.AssistantMessage)
	require.Len(t, result.Files, 1)
	assert.Equal(t, "cmd/api/main.go", result.Files[0].Path)
	assert.Equal(t, "package main\n", result.Files[0].Content)
}

func TestGenerateBuilderFiles_SkipsAuthorizationHeaderWhenRegistryAPIKeyIsBlank(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Empty(t, r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": `{"assistant_message":"Created initial files.","files":[{"path":"cmd/api/main.go","content":"package main\n"}]}`,
					},
				},
			},
		}))
	}))
	defer server.Close()

	app.Config = newBuilderAgentRegistryTestConfigWithAPIKey(t, server.URL, "")

	result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{
		{Role: "user", Content: "Create the initial files."},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGenerateBuilderFiles_RejectsUnsafeFilePaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "path traversal", path: "../escape.sh"},
		{name: "absolute path", path: "/tmp/escape.sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newBuilderAgentTestServer(t, tt.path, "file contents")
			defer server.Close()

			originalConfig := app.Config
			t.Cleanup(func() {
				app.Config = originalConfig
			})

			app.Config = newBuilderAgentRegistryTestConfig(t, server.URL)

			result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{{Role: "user", Content: "Write a file."}})
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unsafe file path")
		})
	}
}

func TestGenerateBuilderFiles_RejectsBlankFileWrites(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		wantErr string
	}{
		{name: "blank path", path: "   ", content: "package main", wantErr: "file path is required"},
		{name: "empty content", path: "cmd/api/main.go", content: "", wantErr: "file content cannot be blank"},
		{name: "blank content", path: "cmd/api/main.go", content: "   ", wantErr: "file content cannot be blank"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newBuilderAgentTestServer(t, tt.path, tt.content)
			defer server.Close()

			originalConfig := app.Config
			t.Cleanup(func() {
				app.Config = originalConfig
			})

			app.Config = newBuilderAgentRegistryTestConfig(t, server.URL)

			result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{{Role: "user", Content: "Write a file."}})
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func newBuilderAgentTestServer(t *testing.T, filePath, content string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": `{"assistant_message":"Generated files.","files":[{"path":` + marshalJSONString(t, filePath) + `,"content":` + marshalJSONString(t, content) + `}]}`,
					},
				},
			},
		}))
	}))
}

func marshalJSONString(t *testing.T, value string) string {
	t.Helper()

	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

func newBuilderAgentRegistryTestConfig(t *testing.T, baseURL string) app.AppConfig {
	return newBuilderAgentRegistryTestConfigWithAPIKey(t, baseURL, "test-api-key")
}

func newBuilderAgentRegistryTestConfigWithAPIKey(t *testing.T, baseURL, apiKey string) app.AppConfig {
	t.Helper()

	return app.AppConfig{
		BuilderProviderRegistryJSON:     `[{"key":"openai-compatible-primary","base_url":` + marshalJSONString(t, baseURL) + `,"api_key":` + marshalJSONString(t, apiKey) + `}]`,
		BuilderModelProfileRegistryJSON: `[{"key":"builder-default","model":"builder-model"}]`,
		BuilderDefaultProviderKey:       "openai-compatible-primary",
		BuilderDefaultModelProfileKey:   "builder-default",
	}
}
