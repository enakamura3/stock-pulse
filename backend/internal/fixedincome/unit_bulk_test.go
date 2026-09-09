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
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Empty CSV file
	emptyFile := &dummyFile{Reader: bytes.NewReader([]byte(""))}
	_, err := svc.BulkAddTransactions(ctx, "p1", emptyFile)
	assert.ErrorContains(t, err, "vazio")

	// 2. CSV with header and valid/invalid rows
	csvContent := `data;instituicao;tipo;valor;indexador;taxa;vencimento
2026-01-15;Itaú CDB;APLICACAO;1000.00;CDI;100.0;2028-12-31
invalid-row
2026-01-16;Itaú CDB;INVALID_TYPE;1000.00;CDI;100.0;2028-12-31`

	file := &dummyFile{Reader: bytes.NewReader([]byte(csvContent))}

	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{}, nil).Once()
	mockRepo.On("CreateAsset", ctx, mock.Anything).Return(&Asset{ID: "a1"}, nil).Maybe()
	mockRepo.On("CreateTransaction", ctx, mock.Anything).Return(&Transaction{ID: "t1"}, nil).Maybe()

	res, err := svc.BulkAddTransactions(ctx, "p1", file)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 2, res.Success)
	assert.NotEmpty(t, res.Errors)
}

func TestService_BulkAddTreasuryTransactions(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Empty CSV
	emptyFile := &dummyFile{Reader: bytes.NewReader([]byte(""))}
	_, err := svc.BulkAddTreasuryTransactions(ctx, "p1", emptyFile)
	assert.ErrorContains(t, err, "vazio")

	// 2. CSV with comma fallback, header, and mixed rows
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
`

	mockRepo.On("GetTreasuryAssetByTicker", mock.Anything, mock.Anything, mock.Anything).Return("a1", nil).Maybe()
	mockRepo.On("CreateTreasurySubscription", mock.Anything, mock.Anything, "p1", "a1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("sub1", nil).Maybe()

	file := &dummyFile{Reader: bytes.NewReader([]byte(csvContent))}
	res, err := svc.BulkAddTreasuryTransactions(ctx, "p1", file)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 4, res.Success)
	assert.Len(t, res.Errors, 6)

	// 3. Comma delimited CSV
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

