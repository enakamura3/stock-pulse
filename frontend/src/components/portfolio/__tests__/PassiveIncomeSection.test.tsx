import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PassiveIncomeSection from '../analysis/PassiveIncomeSection';
import { Position, CalculatedDividend } from '../types';

let mockTheme = 'light';
vi.mock('@/components/ThemeProvider', () => ({
  useThemeOptional: () => ({ theme: mockTheme }),
}));

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
    Tooltip: ({ content }: any) => {
      if (content && React.isValidElement(content)) {
        return (
          <div>
            {React.cloneElement(content as any, { active: false, payload: [] })}
            {React.cloneElement(content as any, { active: true, payload: [] })}
            {React.cloneElement(content as any, {
              active: true,
              label: 'Jan/24',
              payload: [
                { name: 'Nacionais (R$)', value: 100, color: '#4ade80', payload: { monthlyIncome: 150, portfolio: 10000 } },
                { name: 'Internacionais (R$)', value: 50, color: '#00f2fe', payload: { monthlyIncome: 150, portfolio: 10000 } },
              ],
            })}
          </div>
        );
      }
      return null;
    },
    XAxis: () => <div />,
    YAxis: ({ tickFormatter }: any) => <div>{tickFormatter ? tickFormatter(100) : null}</div>,
    CartesianGrid: () => <div />,
    Legend: () => <div />,
  };
});

