import { render, screen } from '@testing-library/react';
import DividendsChart from './DividendsChart';
import React from 'react';
import { ThemeProvider } from './ThemeProvider';
import { vi } from 'vitest';

vi.mock('recharts', () => {
  return {
    ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
    BarChart: ({ children }: any) => <div data-testid="bar-chart">{children}</div>,
    Bar: () => null,
    XAxis: () => null,
    YAxis: ({ tickFormatter }: any) => (
      <div data-testid="yaxis">{tickFormatter ? tickFormatter(150) : null}</div>
    ),
    CartesianGrid: () => null,
    Tooltip: ({ content }: any) => (
      <div data-testid="tooltip-wrapper">
        {React.isValidElement(content)
          ? React.cloneElement(content as any, {
              active: true,
              payload: [
                { name: 'Nacionais (R$)', value: 100, color: '#10b981' },
                { name: 'Internacionais (R$)', value: 50, color: '#3b82f6' },
              ],
              label: 'MAI/2026',
            })
          : null}
        {React.isValidElement(content)
          ? React.cloneElement(content as any, {
              active: false,
              payload: [],
              label: '',
            })
          : null}
      </div>
    ),
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
    {
      asset_id: 'div-3',
      ticker: 'TESOURO',
      cum_date: '2026-07-01',
      payment_date: '',
      gross_amount: 25,
      net_amount: 25,
      currency: 'BRL',
      is_accrued: true,
    },
  ];

  it('renders recharts bar chart, custom tooltip, and handles light theme', () => {
    // Set theme to light in localStorage
    localStorage.setItem('theme', 'light');

    render(
      <ThemeProvider>
        <DividendsChart data={mockData} />
      </ThemeProvider>
    );

    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
    expect(screen.getByTestId('yaxis')).toHaveTextContent('R$ 150');
    expect(screen.getByText('MAI/2026')).toBeInTheDocument();
    expect(screen.getByText('Total:')).toBeInTheDocument();
  });

  it('renders chart in dark theme', () => {
    localStorage.setItem('theme', 'dark');

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
