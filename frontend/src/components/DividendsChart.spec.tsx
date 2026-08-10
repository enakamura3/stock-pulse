import { render, screen } from '@testing-library/react';
import DividendsChart from './DividendsChart';
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
    Legend: () => null,
  };
});

describe('DividendsChart Component', () => {
  const mockData = [
    {
      asset_id: 'div-1',
      ticker: 'PETR4',
      cum_date: '2026-05-01',
      payment_date: '2026-05-20',
      gross_amount: 100,
      net_amount: 100,
      currency: 'BRL',
    },
    {
      asset_id: 'div-2',
      ticker: 'AAPL34',
      cum_date: '2026-06-01',
      payment_date: '2026-06-15',
      gross_amount: 50,
      net_amount: 50,
      currency: 'USD',
      original_net_amount: 10,
    },
  ];

  it('renders recharts bar chart when data is available', () => {
    render(
      <ThemeProvider>
        <DividendsChart data={mockData} />
      </ThemeProvider>
    );

    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
  });

  it('renders fallback message when data is empty', () => {
    render(
      <ThemeProvider>
        <DividendsChart data={[]} />
      </ThemeProvider>
    );

    expect(screen.getByText('Gráfico indisponível')).toBeInTheDocument();
  });
});
