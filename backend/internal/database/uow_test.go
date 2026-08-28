package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockTx struct {
	mock.Mock
}

func (m *mockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	a := m.Called(ctx, sql, args)
	return a.Get(0).(pgx.Rows), a.Error(1)
}

func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	a := m.Called(ctx, sql, args)
	return a.Get(0).(pgx.Row)
}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	if tx := args.Get(0); tx != nil {
		return tx.(pgx.Tx), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}
func (m *mockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}
func (m *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockTx) Conn() *pgx.Conn {
	return nil
}

type mockDBTX struct {
	mock.Mock
}

func (m *mockDBTX) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *mockDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	a := m.Called(ctx, sql, args)
	return a.Get(0).(pgx.Rows), a.Error(1)
}

func (m *mockDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	a := m.Called(ctx, sql, args)
	return a.Get(0).(pgx.Row)
}

func (m *mockDBTX) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	if tx := args.Get(0); tx != nil {
		return tx.(pgx.Tx), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestInjectExtractTx(t *testing.T) {
	ctx := context.Background()
	tx := new(mockTx)

	assert.Nil(t, ExtractTx(ctx))

	ctxTx := InjectTx(ctx, tx)
	assert.NotNil(t, ExtractTx(ctxTx))
	assert.Equal(t, tx, ExtractTx(ctxTx))
}

func TestGetDB(t *testing.T) {
	ctx := context.Background()
	tx := new(mockTx)
	defaultDB := new(mockDBTX)

	// Sem transação no contexto, deve retornar defaultDB
	res := GetDB(ctx, defaultDB)
	assert.Equal(t, defaultDB, res)

	// Com transação no contexto, deve retornar a transação
	ctxTx := InjectTx(ctx, tx)
	resTx := GetDB(ctxTx, defaultDB)
	assert.Equal(t, tx, resTx)
}

func TestUnitOfWork_Do_Success(t *testing.T) {
	pool := new(mockDBTX)
	tx := new(mockTx)

	pool.On("Begin", mock.Anything).Return(tx, nil)
	tx.On("Commit", mock.Anything).Return(nil)

	uow := NewUnitOfWork(pool)

	err := uow.Do(context.Background(), func(ctx context.Context) error {
		assert.NotNil(t, ExtractTx(ctx))
		return nil
	})

	assert.NoError(t, err)
	pool.AssertExpectations(t)
	tx.AssertExpectations(t)
}

func TestUnitOfWork_Do_Nested(t *testing.T) {
	pool := new(mockDBTX)
	tx := new(mockTx)

	uow := NewUnitOfWork(pool)

	ctxTx := InjectTx(context.Background(), tx)

	err := uow.Do(ctxTx, func(ctx context.Context) error {
		assert.Equal(t, tx, ExtractTx(ctx))
		return nil
	})

	assert.NoError(t, err)
	// pool.Begin nunca deve ser chamado
	pool.AssertExpectations(t)
}

func TestUnitOfWork_Do_BeginError(t *testing.T) {
	pool := new(mockDBTX)

	pool.On("Begin", mock.Anything).Return(nil, errors.New("begin error"))

	uow := NewUnitOfWork(pool)

	err := uow.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})

	assert.ErrorContains(t, err, "begin error")
}

func TestUnitOfWork_Do_FnError(t *testing.T) {
	pool := new(mockDBTX)
	tx := new(mockTx)

	pool.On("Begin", mock.Anything).Return(tx, nil)
	tx.On("Rollback", mock.Anything).Return(nil)

	uow := NewUnitOfWork(pool)

	err := uow.Do(context.Background(), func(ctx context.Context) error {
		return errors.New("business error")
	})

	assert.ErrorContains(t, err, "business error")
	tx.AssertExpectations(t)
}

func TestUnitOfWork_Do_Panic(t *testing.T) {
	pool := new(mockDBTX)
	tx := new(mockTx)

	pool.On("Begin", mock.Anything).Return(tx, nil)
	tx.On("Rollback", mock.Anything).Return(nil)

	uow := NewUnitOfWork(pool)

	assert.PanicsWithValue(t, "critical failure", func() {
		_ = uow.Do(context.Background(), func(ctx context.Context) error {
			panic("critical failure")
		})
	})

	tx.AssertExpectations(t)
}

func TestUnitOfWork_Do_CommitError(t *testing.T) {
	pool := new(mockDBTX)
	tx := new(mockTx)

	pool.On("Begin", mock.Anything).Return(tx, nil)
	tx.On("Commit", mock.Anything).Return(errors.New("commit failed"))

	uow := NewUnitOfWork(pool)

	err := uow.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})

	assert.ErrorContains(t, err, "commit failed")
	tx.AssertExpectations(t)
}
