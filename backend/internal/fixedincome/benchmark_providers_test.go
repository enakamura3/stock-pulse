package fixedincome

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── IndexRegistry ──────────────────────────────────────────────────────────

type mockIndexProvider struct {
	rates []IndexRate
	err   error
}

func (m *mockIndexProvider) FetchRates(_ context.Context, _ string, _, _ time.Time) ([]IndexRate, error) {
	return m.rates, m.err
}

func TestIndexRegistry_Fetch_PrimarySuccess(t *testing.T) {
	want := []IndexRate{{Indexer: "CDI", Rate: 0.05, Date: time.Now()}}
	primary := &mockIndexProvider{rates: want}

	registry := NewIndexRegistry()
	registry.Register(IndexerConfig{Name: "CDI", PrimaryProvider: primary})

	got, err := registry.Fetch(context.Background(), "CDI", time.Now().AddDate(0, -1, 0), time.Now())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestIndexRegistry_Fetch_FallsBackOnPrimaryError(t *testing.T) {
	primary := &mockIndexProvider{err: errors.New("primary failed")}
	fallback := &mockIndexProvider{rates: []IndexRate{{Indexer: "IFIX", Rate: 3000.0, Date: time.Now()}}}

	registry := NewIndexRegistry()
	registry.Register(IndexerConfig{Name: "IFIX", PrimaryProvider: primary, FallbackProvider: fallback})

	got, err := registry.Fetch(context.Background(), "IFIX", time.Now().AddDate(0, -1, 0), time.Now())
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.InDelta(t, 3000.0, got[0].Rate, 1e-6)
}

func TestIndexRegistry_Fetch_FallsBackOnPrimaryEmpty(t *testing.T) {
	primary := &mockIndexProvider{rates: []IndexRate{}} // retorna vazio
	fallback := &mockIndexProvider{rates: []IndexRate{{Indexer: "IBOV", Rate: 125000.0, Date: time.Now()}}}

	registry := NewIndexRegistry()
	registry.Register(IndexerConfig{Name: "IBOV", PrimaryProvider: primary, FallbackProvider: fallback})

	got, err := registry.Fetch(context.Background(), "IBOV", time.Now().AddDate(0, -1, 0), time.Now())
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestIndexRegistry_Fetch_BothFail(t *testing.T) {
	primary := &mockIndexProvider{err: errors.New("primary error")}
	fallback := &mockIndexProvider{err: errors.New("fallback error")}

	registry := NewIndexRegistry()
	registry.Register(IndexerConfig{Name: "SP500", PrimaryProvider: primary, FallbackProvider: fallback})

	_, err := registry.Fetch(context.Background(), "SP500", time.Now().AddDate(0, -1, 0), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fallback também falhou")
}

func TestIndexRegistry_Fetch_UnknownIndexer(t *testing.T) {
	registry := NewIndexRegistry()
	_, err := registry.Fetch(context.Background(), "INEXISTENTE", time.Now().AddDate(0, -1, 0), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "não configurado")
}

func TestIndexRegistry_Fetch_NoFallback_PrimaryError(t *testing.T) {
	primary := &mockIndexProvider{err: errors.New("bcb offline")}

	registry := NewIndexRegistry()
	registry.Register(IndexerConfig{Name: "CDI", PrimaryProvider: primary})

	_, err := registry.Fetch(context.Background(), "CDI", time.Now().AddDate(0, -1, 0), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bcb offline")
}

// ─── BCBProvider ─────────────────────────────────────────────────────────────

// bcbClientMockAdapter adapta mockIndexProvider para a interface BCBClient.
type bcbClientMockAdapter struct {
	inner *mockIndexProvider
}

func (a *bcbClientMockAdapter) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	return a.inner.FetchRates(ctx, indexer, startDate, endDate)
}

func TestBCBProvider_DelegatesSuccessfully(t *testing.T) {
	want := []IndexRate{{Indexer: "CDI", Rate: 0.04, Date: time.Now()}}
	provider := &BCBProvider{client: &bcbClientMockAdapter{inner: &mockIndexProvider{rates: want}}}

	got, err := provider.FetchRates(context.Background(), "CDI", time.Now().AddDate(0, -1, 0), time.Now())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestBCBProvider_DelegatesError(t *testing.T) {
	provider := &BCBProvider{client: &bcbClientMockAdapter{inner: &mockIndexProvider{err: errors.New("timeout")}}}

	_, err := provider.FetchRates(context.Background(), "CDI", time.Now().AddDate(0, -1, 0), time.Now())
	require.Error(t, err)
}

// ─── BrapiProvider ───────────────────────────────────────────────────────────

func TestBrapiProvider_MapIndexerToTicker(t *testing.T) {
	p := &BrapiProvider{}

	tests := []struct {
		indexer string
		want    string
	}{
		{"IFIX", "IFIX.SA"},
		{"ifix", "IFIX.SA"},
		{"IBOV", "IBOV"}, // ticker nativo da BRAPI — NÃO "^BVSP" (isso é formato Yahoo Finance)
		{"ibov", "IBOV"},
		{"CDI", ""},   // não suportado pelo BrapiProvider
		{"SELIC", ""}, // não suportado pelo BrapiProvider
	}

	for _, tc := range tests {
		t.Run(tc.indexer, func(t *testing.T) {
			got := p.mapIndexerToTicker(tc.indexer)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBrapiProvider_RejectsPeriodsOlderThan3Months(t *testing.T) {
	p := NewBrapiProvider()
	startDate := time.Now().AddDate(-1, 0, 0) // 1 ano atrás — excede limite gratuito BRAPI

	_, err := p.FetchRates(context.Background(), "IFIX", startDate, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "3 meses")
}

func TestBrapiProvider_RejectsUnsupportedIndexer(t *testing.T) {
	p := NewBrapiProvider()
	// Período dentro do limite para chegar na verificação de ticker
	start := time.Now().AddDate(0, 0, -14)

	_, err := p.FetchRates(context.Background(), "CDI", start, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nao suportado")
}

func TestBrapiProvider_ParsesValidResponse(t *testing.T) {
	// Timestamp: 2023-11-14 (dentro dos últimos 3 meses seria falso aqui, mas
	// o servidor mock responde imediatamente — precisamos de datas recentes)
	now := time.Now()
	ts1 := now.AddDate(0, -2, -1).Unix() // ~2 meses atrás
	ts2 := now.AddDate(0, -2, 0).Unix()  // ~2 meses atrás + 1 dia

	body := fmt.Sprintf(`{
		"results": [{
			"symbol": "IFIX.SA",
			"historicalDataPrice": [
				{"date": %d, "close": 3100.50},
				{"date": %d, "close": 3105.75}
			]
		}]
	}`, ts1, ts2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := &BrapiProvider{
		client: &http.Client{
			Transport: &mockTransport{serverURL: srv.URL},
		},
	}

	start := now.AddDate(0, -2, -5) // dentro do limite de 3 meses
	rates, err := p.FetchRates(context.Background(), "IFIX", start, now)
	require.NoError(t, err)
	assert.NotEmpty(t, rates)
	for _, r := range rates {
		assert.Equal(t, "IFIX", r.Indexer)
		assert.True(t, r.Rate > 0)
	}
}

func TestBrapiProvider_ParsesEmptyResults(t *testing.T) {
	body := `{"results": []}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := &BrapiProvider{
		client: &http.Client{Transport: &mockTransport{serverURL: srv.URL}},
	}

	start := time.Now().AddDate(0, -1, 0)
	_, err := p.FetchRates(context.Background(), "IBOV", start, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nenhum resultado")
}

// ─── YahooFinanceIndexProvider ────────────────────────────────────────────────

func TestYahooProvider_MapIndexerToTicker(t *testing.T) {
	p := &YahooFinanceIndexProvider{}

	tests := []struct {
		indexer string
		want    string
	}{
		{"IFIX", "IFIX.SA"},
		{"IBOV", "^BVSP"},  // Yahoo Finance usa "^BVSP" para o Ibovespa
		{"SP500", "^GSPC"},
		{"CDI", ""},  // não suportado pelo YahooProvider
	}

	for _, tc := range tests {
		t.Run(tc.indexer, func(t *testing.T) {
			got := p.mapIndexerToTicker(tc.indexer)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestYahooProvider_ParsesValidResponse(t *testing.T) {
	closeVal := 5000.25
	body := fmt.Sprintf(`{
		"chart": {
			"result": [{
				"timestamp": [1700000000, 1700086400],
				"indicators": {
					"quote": [{
						"close": [%g, null]
					}]
				}
			}],
			"error": null
		}
	}`, closeVal)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := &YahooFinanceIndexProvider{
		client: &http.Client{
			Transport: &mockTransport{serverURL: srv.URL},
		},
	}

	rates, err := p.FetchRates(context.Background(), "SP500", time.Now().AddDate(-1, 0, 0), time.Now())
	require.NoError(t, err)
	// Deve ignorar o null e retornar apenas 1 ponto
	assert.Len(t, rates, 1)
	assert.Equal(t, "SP500", rates[0].Indexer)
	assert.InDelta(t, closeVal, rates[0].Rate, 1e-6)
}

func TestYahooProvider_ReturnsErrorOnHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := &YahooFinanceIndexProvider{
		client: &http.Client{
			Transport: &mockTransport{serverURL: srv.URL},
		},
	}

	_, err := p.FetchRates(context.Background(), "SP500", time.Now().AddDate(-1, 0, 0), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

func TestYahooProvider_ReturnsErrorOnEmptyResult(t *testing.T) {
	body := `{"chart": {"result": [], "error": null}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := &YahooFinanceIndexProvider{
		client: &http.Client{Transport: &mockTransport{serverURL: srv.URL}},
	}

	_, err := p.FetchRates(context.Background(), "IBOV", time.Now().AddDate(-1, 0, 0), time.Now())
	require.Error(t, err)
}

// ─── Helper: mockTransport ────────────────────────────────────────────────────

// mockTransport redireciona qualquer requisição para um servidor de teste local,
// preservando o path e query originais da requisição.
type mockTransport struct {
	serverURL string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	newReq.URL.Scheme = "http"
	newReq.URL.Host = strings.TrimPrefix(t.serverURL, "http://")
	return http.DefaultTransport.RoundTrip(newReq)
}
