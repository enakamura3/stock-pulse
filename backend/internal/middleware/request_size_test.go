package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestSizeLimit_JSONUnderLimit(t *testing.T) {
	handler := RequestSizeLimit(1024, 2048)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	bodyContent := `{"message": "hello world"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(bodyContent))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, bodyContent, rec.Body.String())
}

func TestRequestSizeLimit_JSONOverLimit(t *testing.T) {
	handler := RequestSizeLimit(100, 2048)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			if IsMaxBytesError(err) {
				http.Error(w, "Payload Too Large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Envia 200 bytes quando o limite é 100
	hugeBody := strings.Repeat("A", 200)
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(hugeBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Contains(t, rec.Body.String(), "Payload Too Large")
}

func TestRequestSizeLimit_MultipartForm(t *testing.T) {
	// JSON limite: 50 bytes, Multipart limite: 500 bytes
	handler := RequestSizeLimit(50, 500)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(1000)
		if err != nil {
			if IsMaxBytesError(err) {
				http.Error(w, "Multipart Too Large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Multipart com 100 bytes (maior que 50 bytes de JSON, menor que 500 de multipart) -> deve passar
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.csv")
	assert.NoError(t, err)
	_, _ = part.Write([]byte(strings.Repeat("X", 80)))
	_ = writer.Close()

	reqOK := httptest.NewRequest(http.MethodPost, "/api/upload", &buf)
	reqOK.Header.Set("Content-Type", writer.FormDataContentType())
	recOK := httptest.NewRecorder()

	handler.ServeHTTP(recOK, reqOK)
	assert.Equal(t, http.StatusOK, recOK.Code)

	// 2. Multipart excedendo limite de 500 bytes -> deve falhar com 413
	var bufLarge bytes.Buffer
	writerLarge := multipart.NewWriter(&bufLarge)
	partLarge, err := writerLarge.CreateFormFile("file", "huge.csv")
	assert.NoError(t, err)
	_, _ = partLarge.Write([]byte(strings.Repeat("X", 600)))
	_ = writerLarge.Close()

	reqLarge := httptest.NewRequest(http.MethodPost, "/api/upload", &bufLarge)
	reqLarge.Header.Set("Content-Type", writerLarge.FormDataContentType())
	recLarge := httptest.NewRecorder()

	handler.ServeHTTP(recLarge, reqLarge)
	assert.Equal(t, http.StatusRequestEntityTooLarge, recLarge.Code)
	assert.Contains(t, recLarge.Body.String(), "Multipart Too Large")
}

func TestIsMaxBytesError(t *testing.T) {
	assert.False(t, IsMaxBytesError(nil))
	assert.False(t, IsMaxBytesError(errors.New("other error")))
	assert.True(t, IsMaxBytesError(&http.MaxBytesError{Limit: 100}))
	assert.True(t, IsMaxBytesError(fmt.Errorf("wrapped error: %w", &http.MaxBytesError{Limit: 100})))
	assert.True(t, IsMaxBytesError(errors.New("http: request body too large")))
}

func TestRequestSizeLimit_DefaultLimits(t *testing.T) {
	// Passando 0 ou negativo para acionar os fallbacks para DefaultMaxJSONBytes e DefaultMaxMultipartBytes
	mw := RequestSizeLimit(0, -1)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/ping", strings.NewReader("ok"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequestSizeLimit_NilBody(t *testing.T) {
	mw := RequestSizeLimit(1024, 2048)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Nil(t, r.Body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/nobody", nil)
	req.Body = nil
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

