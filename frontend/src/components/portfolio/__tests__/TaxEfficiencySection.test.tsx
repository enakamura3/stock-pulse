import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import TaxEfficiencySection from '../analysis/TaxEfficiencySection';
import { Position, CalculatedDividend, FixedIncomePosition, TreasuryPosition } from '../types';

describe('TaxEfficiencySection', () => {
  const mockPositions: Position[] = [
    {
      asset_id: '1',
      ticker: 'PETR4',
      name: 'Petrobras PN',
      type: 'STOCK_BR', // Isento
      currency: 'BRL',
      quantity: 100,
      average_price: 20,
      total_cost: 2000,
      current_price: 30,
      current_value: 3000,
    },
    {
      asset_id: '2',
      ticker: 'AAPL34',
      name: 'Apple BDR',
      type: 'BDR', // Tributavel
      currency: 'BRL',
      quantity: 10,
      average_price: 100,
      total_cost: 1000,
      current_price: 100,
      current_value: 1000,
    },
  ];

  const mockFiPositions: FixedIncomePosition[] = [
    {
      asset: {
        id: 'fi-1',
        portfolio_id: 'p1',
        institution: 'Banco Y',
        type: 'LCI', // Isento
        debt_type: 'POS',
        indexer: 'CDI',
        rate: 95,
        maturity_date: '2027-01-01',
      },
      start_date: '2024-01-01',
      total_invested: 2000,
      gross_value: 2200,
      net_value: 2200,
      net_return_percent: 10,
      gross_return_percent: 10,
      iof_amount: 0,
      ir_amount: 0,
      ir_rate: 0,
      iof_rate: 0,
      days_in_portfolio: 100,
      days_to_maturity: 500,
      is_matured: false,
    },
  ];

  const mockTreasuryPositions: TreasuryPosition[] = [
    {
      transaction_id: 't1',
      asset_id: 'a1',
      ticker: 'TESOURO SELIC 2029',
      treasury_type: 'SELIC',
      maturity_date: '2029-03-01',
      has_coupons: false,
      start_date: '2024-01-01',
      quantity: 1.0,
      unit_price: 10000,
      contracted_rate: 0,
      total_invested: 10000,
      gross_value: 11000,
      net_value: 10800,
      is_matured: false,
      days_to_maturity: 1000,
      taxes_calculated: 200,
      b3_fee: 15,
      ir_tax: 185,
      iof_tax: 0,
    },
  ];

  const mockDividends: CalculatedDividend[] = [
    {
      asset_id: '1',
      ticker: 'PETR4',
      cum_date: '2024-01-01',
      payment_date: '2024-01-15',
      gross_amount: 100,
      net_amount: 100,
      currency: 'BRL',
      type: 'Dividendo', // Isento
      quantity: 100,
      per_share_amount: 1,
      asset_type: 'STOCK_BR',
      asset_name: 'Petrobras',
    },
    {
      asset_id: '1',
      ticker: 'PETR4',
      cum_date: '2024-02-01',
      payment_date: '2024-02-15',
      gross_amount: 50,
      net_amount: 42.5,
      currency: 'BRL',
      type: 'JCP', // Tributavel na fonte
      quantity: 100,
      per_share_amount: 0.5,
      asset_type: 'STOCK_BR',
      asset_name: 'Petrobras',
    },
  ];

  it('renders tax efficiency section with calculated exemptions', () => {
    render(
      <TaxEfficiencySection
        positions={mockPositions}
        dividends={mockDividends}
        fiPositions={mockFiPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Eficiência Tributária/i)).toBeInTheDocument();
    expect(screen.getByText(/Patrimônio Isento de IR/i)).toBeInTheDocument();
    expect(screen.getByText(/Proventos Isentos/i)).toBeInTheDocument();
    expect(screen.getByText(/Impostos Retidos \(RF\/TD\)/i)).toBeInTheDocument();

    // Total Portfolio = 3000 (PETR4) + 1000 (AAPL34) + 2200 (LCI) + 10800 (TD) = 17000
    // Isento = 3000 (PETR4) + 2200 (LCI) = 5200 => 5200 / 17000 = 30.6%
    expect(screen.getByText(/30.6%/i)).toBeInTheDocument();

    // Proventos Isentos = 100 / (100 + 42.5) = 70.2%
    expect(screen.getByText(/70.2%/i)).toBeInTheDocument();

    // IR Retido TD = 185
    expect(screen.getByText(/R\$\s*185,00/i)).toBeInTheDocument();
  });
});

