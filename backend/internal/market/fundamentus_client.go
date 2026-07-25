package market

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
)

// FundamentusRawDividend contém os dados crus extraídos da tabela HTML do Fundamentus.
// Nenhum campo é convertido — todos são strings exatamente como aparecem no HTML.
type FundamentusRawDividend struct {
	Date        string // "12/02/2026" (DD/MM/YYYY)
	Amount      string // "1.002,5499" (ponto = milhar, vírgula = decimal)
	Type        string // "JRS CAP PROPRIO", "DIVIDENDO", "RENDIMENTO", "AMORTIZACAO"
	PaymentDate string // "15/03/2026" (DD/MM/YYYY)
}

type FundamentusClient struct {
	httpClient *http.Client
}

func NewFundamentusClient() *FundamentusClient {
	return &FundamentusClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchDividends busca a tabela de proventos do Fundamentus.
// Tenta a URL de ações primeiro; se não retornar dados, tenta a URL de FIIs.
// Retorna os dados crus e o layout detectado ("acao" ou "fii").
func (c *FundamentusClient) FetchDividends(ctx context.Context, ticker string) ([]FundamentusRawDividend, string, error) {
	symbol := strings.TrimSuffix(ticker, ".SA")

	// Tenta como Ação primeiro
	raw, err := c.fetchAndScrape(ctx, fmt.Sprintf("https://www.fundamentus.com.br/proventos.php?papel=%s&tipo=2", symbol), "acao")
	if err == nil && len(raw) > 0 {
		return raw, "acao", nil
	}

	// Se retornou vazio (ou deu erro na raspagem), tenta como FII
	rawFii, errFii := c.fetchAndScrape(ctx, fmt.Sprintf("https://www.fundamentus.com.br/fii_proventos.php?papel=%s&tipo=2", symbol), "fii")
	if errFii == nil && len(rawFii) > 0 {
		return rawFii, "fii", nil
	}

	return nil, "", fmt.Errorf("nenhum provento encontrado no fundamentus para %s", ticker)
}

func (c *FundamentusClient) fetchAndScrape(ctx context.Context, url string, layout string) ([]FundamentusRawDividend, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fundamentus retornou status %d", resp.StatusCode)
	}

	decoder := charmap.ISO8859_1.NewDecoder()
	body, err := io.ReadAll(decoder.Reader(resp.Body))
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var rawDividends []FundamentusRawDividend

	doc.Find("table#resultado tbody tr, table tbody tr").Each(func(i int, sel *goquery.Selection) {
		tds := sel.Find("td")
		if tds.Length() >= 4 {
			var dateStr, amountStr, paymentDateStr, tipoStr string

			if layout == "acao" {
				// Ação: Data (0), Valor (1), Tipo (2), Data de Pagamento (3)
				dateStr = strings.TrimSpace(tds.Eq(0).Text())
				amountStr = strings.TrimSpace(tds.Eq(1).Text())
				tipoStr = strings.TrimSpace(tds.Eq(2).Text())
				paymentDateStr = strings.TrimSpace(tds.Eq(3).Text())
			} else {
				// FII: Data (0), Tipo (1), Data de Pagamento (2), Valor (3)
				dateStr = strings.TrimSpace(tds.Eq(0).Text())
				tipoStr = strings.TrimSpace(tds.Eq(1).Text())
				paymentDateStr = strings.TrimSpace(tds.Eq(2).Text())
				amountStr = strings.TrimSpace(tds.Eq(3).Text())
			}

			rawDividends = append(rawDividends, FundamentusRawDividend{
				Date:        dateStr,
				Amount:      amountStr,
				Type:        tipoStr,
				PaymentDate: paymentDateStr,
			})
		}
	})

	return rawDividends, nil
}
