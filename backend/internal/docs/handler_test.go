package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler("path/to/yaml")
	assert.Equal(t, "path/to/yaml", h.yamlPath)
}

func TestHandler_ServeYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "openapi-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("openapi: 3.0.0")
	assert.NoError(t, err)
	tmpFile.Close()

	h := NewHandler(tmpFile.Name())

	req := httptest.NewRequest("GET", "/api/v1/swagger/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	h.ServeYAML(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/yaml; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "openapi: 3.0.0")
}

func TestHandler_ServeUI(t *testing.T) {
	h := NewHandler("path")

	req := httptest.NewRequest("GET", "/api/v1/swagger/", nil)
	rec := httptest.NewRecorder()

	h.ServeUI(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "<title>StockPulse - Documentação de API</title>")
}
