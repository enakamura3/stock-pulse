import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import TransactionHistory from '../TransactionHistory';
import { UnifiedTransaction } from '../types';

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
    expect(screen.getByText('Sem transações registradas nesta carteira.')).toBeInTheDocument();
  });

  it('renders transactions and conditionally displays the exchange rate based on kpiCurrency', () => {
    const mockTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-1',
        portfolio_id: 'port-1',
        module: 'RV',
        date: '2026-06-01',
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

    // Não deve exibir o câmbio quando kpiCurrency é igual a currency
    expect(screen.queryByText('(Câmbio: 5.2500)')).not.toBeInTheDocument();
  });
});
