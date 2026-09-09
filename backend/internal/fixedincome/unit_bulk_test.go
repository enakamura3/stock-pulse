package fixedincome

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type dummyFile struct {
	*bytes.Reader
}

func (d *dummyFile) Close() error { return nil }

func TestService_BulkAddTransactions(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty CSV file", func(t *testing.T) {
		mockRepo := &MockFullRepo{}
		svc := NewService(mockRepo, nil)
		emptyFile := &dummyFile{Reader: bytes.NewReader([]byte(""))}
		_, err := svc.BulkAddTransactions(ctx, "p1", emptyFile)
		assert.ErrorContains(t, err, "vazio")
	})

	t.Run("Error loading existing assets", func(t *testing.T) {
		mockRepo := &MockFullRepo{}
		svc := NewService(mockRepo, nil)
		mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return(nil, assert.AnError).Once()
		file := &dummyFile{Reader: bytes.NewReader([]byte("date;name;type;amount;indexer;rate;maturity\n2026-01-01;CDB;APLICACAO;100;CDI;100;2028-01-01"))}
		_, err := svc.BulkAddTransactions(ctx, "p1", file)
		assert.ErrorContains(t, err, "erro ao carregar ativos")
	})

	t.Run("Malformed CSV file", func(t *testing.T) {
		mockRepo := &MockFullRepo{}
		svc := NewService(mockRepo, nil)
		badFile := &dummyFile{Reader: bytes.NewReader([]byte("\"unclosed quote"))}
		_, err := svc.BulkAddTransactions(ctx, "p1", badFile)
		assert.ErrorContains(t, err, "erro ao ler arquivo CSV")
	})

	t.Run("CSV with various error and success branches", func(t *testing.T) {
		mockRepo := &MockFullRepo{}
		svc := NewService(mockRepo, nil)

		csvContent := `data;instituicao;tipo;valor;indexador;taxa;vencimento
2026-01-15;Itaú CDB;APLICACAO;1000.00;CDI;100.0;2028-12-31
invalid-short-row
;;;;;;
2026-01-15;Itaú CDB;APLICACAO;-500;CDI;100.0;2028-12-31
bad_date;Itaú CDB;APLICACAO;1000.00;CDI;100.0;2028-12-31
16/01/2026;Itaú CDB;APLICACAO;1000.00;CDI;100.0;--
2026-01-17;Novo CDB;APLICACAO;1000.00;PRE;12.0;2028-01-01
2026-01-18;Fail Asset;APLICACAO;1000.00;PRE;10.0;2028-01-01
2026-01-19;Fail Tx;APLICACAO;1000.00;PRE;10.0;2028-01-01
2026-01-20;IPCA CDB;APLICACAO;1000.00;IPCA;5.5;2030-01-01
`
		existingAssets := []Asset{
			{ID: "a1", Institution: "Itaú CDB", Indexer: "CDI", Rate: 100.0, DebtType: "POS"},
		}
		mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return(existingAssets, nil).Once()
		mockRepo.On("GetLatestIndexRate", mock.Anything, mock.Anything).Return(&IndexRate{}, nil).Maybe()
		mockRepo.On("SaveIndexRates", mock.Anything, mock.Anything).Return(nil).Maybe()

		// Novo CDB -> creates asset and tx
		mockRepo.On("CreateAsset", ctx, mock.MatchedBy(func(a *Asset) bool {
			return a.Institution == "Novo CDB"
		})).Return(&Asset{ID: "a2", Institution: "Novo CDB", DebtType: "PRE"}, nil).Once()

		// IPCA CDB -> creates asset with HIBRIDO and tx
		mockRepo.On("CreateAsset", ctx, mock.MatchedBy(func(a *Asset) bool {
			return a.Institution == "IPCA CDB"
		})).Return(&Asset{ID: "a4", Institution: "IPCA CDB", DebtType: "HIBRIDO"}, nil).Once()

		// Fail Asset -> fails to create asset
		mockRepo.On("CreateAsset", ctx, mock.MatchedBy(func(a *Asset) bool {
			return a.Institution == "Fail Asset"
		})).Return(nil, assert.AnError).Once()

		// Fail Tx -> creates asset, fails tx
		mockRepo.On("CreateAsset", ctx, mock.MatchedBy(func(a *Asset) bool {
			return a.Institution == "Fail Tx"
		})).Return(&Asset{ID: "a3", Institution: "Fail Tx", DebtType: "PRE"}, nil).Once()

		// Transaction mocks
		mockRepo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *Transaction) bool {
			return tx.AssetID == "a1" || tx.AssetID == "a2" || tx.AssetID == "a4"
		})).Return(&Transaction{ID: "t1"}, nil)

		mockRepo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *Transaction) bool {
			return tx.AssetID == "a3"
		})).Return(nil, assert.AnError).Once()

		file := &dummyFile{Reader: bytes.NewReader([]byte(csvContent))}
		res, err := svc.BulkAddTransactions(ctx, "p1", file)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 4, res.Success)
		assert.Len(t, res.Errors, 6)
	})
}

