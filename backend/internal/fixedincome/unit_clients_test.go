package fixedincome

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBCBClient_Unit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("dataInicial") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"data":"02/01/2026","valor":"0.04"}]`))
	}))
	defer server.Close()

	client := &bcbClient{
		httpClient: &http.Client{
			Transport: &mockTransport{serverURL: server.URL},
		},
	}

	// 1. Unsupported indexer
	_, err := client.FetchRates(context.Background(), "INVALID", time.Now(), time.Now())
	assert.ErrorContains(t, err, "unsupported indexer")

	// 2. Success CDI
	rates, err := client.FetchRates(context.Background(), "CDI", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Len(t, rates, 1)
	assert.Equal(t, 0.04, rates[0].Rate)

	// 3. NewBCBClient constructor
	c := NewBCBClient()
	assert.NotNil(t, c)
}

func TestAnbimaClient_Unit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"date":"2026-01-01","name":"Confraternização Universal","type":"national"}]`))
	}))
	defer server.Close()

	client := &anbimaClient{
		httpClient: &http.Client{
			Transport: &mockTransport{serverURL: server.URL},
		},
	}

	holidays, err := client.FetchHolidays(context.Background(), 2026)
	assert.NoError(t, err)
	assert.Len(t, holidays, 1)
	assert.Equal(t, "Confraternização Universal", holidays[0].Name)

	// NewAnbimaClient constructor
	c := NewAnbimaClient()
	assert.NotNil(t, c)
}
