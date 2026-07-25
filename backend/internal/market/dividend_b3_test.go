package market

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestB3DividendSource_GetDividends(t *testing.T) {
	client := NewB3Client()
	client.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		jsonResp := `{"results": [
			{
				"corporateAction": "DIVIDENDO",
				"valueCash": "2,5499",
				"lastDatePriorEx": "12/02/2026",
				"paymentDate": "15/03/2026"
			},
			{
				"corporateAction": "JRS CAP PROPRIO",
				"valueCash": "1,50",
				"lastDatePriorEx": "01/01/2026",
				"paymentDate": "10/01/2026"
			}
		]}`
		
		if strings.Contains(req.URL.String(), "GetListedFundDividends") {
			jsonResp = `{"results": [
				{
					"corporateAction": "RENDIMENTO",
					"valueCash": "0,75",
					"lastDatePriorEx": "15/05/2026",
					"paymentDate": "22/05/2026"
				}
			]}`
		} else if strings.Contains(req.URL.String(), "GetListedCompanies") {
			jsonResp = `{"results": [{"tradingName": "PETROBRAS"}]}`
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(jsonResp)),
		}
	})

	source := NewB3DividendSource(client)

	t.Run("Stock", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "PETR4.SA", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 2)
		
		assert.Equal(t, "Dividendo", events[0].Type)
		assert.Equal(t, 2.5499, events[0].Amount)
		assert.Equal(t, time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), events[0].Date)
		assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), events[0].PaymentDate)

		assert.Equal(t, "JCP", events[1].Type)
		assert.Equal(t, 1.5, events[1].Amount)
	})

	t.Run("FII", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "MXRF11.SA", "FII")
		assert.NoError(t, err)
		assert.Len(t, events, 1)

		assert.Equal(t, "Rendimento", events[0].Type)
		assert.Equal(t, 0.75, events[0].Amount)
	})

	t.Run("Name and SupportedTypes", func(t *testing.T) {
		assert.Equal(t, "b3", source.Name())
		assert.Contains(t, source.SupportedAssetTypes(), "STOCK_BR")
	})
	
	t.Run("Fetch Error", func(t *testing.T) {
		errClient := NewB3Client()
		errClient.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}
		})
		errSource := NewB3DividendSource(errClient)
		
		res, err := errSource.GetDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.Error(t, err)
		assert.Nil(t, res)
		
		res, err = errSource.GetDividends(context.Background(), "MXRF11", "FII")
		assert.Error(t, err)
		assert.Nil(t, res)
		
		_, err = errClient.FetchCompanies(context.Background())
		assert.Error(t, err)
	})
}
