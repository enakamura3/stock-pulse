package market

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInvestidor10Client_FetchDividends(t *testing.T) {
	validHTML := `
	<!DOCTYPE html>
	<html>
	<body>
		<table id="table-dividends-history">
			<thead>
				<tr>
					<th>Tipo</th>
					<th>Data COM</th>
					<th>Pagamento</th>
					<th>Valor</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Dividendos</td>
					<td>20/10/2026</td>
					<td>10/11/2026</td>
					<td>1,00440000</td>
				</tr>
				<tr>
					<td>Rendimento</td>
					<td>15/09/2026</td>
					<td>-</td>
					<td>0,50000000</td>
				</tr>
				<tr>
					<td>JCP</td>
					<td>21/11/2023</td>
					<td>08/12/2023</td>
					<td>0,95470000</td>
				</tr>
				<!-- Linha vazia ou com menos colunas -->
				<tr>
					<td>Incompleto</td>
				</tr>
			</tbody>
		</table>
	</body>
	</html>
	`

	emptyTableHTML := `
	<!DOCTYPE html>
	<html>
	<body>
		<table id="table-dividends-history">
			<tbody>
			</tbody>
		</table>
	</body>
	</html>
	`

	noTableHTML := `
	<!DOCTYPE html>
	<html><body><div>Sem tabela</div></body></html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/etfs/spyi11"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validHTML))
		case strings.Contains(r.URL.Path, "/acoes/petr4"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validHTML))
		case strings.Contains(r.URL.Path, "/fiis/mxrf11"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validHTML))
		case strings.Contains(r.URL.Path, "/bdrs/aapl34"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validHTML))
		case strings.Contains(r.URL.Path, "/empty"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(emptyTableHTML))
		case strings.Contains(r.URL.Path, "/notable"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(noTableHTML))
		case strings.Contains(r.URL.Path, "/servererror"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewInvestidor10Client().WithBaseURL(server.URL)
	assert.NotNil(t, client.httpClient)
	client = client.WithHTTPClient(server.Client())

	t.Run("Empty or invalid ticker", func(t *testing.T) {
		_, err := client.FetchDividends(context.Background(), "", "ETF_BR")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ticker inválido")

		_, err = client.FetchDividends(context.Background(), "   .SA", "ETF_BR")
		assert.Error(t, err)
	})

	t.Run("Success ETF_BR", func(t *testing.T) {
		raw, err := client.FetchDividends(context.Background(), "SPYI11.SA", "ETF_BR")
		assert.NoError(t, err)
		assert.Len(t, raw, 3)
		assert.Equal(t, "Dividendos", raw[0].Type)
		assert.Equal(t, "20/10/2026", raw[0].CumDate)
		assert.Equal(t, "10/11/2026", raw[0].PaymentDate)
		assert.Equal(t, "1,00440000", raw[0].Amount)
	})

	t.Run("Success FII and FIAGRO categories", func(t *testing.T) {
		raw, err := client.FetchDividends(context.Background(), "MXRF11", "FII")
		assert.NoError(t, err)
		assert.NotEmpty(t, raw)

		raw, err = client.FetchDividends(context.Background(), "MXRF11", "FIAGRO")
		assert.NoError(t, err)
		assert.NotEmpty(t, raw)
	})

	t.Run("Success BDR category", func(t *testing.T) {
		raw, err := client.FetchDividends(context.Background(), "AAPL34", "BDR")
		assert.NoError(t, err)
		assert.NotEmpty(t, raw)
	})

	t.Run("Success default STOCK_BR category", func(t *testing.T) {
		raw, err := client.FetchDividends(context.Background(), "PETR4", "STOCK_BR")
		assert.NoError(t, err)
		assert.NotEmpty(t, raw)
	})

	t.Run("Server Error", func(t *testing.T) {
		_, err := client.FetchDividends(context.Background(), "servererror", "ETF_BR")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "investidor10 retornou status 500")
	})

	t.Run("Empty Table Error", func(t *testing.T) {
		_, err := client.FetchDividends(context.Background(), "empty", "ETF_BR")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nenhum provento encontrado")
	})

	t.Run("No Table Error", func(t *testing.T) {
		_, err := client.FetchDividends(context.Background(), "notable", "ETF_BR")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nenhum provento encontrado")
	})

	t.Run("Not Found in any category", func(t *testing.T) {
		_, err := client.FetchDividends(context.Background(), "NOT_FOUND_TICKER", "ETF_BR")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "investidor10 retornou status 404")
	})

	t.Run("Context Cancelled Error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.FetchDividends(ctx, "SPYI11", "ETF_BR")
		assert.Error(t, err)
	})

	t.Run("NewRequest Error with nil context", func(t *testing.T) {
		_, err := client.FetchDividends(nil, "SPYI11", "ETF_BR")
		assert.Error(t, err)
	})

	t.Run("Body Read Error in goquery", func(t *testing.T) {
		c := NewInvestidor10Client()
		c.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(&errReader{err: assert.AnError}),
			}
		})
		_, err := c.FetchDividends(context.Background(), "SPYI11", "ETF_BR")
		assert.Error(t, err)
	})
}

func TestInvestidor10DividendSource_GetDividends(t *testing.T) {
	sourceHTML := `
	<table id="table-dividends-history">
		<tbody>
			<tr>
				<td>Dividendos</td>
				<td>20/10/2026</td>
				<td>10/11/2026</td>
				<td>1,00440000</td>
			</tr>
			<!-- Pagamento em aberto / traço -->
			<tr>
				<td>Rendimentos</td>
				<td>15/09/2026</td>
				<td>-</td>
				<td>0,50000000</td>
			</tr>
			<!-- Pagamento n/a -->
			<tr>
				<td>Rendimentos</td>
				<td>15/08/2026</td>
				<td>n/a</td>
				<td>0,50000000</td>
			</tr>
			<!-- Pagamento vazio -->
			<tr>
				<td>Rendimentos</td>
				<td>15/07/2026</td>
				<td></td>
				<td>0,50000000</td>
			</tr>
			<!-- Data COM inválida (deve ignorar) -->
			<tr>
				<td>Dividendos</td>
				<td>data-invalida</td>
				<td>10/11/2026</td>
				<td>1,00</td>
			</tr>
			<!-- Valor inválido (deve ignorar) -->
			<tr>
				<td>Dividendos</td>
				<td>20/06/2026</td>
				<td>10/07/2026</td>
				<td>valor-invalido</td>
			</tr>
			<!-- Histórico antigo (2023) -->
			<tr>
				<td>Juros Sobre Capital Próprio</td>
				<td>21/11/2023</td>
				<td>08/12/2023</td>
				<td>0,95470000</td>
			</tr>
		</tbody>
	</table>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "err") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if strings.Contains(r.URL.Path, "empty") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<table id="table-dividends-history"><tbody><tr><td>D</td><td>invalid</td><td>10/11/2026</td><td>1,00</td></tr></tbody></table>`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sourceHTML))
	}))
	defer server.Close()

	client := NewInvestidor10Client().WithBaseURL(server.URL).WithHTTPClient(server.Client())
	source := NewInvestidor10DividendSource(client)

	assert.Equal(t, "investidor10", source.Name())
	assert.Contains(t, source.SupportedAssetTypes(), "ETF_BR")
	assert.Contains(t, source.SupportedAssetTypes(), "STOCK_BR")

	t.Run("Success Parsing and Normalization", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "SPYI11", "ETF_BR")
		assert.NoError(t, err)
		assert.Len(t, events, 5)

		// Mais recente primeiro (ordenado decrescente)
		assert.Equal(t, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), events[0].Date)
		assert.Equal(t, time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), events[0].PaymentDate)
		assert.Equal(t, 1.0044, events[0].Amount)
		assert.Equal(t, "Dividendo", events[0].Type)

		// Teste de fallback de PaymentDate para CumDate quando "-"
		assert.Equal(t, events[1].Date, events[1].PaymentDate)
		assert.Equal(t, "Rendimento", events[1].Type)

		// JCP normalizado
		assert.Equal(t, "JCP", events[4].Type)
		assert.Equal(t, time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC), events[4].Date)
	})

	t.Run("Client Error", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "err", "ETF_BR")
		assert.Error(t, err)
		assert.Nil(t, events)
	})

	t.Run("Empty Valid Events", func(t *testing.T) {
		events, err := source.GetDividends(context.Background(), "empty", "ETF_BR")
		assert.NoError(t, err)
		assert.Nil(t, events)
	})
}
