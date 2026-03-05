package openapi

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// Config controls top-level OpenAPI metadata.
type Config struct {
	Title       string
	Description string
	Version     string
	ServerURL   string
}

// BuildFromGinEngine builds an OpenAPI 3.0.3 document from gin routes.
func BuildFromGinEngine(r *gin.Engine, cfg Config) map[string]any {
	if cfg.Title == "" {
		cfg.Title = "Ketches API"
	}
	if cfg.Description == "" {
		cfg.Description = "Auto-generated from Gin route table."
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "/"
	}

	routes := r.Routes()
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})

	paths := map[string]any{}
	for _, route := range routes {
		if route.Path == "/openapi.json" || route.Path == "/openapi.yaml" {
			continue
		}

		oasPath, parameters := convertGinPathToOAS(route.Path)
		pathItem, ok := paths[oasPath].(map[string]any)
		if !ok {
			pathItem = map[string]any{}
			paths[oasPath] = pathItem
		}

		method := strings.ToLower(route.Method)
		tag := inferTag(oasPath)

		op := map[string]any{
			"operationId": buildOperationID(route.Method, oasPath),
			"summary":     fmt.Sprintf("%s %s", route.Method, oasPath),
			"tags":        []string{tag},
			"responses": map[string]any{
				"200": map[string]any{"description": "OK"},
				"400": map[string]any{"description": "Bad Request"},
				"401": map[string]any{"description": "Unauthorized"},
				"403": map[string]any{"description": "Forbidden"},
				"500": map[string]any{"description": "Internal Server Error"},
			},
		}
		if len(parameters) > 0 {
			op["parameters"] = parameters
		}
		if requiresAuth(oasPath) {
			op["security"] = []any{
				map[string]any{"BearerAuth": []string{}},
			}
		}

		pathItem[method] = op
	}

	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       cfg.Title,
			"description": cfg.Description,
			"version":     cfg.Version,
		},
		"servers": []any{
			map[string]any{"url": cfg.ServerURL},
		},
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"BearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
	}

	return spec
}

func convertGinPathToOAS(path string) (string, []any) {
	segments := strings.Split(path, "/")
	params := make([]any, 0)

	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			name := strings.TrimPrefix(seg, ":")
			segments[i] = "{" + name + "}"
			params = append(params, map[string]any{
				"name":        name,
				"in":          "path",
				"required":    true,
				"description": "Path parameter",
				"schema":      map[string]any{"type": "string"},
			})
			continue
		}
		if strings.HasPrefix(seg, "*") {
			name := strings.TrimPrefix(seg, "*")
			segments[i] = "{" + name + "}"
			params = append(params, map[string]any{
				"name":        name,
				"in":          "path",
				"required":    true,
				"description": "Wildcard path parameter",
				"schema":      map[string]any{"type": "string"},
			})
		}
	}

	return strings.Join(segments, "/"), params
}

func buildOperationID(method, path string) string {
	id := strings.ToLower(method + "_" + path)
	id = strings.ReplaceAll(id, "{", "")
	id = strings.ReplaceAll(id, "}", "")
	id = nonAlnum.ReplaceAllString(id, "_")
	id = strings.Trim(id, "_")
	return id
}

func inferTag(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" {
		if parts[2] != "" {
			return parts[2]
		}
	}
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "default"
}

func requiresAuth(path string) bool {
	if !strings.HasPrefix(path, "/api/v1") {
		return false
	}

	publicPaths := map[string]bool{
		"/api/v1/version":       true,
		"/api/v1/users/sign-in": true,
		"/api/v1/users/sign-up": true,
	}
	if publicPaths[path] {
		return false
	}
	if strings.HasPrefix(path, "/api/v1/webhooks/") {
		return false
	}
	return true
}
