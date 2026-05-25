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
							"chartPreviousClose": 140.0
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
