package market

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetFundamentals_RedisSetError(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	scraper := NewScraper()
	scraper.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		html := `<html><body>
			<span class="txt">P/L</span><span class="data">5,5</span>
			<span class="txt">Div. Yield</span><span class="data">8,0%</span>
		</body></html>`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(html)),
		}
	})

	mockProv := &MockQuoteProvider{}
	mockProv.On("GetQuote", mock.Anything, "PETR4").Return(&Quote{
		Ticker: "PETR4",
		Price:  30.0,
	}, nil)

	svc := NewService(mockProv, rdb)
	svc.scraper = scraper

	// Cause Redis Set to fail by shutting down miniredis before Set
	mr.Close()

	fund, err := svc.GetFundamentals(context.Background(), "PETR4")
	assert.NoError(t, err)
	assert.NotNil(t, fund)
}
