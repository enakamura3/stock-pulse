package telegram

import (
	"errors"
	"strings"
	"testing"

	"github.com/onigiri/stock-pulse/backend/internal/alert"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_Alerts(t *testing.T) {
	h, svc, _, mSvc, _, alertSvc := setupHandlersTest()

	t.Run("HandleAlerts - nil alert service", func(t *testing.T) {
		hNil := NewHandlers(svc, nil, nil, nil, nil)
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Edit", "⚠️ Módulo de alertas não está ativo.", mock.Anything).Return(nil).Once()

		err := hNil.HandleAlerts(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAlerts - unauthenticated", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.store = map[string]interface{}{"user_id": nil}
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleAlerts(mCtx)
		assert.Error(t, err)
	})

	t.Run("HandleAlerts - get alerts error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(([]*alert.Alert)(nil), errors.New("db error")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar seus alertas.", mock.Anything).Return(nil).Once()

		err := h.HandleAlerts(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAlerts - empty alerts", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Você não possui alertas cadastrados")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAlerts(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "Você não possui alertas cadastrados")
	})

	t.Run("HandleAlerts - with alerts covering all statuses, conditions, currencies, and limits (>5)", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		alertsList := []*alert.Alert{
			{ID: "a1", Ticker: "PETR4", TargetPrice: 35.50, Condition: "ABOVE", Status: "ACTIVE", Currency: "BRL"},
			{ID: "a2", Ticker: "VALE3", TargetPrice: 60.00, Condition: "BELOW", Status: "TRIGGERED", Currency: ""},
			{ID: "a3", Ticker: "AAPL", TargetPrice: 200.00, Condition: "ABOVE", Status: "DISABLED", Currency: "USD"},
			{ID: "a4", Ticker: "BTC-USD", TargetPrice: 90000.00, Condition: "BELOW", Status: "UNKNOWN_STATUS", Currency: "USD"},
			{ID: "a5", Ticker: "WEGE3", TargetPrice: 45.00, Condition: "ABOVE", Status: "ACTIVE", Currency: "BRL"},
			{ID: "a6", Ticker: "ITUB4", TargetPrice: 30.00, Condition: "BELOW", Status: "ACTIVE", Currency: "BRL"},
		}
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(alertsList, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Meus Alertas de Preço") &&
				strings.Contains(msg, "PETR4") &&
				strings.Contains(msg, "VALE3") &&
				strings.Contains(msg, "disparado") &&
				strings.Contains(msg, "pausado") &&
				strings.Contains(msg, "Total: 6 alerta(s)")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAlerts(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "PETR4")
	})

	t.Run("HandleAlerts - message not modified", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err := h.HandleAlerts(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAlerts - generic edit error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("network error")).Once()

		err := h.HandleAlerts(mCtx)
		assert.Error(t, err)
	})

	t.Run("HandleAlertCreate - state error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ALERT_EXPECT_TICKER"}).Return(errors.New("redis error")).Once()
		mCtx.On("Edit", "❌ Erro interno ao iniciar criação de alerta.", mock.Anything).Return(nil).Once()

		err := h.HandleAlertCreate(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAlertCreate - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ALERT_EXPECT_TICKER"}).Return(nil).Once()
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Criar Alerta de Preço")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAlertCreate(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAlertConditionAbove and Below - invalid state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		mCtx.On("Edit", "⚠️ Nenhuma criação de alerta em andamento.", mock.Anything).Return(nil).Once()

		err := h.HandleAlertConditionAbove(mCtx)
		assert.NoError(t, err)

		// State step mismatch
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "OTHER_STEP"}, nil).Once()
		mCtx2.On("Edit", "⚠️ Nenhuma criação de alerta em andamento.", mock.Anything).Return(nil).Once()

		err = h.HandleAlertConditionBelow(mCtx2)
		assert.NoError(t, err)
	})

	t.Run("HandleAlertConditionAbove and Below - success", func(t *testing.T) {
		// ABOVE
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_COND", Ticker: "PETR4"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4", Type: "ABOVE"}).Return(nil).Once()
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "PETR4") && strings.Contains(msg, "ACIMA DE")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAlertConditionAbove(mCtx)
		assert.NoError(t, err)

		// BELOW
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_COND", Ticker: "VALE3"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "VALE3", Type: "BELOW"}).Return(nil).Once()
		mCtx2.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "VALE3") && strings.Contains(msg, "ABAIXO DE")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleAlertConditionBelow(mCtx2)
		assert.NoError(t, err)
	})

	t.Run("handleAlertToggle - unauthenticated, error and success", func(t *testing.T) {
		// Unauthenticated
		mCtxUnauth := new(MockTelebotContext)
		mCtxUnauth.store = map[string]interface{}{"user_id": nil}
		mCtxUnauth.On("Respond", mock.Anything).Return(nil)
		mCtxUnauth.On("Callback").Return(&telebot.Callback{})
		mCtxUnauth.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.handleAlertToggle(mCtxUnauth, "a1")
		assert.Error(t, err)

		// Toggle error and then calls HandleAlerts
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Respond", mock.Anything).Return(nil).Twice()
		alertSvc.On("ToggleAlert", mock.Anything, "a1", "00000000-0000-0000-0000-000000000000").Return("", errors.New("toggle err")).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtxErr.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.handleAlertToggle(mCtxErr, "a1")
		assert.NoError(t, err)

		// Toggle success
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Respond", mock.Anything).Return(nil).Twice()
		alertSvc.On("ToggleAlert", mock.Anything, "a1", "00000000-0000-0000-0000-000000000000").Return("DISABLED", nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtxOk.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.handleAlertToggle(mCtxOk, "a1")
		assert.NoError(t, err)
	})

	t.Run("handleAlertDelete - unauthenticated, error and success", func(t *testing.T) {
		// Unauthenticated
		mCtxUnauth := new(MockTelebotContext)
		mCtxUnauth.store = map[string]interface{}{"user_id": nil}
		mCtxUnauth.On("Respond", mock.Anything).Return(nil)
		mCtxUnauth.On("Callback").Return(&telebot.Callback{})
		mCtxUnauth.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.handleAlertDelete(mCtxUnauth, "a1")
		assert.Error(t, err)

		// Delete error and then calls HandleAlerts
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Respond", mock.Anything).Return(nil).Twice()
		alertSvc.On("DeleteAlert", mock.Anything, "a1", "00000000-0000-0000-0000-000000000000").Return(errors.New("delete err")).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtxErr.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.handleAlertDelete(mCtxErr, "a1")
		assert.NoError(t, err)

		// Delete success
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Respond", mock.Anything).Return(nil).Twice()
		alertSvc.On("DeleteAlert", mock.Anything, "a1", "00000000-0000-0000-0000-000000000000").Return(nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtxOk.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.handleAlertDelete(mCtxOk, "a1")
		assert.NoError(t, err)
	})

	t.Run("HandleDynamicCallback - alert prefixes", func(t *testing.T) {
		// btn_alert_toggle_
		mCtx1 := new(MockTelebotContext)
		mCtx1.On("Callback").Return(&telebot.Callback{Data: "\fbtn_alert_toggle_a123"}).Once()
		mCtx1.On("Respond", mock.Anything).Return(nil).Twice()
		alertSvc.On("ToggleAlert", mock.Anything, "a123", "00000000-0000-0000-0000-000000000000").Return("ACTIVE", nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtx1.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDynamicCallback(mCtx1)
		assert.NoError(t, err)

		// btn_alert_del_
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Callback").Return(&telebot.Callback{Data: "\fbtn_alert_del_a123"}).Once()
		mCtx2.On("Respond", mock.Anything).Return(nil).Twice()
		alertSvc.On("DeleteAlert", mock.Anything, "a123", "00000000-0000-0000-0000-000000000000").Return(nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtx2.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtx2)
		assert.NoError(t, err)

		// btn_alert_cond_
		mCtx3 := new(MockTelebotContext)
		mCtx3.On("Callback").Return(&telebot.Callback{Data: "\fbtn_alert_cond_above"}).Once()
		mCtx3.On("Respond", mock.Anything).Return(nil).Once()
		mCtx3.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_COND", Ticker: "PETR4"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4", Type: "ABOVE"}).Return(nil).Once()
		mCtx3.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtx3)
		assert.NoError(t, err)
	})

	t.Run("HandleText - ALERT_EXPECT_TICKER", func(t *testing.T) {
		// Quote error
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxErr.On("Text").Return("INVALID_TICKER")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_TICKER"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "INVALID_TICKER").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mCtxErr.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo não encontrado na bolsa")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtxErr)
		assert.NoError(t, err)

		// Success
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Text").Return("PETR4")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_TICKER"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Symbol: "PETR4"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ALERT_EXPECT_COND", Ticker: "PETR4"}).Return(nil).Once()
		mCtxOk.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Alerta para PETR4")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxOk)
		assert.NoError(t, err)
	})

	t.Run("HandleText - ALERT_EXPECT_PRICE", func(t *testing.T) {
		// Invalid price
		mCtxInvalid := new(MockTelebotContext)
		mCtxInvalid.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxInvalid.On("Text").Return("abc")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4", Type: "ABOVE"}, nil).Once()
		mCtxInvalid.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Preço inválido")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtxInvalid)
		assert.NoError(t, err)

		// Unauthenticated
		mCtxUnauth := new(MockTelebotContext)
		mCtxUnauth.store = map[string]interface{}{"user_id": nil}
		mCtxUnauth.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUnauth.On("Text").Return("35.50")
		mCtxUnauth.On("Callback").Return((*telebot.Callback)(nil))
		mCtxUnauth.On("Send", mock.Anything, mock.Anything).Return(nil)
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4", Type: "ABOVE"}, nil).Once()

		err = h.HandleText(mCtxUnauth)
		assert.Error(t, err)

		// CreateAlert error
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxErr.On("Text").Return("35,50")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4", Type: "ABOVE"}, nil).Once()
		alertSvc.On("CreateAlert", mock.Anything, "00000000-0000-0000-0000-000000000000", "PETR4", 35.50, "ABOVE").Return((*alert.Alert)(nil), errors.New("create err")).Once()
		mCtxErr.On("Send", "❌ Ocorreu um erro ao salvar o alerta. Tente novamente mais tarde.", mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxErr)
		assert.NoError(t, err)

		// CreateAlert success (BELOW, currency "")
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Text").Return("60.00")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "VALE3", Type: "BELOW"}, nil).Once()
		alertSvc.On("CreateAlert", mock.Anything, "00000000-0000-0000-0000-000000000000", "VALE3", 60.00, "BELOW").Return(&alert.Alert{
			Ticker:      "VALE3",
			Condition:   "BELOW",
			TargetPrice: 60.00,
			Currency:    "",
		}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mCtxOk.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Alerta Criado com Sucesso") &&
				strings.Contains(msg, "VALE3") &&
				strings.Contains(msg, "abaixo de") &&
				strings.Contains(msg, "R$ 60,00")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxOk)
		assert.NoError(t, err)
	})
}
