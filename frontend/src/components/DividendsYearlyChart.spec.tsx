import { render, screen } from '@testing-library/react';
import DividendsYearlyChart from './DividendsYearlyChart';
import React from 'react';
import { ThemeProvider } from './ThemeProvider';
import { vi } from 'vitest';

vi.mock('recharts', () => {
  const Original = vi.importActual('recharts');
  return {
    ...Original,
    ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
    BarChart: ({ children }: any) => <div data-testid="bar-chart">{children}</div>,
    Bar: () => null,
    XAxis: () => null,
    YAxis: () => null,
    CartesianGrid: () => null,
    Tooltip: () => null,
  };
});

describe('DividendsYearlyChart Component', () => {
  const mockData = [
    {
      asset_id: 'div-1',
      ticker: 'PETR4',
      cum_date: '2025-05-01',
      payment_date: '2025-05-20',
      gross_amount: 100,
      net_amount: 100,
      currency: 'BRL',
    },
    {
      asset_id: 'div-2',
      ticker: 'PETR4',
      cum_date: '2026-05-01',
      payment_date: '2026-05-20',
      gross_amount: 150,
      net_amount: 150,
      currency: 'BRL',
    },
  ];

  it('renders yearly recharts bar chart when data is available', () => {
    render(
      <ThemeProvider>
        <DividendsYearlyChart data={mockData} />
      </ThemeProvider>
    );

    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
  });

  it('renders fallback message when empty', () => {
    render(
      <ThemeProvider>
        <DividendsYearlyChart data={[]} />
      </ThemeProvider>
    );

    expect(screen.getByText('Nenhum dado anual')).toBeInTheDocument();
  });
});
