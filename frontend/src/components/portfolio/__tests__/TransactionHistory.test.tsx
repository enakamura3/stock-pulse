import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
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
    expect(screen.getByText('• PETR4')).toBeInTheDocument();
    expect(screen.getByText('• HGLG11')).toBeInTheDocument();

    // Hover and click category card to filter
    const actionsCard = screen.getByText(/📈 Ações/i).closest('div[style*="cursor: pointer"]')!;
    fireEvent.mouseEnter(actionsCard);
    fireEvent.mouseLeave(actionsCard);
    fireEvent.click(actionsCard); // Selects STOCK

    // Click again to unselect ('Todos')
    fireEvent.click(actionsCard);
  });

  it('triggers onLaunchOperation button when empty', () => {
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

    fireEvent.click(screen.getByText('+ Lançar primeira operação'));
    expect(mockOnLaunchOperation).toHaveBeenCalled();
  });

  it('handles pagination across multiple pages and actions on transactions', () => {
    const manyTransactions: UnifiedTransaction[] = Array.from({ length: 25 }, (_, i) => ({
      id: `tx-${i}`,
      portfolio_id: 'port-1',
      module: 'RV',
      date: currentDate(1),
      asset_name: `TICKER${i}`,
      asset_type: 'STOCK_BR',
      type: 'BUY',
      quantity: 10,
      unit_price: 20,
      total_value: 200,
      currency: 'BRL',
    }));

    render(
      <TransactionHistory
        transactions={manyTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Página 1 de 2')).toBeInTheDocument();

    const nextBtn = screen.getByRole('button', { name: /Próxima →/i });
    fireEvent.click(nextBtn);
    expect(screen.getByText('Página 2 de 2')).toBeInTheDocument();

    const prevBtn = screen.getByRole('button', { name: /← Anterior/i });
    fireEvent.click(prevBtn);
    expect(screen.getByText('Página 1 de 2')).toBeInTheDocument();

    // Trigger edit and delete on first item
    const editBtns = screen.getAllByTitle('Editar operação');
    fireEvent.click(editBtns[0]);
    expect(mockHandleEditTransaction).toHaveBeenCalled();

    const deleteBtns = screen.getAllByTitle('Excluir operação');
    fireEvent.click(deleteBtns[0]);
    expect(mockHandleDeleteTransaction).toHaveBeenCalled();
  });

  it('handles filter bar changes, clear filters, and empty category breakdown for sell-only transactions', () => {
    const sellTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-sell',
        portfolio_id: 'port-1',
        module: 'RV',
        date: currentDate(5),
        asset_name: 'PETR4',
        asset_type: 'STOCK_BR',
        type: 'SELL',
        quantity: 10,
        unit_price: 35,
        total_value: 350,
        currency: 'BRL',
      },
    ];

    render(
      <TransactionHistory
        transactions={sellTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Nenhuma aquisição no período filtrado.')).toBeInTheDocument();

    // Change ticker filter
    const tickerSelect = screen.getByDisplayValue('Ativo: Todos');
    fireEvent.change(tickerSelect, { target: { value: 'PETR4' } });

    // Click Limpar Filtros
    const clearBtn = screen.getByText(/Limpar Filtros/i);
    fireEvent.click(clearBtn);

    // Toggle Lista Simples / Por Data
    const listaSimplesBtn = screen.getByText('📄 Lista Simples');
    fireEvent.click(listaSimplesBtn);
  });

  it('handles BONUS, REVERSE_SPLIT, SUBSCRIPTION filter, and RF with currency conversion', () => {
    const mixedTransactions: UnifiedTransaction[] = [
      {
        id: 'tx-bonus',
        portfolio_id: 'port-1',
        module: 'RV',
        date: '2024-03-01',
        asset_name: 'PETR4',
        asset_type: 'STOCK_BR',
        type: 'BONUS',
        quantity: 5,
        unit_price: 30,
        total_value: 150,
        currency: 'BRL',
      },
      {
        id: 'tx-rev-split',
        portfolio_id: 'port-1',
        module: 'RV',
        date: '2024-03-02',
        asset_name: 'PETR4',
        asset_type: 'STOCK_BR',
        type: 'REVERSE_SPLIT',
        quantity: 2,
        unit_price: null,
        total_value: 0,
        currency: 'BRL',
      },
      {
        id: 'tx-rf-usd',
        portfolio_id: 'port-1',
        module: 'RF',
        date: '2024-03-03',
        asset_name: 'US Treasury Bond',
        asset_type: 'TESOURO',
        type: 'SUBSCRIPTION',
        quantity: null,
        unit_price: null,
        exchange_rate: 5.2,
        total_value: 1000,
        currency: 'USD',
      },
      {
        id: 'tx-rf-red',
        portfolio_id: 'port-1',
        module: 'RF',
        date: '2024-03-04',
        asset_name: 'US Treasury Bond',
        asset_type: 'TESOURO',
        type: 'REDEMPTION',
        quantity: null,
        unit_price: null,
        total_value: 500,
        currency: 'BRL',
      },
      {
        id: 'tx-other-date',
        portfolio_id: 'port-1',
        module: 'RV',
        date: '2023-01-01',
        asset_name: 'VALE3',
        asset_type: 'STOCK_BR',
        type: 'BUY',
        quantity: 10,
        unit_price: 60,
        total_value: 600,
        currency: 'BRL',
      },
    ];

    render(
      <TransactionHistory
        transactions={mixedTransactions}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('2 para 1')).toBeInTheDocument();
    expect(screen.getAllByText('(Câmbio: 5.2000)')[0]).toBeInTheDocument();

    // Filter by year 2024
    const yearSelect = screen.getByDisplayValue('Ano: Todos');
    fireEvent.change(yearSelect, { target: { value: '2024' } });

    // Filter by month 03
    const monthSelect = screen.getByDisplayValue('Mês: Todos');
    fireEvent.change(monthSelect, { target: { value: '03' } });

    // Filter by type SUBSCRIPTION
    const typeSelect = screen.getByDisplayValue('Tipo: Todos');
    fireEvent.change(typeSelect, { target: { value: 'SUBSCRIPTION' } });
  });

  it('handles current year with non-current month and specific transaction type filter', () => {
    const now = new Date();
    const curYear = String(now.getFullYear());
    const curMonth = String(now.getMonth() + 1).padStart(2, '0');
    const diffMonth = curMonth === '01' ? '02' : '01';

    const txs: UnifiedTransaction[] = [
      {
        id: 'tx-cur-year-diff-month-buy',
        portfolio_id: 'port-1',
        module: 'RV',
        date: `${curYear}-${diffMonth}-15`,
        asset_name: 'BBAS3',
        asset_type: 'STOCK_BR',
        type: 'BUY',
        quantity: 10,
        unit_price: 25,
        total_value: 250,
        currency: 'BRL',
      },
      {
        id: 'tx-cur-year-diff-month-sell',
        portfolio_id: 'port-1',
        module: 'RV',
        date: `${curYear}-${diffMonth}-16`,
        asset_name: 'BBAS3',
        asset_type: 'STOCK_BR',
        type: 'SELL',
        quantity: 5,
        unit_price: 30,
        total_value: 150,
        currency: 'BRL',
      },
    ];

    render(
      <TransactionHistory
        transactions={txs}
        filterTxTicker=""
        setFilterTxTicker={mockSetFilterTxTicker}
        handleEditTransaction={mockHandleEditTransaction}
        handleDeleteTransaction={mockHandleDeleteTransaction}
        onLaunchOperation={mockOnLaunchOperation}
        kpiCurrency="BRL"
      />
    );

    // Initial state: year is curYear, month is 'Todos'
    expect(screen.getByDisplayValue(curYear)).toBeInTheDocument();
    expect(screen.getByDisplayValue('Mês: Todos')).toBeInTheDocument();

    // Now filter by BUY type
    const typeSelect = screen.getByDisplayValue('Tipo: Todos');
    fireEvent.change(typeSelect, { target: { value: 'BUY' } });

    // BBAS3 buy should be present
    expect(screen.getAllByText('R$ 250,00')[0]).toBeInTheDocument();
  });
});


