package docs

import (
	_ "embed"
	"net/http"
	"os"
)

//go:embed openapi.yaml
var defaultOpenAPIYAML []byte

// Handler gerencia a renderização do Swagger UI e exposição do OpenAPI YAML.
type Handler struct {
	yamlBytes []byte
}

// NewHandler inicializa o Handler carregando o caminho do openapi.yaml com fallback para o embedded.
func NewHandler(yamlPath string) *Handler {
	if yamlPath != "" {
		if data, err := os.ReadFile(yamlPath); err == nil && len(data) > 0 {
			return &Handler{
				yamlBytes: data,
			}
		}
	}
	return &Handler{
		yamlBytes: defaultOpenAPIYAML,
	}
}

// ServeYAML serve o arquivo openapi.yaml bruto para consumo da interface.
func (h *Handler) ServeYAML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.yamlBytes)
}

// ServeUI renderiza a UI interativa do Swagger UI estilizada para o stock-pulse.
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>stock-pulse - Documentação de API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5/favicon-32x32.png" />
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" charset="UTF-8"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/api/v1/swagger/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.api,
          SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout",
        persistAuthorization: true,
        displayRequestDuration: true,
        docExpansion: "none",
        filter: true,
        tryItOutEnabled: true
      });
    };
  </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(tmpl))
}
