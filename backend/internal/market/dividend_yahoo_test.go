package market

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"


	"github.com/stretchr/testify/assert"
)

func TestYahooDividendSource_GetDividends(t *testing.T) {
	client := NewYahooClient()
	client.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		jsonResp := `{"chart": {"result": [{"events": {"dividends": {
			"1609459200": {"amount": 0.5, "date": 1609459200},
			"1625097600": {"amount": 1.5, "date": 1625097600}
		}}}]}}`
		
		if strings.Contains(req.URL.String(), "ERR") {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(jsonResp)),
		}
	})

	source := NewYahooDividendSource(client)

	t.Run("Name and SupportedTypes", func(t *testing.T) {
		assert.Equal(t, "yahoo", source.Name())
		assert.Contains(t, source.SupportedAssetTypes(), "STOCK_BR")
	})

	t.Run("Success", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "PETR4.SA", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 2)
		
		var found05, found15 bool
		for _, e := range events {
			if e.Amount == 0.5 {
				found05 = true
			}
			if e.Amount == 1.5 {
				found15 = true
			}
		}
		assert.True(t, found05)
		assert.True(t, found15)
	})

	t.Run("Appends .SA for Brazilian assets", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "VALE3", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 2)
	})

	t.Run("Error", func(t *testing.T) {
		res, err := source.GetDividends(context.Background(), "ERR", "STOCK_BR")
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