func TestService_BulkAddTreasuryTransactions(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Empty CSV
	emptyFile := &dummyFile{Reader: bytes.NewReader([]byte(""))}
	_, err := svc.BulkAddTreasuryTransactions(ctx, "p1", emptyFile)
	assert.ErrorContains(t, err, "vazio")

	// 2. Invalid syntax (unclosed quote)
	badFile := &dummyFile{Reader: bytes.NewReader([]byte("\"unclosed quote"))}
	_, err = svc.BulkAddTreasuryTransactions(ctx, "p1", badFile)
	assert.ErrorContains(t, err, "erro ao ler arquivo CSV")

	// 3. CSV with comma fallback, header, and mixed rows
	csvContent := `Date;Ticker;Type;Quantity;UnitPrice;ContractedRate;TreasuryType;MaturityDate;HasCoupons
2026-01-15;Tesouro Selic 2029;SUBSCRIPTION;1.5;14000.00;0.05;SELIC;2029-01-01;false
16/01/2026;Tesouro IPCA+ 2035;APLICAÇÃO;2,0;3500,50;5,8%;IPCA+;15/05/2035;sim
invalid_short_row
;missing_fields;;;
2026-01-15;Tesouro Prefixado 2027;COMPRA;0.0;1000.00
2026-01-15;Tesouro Prefixado 2027;COMPRA;1.0;0.0
bad_date;Tesouro Prefixado 2027;COMPRA;1.0;1000.00
2026-01-15;Tesouro Prefixado 2027;UNKNOWN_TYPE;1.0;1000.00
2026-01-20;Tesouro Prefixado 2027;BUY;1.0;1000.00;10.5;PREFIXADO;;
2026-01-22;Tesouro Prefixado com Juros 2033;BUY;1.0;1000.00
2026-01-23;Tesouro IPCA+ 2045;COMPRA;1.0;2000.00
2026-01-24;Tesouro Titulo Sem Ano;COMPRA;1.0;1000.00
2026-01-25;Tesouro Selic 2029;RESGATE;0.5;14100.00;0.05;SELIC;2029-01-01;false
2026-01-26;Failing Asset;BUY;1.0;1000.00
`

	mockRepo.On("GetTreasuryAssetByTicker", mock.Anything, mock.Anything, mock.MatchedBy(func(t string) bool {
		return t != "Failing Asset"
	})).Return("a1", nil).Maybe()

	mockRepo.On("GetTreasuryAssetByTicker", mock.Anything, mock.Anything, "Failing Asset").Return("", assert.AnError).Maybe()

	mockRepo.On("CreateTreasurySubscription", mock.Anything, mock.Anything, "p1", "a1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("sub1", nil).Maybe()
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", mock.Anything, mock.Anything, "p1", "a1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("red1", nil).Maybe()
	mockRepo.On("UpdateRedemptionFinancials", mock.Anything, mock.Anything, "red1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	mockRepo.On("GetActiveLotsForAsset", mock.Anything, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{}, nil).Maybe()
	mockRepo.On("GetAnbimaHolidays", mock.Anything).Return(map[string]bool{}, nil).Maybe()
	mockRepo.On("GetSelicRates", mock.Anything).Return(map[string]float64{}, nil).Maybe()
	mockRepo.On("GetTotalSelicInvested", mock.Anything, mock.Anything, "p1").Return(0.0, nil).Maybe()

	file := &dummyFile{Reader: bytes.NewReader([]byte(csvContent))}
	res, err := svc.BulkAddTreasuryTransactions(ctx, "p1", file)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 7, res.Success)
	assert.Len(t, res.Errors, 7)

	// 4. Comma delimited CSV
	csvComma := "Date,Ticker,Type,Quantity,UnitPrice\n2026-01-15,Tesouro Selic 2029,BUY,1.0,14000.00"
	fileComma := &dummyFile{Reader: bytes.NewReader([]byte(csvComma))}
	resComma, err := svc.BulkAddTreasuryTransactions(ctx, "p1", fileComma)
	assert.NoError(t, err)
	assert.Equal(t, 1, resComma.Success)
}

func TestService_ExportTreasuryTransactions(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Error fetching transactions
	mockRepo.On("GetTreasuryTransactionsList", ctx, "p_err").Return(nil, assert.AnError).Once()
	_, err := svc.ExportTreasuryTransactions(ctx, "p_err")
	assert.Error(t, err)

	// 2. Success exporting
	mockRepo.On("GetTreasuryTransactionsList", ctx, "p1").Return([]TreasuryTxRequest{
		{
			Ticker:          "Tesouro Selic 2029",
			Type:            "SUBSCRIPTION",
			Quantity:        2.5,
			UnitPrice:       14500.25,
			ContractedRate:  0.05,
			TreasuryType:    "SELIC",
			MaturityDate:    "2029-01-01",
			HasCoupons:      false,
			TransactionDate: "2026-01-15",
		},
	}, nil).Once()

	data, err := svc.ExportTreasuryTransactions(ctx, "p1")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	csvStr := string(data)
	assert.Contains(t, csvStr, "Date;Ticker;Type;Quantity;UnitPrice;ContractedRate;TreasuryType;MaturityDate;HasCoupons")
	assert.Contains(t, csvStr, "Tesouro Selic 2029")
	assert.Contains(t, csvStr, "SUBSCRIPTION")
	assert.Contains(t, csvStr, "2.500000")
}


