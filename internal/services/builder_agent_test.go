package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
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
		assert.Equal(t, builderAgentSystemPrompt, payload.Messages[0].Content)
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
		{Role: "user", Content: "Create the initial files."},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Created initial files.", result.AssistantMessage)
	require.Len(t, result.Files, 1)
	assert.Equal(t, "cmd/api/main.go", result.Files[0].Path)
	assert.Equal(t, "package main\n", result.Files[0].Content)
}

func TestGenerateBuilderFiles_DoesNotDuplicateExistingSystemPrompt(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload struct {
			Messages []BuilderAgentMessage `json:"messages"`
		}
		require.NoError(t, json.Unmarshal(body, &payload))
		require.Len(t, payload.Messages, 2)
		assert.Equal(t, "system", payload.Messages[0].Role)
		assert.Equal(t, "Existing system prompt", payload.Messages[0].Content)
		assert.Equal(t, "user", payload.Messages[1].Role)
		assert.Equal(t, "Create the initial files.", payload.Messages[1].Content)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"content": `{"assistant_message":"Created initial files.","files":[{"path":"cmd/api/main.go","content":"package main\n"}]}`,
				},
			}},
		}))
	}))
	defer server.Close()

	app.Config = newBuilderAgentRegistryTestConfig(t, server.URL)

	result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{
		{Role: "system", Content: "Existing system prompt"},
		{Role: "user", Content: "Create the initial files."},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
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

func TestGenerateBuilderFilesWithSelection_UsesExplicitRunLevelProviderAndModel(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer selected-api-key", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload struct {
			Model string `json:"model"`
		}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "selected-builder-model", payload.Model)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"content": `{"assistant_message":"Created initial files.","files":[{"path":"cmd/api/main.go","content":"package main\n"}]}`,
				},
			}},
		}))
	}))
	defer server.Close()

	app.Config = app.AppConfig{
		BuilderProviderRegistryJSON: `[
			{"key":"default-provider","base_url":"https://defaults.example.com","api_key":"default-key"},
			{"key":"selected-provider","base_url":` + marshalJSONString(t, server.URL) + `,"api_key":"selected-api-key"}
		]`,
		BuilderModelProfileRegistryJSON: `[
			{"key":"builder-default","model":"default-model"},
			{"key":"selected-model","model":"selected-builder-model"}
		]`,
		BuilderDefaultProviderKey:     "default-provider",
		BuilderDefaultModelProfileKey: "builder-default",
	}

	result, err := GenerateBuilderFilesWithSelection(context.Background(), []BuilderAgentMessage{{Role: "user", Content: "Create the initial files."}}, "selected-provider", "selected-model")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Created initial files.", result.AssistantMessage)
}

func TestGenerateBuilderFiles_AllowsReplyOnlyResultsWithoutFiles(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"content": `{"action":"reply_only","assistant_message":"Hi! What would you like to build or change?","files":[]}`,
				},
			}},
		}))
	}))
	defer server.Close()

	app.Config = newBuilderAgentRegistryTestConfig(t, server.URL)

	result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{
		{Role: "user", Content: "hi"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, BuilderAgentActionReplyOnly, result.Action)
	assert.Equal(t, "Hi! What would you like to build or change?", result.AssistantMessage)
	assert.Empty(t, result.Files)
}

func TestGenerateBuilderFiles_NormalizesProviderBaseURLVariantsToChatCompletionsEndpoint(t *testing.T) {
	testCases := []struct {
		name             string
		baseURLPath      string
		expectedHTTPPath string
	}{
		{
			name:             "root base url",
			baseURLPath:      "",
			expectedHTTPPath: "/v1/chat/completions",
		},
		{
			name:             "v1 base url",
			baseURLPath:      "/v1",
			expectedHTTPPath: "/v1/chat/completions",
		},
		{
			name:             "full chat completions url",
			baseURLPath:      "/v1/chat/completions",
			expectedHTTPPath: "/v1/chat/completions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalConfig := app.Config
			t.Cleanup(func() {
				app.Config = originalConfig
			})

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, tc.expectedHTTPPath, r.URL.Path)
				require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"choices": []map[string]any{{
						"message": map[string]any{
							"content": `{"assistant_message":"Created initial files.","files":[{"path":"cmd/api/main.go","content":"package main\n"}]}`,
						},
					}},
				}))
			}))
			defer server.Close()

			app.Config = newBuilderAgentRegistryTestConfig(t, server.URL+tc.baseURLPath)

			result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{
				{Role: "user", Content: "Create the initial files."},
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "Created initial files.", result.AssistantMessage)
		})
	}
}

