package market

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScrapeFundamentus(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
		<td class="label w2"><span class="txt">LPA</span></td>
		<td class="data w2"><span class="txt">8,35</span></td>
		<td class="label w2"><span class="txt">VPA</span></td>
		<td class="data w2"><span class="txt">34,54</span></td>
		<td class="label"><span class="txt">Div. Yield</span></td>
		<td class="data"><span class="txt">7,1%</span></td>
		`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer mockServer.Close()

	s := NewScraper()
	s.fundamentusBaseURL = mockServer.URL

	fund, err := s.ScrapeFundamentus(context.Background(), "PETR4")
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	if math.Abs(fund.EPS-8.35) > 1e-6 {
		t.Errorf("Expected EPS 8.35, got %f", fund.EPS)
	}
	if math.Abs(fund.BookValue-34.54) > 1e-6 {
		t.Errorf("Expected BookValue 34.54, got %f", fund.BookValue)
	}
	if math.Abs(fund.DividendYield-7.1) > 1e-6 {
		t.Errorf("Expected DivYield 7.1, got %f", fund.DividendYield)
	}
}

func TestScrapeFinviz(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
		<td class="snapshot-td2"><div class="snapshot-td-label">EPS (ttm)</div></td>
		<td class="snapshot-td2"><div><b>8.27</b></div></td>
		<td class="snapshot-td2"><div class="snapshot-td-label">Book/sh</div></td>
		<td class="snapshot-td2"><div><b>7.26</b></div></td>
		<td class="snapshot-td2"><div class="snapshot-td-label">Dividend %</div></td>
		<td class="snapshot-td2"><div><b>0.45%</b></div></td>
		`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer mockServer.Close()

	s := NewScraper()
	s.finvizBaseURL = mockServer.URL

	fund, err := s.ScrapeFinviz(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	if math.Abs(fund.EPS-8.27) > 1e-6 {
		t.Errorf("Expected EPS 8.27, got %f", fund.EPS)
	}
	if math.Abs(fund.BookValue-7.26) > 1e-6 {
		t.Errorf("Expected BookValue 7.26, got %f", fund.BookValue)
	}
	if math.Abs(fund.DividendYield-0.45) > 1e-6 {
		t.Errorf("Expected DivYield 0.45, got %f", fund.DividendYield)
	}
}

func TestScrape_URLEscape(t *testing.T) {
	var requestedRawQuery string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedRawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html></html>"))
	}))
	defer mockServer.Close()

	s := NewScraper()
	s.fundamentusBaseURL = mockServer.URL
	s.finvizBaseURL = mockServer.URL

	_, _ = s.ScrapeFundamentus(context.Background(), "TEST&INJECT=1")
	if requestedRawQuery != "papel=TEST%26INJECT%3D1" {
		t.Errorf("Expected escaped query, got %s", requestedRawQuery)
	}

	_, _ = s.ScrapeFinviz(context.Background(), "BRK/B")
	if requestedRawQuery != "t=BRK%2FB" {
		t.Errorf("Expected escaped query, got %s", requestedRawQuery)
	}
}

func TestGetFundamentals(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("papel") == "PETR4" {
			html := `
			<td class="label w2"><span class="txt">LPA</span></td>
			<td class="data w2"><span class="txt">8,35</span></td>
			<td class="label w2"><span class="txt">VPA</span></td>
			<td class="data w2"><span class="txt">34,54</span></td>
			<td class="label"><span class="txt">Div. Yield</span></td>
			<td class="data"><span class="txt">7,1%</span></td>
			`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
			return
		}
		if r.URL.Query().Get("t") == "AAPL" {
			html := `
			<td class="snapshot-td2"><div class="snapshot-td-label">EPS (ttm)</div></td>
			<td class="snapshot-td2"><div><b>8.27</b></div></td>
			<td class="snapshot-td2"><div class="snapshot-td-label">Book/sh</div></td>
			<td class="snapshot-td2"><div><b>7.26</b></div></td>
			<td class="snapshot-td2"><div class="snapshot-td-label">Dividend %</div></td>
			<td class="snapshot-td2"><div><b>-</b></div></td>
			`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	s := NewScraper()
	s.fundamentusBaseURL = mockServer.URL
	s.finvizBaseURL = mockServer.URL

	// 1. Success with .SA
	fundSA, err := s.GetFundamentals(context.Background(), "PETR4.SA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fundSA.Symbol != "PETR4.SA" {
		t.Errorf("expected PETR4.SA, got %s", fundSA.Symbol)
	}
	if math.Abs(fundSA.GrahamValue-0) < 1e-6 {
		t.Errorf("expected GrahamValue > 0, got %f", fundSA.GrahamValue)
	}

	// 2. Error with .SA
	_, err = s.GetFundamentals(context.Background(), "FAIL.SA")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// 3. Success without .SA
	fundUS, err := s.GetFundamentals(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fundUS.Symbol != "AAPL" {
		t.Errorf("expected AAPL, got %s", fundUS.Symbol)
	}
	if math.Abs(fundUS.DividendYield-0.0) > 1e-6 {
		t.Errorf("expected 0 for '-' yield, got %f", fundUS.DividendYield)
	}

	// 4. Error without .SA
	_, err = s.GetFundamentals(context.Background(), "FAIL")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	// 5. Negative EPS Graham value = 0
	fNeg := s.calculateFormulas(&Fundamentals{EPS: -1, BookValue: 10})
	if math.Abs(fNeg.GrahamValue-0) > 1e-6 {
		t.Errorf("expected GrahamValue 0 for negative EPS, got %f", fNeg.GrahamValue)
	}
}
