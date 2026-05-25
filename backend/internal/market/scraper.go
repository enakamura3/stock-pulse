package market

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Fundamentals struct {
	Symbol        string
	EPS           float64
	BookValue     float64
	DividendYield float64
	GrahamValue   float64
	BazinValue    float64
}

type Scraper struct {
	client *http.Client
}

func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *Scraper) GetFundamentals(ctx context.Context, symbol string) (*Fundamentals, error) {
	if strings.HasSuffix(symbol, ".SA") {
		cleanSymbol := strings.TrimSuffix(symbol, ".SA")
		fund, err := s.ScrapeFundamentus(ctx, cleanSymbol)
		if err != nil {
			return nil, err
		}
		fund.Symbol = symbol
		return fund, nil
	}

	fund, err := s.ScrapeFinviz(ctx, symbol)
	if err != nil {
		return nil, err
	}
	fund.Symbol = symbol
	return fund, nil
}

func (s *Scraper) ScrapeFundamentus(ctx context.Context, symbol string) (*Fundamentals, error) {
	url := fmt.Sprintf("https://www.fundamentus.com.br/detalhes.php?papel=%s", symbol)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	lpaRe := regexp.MustCompile(`(?is)<span[^>]*>LPA</span>.*?<td[^>]*><span[^>]*>([^<]+)</span>`)
	vpaRe := regexp.MustCompile(`(?is)<span[^>]*>VPA</span>.*?<td[^>]*><span[^>]*>([^<]+)</span>`)
	divRe := regexp.MustCompile(`(?is)<span[^>]*>Div\. Yield</span>.*?<td[^>]*><span[^>]*>([^<]+)</span>`)

	eps := parseBrFloat(extractRegex(lpaRe, html))
	bv := parseBrFloat(extractRegex(vpaRe, html))
	dy := parseBrFloat(extractRegex(divRe, html))

	return s.calculateFormulas(&Fundamentals{
		EPS:           eps,
		BookValue:     bv,
		DividendYield: dy,
	}), nil
}

func (s *Scraper) ScrapeFinviz(ctx context.Context, symbol string) (*Fundamentals, error) {
	url := fmt.Sprintf("https://finviz.com/quote.ashx?t=%s", symbol)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	epsRe := regexp.MustCompile(`(?is)EPS \(ttm\)</div></td>.*?<b>([^<]+)</b>`)
	bvRe := regexp.MustCompile(`(?is)Book/sh</div></td>.*?<b>([^<]+)</b>`)
	divRe := regexp.MustCompile(`(?is)Dividend %</div></td>.*?<b>([^<]+)</b>`)

	eps := parseUsFloat(extractRegex(epsRe, html))
	bv := parseUsFloat(extractRegex(bvRe, html))
	dy := parseUsFloat(extractRegex(divRe, html))

	return s.calculateFormulas(&Fundamentals{
		EPS:           eps,
		BookValue:     bv,
		DividendYield: dy,
	}), nil
}

func extractRegex(re *regexp.Regexp, text string) string {
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func parseBrFloat(val string) float64 {
	val = strings.ReplaceAll(val, "%", "")
	val = strings.ReplaceAll(val, ".", "")
	val = strings.ReplaceAll(val, ",", ".")
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

func parseUsFloat(val string) float64 {
	val = strings.ReplaceAll(val, "%", "")
	val = strings.ReplaceAll(val, ",", "")
	if val == "-" {
		return 0
	}
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

// calculateFormulas applies Graham and Bazin formulas
func (s *Scraper) calculateFormulas(fund *Fundamentals) *Fundamentals {
	// Graham: sqrt(22.5 * EPS * VPA)
	if fund.EPS > 0 && fund.BookValue > 0 {
		fund.GrahamValue = math.Sqrt(22.5 * fund.EPS * fund.BookValue)
	} else {
		fund.GrahamValue = 0 // Invalid for negative earnings
	}

	// Bazin: Average Dividend / 0.06
	// The DivYield scraped is in %, e.g., 7.1 meaning 7.1%
	// If we have DivYield and we can estimate the absolute dividend paid:
	// Wait, Fundamentus and Finviz give Dividend Yield (%).
	// Yield = (Div / Price). We don't have Price here immediately unless we fetch it.
	// Actually, Bazin's Ceiling Price is usually calculated based on the historical absolute dividend.
	// We can leave BazinValue as 0 or calculate it in the service if we know the price.
	return fund
}
