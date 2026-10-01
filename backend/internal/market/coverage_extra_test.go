package market

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/time/rate"
)

func TestB3Client_FetchBase_Errors(t *testing.T) {
	client := NewB3Client()
	client.rateLimiter = rate.NewLimiter(rate.Inf, 100)

	t.Run("Context Cancelled RateLimiter Error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.fetchBase(ctx, "http://example.com", map[string]string{"k": "v"})
		assert.Error(t, err)
	})

	t.Run("JSON Marshal Error", func(t *testing.T) {
		_, err := client.fetchBase(context.Background(), "http://example.com", math.NaN())
		assert.Error(t, err)
	})

	t.Run("Invalid URL Prefix", func(t *testing.T) {
		_, err := client.fetchBase(context.Background(), "http://invalid \x7f url", map[string]string{"k": "v"})
		assert.Error(t, err)
	})

	t.Run("HTTP Do Error", func(t *testing.T) {
		c := NewB3Client()
		c.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return nil
		})
		_, err := c.fetchBase(context.Background(), "http://example.com", map[string]string{"k": "v"})
		assert.Error(t, err)
	})

	t.Run("Unmarshal Errors", func(t *testing.T) {
		c := NewB3Client()
		c.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("invalid-json")),
			}
		})

		_, err := c.FetchCashDividends(context.Background(), "PETR")
		assert.Error(t, err)

		_, err = c.FetchFundDividends(context.Background(), "MXRF")
		assert.Error(t, err)

		_, err = c.FetchCompanies(context.Background())
		assert.Error(t, err)
	})

	t.Run("FetchCashDividends and FetchFundDividends Context Cancel Error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.FetchCashDividends(ctx, "PETR")
		assert.Error(t, err)

		_, err = client.FetchFundDividends(ctx, "MXRF")
		assert.Error(t, err)
	})
}

func TestB3DividendSource_ParseAndValidationBranches(t *testing.T) {
	client := NewB3Client()
	client.rateLimiter = rate.NewLimiter(rate.Inf, 100)
	client.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.String(), "GetListedCompanies") {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"results": [{"tradingName": "PETROBRAS"}]}`)),
			}
		}
		// Return dividends with various parsing edge cases: invalid date, invalid amount, amortizacao, invalid payment date
		jsonResp := `{"results": [
			{"corporateAction": "DIVIDENDO", "valueCash": "1.0", "lastDatePriorEx": "invalid-date", "paymentDate": "15/03/2026"},
			{"corporateAction": "DIVIDENDO", "valueCash": "not-a-number", "lastDatePriorEx": "12/02/2026", "paymentDate": "15/03/2026"},
			{"corporateAction": "AMORTIZACAO", "valueCash": "1,25", "lastDatePriorEx": "12/02/2026", "paymentDate": ""},
			{"corporateAction": "DIVIDENDO", "valueCash": "2,00", "lastDatePriorEx": "12/02/2026", "paymentDate": "invalid-payment-date"}
		]}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(jsonResp)),
		}
	})

	source := NewB3DividendSource(client)

	t.Run("Parsing Edge Cases and Cache Hit", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 2)
		assert.Equal(t, "Amortização", events[0].Type)
		assert.Equal(t, 1.25, events[0].Amount)
		assert.Equal(t, events[0].Date, events[0].PaymentDate) // Empty payment date defaults to exDate

		assert.Equal(t, "Dividendo", events[1].Type)
		assert.Equal(t, 2.00, events[1].Amount)
		assert.Equal(t, events[1].Date, events[1].PaymentDate) // Invalid payment date defaults to exDate

		// Second call should hit the cache in resolveTradingName
		eventsCache, err := source.GetDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, eventsCache, 2)
	})

	t.Run("ResolveTradingName FetchCompanies Error", func(t *testing.T) {
		errClient := NewB3Client()
		errClient.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		errClient.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader(`internal error`)),
			}
		})
		errSource := NewB3DividendSource(errClient)
		_, err := errSource.GetDividends(context.Background(), "VALE3", "STOCK_BR")
		assert.Error(t, err)
	})

	t.Run("Fetch Cash and Fund Dividends Network Error", func(t *testing.T) {
		errClient := NewB3Client()
		errClient.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		errClient.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.String(), "GetListedCompanies") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"results": [{"tradingName": "PETROBRAS"}]}`)),
				}
			}
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader(`error`)),
			}
		})
		errSource := NewB3DividendSource(errClient)
		errSource.tickerNameCache["PETR4"] = "PETROBRAS"
		errSource.tickerNameCache["MXRF11"] = "MXRF11"

		// StockBR error
		_, err := errSource.GetDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.Error(t, err)

		// FII error
		_, err = errSource.GetDividends(context.Background(), "MXRF11", "FII")
		assert.Error(t, err)
	})

	t.Run("FII Unknown CorporateAction sets Rendimento", func(t *testing.T) {
		c := NewB3Client()
		c.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.String(), "GetListedCompanies") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"results": [{"tradingName": "FII_NAME"}]}`)),
				}
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"results": [
					{"corporateAction": "DISTRIBUICAO_SPECIAL", "valueCash": "0,85", "lastDatePriorEx": "10/01/2026", "paymentDate": "20/01/2026"}
				]}`)),
			}
		})
		s := NewB3DividendSource(c)
		s.tickerNameCache["KNRI11"] = "FII_NAME"
		evs, err := s.GetDividends(context.Background(), "KNRI11", "FII")
		assert.NoError(t, err)
		assert.Len(t, evs, 1)
		assert.Equal(t, "Rendimento", evs[0].Type)
	})

	t.Run("Concurrent resolveTradingName hits double-checked locking", func(t *testing.T) {
		c := NewB3Client()
		c.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			time.Sleep(5 * time.Millisecond)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"results": [{"tradingName": "CONC_NAME"}]}`)),
			}
		})
		s := NewB3DividendSource(c)

		barrier := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < 200; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-barrier
				_, _ = s.resolveTradingName(context.Background(), "CONC1")
			}()
		}
		close(barrier)
		wg.Wait()
	})
}

