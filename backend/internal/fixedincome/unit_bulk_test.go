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
