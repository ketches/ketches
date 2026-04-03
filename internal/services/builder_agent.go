package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
)

var ErrBuilderAgentUnsafeFilePath = errors.New("unsafe file path")

const maxBuilderRelativePathLength = 255

const builderAgentSystemPrompt = "You are a builder. First decide whether the user's latest turn requires file changes. If the user is only greeting, chatting, asking for clarification, or not yet requesting a concrete project change, return action=reply_only with an assistant_message and an empty files array. If the user is asking to create or modify project files, return action=generate_files with the smallest correct set of file writes. Return only content that satisfies the required JSON schema. Never omit required fields, never return prose outside the schema, and never generate unsafe file paths."

const builderAnthropicDefaultMaxTokens = 4096

type builderAgentProtocol string

const (
	builderAgentProtocolOpenAICompatible builderAgentProtocol = "openai_compatible"
	builderAgentProtocolAnthropicNative  builderAgentProtocol = "anthropic_native"
)

type BuilderAgentMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BuilderAgentFileWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type BuilderAgentAction string

const (
	BuilderAgentActionGenerateFiles BuilderAgentAction = "generate_files"
	BuilderAgentActionReplyOnly     BuilderAgentAction = "reply_only"
)

type BuilderAgentResult struct {
	Action           BuilderAgentAction      `json:"action"`
	AssistantMessage string                  `json:"assistant_message"`
	Files            []BuilderAgentFileWrite `json:"files"`
}

type builderAgentChatCompletionRequest struct {
	Model          string                     `json:"model"`
	Messages       []BuilderAgentMessage      `json:"messages"`
	ResponseFormat builderAgentResponseFormat `json:"response_format"`
}

type builderAgentResponseFormat struct {
	Type       string                 `json:"type"`
	JSONSchema builderAgentJSONSchema `json:"json_schema"`
}

type builderAgentJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type builderAgentChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type builderAnthropicMessageRequest struct {
	Model     string                         `json:"model"`
	MaxTokens int                            `json:"max_tokens"`
	System    string                         `json:"system,omitempty"`
	Messages  []builderAnthropicMessageInput `json:"messages"`
}

type builderAnthropicMessageInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type builderAnthropicMessageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func GenerateBuilderFiles(ctx context.Context, messages []BuilderAgentMessage) (*BuilderAgentResult, error) {
	return GenerateBuilderFilesWithSelection(ctx, messages, "", "")
}

func GenerateBuilderFilesWithSelection(ctx context.Context, messages []BuilderAgentMessage, providerKey, modelProfileKey string) (*BuilderAgentResult, error) {
	resolvedRequest, err := resolveBuilderAgentRequest(ctx, providerKey, modelProfileKey)
	if err != nil {
		return nil, err
	}

	protocol, err := resolveBuilderAgentProtocol(resolvedRequest.BaseURL)
	if err != nil {
		return nil, err
	}

	req, err := newBuilderAgentRequest(ctx, protocol, resolvedRequest, messages)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, readErr
		}
		return nil, app.NewErrorf("builder agent request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	result, err := decodeBuilderAgentResponse(resp.Body, protocol)
	if err != nil {
		return nil, err
	}
	if err := validateBuilderAgentResult(result); err != nil {
		return nil, err
	}

	return result, nil
}

func GenerateBuilderFilesForRun(ctx context.Context, run *entities.BuilderRun, messages []BuilderAgentMessage) (*BuilderAgentResult, error) {
	if run == nil {
		return nil, errors.New("builder run is required")
	}

	projectID, err := loadBuilderSessionProjectID(ctx, run.SessionID)
	if err != nil {
		return nil, err
	}

	return GenerateBuilderFilesWithSelection(
		withBuilderRunGenerationContext(ctx, projectID, run),
		messages,
		stringPointerValue(run.ProviderKey),
		stringPointerValue(run.ModelProfileKey),
	)
}

type builderAgentResolvedRequest struct {
	BaseURL string
	APIKey  string
	Model   string
}

func resolveBuilderAgentRequest(ctx context.Context, providerKey, modelProfileKey string) (builderAgentResolvedRequest, error) {
	if generationSelection, ok := builderRunGenerationSelectionFromContext(ctx); ok {
		resolvedProvider, err := resolveBuilderAIProviderForExecution(
			ctx,
			generationSelection.ProjectID,
			generationSelection.UserID,
			generationSelection.ProviderScope,
			providerKey,
			modelProfileKey,
		)
		if err != nil {
			return builderAgentResolvedRequest{}, err
		}

		return builderAgentResolvedRequest{
			BaseURL: resolvedProvider.BaseURL,
			APIKey:  resolvedProvider.APIKey,
			Model:   resolvedProvider.ModelProfileKey,
		}, nil
	}

	registry, err := loadBuilderProviderRegistry(app.Config)
	if err != nil {
		return builderAgentResolvedRequest{}, err
	}

	resolvedProviderProfile, err := registry.resolveBuilderProviderProfile(providerKey, modelProfileKey)
	if err != nil {
		return builderAgentResolvedRequest{}, err
	}

	return builderAgentResolvedRequest{
		BaseURL: resolvedProviderProfile.Provider.BaseURL,
		APIKey:  resolvedProviderProfile.Provider.APIKey,
		Model:   resolvedProviderProfile.ModelProfile.Model,
	}, nil
}

