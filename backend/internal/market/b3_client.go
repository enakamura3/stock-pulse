package market

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type B3RawDividend struct {
	CorporateAction string `json:"corporateAction"` // "DIVIDENDO", "JRS CAP PROPRIO"
	ValueCash       string `json:"valueCash"`       // "2,5499"
	LastDatePriorEx string `json:"lastDatePriorEx"` // "12/02/2026"
	PaymentDate     string `json:"paymentDate"`     // "15/03/2026"
	Rate            string `json:"rate"`            // "100"
}

type B3RawCompany struct {
	TradingName string `json:"tradingName"` // "BB SEGURIDADE"
}

type B3Client struct {
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

func NewB3Client() *B3Client {
	return &B3Client{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		rateLimiter: rate.NewLimiter(rate.Every(2*time.Second), 1),
	}
}

// fetchBase is a helper to encode payload to base64 and make the request
func (c *B3Client) fetchBase(ctx context.Context, urlPrefix string, payload interface{}) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	base64Payload := base64.StdEncoding.EncodeToString(payloadBytes)
	url := fmt.Sprintf("%s%s", urlPrefix, base64Payload)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("b3 api error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *B3Client) FetchCashDividends(ctx context.Context, tradingName string) ([]B3RawDividend, error) {
	payload := map[string]string{
		"language":    "pt-br",
		"pageNumber":  "1",
		"pageSize":    "100",
		"tradingName": tradingName,
	}

	urlPrefix := "https://sistemaswebb3-listados.b3.com.br/listedCompaniesPage/api/GetListedCashDividends/"
	body, err := c.fetchBase(ctx, urlPrefix, payload)
	if err != nil {
		return nil, err
	}

	var response struct {
		Results []B3RawDividend `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (c *B3Client) FetchFundDividends(ctx context.Context, tradingName string) ([]B3RawDividend, error) {
	payload := map[string]string{
		"language":    "pt-br",
		"pageNumber":  "1",
		"pageSize":    "100",
		"tradingName": tradingName,
	}

	// Wait, we need to know the correct endpoint. The plan says: "GetListedFundDividends"
	urlPrefix := "https://sistemaswebb3-listados.b3.com.br/listedFundsPage/api/GetListedFundDividends/"
	body, err := c.fetchBase(ctx, urlPrefix, payload)
	if err != nil {
		return nil, err
	}

	var response struct {
		Results []B3RawDividend `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (c *B3Client) FetchCompanies(ctx context.Context) ([]B3RawCompany, error) {
	// Let's assume there is an endpoint for searching companies, e.g., GetListedCompanies or similar.
	// Actually, we just need the list. Usually it's in an endpoint like /listedCompaniesPage/api/GetListedCompanies/
	payload := map[string]string{
		"language":   "pt-br",
		"pageNumber": "1",
		"pageSize":   "1000",
	}

	urlPrefix := "https://sistemaswebb3-listados.b3.com.br/listedCompaniesPage/api/GetListedCompanies/"
	body, err := c.fetchBase(ctx, urlPrefix, payload)
	if err != nil {
		return nil, err
	}

	var response struct {
		Results []B3RawCompany `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}
