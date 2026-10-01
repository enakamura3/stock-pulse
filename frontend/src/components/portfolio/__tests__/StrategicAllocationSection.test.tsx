import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import StrategicAllocationSection from '../analysis/StrategicAllocationSection';
import { Position, FixedIncomePosition, TreasuryPosition } from '../types';

// Mock Recharts
vi.mock('recharts', async () => {
  const original = await vi.importActual<any>('recharts');
  return {
    ...original,
    ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
    PieChart: ({ children }: any) => <div data-testid="pie-chart">{children}</div>,
    Pie: ({ children, label }: any) => {
      let labelOutput = null;
      if (typeof label === 'function') {
        labelOutput = (
          <div>
            {label({ cx: 50, cy: 50, midAngle: 45, innerRadius: 55, outerRadius: 90, pct: 1 })}
            {label({ cx: 50, cy: 50, midAngle: 45, innerRadius: 55, outerRadius: 90, pct: 25 })}
          </div>
        );
      }
      return <div>{children}{labelOutput}</div>;
    },
    Tooltip: ({ content }: any) => {
      if (content && React.isValidElement(content)) {
        return (
          <div>
            {/* Exercise inactive, empty, and active tooltip states */}
            {React.cloneElement(content as any, { active: false, payload: [] })}
            {React.cloneElement(content as any, { active: true, payload: [] })}
            {React.cloneElement(content as any, {
              active: true,
              payload: [{ payload: { name: 'Ações (B3)', value: 3000, pct: 50 } }],
            })}
          </div>
        );
      }
      return null;
    },
  };
});

