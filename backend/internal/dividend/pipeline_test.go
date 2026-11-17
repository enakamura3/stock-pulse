package dividend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

type errorTransport struct {
	err error
}

func (t *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, t.err
}

type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestPipeline_Functions(t *testing.T) {
	t.Run("NormalizeType", func(t *testing.T) {
		assert.Equal(t, "JCP", NormalizeType("JRS CAP PROPRIO", "STOCK_BR"))
		assert.Equal(t, "JCP", NormalizeType("Juros sobre Capital Próprio", "STOCK_BR"))
		assert.Equal(t, "JCP", NormalizeType("JCP", "STOCK_BR"))
		assert.Equal(t, "Rendimento", NormalizeType("RENDIMENTO", "STOCK_BR"))
		assert.Equal(t, "Amortização", NormalizeType("AMORTIZACAO", "FII"))
		assert.Equal(t, "Rendimento", NormalizeType("DIVIDENDO", "FII"))
		assert.Equal(t, "Rendimento", NormalizeType("OUTRO", "FIAGRO"))
		assert.Equal(t, "Dividendo", NormalizeType("DIVIDENDO", "STOCK_BR"))
	})

	t.Run("NormalizeDate and SameCalendarDate", func(t *testing.T) {
		assert.True(t, NormalizeDate(time.Time{}).IsZero())
		d1 := time.Date(2026, 5, 10, 15, 30, 0, 0, time.FixedZone("BRT", -3*3600))
		d2 := NormalizeDate(d1)
		assert.Equal(t, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), d2)

		assert.True(t, SameCalendarDate(time.Time{}, time.Time{}))
		assert.False(t, SameCalendarDate(d2, time.Time{}))
		assert.True(t, SameCalendarDate(d1, d2))
	})

	t.Run("PreviousBusinessDay", func(t *testing.T) {
		assert.True(t, PreviousBusinessDay(time.Time{}).IsZero())
		// Monday 2026-08-10 -> Friday 2026-08-07
		mon := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), PreviousBusinessDay(mon))
		// Sunday 2026-08-09 -> Friday 2026-08-07
		sun := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), PreviousBusinessDay(sun))
		// Thursday 2026-11-12 -> Wednesday 2026-11-11
		thu := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC), PreviousBusinessDay(thu))
	})

	t.Run("ResolveEventType", func(t *testing.T) {
		assert.Equal(t, "JCP", ResolveEventType("JCP", "Dividendo"))
		assert.Equal(t, "JCP", ResolveEventType("Dividendo", "JCP"))
		assert.Equal(t, "Dividendo", ResolveEventType("Dividendo", "Dividendo"))
	})

	t.Run("ProcessPipeline: Normalize -> Deduplicate -> Validate", func(t *testing.T) {
		cum1 := time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC)
		pay1 := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
		cum2 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		pay2 := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

		raw := []Event{
			// Valid Dividendo followed by duplicate JCP -> should deduplicate and upgrade type to JCP
			{Date: cum1, PaymentDate: pay1, Amount: 0.50, Type: "DIVIDENDO"},
			{Date: cum1, PaymentDate: pay1, Amount: 0.50, Type: "JCP"},
			// Valid second event (later date, should sort first)
			{Date: cum2, PaymentDate: pay2, Amount: 1.00, Type: "DIVIDENDO"},
			// Invalid: missing PaymentDate
			{Date: cum1, PaymentDate: time.Time{}, Amount: 0.80, Type: "DIVIDENDO"},
			// Invalid: missing CumDate
			{Date: time.Time{}, PaymentDate: pay1, Amount: 0.80, Type: "DIVIDENDO"},
			// Invalid: zero/negative Amount
			{Date: cum1, PaymentDate: pay1, Amount: 0, Type: "DIVIDENDO"},
		}

		res := ProcessPipeline(raw, "STOCK_BR")
		assert.Len(t, res, 2)
		assert.Equal(t, NormalizeDate(cum2), res[0].Date)
		assert.Equal(t, "Dividendo", res[0].Type)
		assert.Equal(t, NormalizeDate(cum1), res[1].Date)
		assert.Equal(t, "JCP", res[1].Type)
	})

	t.Run("Resolve Fuzzy Match", func(t *testing.T) {
		cum1 := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
		pay1 := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
		cum2 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		pay2 := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
		cum3 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
		pay3 := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)

		existing := []AssetEvent{
			{ID: "e1", AssetID: "a1", CumDate: cum1, PaymentDate: pay1, GrossAmount: 1.00, Type: "Dividendo"},
			{ID: "e2", AssetID: "a1", CumDate: cum2, PaymentDate: pay2, GrossAmount: 1.00, Type: "Dividendo"},
			{ID: "e3", AssetID: "a1", CumDate: cum2, PaymentDate: pay2, GrossAmount: 0.50, Type: "Dividendo"},
		}

		incoming := []Event{
			// 1. Exact match -> Skip
			{Date: cum1, PaymentDate: pay1, Amount: 1.00, Type: "Dividendo"},
			// 2. Fuzzy match same type (1.03 vs 1.00, diff 0.03 <= 0.05) -> Update e2
			{Date: cum2, PaymentDate: pay2, Amount: 1.03, Type: "Dividendo"},
			// 3. Cross-type match (JCP 0.52 vs Dividendo 0.50, e2 is already matched!) -> Update e3 to JCP
			{Date: cum2, PaymentDate: pay2, Amount: 0.52, Type: "JCP"},
			// 4. New event -> Insert
			{Date: cum3, PaymentDate: pay3, Amount: 2.00, Type: "Dividendo"},
		}

		plan := Resolve("a1", incoming, existing)
		assert.Equal(t, 1, plan.Skips)
		assert.Len(t, plan.Updates, 2)
		assert.Equal(t, "e2", plan.Updates[0].ID)
		assert.Equal(t, 1.03, plan.Updates[0].GrossAmount)
		assert.Equal(t, "e3", plan.Updates[1].ID)
		assert.Equal(t, "JCP", plan.Updates[1].Type)
		assert.Len(t, plan.Inserts, 1)
		assert.Equal(t, 2.00, plan.Inserts[0].GrossAmount)
	})
}

