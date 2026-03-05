package openapi

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/yaml"
)

// RegisterRoutes exposes generated OpenAPI docs through HTTP endpoints.
func RegisterRoutes(r *gin.Engine, cfg Config) {
	devMode := gin.Mode() == gin.DebugMode

	type docBundle struct {
		JSON []byte
		YAML []byte
	}

	generate := func() (*docBundle, error) {
		spec := BuildFromGinEngine(r, cfg)

		jsonBytes, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return nil, err
		}
		yamlBytes, err := yaml.Marshal(spec)
		if err != nil {
			return nil, err
		}

		if devMode {
			if err := os.MkdirAll("openapi", 0o755); err != nil {
				log.Printf("openapi: failed to create output dir in dev mode: %v", err)
			} else {
				_ = os.WriteFile(filepath.Join("openapi", "openapi.json"), jsonBytes, 0o644)
				_ = os.WriteFile(filepath.Join("openapi", "openapi.yaml"), yamlBytes, 0o644)
			}
		}

		return &docBundle{JSON: jsonBytes, YAML: yamlBytes}, nil
	}

	var cached *docBundle
	if !devMode {
		var err error
		cached, err = generate()
		if err != nil {
			log.Printf("openapi: failed to generate initial docs: %v", err)
		}
	}

	load := func(c *gin.Context) *docBundle {
		if devMode {
			docs, err := generate()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate openapi docs"})
				return nil
			}
			return docs
		}
		if cached == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "openapi docs unavailable"})
			return nil
		}
		return cached
	}

	r.GET("/openapi", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/openapi/index.html")
	})

	r.GET("/openapi/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
	})

	r.GET("/openapi.json", func(c *gin.Context) {
		docs := load(c)
		if docs == nil {
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", docs.JSON)
	})
	r.GET("/openapi.yaml", func(c *gin.Context) {
		docs := load(c)
		if docs == nil {
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.YAML)
	})

	r.GET("/openapi/openapi.json", func(c *gin.Context) {
		docs := load(c)
		if docs == nil {
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", docs.JSON)
	})
	r.GET("/openapi/openapi.yaml", func(c *gin.Context) {
		docs := load(c)
		if docs == nil {
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.YAML)
	})
}

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Ketches OpenAPI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    html, body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/openapi/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: 'BaseLayout'
    });
  </script>
</body>
</html>
`
