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

func TestStockAnalysisDividendSource_GetDividends(t *testing.T) {
	client := NewStockAnalysisClient()
	client.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		htmlResp := `
		<table>
			<tbody>
				<tr>
					<td>Feb 12, 2026</td>
					<td>$2.5499</td>
					<td>Feb 11, 2026</td>
					<td>Mar 15, 2026</td>
				</tr>
				<!-- Invalid ExDate -->
				<tr>
					<td>Invalid Date</td>
					<td>$1.00</td>
					<td>Invalid Date</td>
					<td>Mar 15, 2026</td>
				</tr>
				<!-- Invalid Amount -->
				<tr>
					<td>Feb 12, 2026</td>
					<td>$INVALID</td>
					<td>Feb 11, 2026</td>
					<td>Mar 15, 2026</td>
				</tr>
				<!-- Invalid PayDate -->
				<tr>
					<td>Feb 12, 2026</td>
					<td>$1.00</td>
					<td>Feb 11, 2026</td>
					<td>Invalid PayDate</td>
				</tr>
			</tbody>
		</table>`
		
		if strings.Contains(req.URL.String(), "err") {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(htmlResp)),
		}
	})

	source := NewStockAnalysisDividendSource(client)

	t.Run("Name and SupportedTypes", func(t *testing.T) {
		assert.Equal(t, "stockanalysis", source.Name())
		assert.Contains(t, source.SupportedAssetTypes(), "STOCK_BR")
	})

	t.Run("Success", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "PETR4.SA", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 2)
		
		assert.Equal(t, "Dividendo", events[0].Type)
		assert.Equal(t, 2.5499, events[0].Amount)
		assert.Equal(t, time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), events[0].Date)

		// O evento com PayDate inválido usa o ExDate como fallback para PayDate
		assert.Equal(t, "Dividendo", events[1].Type)
		assert.Equal(t, 1.00, events[1].Amount)
		assert.Equal(t, time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), events[1].PaymentDate)
	})

	t.Run("Error", func(t *testing.T) {
		res, err := source.GetDividends(context.Background(), "ERR", "STOCK_BR")
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("FII Success Normalization", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "MXRF11.SA", "FII")
		assert.NoError(t, err)
		assert.Len(t, events, 2)
		
		assert.Equal(t, "Rendimento", events[0].Type)
		assert.Equal(t, 2.5499, events[0].Amount)
		// 11 Feb minus 24h = 10 Feb
		assert.Equal(t, time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), events[0].Date)
	})
}