type mockSource struct {
	name   string
	events []Event
	err    error
}

func (m *mockSource) Name() string                  { return m.name }
func (m *mockSource) SupportedAssetTypes() []string { return []string{"STOCK_BR", "STOCK_US"} }
func (m *mockSource) GetDividends(ctx context.Context, ticker, assetType string) ([]Event, error) {
	return m.events, m.err
}

func TestGateway_GetDividends(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	cum := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	pay := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)

	t.Run("NewDefaultGateway", func(t *testing.T) {
		gw := NewDefaultGateway(rdb, time.Hour)
		assert.NotNil(t, gw)
	})

	t.Run("Unknown AssetType", func(t *testing.T) {
		gw := NewDefaultGateway(rdb, time.Hour)
		_, err := gw.GetDividends(context.Background(), "BTC", "CRYPTO")
		assert.ErrorContains(t, err, "tipo de ativo desconhecido")
	})

	t.Run("Primary Success and Cache Hit", func(t *testing.T) {
		i10 := &mockSource{
			name:   "investidor10",
			events: []Event{{Date: cum, PaymentDate: pay, Amount: 1.5, Type: "DIVIDENDOS"}},
		}
		gw := NewGateway(i10, nil, nil, nil, rdb, time.Hour)

		evs, err := gw.GetDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, evs, 1)
		assert.Equal(t, "Dividendo", evs[0].Type)

		// Second call hits Redis cache
		evsCached, err := gw.GetDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, evsCached, 1)
	})

	t.Run("Primary Fails, Nil Skipped, Fallback Succeeds", func(t *testing.T) {
		i10 := &mockSource{name: "investidor10", err: errors.New("i10 down")}
		fund := &mockSource{
			name:   "fundamentus",
			events: []Event{{Date: cum, PaymentDate: pay, Amount: 0.85, Type: "JCP"}},
		}
		gw := NewGateway(i10, nil, fund, nil, nil, time.Hour)

		evs, err := gw.GetDividends(context.Background(), "VALE3", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, evs, 1)
		assert.Equal(t, "JCP", evs[0].Type)
	})

	t.Run("All Sources Fail", func(t *testing.T) {
		i10 := &mockSource{name: "investidor10", err: errors.New("i10 down")}
		sa := &mockSource{name: "stockanalysis", err: errors.New("sa down")}
		gw := NewGateway(i10, nil, nil, sa, nil, time.Hour)

		_, err := gw.GetDividends(context.Background(), "AAPL", "STOCK_US")
		assert.ErrorContains(t, err, "nenhum provento encontrado")
	})
}

