package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupProviderTest(handler http.HandlerFunc) (*YahooFinanceProvider, *httptest.Server) {
	server := httptest.NewServer(handler)
	p := NewYahooFinanceProvider()

	// Override the HTTP client transport to route to our test server
	// Since the URL is hardcoded, we will intercept requests with a custom RoundTripper
	p.client.Transport = &mockTransport{serverURL: server.URL}

	return p, server
}

type mockTransport struct {
	serverURL string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to the local test server
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(m.serverURL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func TestProvider_GetQuote(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "AAPL")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"chart": {
					"result": [{
						"meta": {
							"currency": "USD",
							"symbol": "AAPL",
							"longName": "Apple Inc.",
							"regularMarketPrice": 150.0,
							"chartPreviousClose": 140.0,
							"regularMarketVolume": 55000000
						}
					}]
				}
			}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		q, err := p.GetQuote(context.Background(), "AAPL")
		assert.NoError(t, err)
		assert.Equal(t, "AAPL", q.Symbol)
		assert.Equal(t, "Apple Inc.", q.Name)
		assert.Equal(t, 150.0, q.Price)
		assert.Equal(t, "USD", q.Currency)
		assert.Equal(t, 10.0, q.Change)
		assert.InDelta(t, (10.0/140.0)*100, q.ChangePercent, 0.0001)
		assert.Equal(t, 140.0, q.PreviousClose)
		assert.Equal(t, int64(55000000), q.Volume)
	})

	t.Run("Name Fallback", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": [{"meta": {"shortName": "Apple", "symbol": "AAPL", "regularMarketPrice": 150.0}}]}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		q, err := p.GetQuote(context.Background(), "AAPL")
		assert.NoError(t, err)
		assert.Equal(t, "Apple", q.Name)
	})

	t.Run("Name Fallback to Symbol", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": [{"meta": {"symbol": "AAPL", "regularMarketPrice": 150.0}}]}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		q, err := p.GetQuote(context.Background(), "AAPL")
		assert.NoError(t, err)
		assert.Equal(t, "AAPL", q.Name)
	})

	t.Run("HTTP Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetQuote(context.Background(), "AAPL")
		assert.ErrorContains(t, err, "status 404")
	})

	t.Run("JSON Parse Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetQuote(context.Background(), "AAPL")
		assert.Error(t, err)
	})

	t.Run("API Error Response", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"error": "Not found"}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetQuote(context.Background(), "AAPL")
		assert.ErrorContains(t, err, "erro retornado")
	})

	t.Run("Empty Result", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": []}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetQuote(context.Background(), "AAPL")
		assert.ErrorContains(t, err, "ativo não encontrado")
	})
}

func TestProvider_SearchAssets(t *testing.T) {
	t.Run("Empty Query", func(t *testing.T) {
		p := NewYahooFinanceProvider()
		res, err := p.SearchAssets(context.Background(), "")
		assert.NoError(t, err)
		assert.Len(t, res, 0)
	})

	t.Run("Success", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.RawQuery, "q=AAPL")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"quotes": [
					{"symbol": "AAPL", "longname": "Apple Inc.", "exchange": "NMS", "quoteType": "EQUITY"},
					{"symbol": "AAP", "shortname": "Advance Auto Parts", "exchange": "NYQ", "quoteType": "EQUITY"},
					{"symbol": "AAPL.BA", "exchange": "BUE", "quoteType": "EQUITY"},
					{"symbol": ""}
				]
			}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		res, err := p.SearchAssets(context.Background(), "AAPL")
		assert.NoError(t, err)
		assert.Len(t, res, 3) // Empty symbol is ignored
		assert.Equal(t, "Apple Inc.", res[0].Name)
		assert.Equal(t, "Advance Auto Parts", res[1].Name)
		assert.Equal(t, "AAPL.BA", res[2].Name) // Fallback to symbol
	})

	t.Run("HTTP Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.SearchAssets(context.Background(), "AAPL")
		assert.ErrorContains(t, err, "status 429")
	})

	t.Run("JSON Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.SearchAssets(context.Background(), "AAPL")
		assert.Error(t, err)
	})
}

