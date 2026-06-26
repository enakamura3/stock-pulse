package fixedincome

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// brasilAPIHoliday representa um feriado retornado pela BrasilAPI.
type brasilAPIHoliday struct {
	Date string `json:"date"` // YYYY-MM-DD
	Name string `json:"name"`
	Type string `json:"type"` // "national" | "municipal" etc.
}

// AnbimaClient busca feriados nacionais via BrasilAPI.
type AnbimaClient interface {
	FetchHolidays(ctx context.Context, year int) ([]brasilAPIHoliday, error)
}

type anbimaClient struct {
	httpClient *http.Client
}

func NewAnbimaClient() AnbimaClient {
	return &anbimaClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *anbimaClient) FetchHolidays(ctx context.Context, year int) ([]brasilAPIHoliday, error) {
	url := fmt.Sprintf("https://brasilapi.com.br/api/feriados/v1/%d", year)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("anbimaClient: failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anbimaClient: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anbimaClient: BrasilAPI returned status %d for year %d", resp.StatusCode, year)
	}

	var holidays []brasilAPIHoliday
	if err := json.NewDecoder(resp.Body).Decode(&holidays); err != nil {
		return nil, fmt.Errorf("anbimaClient: failed to decode response: %w", err)
	}

	return holidays, nil
}
