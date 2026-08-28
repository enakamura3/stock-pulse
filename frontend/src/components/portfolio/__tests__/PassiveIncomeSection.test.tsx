import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import PassiveIncomeSection from '../analysis/PassiveIncomeSection';
import { Position, CalculatedDividend } from '../types';

// Mock Recharts
vi.mock('recharts', async () => {
  const original = await vi.importActual<any>('recharts');
  return {
    ...original,
    ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
    BarChart: ({ children }: any) => <div>{children}</div>,
    Bar: () => <div />,
    AreaChart: ({ children }: any) => <svg data-testid="area-chart">{children}</svg>,

    Area: () => <div />,
  };
});

describe('PassiveIncomeSection', () => {
  const mockPositions: Position[] = [
    {
      asset_id: '1',
      ticker: 'PETR4',
      name: 'Petrobras',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 1000,
      average_price: 20,
      total_cost: 20000,
      current_price: 30,
      current_value: 30000,
    },
  ];

  const mockDividends: CalculatedDividend[] = [
    {
      asset_id: '1',
      ticker: 'PETR4',
      cum_date: '2024-01-01',
      payment_date: '2024-01-15',
      gross_amount: 3000,
      net_amount: 3000,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 1000,
      per_share_amount: 3,
      asset_type: 'STOCK_BR',
      asset_name: 'Petrobras',
    },
  ];

  it('renders income generation and snowball projection sections', () => {
    render(
      <PassiveIncomeSection
        positions={mockPositions}
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Geração de Renda/i)).toBeInTheDocument();
    expect(screen.getByText(/Efeito Bola de Neve & Projeção de Renda/i)).toBeInTheDocument();
    expect(screen.getByText(/Renda em 1 ano/i)).toBeInTheDocument();
    expect(screen.getByText(/Renda em 5 anos/i)).toBeInTheDocument();
    expect(screen.getByText(/Renda em 10 anos/i)).toBeInTheDocument();
  });

  it('allows altering monthly contribution and updates projections', () => {
    render(
      <PassiveIncomeSection
        positions={mockPositions}
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    const editContribBtn = screen.getByRole('button', { name: /Alterar/i });
    fireEvent.click(editContribBtn);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '2.000,00' } });

    const okBtn = screen.getByRole('button', { name: /OK/i });
    fireEvent.click(okBtn);

    // Verify updated contribution displays
    const matches = screen.getAllByText(/R\$\s*2\.000,00/i);
    expect(matches.length).toBeGreaterThanOrEqual(1);
  });

});
