package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/ketches/ketches/internal/app"
)

var ErrBuilderAgentUnsafeFilePath = errors.New("unsafe file path")

type BuilderAgentMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BuilderAgentFileWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type BuilderAgentResult struct {
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

func GenerateBuilderFiles(ctx context.Context, messages []BuilderAgentMessage) (*BuilderAgentResult, error) {
	requestBody, err := json.Marshal(builderAgentChatCompletionRequest{
		Model:    app.Config.BuilderAgentModel,
		Messages: messages,
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

	endpoint := strings.TrimRight(app.Config.BuilderAgentBaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+app.Config.BuilderAgentAPIKey)
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("builder agent request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var completion builderAgentChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return nil, err
	}
	if len(completion.Choices) == 0 {
		return nil, errors.New("builder agent response missing choices")
	}

	var result BuilderAgentResult
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &result); err != nil {
		return nil, err
	}
	if err := validateBuilderAgentResult(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func builderAgentResultSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"assistant_message", "files"},
		"properties": map[string]any{
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
		return "", fmt.Errorf("%w: %s", ErrBuilderAgentUnsafeFilePath, filePath)
	}

	segments := strings.Split(normalized, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("%w: %s", ErrBuilderAgentUnsafeFilePath, filePath)
		}
	}

	return path.Clean(normalized), nil
}
