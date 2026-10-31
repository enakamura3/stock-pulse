package market

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/calculator"
)

var (
	fundamentusLpaRe    = regexp.MustCompile(`(?is)<span[^>]*>LPA</span>.*?<td[^>]*><span[^>]*>([^<]+)</span>`)
	fundamentusVpaRe    = regexp.MustCompile(`(?is)<span[^>]*>VPA</span>.*?<td[^>]*><span[^>]*>([^<]+)</span>`)
	fundamentusVpCotaRe = regexp.MustCompile(`(?is)<span[^>]*>VP/Cota</span>.*?<td[^>]*><span[^>]*>([^<]+)</span>`)
	fundamentusDivRe    = regexp.MustCompile(`(?is)<span[^>]*>Div\. Yield</span>.*?<td[^>]*><span[^>]*>([^<]+)</span>`)

	finvizEpsRe = regexp.MustCompile(`(?is)EPS \(ttm\)</div></td>.*?<b>([^<]+)</b>`)
	finvizBvRe  = regexp.MustCompile(`(?is)Book/sh</div></td>.*?<b>([^<]+)</b>`)
	finvizDivRe = regexp.MustCompile(`(?is)Dividend %</div></td>.*?<b>([^<]+)</b>`)
)

type Fundamentals struct {
	Ticker        string
	EPS           float64
	BookValue     float64
	DividendYield float64
	GrahamValue   float64
	BazinValue    float64
}

type FundamentalsScraper interface {
	GetFundamentals(ctx context.Context, ticker string) (*Fundamentals, error)
}

type Scraper struct {
	client             *http.Client
	fundamentusBaseURL string
	finvizBaseURL      string
}

func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		fundamentusBaseURL: "https://www.fundamentus.com.br",
		finvizBaseURL:      "https://finviz.com",
	}
}

func (s *Scraper) GetFundamentals(ctx context.Context, ticker string) (*Fundamentals, error) {
	if strings.HasSuffix(ticker, ".SA") {
		cleanTicker := strings.TrimSuffix(ticker, ".SA")
		fund, err := s.ScrapeFundamentus(ctx, cleanTicker)
		if err != nil {
			return nil, err
		}
		fund.Ticker = ticker
		return fund, nil
	}

	fund, err := s.ScrapeFinviz(ctx, ticker)
	if err != nil {
		return nil, err
	}
	fund.Ticker = ticker
	return fund, nil
}

func (s *Scraper) ScrapeFundamentus(ctx context.Context, ticker string) (*Fundamentals, error) {
	url := fmt.Sprintf("%s/detalhes.php?papel=%s", s.fundamentusBaseURL, url.QueryEscape(ticker))
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

	eps := parseBrFloat(extractRegex(fundamentusLpaRe, html))
	bv := parseBrFloat(extractRegex(fundamentusVpaRe, html))
	if calculator.FloatIsZero(bv) {
		bv = parseBrFloat(extractRegex(fundamentusVpCotaRe, html))
	}
	dy := parseBrFloat(extractRegex(fundamentusDivRe, html))

	return s.calculateFormulas(&Fundamentals{
		EPS:           eps,
		BookValue:     bv,
		DividendYield: dy,
	}), nil
}

func (s *Scraper) ScrapeFinviz(ctx context.Context, ticker string) (*Fundamentals, error) {
	url := fmt.Sprintf("%s/quote.ashx?t=%s", s.finvizBaseURL, url.QueryEscape(ticker))
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

	eps := parseUsFloat(extractRegex(finvizEpsRe, html))
	bv := parseUsFloat(extractRegex(finvizBvRe, html))
	dy := parseUsFloat(extractRegex(finvizDivRe, html))

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
