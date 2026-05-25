package market

import (
	"context"
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

	if fund.EPS != 8.35 {
		t.Errorf("Expected EPS 8.35, got %f", fund.EPS)
	}
	if fund.BookValue != 34.54 {
		t.Errorf("Expected BookValue 34.54, got %f", fund.BookValue)
	}
	if fund.DividendYield != 7.1 {
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

	if fund.EPS != 8.27 {
		t.Errorf("Expected EPS 8.27, got %f", fund.EPS)
	}
	if fund.BookValue != 7.26 {
		t.Errorf("Expected BookValue 7.26, got %f", fund.BookValue)
	}
	if fund.DividendYield != 0.45 {
		t.Errorf("Expected DivYield 0.45, got %f", fund.DividendYield)
	}
}
