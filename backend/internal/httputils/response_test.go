package httputils

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespondWithJSON(t *testing.T) {
	w := httptest.NewRecorder()
	RespondWithJSON(w, http.StatusOK, map[string]string{"message": "ok"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"message": "ok"}`, w.Body.String())
}

func TestRespondWithError(t *testing.T) {
	w := httptest.NewRecorder()
	RespondWithError(w, http.StatusBadRequest, "invalid request")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "invalid request"}`, w.Body.String())
}

func TestRespondWithJSON_MarshalError(t *testing.T) {
	w := httptest.NewRecorder()
	// math.Inf(1) cannot be marshaled into JSON
	RespondWithJSON(w, http.StatusOK, map[string]float64{"val": math.Inf(1)})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Erro de serialização")
}