func TestGenerateBuilderFiles_UsesAnthropicNativeMessagesAPI(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/messages", r.URL.Path)
		require.Equal(t, "anthropic-api-key", r.Header.Get("x-api-key"))
		require.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Empty(t, r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
			System    string `json:"system"`
			Messages  []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "claude-sonnet-4-20250514", payload.Model)
		assert.Equal(t, builderAnthropicDefaultMaxTokens, payload.MaxTokens)
		assert.Contains(t, payload.System, builderAgentSystemPrompt)
		require.Len(t, payload.Messages, 2)
		assert.Equal(t, "user", payload.Messages[0].Role)
		assert.Equal(t, "Create the initial files.", payload.Messages[0].Content)
		assert.Equal(t, "assistant", payload.Messages[1].Role)
		assert.Equal(t, "Acknowledged.", payload.Messages[1].Content)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{
					"type": "text",
					"text": `{"assistant_message":"Created initial files.","files":[{"path":"cmd/api/main.go","content":"package main\n"}]}`,
				},
			},
			"stop_reason": "end_turn",
		}))
	}))
	defer server.Close()

	app.Config = app.AppConfig{
		BuilderProviderRegistryJSON:     `[{"key":"anthropic-native","base_url":"` + server.URL + `/v1/messages","api_key":"anthropic-api-key"}]`,
		BuilderModelProfileRegistryJSON: `[{"key":"claude-sonnet-4","model":"claude-sonnet-4-20250514"}]`,
		BuilderDefaultProviderKey:       "anthropic-native",
		BuilderDefaultModelProfileKey:   "claude-sonnet-4",
	}

	result, err := GenerateBuilderFiles(context.Background(), []BuilderAgentMessage{
		{Role: "user", Content: "Create the initial files."},
		{Role: "assistant", Content: "Acknowledged."},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Created initial files.", result.AssistantMessage)
}

func TestResolveBuilderAgentProtocol_PrefersAnthropicNativeForOfficialBaseURL(t *testing.T) {
	protocol, err := resolveBuilderAgentProtocol("https://api.anthropic.com")
	require.NoError(t, err)
	assert.Equal(t, builderAgentProtocolAnthropicNative, protocol)

	protocol, err = resolveBuilderAgentProtocol("https://api.anthropic.com/v1")
	require.NoError(t, err)
	assert.Equal(t, builderAgentProtocolAnthropicNative, protocol)

	protocol, err = resolveBuilderAgentProtocol("https://api.anthropic.com/v1/chat/completions")
	require.NoError(t, err)
	assert.Equal(t, builderAgentProtocolOpenAICompatible, protocol)
}

func TestGenerateBuilderFilesForRun_UsesProjectScopedDatabaseProviderSelection(t *testing.T) {
	setupAIProviderServiceTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer project-api-key", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload struct {
			Model string `json:"model"`
		}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "claude-4-sonnet", payload.Model)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"content": `{"assistant_message":"Created initial files.","files":[{"path":"cmd/api/main.go","content":"package main\n"}]}`,
				},
			}},
		}))
	}))
	defer server.Close()

	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base:       entities.Base{ID: "session-1"},
		ProjectID:  "project-1",
		BuildEnvID: "env-1",
		CreatedBy:  "user-1",
		Status:     entities.BuilderSessionStatusProvisioning,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
		ID:                     "project-provider-1",
		ProjectID:              "project-1",
		ProviderKey:            "shared-provider",
		DisplayName:            "Shared Provider",
		BaseURL:                server.URL,
		APIKey:                 "project-api-key",
		DefaultModelProfileKey: "claude-4-sonnet",
		Enabled:                true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.UserAIProvider{
		ID:                     "user-provider-1",
		UserID:                 "user-1",
		ProviderKey:            "shared-provider",
		DisplayName:            "Personal Provider",
		BaseURL:                "https://user.example.com",
		APIKey:                 "user-api-key",
		DefaultModelProfileKey: "claude-4-sonnet",
		Enabled:                true,
	}).Error)

	run := &entities.BuilderRun{
		ID:              "run-1",
		SessionID:       "session-1",
		RequestedBy:     "user-1",
		ProviderScope:   stringPtr("project"),
		ProviderKey:     stringPtr("shared-provider"),
		ModelProfileKey: stringPtr("claude-4-sonnet"),
	}

	result, err := GenerateBuilderFilesForRun(context.Background(), run, []BuilderAgentMessage{
		{Role: "user", Content: "Create the initial files."},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Created initial files.", result.AssistantMessage)
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
