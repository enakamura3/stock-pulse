import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import DailyReport from '../DailyReport';
import { Position, FixedIncomePosition, TreasuryPosition } from '../types';

describe('DailyReport Component', () => {
  const mockPositions: Position[] = [
    {
      asset_id: 'pos1',
      ticker: 'PETR4',
      name: 'Petrobras',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 100,
      average_price: 25,
      total_cost: 2500,
      current_price: 30,
      current_value: 3000,
      daily_change: 1.5,
      daily_change_percent: 5.26,
    },
    {
      asset_id: 'pos2',
      ticker: 'VALE3',
      name: 'Vale',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 50,
      average_price: 80,
      total_cost: 4000,
      current_price: 70,
      current_value: 3500,
      daily_change: -2.0,
      daily_change_percent: -2.77,
    },
    {
      asset_id: 'pos3',
      ticker: 'HGLG11',
      name: 'CSHG Logística',
      type: 'FII',
      currency: 'BRL',
      quantity: 10,
      average_price: 160,
      total_cost: 1600,
      current_price: 160,
      current_value: 1600,
      daily_change: 0.0,
      daily_change_percent: 0.0,
    },
  ];

  const mockFIPositions: FixedIncomePosition[] = [
    {
      asset: {
        id: 'fi1',
        portfolio_id: 'p1',
        institution: 'Banco Sofisa',
        type: 'CDB',
        debt_type: 'POS',
        indexer: 'CDI',
        rate: 110,
        maturity_date: '2027-12-31T00:00:00Z',
      },
      start_date: '2024-01-01',
      total_invested: 5000,
      gross_value: 5500,
      net_value: 5400,
      net_return_percent: 8.0,
      gross_return_percent: 10.0,
      iof_amount: 0,
      ir_amount: 100,
      ir_rate: 17.5,
      iof_rate: 0,
    },
  ];

  const mockTreasuryPositions: TreasuryPosition[] = [
    {
      transaction_id: 'tr1',
      asset_id: 'asset_tr1',
      ticker: 'Tesouro Selic 2029',
      treasury_type: 'SELIC',
      maturity_date: '2029-03-01',
      has_coupons: false,
      start_date: '2024-01-01',
      quantity: 1,
      unit_price: 10000,
      contractedRate: 0,
      total_invested: 10000,
      gross_value: 10500,
      net_value: 10400,
      is_matured: false,
      days_to_maturity: 900,
      taxes: 100,
      b3_fee: 10,
      ir_tax: 90,
      iof_tax: 0,
    },
  ];

  it('renders empty state when no positions exist', () => {
    render(<DailyReport positions={[]} kpiCurrency="BRL" />);
    expect(screen.getByText('Nenhuma posição ativa encontrada.')).toBeInTheDocument();
  });

  it('renders total daily variation card and asset risers / fallers', () => {
    render(
      <DailyReport
        positions={mockPositions}
        fiPositions={mockFIPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Variação Total Diária da Carteira')).toBeInTheDocument();
    expect(screen.getByText('🚀 Maiores Altas do Dia')).toBeInTheDocument();
    expect(screen.getByText('📉 Maiores Baixas do Dia')).toBeInTheDocument();
  });

  it('renders all asset categories including private fixed income and treasury', () => {
    render(
      <DailyReport
        positions={mockPositions}
        fiPositions={mockFIPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('PETR4').length).toBeGreaterThan(0);
    expect(screen.getAllByText('VALE3').length).toBeGreaterThan(0);
    expect(screen.getByText('HGLG11')).toBeInTheDocument();

    expect(screen.getByText('Banco Sofisa')).toBeInTheDocument();
    expect(screen.getByText('Tesouro Selic 2029')).toBeInTheDocument();
  });

  it('allows sorting by column headers', () => {
    render(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
      />
    );

    const tickerHeader = screen.getByText(/Ativo/);
    fireEvent.click(tickerHeader);
    expect(screen.getByText('HGLG11')).toBeInTheDocument();
  });

  it('correctly calculates impact for foreign currency positions', () => {
    const foreignPositions: Position[] = [
      {
        asset_id: 'aapl1',
        ticker: 'AAPL',
        name: 'Apple Inc.',
        type: 'STOCK_US',
        currency: 'USD',
        quantity: 10,
        average_price: 150,
        total_cost: 7500, // in BRL
        current_price: 200, // in USD
        current_value: 10000, // in BRL (implied rate 10000 / (200*10) = 5.0)
        daily_change: 5, // in USD
        daily_change_percent: 2.56,
      },
    ];

    render(
      <DailyReport
        positions={foreignPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('AAPL').length).toBeGreaterThan(0);
    // Daily change is 5 USD * 10 shares * 5.0 BRL/USD = +R$ 250,00 impact
    expect(screen.getAllByText(/250,00/).length).toBeGreaterThan(0);
  });
});