func TestFundamentusDividendSource_ParsingBranches(t *testing.T) {
	client := NewFundamentusClient()
	client.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		html := `
		<table id="resultado">
			<tbody>
				<tr><td>invalid-date</td><td>1,00</td><td>DIVIDENDO</td><td>01/01/2026</td></tr>
				<tr><td>10/01/2026</td><td>invalid-amount</td><td>DIVIDENDO</td><td>01/01/2026</td></tr>
				<tr><td>10/01/2026</td><td>1,50</td><td>JRS CAP PROPRIO</td><td>15/01/2026</td></tr>
				<tr><td>10/01/2026</td><td>0,80</td><td>AMORTIZACAO</td><td>-</td></tr>
				<tr><td>10/01/2026</td><td>0,60</td><td>RENDIMENTO</td><td>invalid-date</td></tr>
			</tbody>
		</table>`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(html)),
		}
	})

	source := NewFundamentusDividendSource(client)

	t.Run("Parses JCP, Amortizacao and Rendimento", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 3)
		assert.Equal(t, "JCP", events[0].Type)
		assert.Equal(t, 1.50, events[0].Amount)
		assert.Equal(t, "Amortização", events[1].Type)
		assert.Equal(t, 0.80, events[1].Amount)
		assert.Equal(t, "Rendimento", events[2].Type)
		assert.Equal(t, 0.60, events[2].Amount)
	})

	t.Run("FII Layout Dedup by Month in Source", func(t *testing.T) {
		c := NewFundamentusClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.String(), "fii_proventos.php") {
				fiiHTML := `
				<table id="resultado">
					<tbody>
						<tr><td>15/01/2026</td><td>RENDIMENTO</td><td>25/01/2026</td><td>1,10</td></tr>
						<tr><td>20/01/2026</td><td>RENDIMENTO</td><td>28/01/2026</td><td>0,50</td></tr>
					</tbody>
				</table>`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(fiiHTML)),
				}
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<table id="resultado"><tbody></tbody></table>`)),
			}
		})
		s := NewFundamentusDividendSource(c)
		events, err := s.GetDividends(context.Background(), "HGLG11", "FII")
		assert.NoError(t, err)
		assert.Len(t, events, 1) // Deduped to 1 per month
		assert.Equal(t, "Rendimento", events[0].Type)
	})
}


type mockFallbackSource struct {
	events []DividendEvent
	err    error
}

func (m *mockFallbackSource) Name() string { return "mock_fallback" }
func (m *mockFallbackSource) SupportedAssetTypes() []string {
	return []string{"STOCK_BR", "FII", "CRYPTO"}
}
func (m *mockFallbackSource) GetDividends(ctx context.Context, ticker, assetType string) ([]DividendEvent, error) {
	return m.events, m.err
}

func TestDividendGateway_MergeFIIAndOnlyFallback(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	now := time.Now()
	prim := &mockFallbackSource{
		events: []DividendEvent{
			{Date: now, Amount: 1.0, Type: "Rendimento"},
		},
	}
	sec := &mockFallbackSource{
		events: []DividendEvent{
			{Date: now, Amount: 1.0, Type: "Rendimento"},
		},
	}

	gw := &DividendGateway{
		routes: map[string][]sourceEntry{
			"FII": {
				{source: prim, role: "primary"},
				{source: sec, role: "secondary"},
			},
		},
		cache: rdb,
		ttl:   time.Hour,
	}

	t.Run("Merge FII primary and secondary", func(t *testing.T) {
		evs, err := gw.GetDividends(context.Background(), "MXRF11", "FII")
		assert.NoError(t, err)
		assert.Len(t, evs, 1)
	})

	t.Run("Only Fallback Configured", func(t *testing.T) {
		fb := &mockFallbackSource{
			events: []DividendEvent{
				{Date: now, Amount: 5.0, Type: "Dividendo"},
			},
		}
		gwFallback := &DividendGateway{
			routes: map[string][]sourceEntry{
				"CRYPTO": {
					{source: fb, role: "fallback"},
				},
			},
			cache: rdb,
			ttl:   time.Hour,
		}

		evs, err := gwFallback.GetDividends(context.Background(), "BTC", "CRYPTO")
		assert.NoError(t, err)
		assert.Len(t, evs, 1)
		assert.Equal(t, 5.0, evs[0].Amount)
	})

	t.Run("Unknown Asset Type", func(t *testing.T) {
		_, err := gw.GetDividends(context.Background(), "UNKNOWN", "UNKNOWN_TYPE")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tipo de ativo desconhecido")
	})

	t.Run("Fallback Failure All Fail", func(t *testing.T) {
		fbErr := &mockFallbackSource{
			err: fmt.Errorf("fallback error"),
		}
		gwErr := &DividendGateway{
			routes: map[string][]sourceEntry{
				"CRYPTO": {
					{source: fbErr, role: "fallback"},
				},
			},
			cache: rdb,
			ttl:   time.Hour,
		}
		_, err := gwErr.GetDividends(context.Background(), "ETH", "CRYPTO")
		assert.Error(t, err)
	})
}

func TestService_GetFundamentals_RedisSetError(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	scraper := NewScraper()
	scraper.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		html := `<html><body>
			<span class="txt">P/L</span><span class="data">5,5</span>
			<span class="txt">Div. Yield</span><span class="data">8,0%</span>
		</body></html>`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(html)),
		}
	})

	mockProv := &MockQuoteProvider{}
	mockProv.On("GetQuote", mock.Anything, "PETR4").Return(&Quote{
		Symbol: "PETR4",
		Price:  30.0,
	}, nil)

	svc := NewService(mockProv, rdb)
	svc.scraper = scraper

	// Cause Redis Set to fail by shutting down miniredis before Set
	mr.Close()

	fund, err := svc.GetFundamentals(context.Background(), "PETR4")
	assert.NoError(t, err)
	assert.NotNil(t, fund)
}

