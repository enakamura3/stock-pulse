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

type errReader struct {
	err error
}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

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

	t.Run("Escapes Special Characters in Symbol", func(t *testing.T) {
		var capturedURL string
		c := NewFundamentusClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			capturedURL = req.URL.String()
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<table id="resultado"><tbody></tbody></table>`)),
			}
		})
		_, _, _ = c.FetchDividends(context.Background(), "TEST&INJECT=1")
		assert.Contains(t, capturedURL, "papel=TEST%26INJECT%3D1")
	})

	t.Run("FII Layout Success", func(t *testing.T) {
		c := NewFundamentusClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.String(), "fii_proventos.php") {
				fiiHTML := `
				<table id="resultado">
					<tbody>
						<tr>
							<td>15/01/2026</td>
							<td>RENDIMENTO</td>
							<td>25/01/2026</td>
							<td>1,10</td>
						</tr>
					</tbody>
				</table>`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(fiiHTML)),
				}
			}
			// Ação returns empty table
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`<table id="resultado"><tbody></tbody></table>`)),
			}
		})

		raw, layout, err := c.FetchDividends(context.Background(), "HGLG11.SA")
		assert.NoError(t, err)
		assert.Equal(t, "fii", layout)
		assert.Len(t, raw, 1)
		assert.Equal(t, "RENDIMENTO", raw[0].Type)
		assert.Equal(t, "1,10", raw[0].Amount)
		assert.Equal(t, "15/01/2026", raw[0].Date)
		assert.Equal(t, "25/01/2026", raw[0].PaymentDate)
	})

	t.Run("NewRequest Error with nil context", func(t *testing.T) {
		c := NewFundamentusClient()
		_, _, err := c.FetchDividends(nil, "PETR4")
		assert.Error(t, err)
	})

	t.Run("Network Do Error", func(t *testing.T) {
		c := NewFundamentusClient()
		c.httpClient.Transport = &errorTransport{err: errors.New("network failure")}
		_, _, err := c.FetchDividends(context.Background(), "PETR4")
		assert.Error(t, err)
	})

	t.Run("Body Read Error", func(t *testing.T) {
		c := NewFundamentusClient()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(&errReader{err: errors.New("read error")}),
			}
		})
		_, _, err := c.FetchDividends(context.Background(), "PETR4")
		assert.Error(t, err)
	})
}
