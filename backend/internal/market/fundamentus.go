package market

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
)

type FundamentusScraper struct {
	client *http.Client
}

func NewFundamentusScraper() *FundamentusScraper {
	return &FundamentusScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *FundamentusScraper) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	symbol := strings.TrimSuffix(ticker, ".SA")

	// Tenta como Ação primeiro
	events, err := s.scrapeTable(ctx, fmt.Sprintf("https://www.fundamentus.com.br/proventos.php?papel=%s&tipo=2", symbol), "acao")
	if err == nil && len(events) > 0 {
		return events, nil
	}

	// Se retornou vazio (ou deu erro na raspagem), tenta como FII
	eventsFii, errFii := s.scrapeTable(ctx, fmt.Sprintf("https://www.fundamentus.com.br/fii_proventos.php?papel=%s&tipo=2", symbol), "fii")
	if errFii == nil && len(eventsFii) > 0 {
		return eventsFii, nil
	}

	// Se ambos falharem ou retornarem vazio, retorna erro para acionar o fallback do Yahoo
	return nil, fmt.Errorf("nenhum provento encontrado no fundamentus para %s", ticker)
}

func (s *FundamentusScraper) scrapeTable(ctx context.Context, url string, assetType string) ([]DividendEvent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
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

	var events []DividendEvent
	layout := "02/01/2006"

	doc.Find("table#resultado tbody tr, table tbody tr").Each(func(i int, sel *goquery.Selection) {
		tds := sel.Find("td")
		if tds.Length() >= 4 {
			var dateStr, amountStr, paymentDateStr, tipoStr string

			if assetType == "acao" {
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

			exDate, err := time.Parse(layout, dateStr)
			if err != nil {
				return
			}

			amountStr = strings.ReplaceAll(amountStr, ".", "")
			amountStr = strings.ReplaceAll(amountStr, ",", ".")
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				return
			}

			var paymentDate time.Time
			if paymentDateStr != "-" && paymentDateStr != "" {
				pd, err := time.Parse(layout, paymentDateStr)
				if err == nil {
					paymentDate = pd
				}
			}

			// Formatar o Tipo para um padrão limpo
			tipoUpper := strings.ToUpper(tipoStr)
			cleanType := "Dividendo"
			if strings.Contains(tipoUpper, "JRS CAP PROPRIO") || strings.Contains(tipoUpper, "JCP") {
				cleanType = "JCP"
			} else if strings.Contains(tipoUpper, "RENDIMENTO") {
				cleanType = "Rendimento"
			} else if strings.Contains(tipoUpper, "AMORTIZACAO") {
				cleanType = "Amortização"
			}

			events = append(events, DividendEvent{
				Date:        exDate,
				PaymentDate: paymentDate,
				Amount:      amount,
				Type:        cleanType,
			})
		}
	})

	// Deduplicar eventos baseados na combinação exata pedida:
	// Data de Pagamento, Tipo, Valor, e Mês/Ano da Data Com
	deduped := make([]DividendEvent, 0, len(events))
	seen := make(map[string]bool)

	for _, ev := range events {
		var key string
		if assetType == "fii" {
			// Regra FII: Apenas 1 pagamento por mês garantido.
			// Ignoramos completamente o valor e a data de pagamento na hora de agrupar.
			key = fmt.Sprintf("fii|%s|%02d|%d", ev.Type, ev.Date.Month(), ev.Date.Year())
		} else {
			// Regra Ações: Podem ter múltiplos pagamentos no mesmo mês (TAEE11).
			// Usamos o valor para diferenciar pagamentos no mesmo mês, mas ignoramos a data de pagamento.
			key = fmt.Sprintf("acao|%s|%.6f|%02d|%d", ev.Type, ev.Amount, ev.Date.Month(), ev.Date.Year())
		}
		
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, ev)
		}
	}

	return deduped, nil
}
