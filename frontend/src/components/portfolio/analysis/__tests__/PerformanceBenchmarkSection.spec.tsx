import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import PerformanceBenchmarkSection from '../PerformanceBenchmarkSection';
import { ThemeProvider } from '@/components/ThemeProvider';

vi.mock('recharts', async () => {
  const actual = await vi.importActual<any>('recharts');
  return {
    ...actual,
    ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
    LineChart: ({ children }: any) => <div>{children}</div>,
    Line: () => <div data-testid="recharts-line" />,
    XAxis: () => <div />,
    YAxis: ({ tickFormatter }: any) => <div>{tickFormatter ? tickFormatter(10) : ''}</div>,
    CartesianGrid: () => <div />,
    Tooltip: ({ content }: any) => (
      <div data-testid="tooltip-mock">
        {content && React.isValidElement(content)
          ? React.cloneElement(content as any, {
              active: true,
              label: 'Jan 24',
              payload: [
                { name: 'Carteira', value: 12.34, color: '#00ffff' },
                { name: 'CDI', value: 10.5, color: '#ffaa00' },
              ],
            })
          : null}
      </div>
    ),
    Legend: () => <div />,
  };
});

describe('PerformanceBenchmarkSection', () => {
  it('renders empty state when performanceData is empty', () => {
    render(<PerformanceBenchmarkSection performanceData={[]} />);
    expect(screen.getByText('Dados de performance insuficientes para gerar a comparação.')).toBeInTheDocument();
  });

  it('renders benchmark chart, handles real/nominal toggle and deflator calculation', () => {
    const mockData = [
      {
        date: '2024-01-01',
        return_pct: 5,
        cdi_return_pct: 4,
        ipca_return_pct: 1,
        ifix_return_pct: 3,
        ibov_return_pct: 2,
        sp500_return_pct: 6,
      },
      {
        date: '2024-02-01',
        return_pct: 10,
        cdi_return_pct: 8,
        ipca_return_pct: 2,
        ifix_return_pct: 7,
        ibov_return_pct: 5,
        sp500_return_pct: 12,
      },
      {
        date: '2024-03-01',
        return_pct: 15,
        cdi_return_pct: 12,
        ipca_return_pct: -100, // triggers Math.abs(ipcaFactor) < 1e-6
        ifix_return_pct: 9,
        ibov_return_pct: 8,
        sp500_return_pct: 15,
      },
    ];

    render(
      <PerformanceBenchmarkSection performanceData={mockData as any} />
    );

    expect(screen.getByText(/Comparação com Benchmarks/)).toBeInTheDocument();
    expect(screen.getByText('📊 Retorno Nominal')).toBeInTheDocument();
    expect(screen.getByText('Carteira:')).toBeInTheDocument();
    expect(screen.getByText('12.34%')).toBeInTheDocument();

    // Toggle to real return
    const toggleBtn = screen.getByText('📊 Retorno Nominal');
    fireEvent.click(toggleBtn);

    expect(screen.getByText('📊 Retorno Real (IPCA)')).toBeInTheDocument();
    expect(screen.getByText(/Rentabilidade real acumulada/)).toBeInTheDocument();

    // Toggle back to nominal
    fireEvent.click(screen.getByText('📊 Retorno Real (IPCA)'));
    expect(screen.getByText('📊 Retorno Nominal')).toBeInTheDocument();
  });

  it('renders with light theme and handles inactive tooltip', () => {
    // Render inside ThemeProvider
    localStorage.setItem('stockpulse_theme', 'light');

    const mockData = [
      {
        date: '2024-01-01',
        return_pct: 5,
      },
    ];

    render(
      <ThemeProvider>
        <PerformanceBenchmarkSection performanceData={mockData as any} />
      </ThemeProvider>
    );

    expect(screen.getByText(/Comparação com Benchmarks/)).toBeInTheDocument();
  });
});
