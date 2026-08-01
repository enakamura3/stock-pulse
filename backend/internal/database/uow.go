package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX define a interface unificada para interações com o PostgreSQL.
// Abstrai tanto pgxpool.Pool quanto pgx.Tx, permitindo o uso fluido de UnitOfWork.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type txKey struct{}

// InjectTx insere uma transação ativa no Contexto para ser propagada.
func InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ExtractTx recupera uma transação ativa do Contexto.
func ExtractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// GetDB resolve o executor de banco de dados apropriado.
// Se houver uma transação (UnitOfWork) rodando no Contexto, ele a utiliza.
// Caso contrário, faz um fallback para o *pgxpool.Pool padrão fornecido pelo repositório.
func GetDB(ctx context.Context, defaultDB DBTX) DBTX {
	if tx := ExtractTx(ctx); tx != nil {
		return tx
	}
	return defaultDB
}

// UnitOfWork gerencia a atomicidade de transações a nível de Service (Cross-Repository).
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type uow struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork cria um novo gerenciador transacional.
func NewUnitOfWork(pool *pgxpool.Pool) UnitOfWork {
	return &uow{pool: pool}
}

// Do executa a função anonima garantindo contexto transacional.
func (u *uow) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// Reaproveita transação se já existir uma na hierarquia do contexto
	if ExtractTx(ctx) != nil {
		return fn(ctx)
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("falha ao iniciar UnitOfWork: %w", err)
	}

	// Safety net para Panic
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	// Injeta a transação e executa
	txCtx := InjectTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("falha ao commitar UnitOfWork: %w", err)
	}

	return nil
}
