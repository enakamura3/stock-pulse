import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import FundamentalHealthSection from '../FundamentalHealthSection';

describe('FundamentalHealthSection', () => {
  it('renders with low PVP (< 0.95), low PE (< 10), and DY, plus top/worst performers and valuation', () => {
    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'HGLG11',
        type: 'FII',
        pvp: 0.90,
        current_value: 10000,
        return_percent: 15.5,
        dividend_yield: 9.2,
      },
      {
        asset_id: '2',
        ticker: 'PETR4',
        type: 'STOCK_BR',
        pe: 4.5,
        current_value: 12000,
        return_percent: 25.0,
        dividend_yield: 12.0,
        current_price: 30,
        graham_value: 45, // discount > 0
        bazin_value: 40,   // discount > 0
      },
      {
        asset_id: '3',
        ticker: 'VALE3',
        type: 'STOCK_BR',
        pe: 6.0,
        current_value: 8000,
        return_percent: -10.2,
        current_price: 70,
        graham_value: 60, // discount < 0
        bazin_value: 50,   // discount < 0
      },
    ];

    const mockFi = [
      {
        id: 'fi-1',
        net_value: 5000,
        days_to_maturity: 0, // daily
      },
      {
        id: 'fi-2',
        net_value: 10000,
        days_to_maturity: 200, // up to 1 year
      },
      {
        id: 'fi-3',
        net_value: 15000,
        days_to_maturity: 700, // 1 to 3 years
      },
      {
        id: 'fi-4',
        net_value: 20000,
        days_to_maturity: 1500, // long term
      },
    ];

    const mockTreasury = [
      {
        transaction_id: 'td-1',
        ticker: 'TESOURO SELIC',
        net_value: 3000,
        days_to_maturity: 0,
      },
      {
        transaction_id: 'td-2',
        ticker: 'TESOURO 2026',
        net_value: 4000,
        days_to_maturity: 300,
      },
      {
        transaction_id: 'td-3',
        ticker: 'TESOURO 2028',
        net_value: 5000,
        days_to_maturity: 800,
      },
      {
        transaction_id: 'td-4',
        ticker: 'TESOURO 2035',
        net_value: 6000,
        days_to_maturity: 3000,
      },
    ];

    render(
      <FundamentalHealthSection
        positions={mockPositions as any}
        fiPositions={mockFi as any}
        treasuryPositions={mockTreasury as any}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Fundamentos da Carteira/)).toBeInTheDocument();
    expect(screen.getByText(/Abaixo do VP/)).toBeInTheDocument();
    expect(screen.getByText(/P\/L atrativo/)).toBeInTheDocument();
    expect(screen.getByText('DY Médio')).toBeInTheDocument();
    expect(screen.getByText(/Top Performers vs Piores/)).toBeInTheDocument();
    expect(screen.getByText('+25.0%')).toBeInTheDocument();
    expect(screen.getByText('-10.2%')).toBeInTheDocument();
    expect(screen.getByText('Preço Teto - Graham')).toBeInTheDocument();
    expect(screen.getByText('Preço Teto - Bazin')).toBeInTheDocument();
    expect(screen.getByText(/Liquidez da Renda Fixa/)).toBeInTheDocument();
  });

  it('renders PVP between 0.95 and 1.05 and PE between 10 and 18 with multiple FIIs and multiple stocks', () => {
    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'KNRI11',
        type: 'FII',
        pvp: 1.00,
        current_value: 10000,
      },
      {
        asset_id: '2',
        ticker: 'MXRF11',
        type: 'FII',
        pvp: 1.02,
        current_value: 10000,
      },
      {
        asset_id: '3',
        ticker: 'ITUB4',
        type: 'STOCK_BR',
        pe: 12.0,
        current_value: 10000,
      },
      {
        asset_id: '4',
        ticker: 'BBDC4',
        type: 'STOCK_BR',
        pe: 14.0,
        current_value: 10000,
      },
    ];

    render(
      <FundamentalHealthSection
        positions={mockPositions as any}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Próximo ao VP/)).toBeInTheDocument();
    expect(screen.getByText(/P\/L na média do mercado/)).toBeInTheDocument();
    expect(screen.queryByText(/Liquidez da Renda Fixa/)).not.toBeInTheDocument();
  });

  it('renders PVP > 1.05 and PE > 18, and empty valuation & performance', () => {
    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'XPML11',
        type: 'FII',
        pvp: 1.15,
        current_value: 10000,
      },
      {
        asset_id: '2',
        ticker: 'WEGE3',
        type: 'STOCK_BR',
        pe: 35.0,
        current_value: 10000,
      },
    ];

    render(
      <FundamentalHealthSection
        positions={mockPositions as any}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Acima do VP/)).toBeInTheDocument();
    expect(screen.getByText(/P\/L elevado/)).toBeInTheDocument();
    expect(screen.getByText('Sem dados de rentabilidade disponíveis.')).toBeInTheDocument();
    expect(screen.getByText('Não há dados suficientes de fundamentos para calcular margem de segurança.')).toBeInTheDocument();
  });

  it('renders placeholders when no FIIs and no stocks are provided', () => {
    render(
      <FundamentalHealthSection
        positions={[]}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Sem FIIs/FIAGROs com P/VP disponível')).toBeInTheDocument();
    expect(screen.getByText('Sem ações/ETFs com P/L disponível')).toBeInTheDocument();
  });
});
