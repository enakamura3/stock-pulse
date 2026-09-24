package market

import (
	"context"
	"errors"
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
				<!-- Missing Record Date (Fallback to Ex-Date) -->
				<tr>
					<td>Mar 20, 2026</td>
					<td>$1.50</td>
					<td>-</td>
					<td>Mar 30, 2026</td>
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
		assert.Len(t, events, 3)

		assert.Equal(t, "Dividendo", events[0].Type)
		assert.Equal(t, 2.5499, events[0].Amount)
		assert.Equal(t, time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), events[0].Date)

		// O evento com PayDate inválido usa o ExDate como fallback para PayDate
		assert.Equal(t, "Dividendo", events[1].Type)
		assert.Equal(t, 1.00, events[1].Amount)
		assert.Equal(t, time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), events[1].PaymentDate)

		// Evento com Record Date faltando usa Ex-Date - 1 dia. ExDate = Mar 20, CumDate = Mar 19
		assert.Equal(t, "Dividendo", events[2].Type)
		assert.Equal(t, 1.50, events[2].Amount)
		assert.Equal(t, time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC), events[2].Date)
		// O paymentDate se manteve Mar 30
		assert.Equal(t, time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC), events[2].PaymentDate)
	})

	t.Run("Error", func(t *testing.T) {
		res, err := source.GetDividends(context.Background(), "ERR", "STOCK_BR")
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("FII Success Normalization", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "MXRF11.SA", "FII")
		assert.NoError(t, err)
		assert.Len(t, events, 3)

		assert.Equal(t, "Rendimento", events[0].Type)
		assert.Equal(t, 2.5499, events[0].Amount)
		// 11 Feb was Record Date, which is exactly the Cum Date. No -24h is applied because Record Date was available.
		assert.Equal(t, time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), events[0].Date)
	})

	t.Run("Escapes Special Characters in Symbol", func(t *testing.T) {
		var capturedURL string
		c := NewStockAnalysisClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			capturedURL = req.URL.String()
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<table><tbody></tbody></table>`)),
			}
		})
		_, _ = c.FetchDividends(context.Background(), "BRK/B", "STOCK_US")
		assert.Contains(t, capturedURL, "/stocks/brk%2Fb/dividend/")

		_, _ = c.FetchDividends(context.Background(), "PETR/4.SA", "STOCK_BR")
		assert.Contains(t, capturedURL, "/quote/bvmf/petr%2F4/dividend/")
	})

	t.Run("ETF Support and Short Rows", func(t *testing.T) {
		var capturedURL string
		c := NewStockAnalysisClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			capturedURL = req.URL.String()
			html := `<table><tbody>
				<tr><td>Header only</td></tr>
				<tr><td>Feb 12, 2026</td><td>$1.50</td><td>Feb 11, 2026</td><td>Mar 15, 2026</td></tr>
			</tbody></table>`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(html)),
			}
		})
		divs, err := c.FetchDividends(context.Background(), "SPY", "ETF_US")
		assert.NoError(t, err)
		assert.Len(t, divs, 1)
		assert.Contains(t, capturedURL, "/etf/spy/dividend/")
	})

	t.Run("NewRequest Error with nil context", func(t *testing.T) {
		c := NewStockAnalysisClient()
		_, err := c.FetchDividends(nil, "AAPL", "STOCK_US")
		assert.Error(t, err)
	})

	t.Run("Network Do Error", func(t *testing.T) {
		c := NewStockAnalysisClient()
		c.httpClient.Transport = &errorTransport{err: errors.New("network failure")}
		_, err := c.FetchDividends(context.Background(), "AAPL", "STOCK_US")
		assert.Error(t, err)
	})

	t.Run("Body Read Error", func(t *testing.T) {
		c := NewStockAnalysisClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(&errReader{err: errors.New("read error")}),
			}
		})
		_, err := c.FetchDividends(context.Background(), "AAPL", "STOCK_US")
		assert.Error(t, err)
	})
}
