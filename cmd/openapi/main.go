package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/openapi"
	"github.com/ketches/ketches/internal/routes"
	"sigs.k8s.io/yaml"
)

func main() {
	app.InitConfig()

	r := gin.New()
	routes.SetupRoutes(r)

	spec := openapi.BuildFromGinEngine(r, openapi.Config{
		Title:       "Ketches API",
		Description: "Auto-generated from Gin route table.",
		Version:     app.Version,
		ServerURL:   "/",
	})

	jsonBytes, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal openapi json: %v", err)
	}
	yamlBytes, err := yaml.Marshal(spec)
	if err != nil {
		log.Fatalf("failed to marshal openapi yaml: %v", err)
	}

	if err := os.MkdirAll("openapi", 0o755); err != nil {
		log.Fatalf("failed to create openapi output directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join("openapi", "openapi.json"), jsonBytes, 0o644); err != nil {
		log.Fatalf("failed to write openapi.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join("openapi", "openapi.yaml"), yamlBytes, 0o644); err != nil {
		log.Fatalf("failed to write openapi.yaml: %v", err)
	}

	log.Printf("openapi docs generated: %s, %s", filepath.Join("openapi", "openapi.json"), filepath.Join("openapi", "openapi.yaml"))
}
