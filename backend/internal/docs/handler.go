package docs

import (
	"net/http"
)

// Handler gerencia a renderização do Swagger UI e exposição do OpenAPI YAML.
type Handler struct {
	yamlPath string
}

// NewHandler inicializa o Handler carregando o caminho do openapi.yaml.
func NewHandler(yamlPath string) *Handler {
	return &Handler{
		yamlPath: yamlPath,
	}
}

// ServeYAML serve o arquivo openapi.yaml bruto para consumo da interface.
func (h *Handler) ServeYAML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	http.ServeFile(w, r, h.yamlPath)
}

// ServeUI renderiza a UI interativa do Swagger UI estilizada com o tema escuro do StockPulse.
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>StockPulse - Documentação de API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5/favicon-32x32.png" />
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #0b0f19; color: #e2e8f0; }
    
    /* Filtro invertido premium para Dark Mode no Swagger */
    .swagger-ui { 
      background: #0b0f19; 
      filter: invert(0.88) hue-rotate(180deg);
      padding: 10px 0;
    }
    
    .swagger-ui .info {
      margin: 30px 0;
    }
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
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(tmpl))
}
