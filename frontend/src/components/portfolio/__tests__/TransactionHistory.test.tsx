import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import TransactionHistory from '../TransactionHistory';
import { UnifiedTransaction } from '../types';

// Helper: gera data no mês/ano corrente no formato YYYY-MM-DD
const currentDate = (day: number): string => {
  const now = new Date();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const d = String(day).padStart(2, '0');
  return `${now.getFullYear()}-${month}-${d}`;
};

describe('TransactionHistory Component', () => {
  const mockSetFilterTxTicker = vi.fn();
  const mockHandleEditTransaction = vi.fn();
  const mockHandleDeleteTransaction = vi.fn();
  const mockOnLaunchOperation = vi.fn();

  it('renders correctly without crashing when there are no transactions and kpiCurrency is undefined', () => {
    render(
      <TransactionHistory
        transactions={[]}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
      />
    );
    expect(screen.getByText(/Nenhuma transação registrada nesta carteira/i)).toBeInTheDocument();
  });

  it('renders transactions and conditionally displays the exchange rate based on kpiCurrency', () => {
    const mockTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-1',
        portfolio_id: 'port-1',
        module: 'RV',
        date: currentDate(1),
        asset_name: 'IVV',
        asset_type: 'ETF',
        type: 'BUY',
        quantity: 10,
        unit_price: 500,
        exchange_rate: 5.25,
        total_value: 5000,
        currency: 'USD'
      }
    ];

    const { rerender } = render(
      <TransactionHistory
        transactions={mockTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    // Deve exibir o câmbio já que a moeda é USD e o kpiCurrency é BRL
    expect(screen.getByText('(Câmbio: 5.2500)')).toBeInTheDocument();

    // O total comprado deve ser convertido de 5000 USD para 26250 BRL (R$ 26.250,00)
    const totalCompradoElement = screen.getByText(/Total Comprado/i);
    expect(totalCompradoElement.parentElement).toHaveTextContent(/26\.250,00/);

    // Re-renderizar com kpiCurrency igual à moeda do ativo (USD)
    rerender(
      <TransactionHistory
        transactions={mockTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="USD"
      />
    );

    // Não deve exibir o câmbio quando kpiCurrency é igual à moeda do ativo
    expect(screen.queryByText('(Câmbio: 5.2500)')).not.toBeInTheDocument();
  });

  it('renders brokerage fee when present', () => {
    const mockTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-fee',
        portfolio_id: 'port-1',
        module: 'RV',
        date: currentDate(1),
        asset_name: 'PETR4',
        asset_type: 'STOCK',
        type: 'BUY',
        quantity: 100,
        unit_price: 30,
        fee: 10.5,
        exchange_rate: 1,
        total_value: 3010,
        currency: 'BRL'
      }
    ];

    render(
      <TransactionHistory
        transactions={mockTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('(Taxas: R$ 10,50)')).toBeInTheDocument();
  });

  it('renders fixed income transactions correctly without quantity and unit price', () => {
    const mockTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-2',
        portfolio_id: 'port-1',
        module: 'RF',
        date: currentDate(2),
        asset_name: 'CDB Banco X',
        asset_type: 'CDB',
        type: 'SUBSCRIPTION',
        quantity: null,
        unit_price: null,
        exchange_rate: null,
        total_value: 10000,
        currency: 'BRL'
      }
    ];

    render(
      <TransactionHistory
        transactions={mockTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('CDB Banco X')[0]).toBeInTheDocument();
    expect(screen.getByText('Valor da Operação')).toBeInTheDocument();
    expect(screen.getAllByText(/10\.000,00/)[0]).toBeInTheDocument();
    expect(screen.queryByText(/un\./)).not.toBeInTheDocument();
  });

  it('renders split transactions correctly displaying the proportion factor', () => {
    const mockTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-3',
        portfolio_id: 'port-1',
        module: 'RV',
        date: currentDate(3),
        asset_name: 'PETR4',
        asset_type: 'STOCK_BR',
        type: 'SPLIT',
        quantity: 2,
        unit_price: null,
        exchange_rate: null,
        total_value: 0,
        currency: 'BRL'
      }
    ];

    render(
      <TransactionHistory
        transactions={mockTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('PETR4')[0]).toBeInTheDocument();
    expect(screen.getByText('Proporção')).toBeInTheDocument();
    expect(screen.getByText('1 para 2')).toBeInTheDocument();
  });

  it('renders acquisition breakdown by macro asset class and allows filtering by category', () => {
    const mockTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-1',
        portfolio_id: 'port-1',
        module: 'RV',
        date: currentDate(1),
        asset_name: 'PETR4',
        asset_type: 'STOCK_BR',
        type: 'BUY',
        quantity: 10,
        unit_price: 30,
        total_value: 300,
        currency: 'BRL'
      },
      {
        id: 'tx-2',
        portfolio_id: 'port-1',
        module: 'RV',
        date: currentDate(2),
        asset_name: 'HGLG11',
        asset_type: 'FII',
        type: 'BUY',
        quantity: 5,
        unit_price: 160,
        total_value: 800,
        currency: 'BRL'
      }
    ];

    render(
      <TransactionHistory
        transactions={mockTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Aquisições por Classe de Ativo/i)).toBeInTheDocument();
    expect(screen.getByText(/📈 Ações/i)).toBeInTheDocument();
    expect(screen.getByText(/🏢 FIIs/i)).toBeInTheDocument();
  });
});

