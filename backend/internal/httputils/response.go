package httputils

import (
	"encoding/json"
	"net/http"
)

// RespondWithError envia uma resposta JSON contendo apenas uma mensagem de erro e o respectivo status HTTP.
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}

// RespondWithJSON converte o payload em JSON e escreve na resposta HTTP com o respectivo status code.
func RespondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Erro de serialização JSON interno"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(response)
}
