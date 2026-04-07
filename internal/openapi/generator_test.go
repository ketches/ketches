package openapi

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type createWidgetRequest struct {
	Name string `json:"name" binding:"required"`
	Size int    `json:"size"`
}

type widgetResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func TestBuildFromGinEngineIncludesRegisteredRequestAndResponseSchemas(t *testing.T) {
	t.Helper()

	engine := gin.New()
	engine.POST("/api/v1/widgets/:widgetID", func(c *gin.Context) {})

	spec := BuildFromGinEngine(engine, Config{
		OperationSpecs: []OperationSpec{
			{
				Method:        "POST",
				Path:          "/api/v1/widgets/{widgetID}",
				RequestBody:   createWidgetRequest{},
				ResponseBody:  widgetResponse{},
				SuccessStatus: 201,
			},
		},
	})

	paths := spec["paths"].(map[string]any)
	pathItem := paths["/api/v1/widgets/{widgetID}"].(map[string]any)
	op := pathItem["post"].(map[string]any)

	requestBody := op["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	jsonContent := content["application/json"].(map[string]any)
	requestSchema := jsonContent["schema"].(map[string]any)
	if got := requestSchema["$ref"]; got != "#/components/schemas/CreateWidgetRequest" {
		t.Fatalf("expected request body ref to CreateWidgetRequest, got %v", got)
	}

	responses := op["responses"].(map[string]any)
	successResponse := responses["201"].(map[string]any)
	successContent := successResponse["content"].(map[string]any)
	successJSON := successContent["application/json"].(map[string]any)
	successSchema := successJSON["schema"].(map[string]any)
	if got := successSchema["$ref"]; got != "#/components/schemas/ResponseOfWidgetResponse" {
		t.Fatalf("expected success response ref to ResponseOfWidgetResponse, got %v", got)
	}

	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	widgetSchema := schemas["WidgetResponse"].(map[string]any)
	properties := widgetSchema["properties"].(map[string]any)
	createdAt := properties["created_at"].(map[string]any)
	if got := createdAt["type"]; got != "string" {
		t.Fatalf("expected created_at to be string, got %v", got)
	}
	if got := createdAt["format"]; got != "date-time" {
		t.Fatalf("expected created_at format to be date-time, got %v", got)
	}

	requestComponent := schemas["CreateWidgetRequest"].(map[string]any)
	required := requestComponent["required"].([]string)
	if len(required) != 1 || required[0] != "name" {
		t.Fatalf("expected required fields to contain name, got %#v", required)
	}

	responseWrapper := schemas["ResponseOfWidgetResponse"].(map[string]any)
	responseWrapperProps := responseWrapper["properties"].(map[string]any)
	dataSchema := responseWrapperProps["data"].(map[string]any)
	if got := dataSchema["$ref"]; got != "#/components/schemas/WidgetResponse" {
		t.Fatalf("expected response wrapper data ref to WidgetResponse, got %v", got)
	}
}
