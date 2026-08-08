import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import StrategicAllocationSection from '../analysis/StrategicAllocationSection';
import { Position } from '../types';

// Mock Recharts
vi.mock('recharts', async () => {
  const original = await vi.importActual<any>('recharts');
  return {
    ...original,
    ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
    PieChart: ({ children }: any) => <div data-testid="pie-chart">{children}</div>,
    Pie: ({ children }: any) => <div>{children}</div>,
  };
});

describe('StrategicAllocationSection', () => {
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
  ];

  it('renders allocation breakdown and rebalancing section', () => {
    render(
      <StrategicAllocationSection
        positions={mockPositions}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Alocação Estratégica/i)).toBeInTheDocument();
    expect(screen.getByText(/Rebalanceamento Inteligente/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Definir % Alvo/i })).toBeInTheDocument();
  });


  it('allows defining target allocation and calculates suggestions', () => {
    render(
      <StrategicAllocationSection
        positions={mockPositions}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    // Click target definition button
    const editBtn = screen.getByRole('button', { name: /Definir % Alvo/i });
    fireEvent.click(editBtn);


    expect(screen.getByText(/Defina o percentual alvo desejado/i)).toBeInTheDocument();

    // Inputs for Ações (B3) and FIIs
    const inputs = screen.getAllByRole('spinbutton');
    expect(inputs.length).toBeGreaterThanOrEqual(2);

    // Set Ações target = 40%, FIIs target = 60%
    // Currently: Ações = 3000/4000 = 75%, FIIs = 1000/4000 = 25%
    // Sub-allocated: FIIs (Target 60% > Current 25%)
    fireEvent.change(inputs[0], { target: { value: '40' } });
    fireEvent.change(inputs[1], { target: { value: '60' } });

    // Verify contribution suggestion for FIIs
    expect(screen.getByText(/Comprar em FIIs/i)).toBeInTheDocument();
  });
});
