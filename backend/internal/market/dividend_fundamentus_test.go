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

func TestFundamentusDividendSource_GetDividends(t *testing.T) {
	client := NewFundamentusClient()
	client.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		htmlResp := `
		<table id="resultado">
			<tbody>
				<tr>
					<td>12/02/2026</td>
					<td>2,5499</td>
					<td>DIVIDENDO</td>
					<td>15/03/2026</td>
				</tr>
			</tbody>
		</table>`

		if strings.Contains(req.URL.String(), "ERR") {
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

	source := NewFundamentusDividendSource(client)

	t.Run("Name and SupportedTypes", func(t *testing.T) {
		assert.Equal(t, "fundamentus", source.Name())
		assert.Contains(t, source.SupportedAssetTypes(), "STOCK_BR")
	})

	t.Run("Success", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "PETR4.SA", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 1)

		assert.Equal(t, "Dividendo", events[0].Type)
		assert.Equal(t, 2.5499, events[0].Amount)
		assert.Equal(t, time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), events[0].Date)
	})

	t.Run("Error", func(t *testing.T) {
		res, err := source.GetDividends(context.Background(), "ERR", "STOCK_BR")
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
