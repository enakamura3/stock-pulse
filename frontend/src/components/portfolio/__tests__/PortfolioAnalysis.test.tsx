import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import PortfolioAnalysis from '../PortfolioAnalysis';

// Mock Recharts so we don't have issues with SVG rendering in jsdom
vi.mock('recharts', async () => {
  const OriginalRecharts = await vi.importActual<any>('recharts');
  return {
    ...OriginalRecharts,
    ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
    PieChart: () => <div data-testid="pie-chart" />,
    Pie: () => <div />,
    LineChart: () => <div data-testid="line-chart" />,
    Line: () => <div />,
    BarChart: () => <div data-testid="bar-chart" />,
    Bar: () => <div />,
  };
});

describe('PortfolioAnalysis', () => {
  it('renders without crashing when empty', () => {
    render(
      <PortfolioAnalysis
        positions={[]}
        dividends={[]}
        fiPositions={[]}
        performanceData={[]}
        kpiCurrency="BRL"
      />
    );
    expect(screen.getByText(/Adicione ativos à carteira para visualizar a análise completa/i)).toBeInTheDocument();
  });

  it('renders correctly with some data', () => {
    const mockPositions = [
      {
        asset_id: 1,
        ticker: 'PETR4',
        type: 'STOCK_BR',
        quantity: 100,
        average_price: 20,
        total_cost: 2000,
        current_price: 25,
        current_value: 2500,
        return_percent: 25,
        dividend_yield: 10,
        pe: 5,
        pvp: 1.2
      }
    ];

    const mockFiPositions = [
      {
        id: 1,
        asset_id: 2,
        ticker: 'CDB Banco X',
        type: 'CDB',
        index_type: 'CDI',
        invested_amount: 1000,
        net_value: 1100,
        days_to_maturity: 365,
        current_rate: 110,
        mature_date: '2027-01-01'
      }
    ];

    const mockDividends = [
      {
        asset_id: 1,
        ticker: 'PETR4',
        type: 'DIVIDEND',
        ex_date: '2023-01-01',
        payment_date: '2023-01-15',
        gross_amount: 100,
        net_amount: 100,
        quantity: 100
      }
    ];

    const mockPerformance = [
      { date: '2023-01-01', total_invested: 3000, value: 3000 },
      { date: '2023-02-01', total_invested: 3000, value: 3600 }
    ];

    expect(() => {
      render(
        <PortfolioAnalysis
          positions={mockPositions as any}
          dividends={mockDividends as any}
          fiPositions={mockFiPositions as any}
          performanceData={mockPerformance as any}
          kpiCurrency="BRL"
        />
      );
    }).not.toThrow();

    // Verify some elements are rendered
    expect(screen.getByText('Alocação Estratégica')).toBeInTheDocument();
  });
});
