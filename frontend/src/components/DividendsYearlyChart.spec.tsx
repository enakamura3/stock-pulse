import { render, screen } from '@testing-library/react';
import React from 'react';
import DividendsYearlyChart from './DividendsYearlyChart';
import { ThemeProvider } from './ThemeProvider';
import { CalculatedDividend } from './portfolio/types';

vi.mock('recharts', () => {
  return {
    ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
    BarChart: ({ children }: any) => <div data-testid="bar-chart">{children}</div>,
    Bar: () => null,
    XAxis: () => null,
    YAxis: ({ tickFormatter }: any) => (
      <div data-testid="yaxis">{tickFormatter ? tickFormatter(200) : null}</div>
    ),
    CartesianGrid: () => null,
    Tooltip: ({ content }: any) => (
      <div data-testid="tooltip-wrapper">
        {React.isValidElement(content)
          ? React.cloneElement(content as any, {
              active: true,
              payload: [{ value: 1234.56 }],
              label: '2024',
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
  };
});

describe('DividendsYearlyChart Component', () => {
  const mockData: CalculatedDividend[] = [
    {
      id: '1',
      asset_id: 'a1',
      ticker: 'PETR4',
      type: 'DIVIDENDO',
      payment_date: '2024-05-10T00:00:00Z',
      cum_date: '2024-04-10T00:00:00Z',
      gross_amount: 100,
      net_amount: 100,
      quantity: 10,
      rate_per_share: 10,
    },
    {
      id: '2',
      asset_id: 'a2',
      ticker: 'VALE3',
      type: 'JCP',
      payment_date: '0001-01-01T00:00:00Z', // fallback to cum_date
      cum_date: '2023-11-20T00:00:00Z',
      gross_amount: 50,
      net_amount: 42.5,
      quantity: 5,
      rate_per_share: 10,
    },
  ];

  afterEach(() => {
    localStorage.clear();
  });

  it('renders yearly chart with recharts and tooltip in light theme', () => {
    localStorage.setItem('theme', 'light');

    render(
      <ThemeProvider>
        <DividendsYearlyChart data={mockData} />
      </ThemeProvider>
    );

    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
    expect(screen.getByTestId('yaxis')).toHaveTextContent('R$200');
    expect(screen.getByText('Ano: 2024')).toBeInTheDocument();
    expect(screen.getByText('R$ 1234.56')).toBeInTheDocument();
  });

  it('renders yearly chart in dark theme', () => {
    localStorage.setItem('theme', 'dark');

    render(
      <ThemeProvider>
        <DividendsYearlyChart data={mockData} />
      </ThemeProvider>
    );

    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
  });

  it('renders empty fallback message when data is empty', () => {
    render(
      <ThemeProvider>
        <DividendsYearlyChart data={[]} />
      </ThemeProvider>
    );

    expect(screen.getByText('Nenhum dado anual')).toBeInTheDocument();
  });
});
