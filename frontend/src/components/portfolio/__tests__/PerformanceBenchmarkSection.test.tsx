import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import PerformanceBenchmarkSection from '../analysis/PerformanceBenchmarkSection';
import { PerformancePoint } from '../types';

// Mock Recharts para ambiente de teste jsdom
vi.mock('recharts', async () => {
  const original = await vi.importActual<any>('recharts');
  return {
    ...original,
    ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
    LineChart: ({ children, data }: any) => (
      <div data-testid="line-chart" data-points={JSON.stringify(data)}>
        {children}
      </div>
    ),
    Line: ({ dataKey, name }: any) => (
      <div data-testid={`line-${dataKey}`} data-name={name} />
    ),
  };
});

describe('PerformanceBenchmarkSection', () => {
  const mockPerformanceData: PerformancePoint[] = [
    {
      date: '2024-01-01',
      value: 1000,
      total_invested: 1000,
      return_pct: 10,       // 10% nominal
      cdi_return_pct: 8,     // 8% nominal
      ipca_return_pct: 5,    // 5% IPCA
      ifix_return_pct: 6,
      ibov_return_pct: 12,
      sp500_return_pct: 15,
    },
  ];

  it('renders fallback when data is empty', () => {
    render(<PerformanceBenchmarkSection performanceData={[]} />);
    expect(screen.getByText(/Dados de performance insuficientes/i)).toBeInTheDocument();
  });

  it('renders nominal mode by default and shows IPCA line', () => {
    render(<PerformanceBenchmarkSection performanceData={mockPerformanceData} />);

    expect(screen.getByText(/Comparação com Benchmarks/i)).toBeInTheDocument();
    expect(screen.getByText(/📊 Retorno Nominal/i)).toBeInTheDocument();
    expect(screen.getByTestId('line-ipca')).toBeInTheDocument();
  });

  it('switches to real return mode when toggle button is clicked', () => {
    render(<PerformanceBenchmarkSection performanceData={mockPerformanceData} />);

    const toggleButton = screen.getByText(/📊 Retorno Nominal/i);
    fireEvent.click(toggleButton);

    // Button label changes to Real
    expect(screen.getByText(/📊 Retorno Real \(IPCA\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Rentabilidade real acumulada descontando a inflação/i)).toBeInTheDocument();

    // IPCA line is hidden in real mode
    expect(screen.queryByTestId('line-ipca')).not.toBeInTheDocument();

    // Verify deflated values in chart data attribute
    const chartEl = screen.getByTestId('line-chart');
    const chartData = JSON.parse(chartEl.getAttribute('data-points') || '[]');

    // Nominal: 10%, IPCA: 5% => Real = ((1 + 0.10) / (1 + 0.05) - 1) * 100 = (1.10 / 1.05 - 1) * 100 = 4.76%
    expect(chartData[0].portfolio).toBe(4.76);
  });
});