func TestInvestidor10Source_And_Client(t *testing.T) {
	htmlTable := `
	<html><body>
	<table id="table-dividends-history">
		<tbody>
			<tr><td>JCP</td><td>10/02/2026</td><td>25/02/2026</td><td>1,2500</td></tr>
			<tr><td>Dividendos</td><td>invalid</td><td>25/02/2026</td><td>1,00</td></tr>
			<tr><td>Dividendos</td><td>10/02/2026</td><td>25/02/2026</td><td>invalid</td></tr>
			<tr><td>Dividendos</td><td>10/02/2026</td><td>-</td><td>0,50</td></tr>
			<tr><td>short</td></tr>
		</tbody>
	</table>
	</body></html>`

	t.Run("All AssetType Categories and Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(htmlTable))
		}))
		defer server.Close()

		c := NewInvestidor10Client().WithBaseURL(server.URL).WithHTTPClient(server.Client())
		src := NewInvestidor10DividendSource(c)
		assert.Equal(t, "investidor10", src.Name())
		assert.Contains(t, src.SupportedAssetTypes(), "STOCK_US")

		for _, at := range []string{"STOCK_BR", "ETF_BR", "FII", "BDR", "STOCK_US", "ETF_US"} {
			evs, err := src.GetDividends(context.Background(), "TEST.SA", at)
			assert.NoError(t, err)
			assert.Len(t, evs, 2)
			assert.False(t, evs[0].PaymentDate.IsZero())
			assert.True(t, evs[1].PaymentDate.IsZero())
		}
	})

	t.Run("Empty Ticker, Empty Table, Status Error, Request Error, Network Error", func(t *testing.T) {
		c := NewInvestidor10Client()
		src := NewInvestidor10DividendSource(c)

		_, err := src.GetDividends(context.Background(), "  .SA ", "STOCK_BR")
		assert.Error(t, err)

		// Empty table returns error after trying categories
		emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><table id="table-dividends-history"><tbody></tbody></table></body></html>`))
		}))
		defer emptyServer.Close()

		cEmpty := NewInvestidor10Client().WithBaseURL(emptyServer.URL)
		_, err = cEmpty.FetchDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.ErrorContains(t, err, "nenhum provento encontrado")

		// HTTP 404
		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer errServer.Close()

		c404 := NewInvestidor10Client().WithBaseURL(errServer.URL)
		_, err = c404.FetchDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.ErrorContains(t, err, "retornou status 404")

		// Invalid URL
		cBadURL := NewInvestidor10Client().WithBaseURL("http://invalid \x7f url")
		_, err = cBadURL.FetchDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.Error(t, err)

		// Network error
		cNetErr := NewInvestidor10Client().WithBaseURL("http://localhost:1")
		cNetErr.httpClient.Transport = &errorTransport{err: errors.New("net err")}
		_, err = cNetErr.FetchDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.Error(t, err)

		// Reader error for goquery
		cReadErr := NewInvestidor10Client().WithBaseURL("http://example.com")
		cReadErr.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(&errReader{err: errors.New("read err")}),
			}
		})
		_, err = cReadErr.FetchDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.Error(t, err)
	})
}

func TestFundamentusSource_And_Client(t *testing.T) {
	t.Run("Acao and FII Layouts", func(t *testing.T) {
		c := NewFundamentusClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.String(), "papel=HGLG11") && !strings.Contains(req.URL.String(), "fii_proventos") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`<table><tbody></tbody></table>`)),
				}
			}
			if strings.Contains(req.URL.String(), "fii_proventos") {
				html := `<table id="resultado"><tbody>
					<tr><td>15/01/2026</td><td>RENDIMENTO</td><td>25/01/2026</td><td>1,10</td></tr>
				</tbody></table>`
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(html))}
			}
			html := `<table id="resultado"><tbody>
				<tr><td>invalid</td><td>1,00</td><td>DIVIDENDO</td><td>01/01/2026</td></tr>
				<tr><td>10/01/2026</td><td>invalid</td><td>DIVIDENDO</td><td>01/01/2026</td></tr>
				<tr><td>10/01/2026</td><td>1,50</td><td>JRS CAP PROPRIO</td><td>15/01/2026</td></tr>
				<tr><td>10/01/2026</td><td>0,80</td><td>AMORTIZACAO</td><td>-</td></tr>
			</tbody></table>`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(html))}
		})

		src := NewFundamentusDividendSource(c)
		assert.Equal(t, "fundamentus", src.Name())
		assert.Contains(t, src.SupportedAssetTypes(), "STOCK_BR")

		evs, err := src.GetDividends(context.Background(), "PETR4.SA", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, evs, 2)

		evsFii, err := src.GetDividends(context.Background(), "HGLG11.SA", "FII")
		assert.NoError(t, err)
		assert.Len(t, evsFii, 1)
	})

	t.Run("Error Branches", func(t *testing.T) {
		c := NewFundamentusClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}
		})
		src := NewFundamentusDividendSource(c)
		_, err := src.GetDividends(context.Background(), "ERR", "STOCK_BR")
		assert.Error(t, err)

		_, err = c.fetchAndScrape(context.Background(), "http://invalid \x7f url", "acao")
		assert.Error(t, err)

		c.httpClient.Transport = &errorTransport{err: errors.New("net err")}
		_, err = c.fetchAndScrape(context.Background(), "http://example.com", "acao")
		assert.Error(t, err)

		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(&errReader{err: errors.New("read err")})}
		})
		_, err = c.fetchAndScrape(context.Background(), "http://example.com", "acao")
		assert.Error(t, err)
	})
}

func TestB3Source_And_Client(t *testing.T) {
	c := NewB3Client()
	c.rateLimiter = rate.NewLimiter(rate.Inf, 100)
	c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.String(), "GetListedCompanies") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results": [{"tradingName": "PETROBRAS"}]}`))}
		}
		if strings.Contains(req.URL.String(), "GetListedFundDividends") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results": [
				{"corporateAction": "RENDIMENTO", "valueCash": "0,75", "lastDatePriorEx": "15/05/2026", "paymentDate": "22/05/2026"}
			]}`))}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results": [
			{"corporateAction": "DIVIDENDO", "valueCash": "2,50", "lastDatePriorEx": "invalid", "paymentDate": "15/03/2026"},
			{"corporateAction": "DIVIDENDO", "valueCash": "invalid", "lastDatePriorEx": "12/02/2026", "paymentDate": "15/03/2026"},
			{"corporateAction": "DIVIDENDO", "valueCash": "2,50", "lastDatePriorEx": "12/02/2026", "paymentDate": "15/03/2026"},
			{"corporateAction": "JRS CAP PROPRIO", "valueCash": "1,00", "lastDatePriorEx": "12/02/2026", "paymentDate": ""}
		]}`))}
	})

	src := NewB3DividendSource(c)
	assert.Equal(t, "b3", src.Name())
	assert.Contains(t, src.SupportedAssetTypes(), "STOCK_BR")

	evs, err := src.GetDividends(context.Background(), "PETR4.SA", "STOCK_BR")
	assert.NoError(t, err)
	assert.Len(t, evs, 2)

	// Cache hit on resolveTradingName
	evs2, err := src.GetDividends(context.Background(), "PETR4.SA", "STOCK_BR")
	assert.NoError(t, err)
	assert.Len(t, evs2, 2)

	evsFii, err := src.GetDividends(context.Background(), "MXRF11.SA", "FII")
	assert.NoError(t, err)
	assert.Len(t, evsFii, 1)

	t.Run("B3 Error Branches", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := c.fetchBase(ctx, "http://example.com", map[string]string{"k": "v"})
		assert.Error(t, err)

		_, err = c.fetchBase(context.Background(), "http://example.com", math.NaN())
		assert.Error(t, err)

		_, err = c.fetchBase(context.Background(), "http://invalid \x7f url", map[string]string{"k": "v"})
		assert.Error(t, err)

		cNet := NewB3Client()
		cNet.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		cNet.httpClient.Transport = &errorTransport{err: errors.New("net err")}
		_, err = cNet.fetchBase(context.Background(), "http://example.com", map[string]string{"k": "v"})
		assert.Error(t, err)

		c500 := NewB3Client()
		c500.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		c500.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}
		})
		s500 := NewB3DividendSource(c500)
		_, err = s500.GetDividends(context.Background(), "VALE3", "STOCK_BR")
		assert.Error(t, err)
		s500.tickerNameCache["MXRF11"] = "MXRF11"
		_, err = s500.GetDividends(context.Background(), "MXRF11", "FII")
		assert.Error(t, err)
		s500.tickerNameCache["VALE3"] = "VALE"
		_, err = s500.GetDividends(context.Background(), "VALE3", "STOCK_BR")
		assert.Error(t, err)
		s500.tickerNameCache["MXRF11"] = "MXRF11"
		_, err = s500.GetDividends(context.Background(), "MXRF11", "FII")
		assert.Error(t, err)

		cBadJSON := NewB3Client()
		cBadJSON.rateLimiter = rate.NewLimiter(rate.Inf, 100)
		cBadJSON.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("invalid"))}
		})
		_, err = cBadJSON.FetchCashDividends(context.Background(), "PETR")
		assert.Error(t, err)
		_, err = cBadJSON.FetchFundDividends(context.Background(), "MXRF")
		assert.Error(t, err)
		_, err = cBadJSON.FetchCompanies(context.Background())
		assert.Error(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = src.resolveTradingName(context.Background(), "PETR4")
		}()
		wg.Wait()
	})
}