describe('StrategicAllocationSection', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  const mockPositions: Position[] = [
    {
      asset_id: '1',
      ticker: 'PETR4',
      name: 'Petrobras',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 100,
      average_price: 20,
      total_cost: 2000,
      current_price: 30,
      current_value: 3000,
    },
    {
      asset_id: '2',
      ticker: 'HGLG11',
      name: 'CSHG Logística',
      type: 'FII',
      currency: 'BRL',
      quantity: 10,
      average_price: 150,
      total_cost: 1500,
      current_price: 100,
      current_value: 1000,
    },
    {
      asset_id: '3',
      ticker: 'AAPL',
      name: 'Apple Inc',
      type: 'BDR',
      currency: 'BRL',
      quantity: 10,
      average_price: 50,
      total_cost: 500,
      current_price: 100,
      current_value: 1000,
    },
  ];

  const mockFIPositions: FixedIncomePosition[] = [
    {
      id: 'fi-1',
      portfolio_id: 'p-1',
      asset_name: 'CDB Banco Inter',
      indexer: 'CDI',
      rate: 110,
      initial_investment: 2000,
      current_balance: 2100,
      gross_value: 2100,
      net_value: 2000,
      accrued_interest: 100,
      tax_amount: 15,
      start_date: '2023-01-01',
      due_date: '2025-01-01',
      tax_rate: 15,
      created_at: '2023-01-01',
      updated_at: '2023-01-01',
    },
  ];

  const mockTreasuryPositions: TreasuryPosition[] = [
    {
      id: 'td-1',
      portfolio_id: 'p-1',
      title: 'Tesouro Selic 2029',
      bond_type: 'SELIC',
      quantity: 1,
      invested_amount: 1000,
      gross_value: 1050,
      net_value: 1000,
      accrued_interest: 50,
      tax_amount: 7.5,
      b3_fee: 1,
      purchase_date: '2023-01-01',
      maturity_date: '2029-01-01',
      tax_rate: 15,
      created_at: '2023-01-01',
      updated_at: '2023-01-01',
    },
  ];

  it('renders "Sem dados" when all positions and fixed incomes are empty', () => {
    render(
      <StrategicAllocationSection
        positions={[]}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    const emptyStates = screen.getAllByText('Sem dados');
    expect(emptyStates.length).toBe(2);
    expect(screen.getByText('Clique em "🎯 Definir % Alvo" para configurar a alocação desejada por categoria.')).toBeInTheDocument();
  });

  it('handles localStorage errors on load and save gracefully', () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(Storage.prototype, 'getItem').mockImplementationOnce(() => {
      throw new Error('Storage disabled');
    });

    render(
      <StrategicAllocationSection
        positions={mockPositions}
        fiPositions={mockFIPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    expect(consoleErrorSpy).toHaveBeenCalledWith('Erro ao carregar metas de alocação:', expect.any(Error));

    // Now test save error
    vi.spyOn(Storage.prototype, 'setItem').mockImplementationOnce(() => {
      throw new Error('Quota exceeded');
    });

    const editBtn = screen.getByRole('button', { name: /Definir % Alvo/i });
    fireEvent.click(editBtn);

    const inputs = screen.getAllByRole('spinbutton');
    fireEvent.change(inputs[0], { target: { value: '50' } });

    expect(consoleErrorSpy).toHaveBeenCalledWith('Erro ao salvar metas de alocação:', expect.any(Error));
  });

  it('shows balanced message when all targets are met or exceeded', () => {
    // Current allocation:
    // Total = 8000
    // Ações = 37.5%, RF = 25%, TD = 12.5%, FIIs = 12.5%, BDRs = 12.5%
    // Set target for Ações to 10% (lower than current), rest 0
    localStorage.setItem(
      'stockpulse_target_allocation',
      JSON.stringify({ 'Ações (B3)': 10 })
    );

    render(
      <StrategicAllocationSection
        positions={mockPositions}
        fiPositions={mockFIPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Alocação Estratégica/)).toBeInTheDocument();
    expect(screen.getByText('Sua carteira está perfeitamente equilibrada em relação às metas definidas.')).toBeInTheDocument();
  });

  it('allows defining target allocation, warns when total != 100, and calculates suggestions', () => {
    render(
      <StrategicAllocationSection
        positions={mockPositions}
        fiPositions={mockFIPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    // Click target definition button
    const editBtn = screen.getByRole('button', { name: /Definir % Alvo/i });
    fireEvent.click(editBtn);

    expect(screen.getByText(/Defina o percentual alvo desejado/i)).toBeInTheDocument();

    const inputs = screen.getAllByRole('spinbutton');
    expect(inputs.length).toBeGreaterThanOrEqual(4);

    // Set target sum != 100 (e.g. 50% on first category)
    fireEvent.change(inputs[0], { target: { value: '50' } });
    expect(screen.getByText(/Soma atual das metas:/)).toBeInTheDocument();

    // Test entering empty value fallback to 0
    fireEvent.change(inputs[1], { target: { value: '' } });

    // Set target on FIIs to 50% so it becomes suballocated (target 50% > current 12.5%)
    fireEvent.change(inputs[1], { target: { value: '50' } });

    // Test superallocated, suballocated, and em meta status indicators
    expect(screen.getAllByText(/Subalocado|Superalocado|Em Meta/).length).toBeGreaterThan(0);

    // Toggle button back to close editing
    const saveBtn = screen.getByRole('button', { name: /Salvar Metas/i });
    fireEvent.click(saveBtn);
    expect(screen.queryByText(/Defina o percentual alvo desejado/i)).not.toBeInTheDocument();

    // Test contribution amount input with comma
    const contributionInput = screen.getByPlaceholderText('1000');
    fireEvent.change(contributionInput, { target: { value: '2,500' } });
    expect(screen.getAllByText(/Comprar em/i).length).toBeGreaterThan(0);

    // Test contribution amount <= 0
    fireEvent.change(contributionInput, { target: { value: '0' } });
    // When contributionVal <= 0, contributionSuggestions is empty
    expect(screen.queryByText(/Comprar em/i)).not.toBeInTheDocument();
  });
});