func resolveBuilderAgentProtocol(baseURL string) (builderAgentProtocol, error) {
	resolvedBaseURL := strings.TrimSpace(baseURL)
	if resolvedBaseURL == "" {
		return "", errors.New("builder provider base URL is required")
	}

	parsedURL, err := url.Parse(resolvedBaseURL)
	if err != nil {
		return "", app.WrapErrorf(err, "parse builder provider base URL: %w", err)
	}

	normalizedPath := strings.ToLower(strings.TrimRight(parsedURL.Path, "/"))
	if strings.HasSuffix(normalizedPath, "/v1/chat/completions") || strings.HasSuffix(normalizedPath, "/chat/completions") {
		return builderAgentProtocolOpenAICompatible, nil
	}

	host := strings.ToLower(parsedURL.Hostname())
	if host == "api.anthropic.com" || strings.HasSuffix(host, ".anthropic.com") || strings.HasSuffix(normalizedPath, "/v1/messages") || strings.HasSuffix(normalizedPath, "/messages") {
		return builderAgentProtocolAnthropicNative, nil
	}

	return builderAgentProtocolOpenAICompatible, nil
}

func resolveBuilderAgentChatCompletionsEndpoint(baseURL string) (string, error) {
	resolvedBaseURL := strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if resolvedBaseURL == "" {
		return "", errors.New("builder provider base URL is required")
	}

	lowerBaseURL := strings.ToLower(resolvedBaseURL)
	switch {
	case strings.HasSuffix(lowerBaseURL, "/v1/chat/completions"):
		return resolvedBaseURL, nil
	case strings.HasSuffix(lowerBaseURL, "/chat/completions"):
		return resolvedBaseURL, nil
	case strings.HasSuffix(lowerBaseURL, "/v1"):
		return resolvedBaseURL + "/chat/completions", nil
	default:
		return resolvedBaseURL + "/v1/chat/completions", nil
	}
}

func resolveBuilderAnthropicMessagesEndpoint(baseURL string) (string, error) {
	resolvedBaseURL := strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if resolvedBaseURL == "" {
		return "", errors.New("builder provider base URL is required")
	}

	lowerBaseURL := strings.ToLower(resolvedBaseURL)
	switch {
	case strings.HasSuffix(lowerBaseURL, "/v1/messages"):
		return resolvedBaseURL, nil
	case strings.HasSuffix(lowerBaseURL, "/messages"):
		return resolvedBaseURL, nil
	case strings.HasSuffix(lowerBaseURL, "/v1"):
		return resolvedBaseURL + "/messages", nil
	default:
		return resolvedBaseURL + "/v1/messages", nil
	}
}

