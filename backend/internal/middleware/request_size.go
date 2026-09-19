package middleware

import (
	"errors"
	"net/http"
	"strings"
)

const (
	// DefaultMaxJSONBytes define o limite padrão de tamanho para requisições JSON e padrão (1 MB).
	DefaultMaxJSONBytes int64 = 1 << 20
	// DefaultMaxMultipartBytes define o limite padrão de tamanho para requisições multipart/form-data (15 MB).
	DefaultMaxMultipartBytes int64 = 15 << 20
)

// IsMaxBytesError verifica se o erro fornecido decorre de uma violação de limite de tamanho de payload (http.MaxBytesReader).
func IsMaxBytesError(err error) bool {
	if err == nil {
		return false
	}
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	return strings.Contains(err.Error(), "http: request body too large")
}

// RequestSizeLimit cria um middleware HTTP que aplica http.MaxBytesReader ao corpo das requisições.
// Requisições com Content-Type multipart/form-data recebem maxMultipartBytes, e as demais recebem maxJSONBytes.
func RequestSizeLimit(maxJSONBytes, maxMultipartBytes int64) func(http.Handler) http.Handler {
	if maxJSONBytes <= 0 {
		maxJSONBytes = DefaultMaxJSONBytes
	}
	if maxMultipartBytes <= 0 {
		maxMultipartBytes = DefaultMaxMultipartBytes
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}

			limit := maxJSONBytes
			contentType := r.Header.Get("Content-Type")
			if strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
				limit = maxMultipartBytes
			}

			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
