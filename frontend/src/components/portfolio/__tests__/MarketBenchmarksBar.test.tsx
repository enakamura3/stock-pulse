import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import MarketBenchmarksBar from '../MarketBenchmarksBar';
import { MarketBenchmarks } from '../types';

describe('MarketBenchmarksBar Component', () => {
  const mockBenchmarks: MarketBenchmarks = {
    ibov: {
      symbol: '^BVSP',
      name: 'Ibovespa',
      value: 130500.5,
      change: 1200.5,
      change_percent: 0.93,
    },
    sp500: {
      symbol: '^GSPC',
      name: 'S&P 500',
      value: 5500.25,
      change: -15.5,
      change_percent: -0.28,
    },
    usd_brl: {
      symbol: 'BRL=X',
      name: 'Dólar Comercial',
      value: 5.45,
      change: -0.02,
      change_percent: -0.37,
    },
    ifix: {
      symbol: 'IFIX.SA',
      name: 'IFIX',
      value: 3350.0,
      change: 5.0,
      change_percent: 0.15,
    },
  };

  it('renders null when benchmarks is undefined or empty', () => {
    const { container: c1 } = render(<MarketBenchmarksBar />);
    expect(c1.firstChild).toBeNull();

    const { container: c2 } = render(<MarketBenchmarksBar benchmarks={null} />);
    expect(c2.firstChild).toBeNull();

    const { container: c3 } = render(<MarketBenchmarksBar benchmarks={{}} />);
    expect(c3.firstChild).toBeNull();
  });

  it('renders loading skeleton when isLoading is true and no benchmarks exist', () => {
    const { container } = render(<MarketBenchmarksBar isLoading={true} />);
    expect(container.firstChild).not.toBeNull();
  });

  it('renders all benchmark cards with correct labels, values, and percentage badges', () => {
    render(<MarketBenchmarksBar benchmarks={mockBenchmarks} />);

    expect(screen.getByText('🌐 Índices e Benchmarks de Mercado')).toBeInTheDocument();
    expect(screen.getByText('Ibovespa')).toBeInTheDocument();
    expect(screen.getByText('+0.93%')).toBeInTheDocument();

    expect(screen.getByText('S&P 500')).toBeInTheDocument();
    expect(screen.getByText('-0.28%')).toBeInTheDocument();

    expect(screen.getByText('Dólar')).toBeInTheDocument();
    expect(screen.getByText('R$ 5,45')).toBeInTheDocument();
    expect(screen.getByText('-0.37%')).toBeInTheDocument();

    expect(screen.getByText('IFIX')).toBeInTheDocument();
    expect(screen.getByText('+0.15%')).toBeInTheDocument();
  });

  it('renders partial benchmarks when only some items are available', () => {
    const partialBenchmarks: MarketBenchmarks = {
      ibov: {
        symbol: '^BVSP',
        name: 'Ibovespa',
        value: 130000,
        change: 0,
        change_percent: 0,
      },
    };

    render(<MarketBenchmarksBar benchmarks={partialBenchmarks} />);
    expect(screen.getByText('Ibovespa')).toBeInTheDocument();
    expect(screen.queryByText('S&P 500')).not.toBeInTheDocument();
    expect(screen.queryByText('Dólar')).not.toBeInTheDocument();
  });

  it('handles invalid or non-numeric benchmark values gracefully', () => {
    const invalidBenchmarks: MarketBenchmarks = {
      ibov: {
        symbol: '^BVSP',
        name: 'Ibovespa',
        value: undefined as any,
        change: 0,
        change_percent: 0,
      },
    };

    render(<MarketBenchmarksBar benchmarks={invalidBenchmarks} />);
    expect(screen.getByText('-')).toBeInTheDocument();
  });
});