func TestProvider_NewRequestError(t *testing.T) {
	p := NewYahooFinanceProvider()

	// If URL is invalid, NewRequestWithContext fails
	_, err := p.SearchAssets(nil, "AAPL") // nil context forces error
	assert.Error(t, err)

	_, err = p.GetQuote(nil, "AAPL") // nil context forces error
	assert.Error(t, err)
}

func TestProvider_DoError(t *testing.T) {
	p := NewYahooFinanceProvider()
	// No server mock, default client to a bad scheme to force do error
	p.client.Transport = &mockTransport{serverURL: "http://127.0.0.1:0"} // nothing running on port 0

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := p.SearchAssets(ctx, "AAPL")
	assert.Error(t, err)

	_, err = p.GetQuote(ctx, "AAPL")
	assert.Error(t, err)
}

func TestProvider_GetHistoricalPrices(t *testing.T) {
	t.Run("Success with default range", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "AAPL")
			assert.Contains(t, r.URL.RawQuery, "range=10y")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"chart": {
					"result": [{
						"timestamp": [1700000000, 1700086400],
						"indicators": {
							"quote": [{
								"close": [150.5, null]
							}]
						}
					}]
				}
			}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		prices, err := p.GetHistoricalPrices(context.Background(), "AAPL", "")
		assert.NoError(t, err)
		assert.Len(t, prices, 1) // null close is filtered out
		assert.Equal(t, int64(1700000000), prices[0].Timestamp)
		assert.Equal(t, 150.5, prices[0].Close)
	})

	t.Run("GetHistoricalPricesBetween Success", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.RawQuery, "period1=1600000000")
			assert.Contains(t, r.URL.RawQuery, "period2=1700000000")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"chart": {
					"result": [{
						"timestamp": [1650000000],
						"indicators": {
							"quote": [{
								"close": [160.0]
							}]
						}
					}]
				}
			}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		prices, err := p.GetHistoricalPricesBetween(context.Background(), "AAPL", 1600000000, 1700000000)
		assert.NoError(t, err)
		assert.Len(t, prices, 1)
		assert.Equal(t, 160.0, prices[0].Close)
	})

	t.Run("HTTP Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetHistoricalPrices(context.Background(), "AAPL", "5y")
		assert.ErrorContains(t, err, "status 500")
	})

	t.Run("JSON Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{broken json`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetHistoricalPrices(context.Background(), "AAPL", "10y")
		assert.Error(t, err)
	})

	t.Run("Provider Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"error": "symbol not found"}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetHistoricalPrices(context.Background(), "AAPL", "10y")
		assert.ErrorContains(t, err, "erro no provedor")
	})

	t.Run("Empty Result", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": []}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetHistoricalPrices(context.Background(), "AAPL", "10y")
		assert.ErrorContains(t, err, "resultado histórico vazio")
	})

	t.Run("Empty Timestamps or Quotes", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": [{"timestamp": [], "indicators": {"quote": []}}]}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetHistoricalPrices(context.Background(), "AAPL", "10y")
		assert.ErrorContains(t, err, "sem timestamps ou quotes")
	})

	t.Run("Length Inconsistency", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": [{"timestamp": [1, 2], "indicators": {"quote": [{"close": [10.0]}]}}]}}`))
		}
		p, server := setupProviderTest(handler)
		defer server.Close()

		_, err := p.GetHistoricalPrices(context.Background(), "AAPL", "10y")
		assert.ErrorContains(t, err, "inconsistência de tamanho")
	})

	t.Run("Context error and Do error", func(t *testing.T) {
		p := NewYahooFinanceProvider()
		_, err := p.GetHistoricalPrices(nil, "AAPL", "10y")
		assert.Error(t, err)

		p.client.Transport = &mockTransport{serverURL: "http://127.0.0.1:0"}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		_, err = p.GetHistoricalPrices(ctx, "AAPL", "10y")
		assert.Error(t, err)
	})
}

