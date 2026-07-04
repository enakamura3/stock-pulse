package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

// Mock telebot.Context
type MockTelebotContext struct {
	mock.Mock
	telebot.Context // embed to panic on unimplemented methods instead of failing compilation if interface grows
}

func (m *MockTelebotContext) Send(what interface{}, opts ...interface{}) error {
	args := m.Called(what, opts)
	return args.Error(0)
}

func (m *MockTelebotContext) Chat() *telebot.Chat {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*telebot.Chat)
	}
	return nil
}

func (m *MockTelebotContext) Args() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockTelebotContext) Respond(resp ...*telebot.CallbackResponse) error {
	args := m.Called(resp)
	return args.Error(0)
}

func (m *MockTelebotContext) Data() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTelebotContext) Text() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTelebotContext) Message() *telebot.Message {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*telebot.Message)
	}
	return nil
}

func (m *MockTelebotContext) Edit(what interface{}, opts ...interface{}) error {
	args := m.Called(what, opts)
	return args.Error(0)
}

func (m *MockTelebotContext) Callback() *telebot.Callback {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*telebot.Callback)
	}
	return nil
}

// Mocks for dependencies
type MockPortfolioService struct {
	mock.Mock
}

func (m *MockPortfolioService) GetPortfolios(ctx context.Context, userID string) ([]portfolio.Portfolio, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]portfolio.Portfolio), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockPortfolioService) GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*portfolio.Portfolio, []portfolio.Position, error) {
	args := m.Called(ctx, portfolioID, userID)
	var p *portfolio.Portfolio
	if args.Get(0) != nil {
		p = args.Get(0).(*portfolio.Portfolio)
	}
	var pos []portfolio.Position
	if args.Get(1) != nil {
		pos = args.Get(1).([]portfolio.Position)
	}
	return p, pos, args.Error(2)
}
func (m *MockPortfolioService) AddTransaction(ctx context.Context, userID string, tx *portfolio.Transaction) (*portfolio.Transaction, error) {
	args := m.Called(ctx, userID, tx)
	if args.Get(0) != nil {
		return args.Get(0).(*portfolio.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockPortfolioService) GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]portfolio.CalculatedDividend, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]portfolio.CalculatedDividend), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockPortfolioService) GetPortfolioTransactions(ctx context.Context, portfolioID, userID string) ([]portfolio.Transaction, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]portfolio.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockMarketSvc struct {
	mock.Mock
}

func (m *MockMarketSvc) GetQuote(ctx context.Context, ticker string) (*market.Quote, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*market.Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockFixedIncomeSvc struct {
	mock.Mock
}

func (m *MockFixedIncomeSvc) GetPortfolioPositions(ctx context.Context, portfolioID string) ([]fixedincome.Position, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) != nil {
		return args.Get(0).([]fixedincome.Position), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupHandlersTest() (*Handlers, *MockService, *MockPortfolioService, *MockMarketSvc, *MockFixedIncomeSvc) {
	svc := new(MockService)
	pSvc := new(MockPortfolioService)
	mSvc := new(MockMarketSvc)
	fiSvc := new(MockFixedIncomeSvc)
	return NewHandlers(svc, pSvc, mSvc, fiSvc), svc, pSvc, mSvc, fiSvc
}

func TestHandlers_Register(t *testing.T) {
	bot, _ := telebot.NewBot(telebot.Settings{Offline: true})
	h, _, _, _, _ := setupHandlersTest()
	h.Register(bot)
}

func TestHandlers_HandleStart(t *testing.T) {
	h, svc, _, _, _ := setupHandlersTest()

	t.Run("no args", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{})
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleStart(mCtx)
		assert.NoError(t, err)
	})

	t.Run("with invalid token", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{"token123"})
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		svc.On("LinkAccountWithToken", mock.Anything, "token123", int64(1)).Return(errors.New("inválido ou expirado")).Once()

		err := h.HandleStart(mCtx)
		assert.NoError(t, err)
	})

	t.Run("with other error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{"token123"})
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		svc.On("LinkAccountWithToken", mock.Anything, "token123", int64(1)).Return(errors.New("db error")).Once()

		err := h.HandleStart(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{"token123"})
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		svc.On("LinkAccountWithToken", mock.Anything, "token123", int64(1)).Return(nil).Once()

		err := h.HandleStart(mCtx)
		assert.NoError(t, err)
	})
}