describe('PassiveIncomeSection', () => {
  beforeEach(() => {
    localStorage.clear();
    mockTheme = 'light';
    vi.restoreAllMocks();
  });

  const now = new Date();
  const curY = now.getFullYear();
  const curM = String(now.getMonth() + 1).padStart(2, '0');
  const pastDate = `${curY}-${curM}-01`;
  const futureYear = curY + 1;
  const farFutureDate = `${curY + 3}-01-01`;

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
      cum_date: pastDate,
      payment_date: `${curY}-${curM}-10`,
      gross_amount: 3000,
      net_amount: 3000,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 1000,
      per_share_amount: 3,
      asset_type: 'STOCK_BR',
      asset_name: 'Petrobras',
    },
    {
      asset_id: '2',
      ticker: 'AAPL',
      cum_date: pastDate,
      payment_date: `${curY}-${curM}-12`,
      gross_amount: 500,
      net_amount: 500,
      original_net_amount: 100,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 50,
      per_share_amount: 10,
      asset_type: 'BDR',
      asset_name: 'Apple',
    },
    {
      asset_id: '3',
      ticker: 'CDB',
      cum_date: pastDate,
      payment_date: `${curY}-${curM}-15`,
      gross_amount: 200,
      net_amount: 200,
      is_accrued: true,
      currency: 'BRL',
      type: 'Rendimento',
      quantity: 1,
      per_share_amount: 200,
      asset_type: 'RF',
      asset_name: 'CDB Inter',
    },
    {
      asset_id: '1',
      ticker: 'PETR4',
      cum_date: `${futureYear}-02-01`,
      payment_date: `${futureYear}-02-15`, // upcoming in future
      gross_amount: 1500,
      net_amount: 1500,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 1000,
      per_share_amount: 1.5,
      asset_type: 'STOCK_BR',
      asset_name: 'Petrobras',
    },
    {
      asset_id: '1',
      ticker: 'PETR4',
      cum_date: `${curY - 3}-01-01`, // old dividend > 12m ago
      payment_date: `${curY - 3}-01-15`,
      gross_amount: 800,
      net_amount: 800,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 1000,
      per_share_amount: 0.8,
      asset_type: 'STOCK_BR',
      asset_name: 'Petrobras',
    },
    {
      asset_id: '4',
      ticker: 'VALE3',
      cum_date: '0001-01-01', // no payment_date and invalid cum_date
      payment_date: '',
      gross_amount: 100,
      net_amount: 100,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 10,
      per_share_amount: 10,
      asset_type: 'STOCK_BR',
      asset_name: 'Vale',
    },
    {
      asset_id: '4b',
      ticker: 'VALE3',
      cum_date: 'nodashdate', // parts.length < 2, returns null on line 103
      payment_date: '',
      gross_amount: 50,
      net_amount: 50,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 5,
      per_share_amount: 10,
      asset_type: 'STOCK_BR',
      asset_name: 'Vale',
    },
    {
      asset_id: '5',
      ticker: 'ITUB4',
      cum_date: farFutureDate, // far future > maxFuture (11 months)
      payment_date: farFutureDate,
      gross_amount: 2000,
      net_amount: 2000,
      currency: 'BRL',
      type: 'JCP',
      quantity: 100,
      per_share_amount: 20,
      asset_type: 'STOCK_BR',
      asset_name: 'Itaú',
    },
  ];

  it('renders correctly with empty data in dark theme', () => {
    mockTheme = 'dark';
    render(
      <PassiveIncomeSection
        positions={[]}
        dividends={[]}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Nenhum provento registrado nos últimos 12 meses.')).toBeInTheDocument();
    expect(screen.getByText('Defina uma meta mensal de renda passiva para acompanhar seu progresso.')).toBeInTheDocument();
  });

  it('loads saved goal and contribution from localStorage and renders upcoming dividends', () => {
    localStorage.setItem('stockpulse_monthly_goal', '2500');
    localStorage.setItem('stockpulse_monthly_contribution', '1500');

    render(
      <PassiveIncomeSection
        positions={mockPositions}
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Geração de Renda/i)).toBeInTheDocument();
    expect(screen.getByText(/💰 Proventos a Receber/i)).toBeInTheDocument();
    expect(screen.getByText('Renda em 1 ano')).toBeInTheDocument();
  });

  it('allows editing monthly goal, handles Enter keydown, save, and progress bars', () => {
    render(
      <PassiveIncomeSection
        positions={mockPositions}
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    // Initial state: goal is 0, button says 'Definir'
    const setGoalBtn = screen.getByRole('button', { name: /Definir/i });
    fireEvent.click(setGoalBtn);

    const goalInput = screen.getByPlaceholderText('Ex: 1.000,00');

    // Test saving invalid goal (does not update)
    fireEvent.change(goalInput, { target: { value: 'invalid' } });
    fireEvent.click(screen.getByRole('button', { name: 'Salvar' }));
    expect(screen.getByText('Defina uma meta mensal de renda passiva para acompanhar seu progresso.')).toBeInTheDocument();

    // Reopen and enter low goal so coverage is >= 100%
    fireEvent.click(screen.getByRole('button', { name: /Definir/i }));
    const goalInput2 = screen.getByPlaceholderText('Ex: 1.000,00');
    fireEvent.change(goalInput2, { target: { value: '100,00' } });
    fireEvent.keyDown(goalInput2, { key: 'Enter' });

    expect(screen.getByText(/Parabéns! Sua renda passiva já cobre 100% da sua meta mensal!/i)).toBeInTheDocument();

    // Now edit goal so coverage is between 60% and 99%
    // Avg monthly is 7000 / 12 = 583.33
    // If goal = 700 -> coverage = 583.33 / 700 = ~83.3%
    fireEvent.click(screen.getByRole('button', { name: /Editar/i }));
    const goalInput3 = screen.getByPlaceholderText('Ex: 1.000,00');
    fireEvent.change(goalInput3, { target: { value: '700,00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Salvar' }));

    expect(screen.getByText(/Faltam/i)).toBeInTheDocument();

    // Now edit goal so coverage is < 60% (e.g. 1500 -> 583.33 / 1500 = 38.8%)
    fireEvent.click(screen.getByRole('button', { name: /Editar/i }));
    const goalInput4 = screen.getByPlaceholderText('Ex: 1.000,00');
    fireEvent.change(goalInput4, { target: { value: '1.500,00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Salvar' }));

    expect(screen.getByText(/Faltam/i)).toBeInTheDocument();
  });

  it('allows altering monthly contribution, handles Enter keydown, and invalid input', () => {
    render(
      <PassiveIncomeSection
        positions={mockPositions}
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    const editContribBtn = screen.getByRole('button', { name: /Alterar/i });
    fireEvent.click(editContribBtn);

    const contribInput = screen.getByPlaceholderText('1.000,00');

    // Invalid contribution does not save
    fireEvent.change(contribInput, { target: { value: 'abc' } });
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

    // Reopen and enter valid contribution via Enter key
    fireEvent.click(screen.getByRole('button', { name: /Alterar/i }));
    const contribInput2 = screen.getByPlaceholderText('1.000,00');
    fireEvent.change(contribInput2, { target: { value: '2.500,00' } });
    fireEvent.keyDown(contribInput2, { key: 'Enter' });

    expect(screen.getByText('R$ 2.500,00')).toBeInTheDocument();
  });
});