func TestStockAnalysisSource_And_Client(t *testing.T) {
	c := NewStockAnalysisClient()
	c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		html := `<table><tbody>
			<tr><td>short</td></tr>
			<tr><td>invalid</td><td>$0.27</td><td>Nov 12, 2026</td><td>Nov 16, 2026</td></tr>
			<tr><td>Nov 12, 2026</td><td>invalid</td><td>Nov 12, 2026</td><td>Nov 16, 2026</td></tr>
			<tr><td>Aug 10, 2026</td><td>$0.27</td><td>Aug 10, 2026</td><td>Aug 13, 2026</td></tr>
			<tr><td>Nov 12, 2026</td><td>$0.27</td><td>-</td><td>-</td></tr>
		</tbody></table>`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(html))}
	})

	src := NewStockAnalysisDividendSource(c)
	assert.Equal(t, "stockanalysis", src.Name())
	assert.Contains(t, src.SupportedAssetTypes(), "STOCK_US")

	evs, err := src.GetDividends(context.Background(), "AAPL", "STOCK_US")
	assert.NoError(t, err)
	assert.Len(t, evs, 2)
	// Aug 10, 2026 (Monday) -> previous business day is Aug 7, 2026 (Friday)
	assert.Equal(t, time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), evs[0].Date)
	assert.Equal(t, time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), evs[0].PaymentDate)
	assert.True(t, evs[1].PaymentDate.IsZero())

	evsEtf, err := src.GetDividends(context.Background(), "VOO", "ETF_US")
	assert.NoError(t, err)
	assert.Len(t, evsEtf, 2)

	t.Run("StockAnalysis Error Branches", func(t *testing.T) {
		cErr := NewStockAnalysisClient()
		sErr := NewStockAnalysisDividendSource(cErr)

		_, err := cErr.FetchDividends(nil, "AAPL", "STOCK_US")
		assert.Error(t, err)

		cErr.httpClient.Transport = &errorTransport{err: errors.New("net err")}
		_, err = sErr.GetDividends(context.Background(), "AAPL", "STOCK_US")
		assert.Error(t, err)

		cErr.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))}
		})
		_, err = sErr.GetDividends(context.Background(), "AAPL", "STOCK_US")
		assert.Error(t, err)

		cErr.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(&errReader{err: errors.New("read err")})}
		})
		_, err = sErr.GetDividends(context.Background(), "AAPL", "STOCK_US")
		assert.Error(t, err)
	})
}

func TestGateway_CacheHitCorruptedJSON(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	_ = mr.Set("dividends:PETR4", "not-json")

	cum := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	pay := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	b, _ := json.Marshal([]Event{{Date: cum, PaymentDate: pay, Amount: 1.0, Type: "Dividendo"}})
	assert.NotEmpty(t, b)

	i10 := &mockSource{
		name:   "investidor10",
		events: []Event{{Date: cum, PaymentDate: pay, Amount: 1.0, Type: "Dividendo"}},
	}
	gw := NewGateway(i10, nil, nil, nil, rdb, time.Hour)
	evs, err := gw.GetDividends(context.Background(), "PETR4", "STOCK_BR")
	assert.NoError(t, err)
	assert.Len(t, evs, 1)
}
