package market

import (
    "context"
    "testing"
)

func TestScrapeFundamentus(t *testing.T) {
    s := NewScraper()
    fund, err := s.ScrapeFundamentus(context.Background(), "PETR4")
    if err != nil {
        t.Fatalf("Scrape error: %v", err)
    }
    t.Logf("Fundamentus: %+v", fund)
}

func TestScrapeFinviz(t *testing.T) {
    s := NewScraper()
    fund, err := s.ScrapeFinviz(context.Background(), "AAPL")
    if err != nil {
        t.Fatalf("Scrape error: %v", err)
    }
    t.Logf("Finviz: %+v", fund)
}
