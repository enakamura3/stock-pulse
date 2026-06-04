package fixedincome

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type BCBClient interface {
	FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error)
}

type bcbClient struct {
	httpClient *http.Client
}

func NewBCBClient() BCBClient {
	return &bcbClient{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// bcbData representa o formato de resposta do SGS
type bcbData struct {
	Data  string `json:"data"`
	Valor string `json:"valor"`
}

var indexerToSGSCode = map[string]string{
	"CDI":   "12",
	"SELIC": "11",
	"IPCA":  "433",
}

func (c *bcbClient) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	code, ok := indexerToSGSCode[indexer]
	if !ok {
		return nil, fmt.Errorf("unsupported indexer: %s", indexer)
	}

	startStr := startDate.Format("02/01/2006")
	endStr := endDate.Format("02/01/2006")

	url := fmt.Sprintf("https://api.bcb.gov.br/dados/serie/bcdata.sgs.%s/dados?formato=json&dataInicial=%s&dataFinal=%s", code, startStr, endStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bcb api returned status %d", resp.StatusCode)
	}

	var data []bcbData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var rates []IndexRate
	for _, item := range data {
		parsedDate, err := time.Parse("02/01/2006", item.Data)
		if err != nil {
			continue
		}
		var parsedRate float64
		_, err = fmt.Sscanf(item.Valor, "%f", &parsedRate)
		if err != nil {
			continue
		}

		rates = append(rates, IndexRate{
			Indexer: indexer,
			Date:    parsedDate,
			Rate:    parsedRate,
		})
	}

	return rates, nil
}