func newBuilderAgentRequest(ctx context.Context, protocol builderAgentProtocol, resolvedRequest builderAgentResolvedRequest, messages []BuilderAgentMessage) (*http.Request, error) {
	var (
		endpoint    string
		requestBody []byte
		err         error
	)

	switch protocol {
	case builderAgentProtocolAnthropicNative:
		endpoint, err = resolveBuilderAnthropicMessagesEndpoint(resolvedRequest.BaseURL)
		if err != nil {
			return nil, err
		}
		requestBody, err = json.Marshal(buildBuilderAnthropicMessageRequest(resolvedRequest.Model, messages))
		if err != nil {
			return nil, err
		}
	case builderAgentProtocolOpenAICompatible:
		endpoint, err = resolveBuilderAgentChatCompletionsEndpoint(resolvedRequest.BaseURL)
		if err != nil {
			return nil, err
		}
		requestBody, err = json.Marshal(builderAgentChatCompletionRequest{
			Model:    resolvedRequest.Model,
			Messages: prepareBuilderAgentMessages(messages),
			ResponseFormat: builderAgentResponseFormat{
				Type: "json_schema",
				JSONSchema: builderAgentJSONSchema{
					Name:   "builder_agent_result",
					Strict: true,
					Schema: builderAgentResultSchema(),
				},
			},
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, app.NewErrorf("unsupported builder agent protocol %q", protocol)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	switch protocol {
	case builderAgentProtocolAnthropicNative:
		if strings.TrimSpace(resolvedRequest.APIKey) != "" {
			req.Header.Set("x-api-key", resolvedRequest.APIKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		if strings.TrimSpace(resolvedRequest.APIKey) != "" {
			req.Header.Set("Authorization", "Bearer "+resolvedRequest.APIKey)
		}
	}

	return req, nil
}

func buildBuilderAnthropicMessageRequest(model string, messages []BuilderAgentMessage) builderAnthropicMessageRequest {
	preparedMessages := prepareBuilderAgentMessages(messages)
	systemParts := make([]string, 0, 1)
	anthropicMessages := make([]builderAnthropicMessageInput, 0, len(preparedMessages))

	for _, message := range preparedMessages {
		switch message.Role {
		case "system":
			if strings.TrimSpace(message.Content) != "" {
				systemParts = append(systemParts, message.Content)
			}
		case "user", "assistant":
			anthropicMessages = append(anthropicMessages, builderAnthropicMessageInput{
				Role:    message.Role,
				Content: message.Content,
			})
		}
	}

	return builderAnthropicMessageRequest{
		Model:     model,
		MaxTokens: builderAnthropicDefaultMaxTokens,
		System:    strings.Join(systemParts, "\n\n"),
		Messages:  anthropicMessages,
	}
}

func decodeBuilderAgentResponse(body io.Reader, protocol builderAgentProtocol) (*BuilderAgentResult, error) {
	if body == nil {
		return nil, errors.New("builder agent response body is required")
	}

	switch protocol {
	case builderAgentProtocolAnthropicNative:
		var response builderAnthropicMessageResponse
		if err := json.NewDecoder(body).Decode(&response); err != nil {
			return nil, err
		}
		content := extractAnthropicTextResponse(response)
		if content == "" {
			return nil, errors.New("builder agent response missing text content")
		}
		var result BuilderAgentResult
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return nil, err
		}
		return &result, nil
	default:
		var completion builderAgentChatCompletionResponse
		if err := json.NewDecoder(body).Decode(&completion); err != nil {
			return nil, err
		}
		if len(completion.Choices) == 0 {
			return nil, errors.New("builder agent response missing choices")
		}
		var result BuilderAgentResult
		if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
}

func extractAnthropicTextResponse(response builderAnthropicMessageResponse) string {
	var builder strings.Builder
	for _, block := range response.Content {
		if block.Type != "text" || strings.TrimSpace(block.Text) == "" {
			continue
		}
		builder.WriteString(block.Text)
	}

	return builder.String()
}

func prepareBuilderAgentMessages(messages []BuilderAgentMessage) []BuilderAgentMessage {
	prepared := make([]BuilderAgentMessage, 0, len(messages)+1)
	if len(messages) == 0 || messages[0].Role != "system" {
		prepared = append(prepared, BuilderAgentMessage{Role: "system", Content: builderAgentSystemPrompt})
	}
	prepared = append(prepared, messages...)
	return prepared
}

func builderAgentResultSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"assistant_message", "files"},
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{string(BuilderAgentActionGenerateFiles), string(BuilderAgentActionReplyOnly)},
			},
			"assistant_message": map[string]any{
				"type": "string",
			},
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"path", "content"},
					"properties": map[string]any{
						"path": map[string]any{
							"type": "string",
						},
						"content": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}
}

func validateBuilderAgentResult(result *BuilderAgentResult) error {
	if result == nil {
		return errors.New("builder agent result is required")
	}
	if result.Action == "" {
		if len(result.Files) == 0 {
			result.Action = BuilderAgentActionReplyOnly
		} else {
			result.Action = BuilderAgentActionGenerateFiles
		}
	}

	switch result.Action {
	case BuilderAgentActionReplyOnly:
		if strings.TrimSpace(result.AssistantMessage) == "" {
			return errors.New("assistant message cannot be blank")
		}
		if len(result.Files) > 0 {
			return errors.New("reply_only builder agent result must not include files")
		}
		return nil
	case BuilderAgentActionGenerateFiles:
		if len(result.Files) == 0 {
			return errors.New("generate_files builder agent result must include at least one file")
		}
	default:
		return app.NewErrorf("unsupported builder agent action %q", result.Action)
	}

	for i := range result.Files {
		validated, err := validateBuilderAgentFileWrite(result.Files[i])
		if err != nil {
			return err
		}
		result.Files[i] = validated
	}

	return nil
}

func validateBuilderAgentFileWrite(file BuilderAgentFileWrite) (BuilderAgentFileWrite, error) {
	validatedPath, err := validateBuilderAgentFilePath(file.Path)
	if err != nil {
		return BuilderAgentFileWrite{}, err
	}
	if strings.TrimSpace(file.Content) == "" {
		return BuilderAgentFileWrite{}, errors.New("file content cannot be blank")
	}

	file.Path = validatedPath
	return file, nil
}

func validateBuilderAgentFilePath(filePath string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(filePath, "\\", "/"))
	if normalized == "" {
		return "", errors.New("file path is required")
	}
	if strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "~") {
		return "", app.WrapErrorf(ErrBuilderAgentUnsafeFilePath, "%w: %s", ErrBuilderAgentUnsafeFilePath, filePath)
	}

	segments := strings.Split(normalized, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", app.WrapErrorf(ErrBuilderAgentUnsafeFilePath, "%w: %s", ErrBuilderAgentUnsafeFilePath, filePath)
		}
	}
	if len(normalized) > maxBuilderRelativePathLength {
		return "", app.WrapErrorf(ErrBuilderAgentUnsafeFilePath, "%w: %s", ErrBuilderAgentUnsafeFilePath, filePath)
	}

	return path.Clean(normalized), nil
}
