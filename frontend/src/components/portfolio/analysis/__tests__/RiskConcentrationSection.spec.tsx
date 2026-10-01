import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import RiskConcentrationSection from '../RiskConcentrationSection';

describe('RiskConcentrationSection', () => {
  it('renders empty placeholder when performanceData has fewer than 10 points', () => {
    render(
      <RiskConcentrationSection
        positions={[]}
        fiPositions={[]}
        treasuryPositions={[]}
        performanceData={[{ date: '2024-01-01', value: 100 }]}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Dados de performance insuficientes para calcular métricas de risco.')).toBeInTheDocument();
  });

  it('renders full risk scorecards when performanceData >= 10 points and handles expansion and all metrics', () => {
    // Generate 15 points of performance data with positive trend
    const mockPerf = Array.from({ length: 15 }, (_, i) => ({
      date: `2024-01-${String(i + 1).padStart(2, '0')}`,
      value: 1000 + i * 20 - (i === 5 ? 50 : 0), // some volatility and drawdown
    }));

    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        name: 'Petrobras',
        type: 'STOCK_BR',
        current_value: 5000,
        total_cost: 4000,
        profit_loss: 1000, // gainer
        return_percent: 25,
      },
      {
        asset_id: '2',
        ticker: 'VALE3',
        name: 'Vale',
        type: 'STOCK_BR',
        current_value: 3000,
        total_cost: 3500,
        profit_loss: -500, // loser
        return_percent: -14.28,
      },
      {
        asset_id: '3',
        ticker: 'HGLG11',
        name: 'CSHG Logística',
        type: 'FII',
        current_value: 2000,
        total_cost: 2000,
        profit_loss: 0,
        return_percent: 0,
      },
    ];

    const mockFi = [
      {
        id: 'fi-1',
        net_value: 4000,
        total_invested: 3500,
        net_return_percent: 14.28,
        asset: { type: 'CDB', indexer: 'PREFIXADO', institution: 'Banco Inter' },
      },
    ];

    const mockTreasury = [
      {
        transaction_id: 'td-1',
        ticker: 'TESOURO SELIC',
        net_value: 6000,
        total_invested: 5500,
      },
    ];

    render(
      <RiskConcentrationSection
        positions={mockPositions as any}
        fiPositions={mockFi as any}
        treasuryPositions={mockTreasury as any}
        performanceData={mockPerf as any}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Termômetro de Risco/)).toBeInTheDocument();
    expect(screen.getByText('Índice de Sharpe')).toBeInTheDocument();
    expect(screen.getByText('Atribuição de Performance')).toBeInTheDocument();
    expect(screen.getByText('Beta')).toBeInTheDocument();
    expect(screen.getByText('Drawdown Máximo')).toBeInTheDocument();

    // Expand all KPI scorecards to render details
    const expandButtons = screen.getAllByText('Ver detalhes e ativos');
    expandButtons.forEach(btn => fireEvent.click(btn));

    expect(screen.getByText(/Ativos mais lucrativos/)).toBeInTheDocument();
    expect(screen.getByText(/Ativos detratores/)).toBeInTheDocument();
    expect(screen.getByText(/Maiores Contribuidores/)).toBeInTheDocument();
    expect(screen.getByText(/Maiores Detratores/)).toBeInTheDocument();
    expect(screen.getByText('Perfil de Oscilação da Carteira')).toBeInTheDocument();
    expect(screen.getByText('Concentração nos 3 Maiores Ativos')).toBeInTheDocument();
  });

  it('handles negative totalContribution, zero gainers/losers, and high concentration', () => {
    // Generate 15 points with negative trend (declining value)
    const mockPerf = Array.from({ length: 15 }, (_, i) => ({
      date: `2024-01-${String(i + 1).padStart(2, '0')}`,
      value: 2000 - i * 50, // continuous drop: severe drawdown, negative return
    }));

    // Single position with 100% weight and 0 profitLoss
    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'FLAT1',
        type: 'CASH',
        current_value: 1000,
        total_cost: 1000,
        profit_loss: 0,
        return_percent: 0,
      },
    ];

    render(
      <RiskConcentrationSection
        positions={mockPositions as any}
        fiPositions={[]}
        treasuryPositions={[]}
        performanceData={mockPerf as any}
        kpiCurrency="BRL"
      />
    );

    const expandButtons = screen.getAllByText('Ver detalhes e ativos');
    expandButtons.forEach(btn => fireEvent.click(btn));

    expect(screen.getByText('Nenhum ativo com ganhos ou perdas significativos.')).toBeInTheDocument();
    expect(screen.getByText('Sem dados suficientes para calcular a contribuição individual dos ativos.')).toBeInTheDocument();
  });

  it('handles moderate sharpe, beta in market range, and top3Sum thresholds', () => {
    // Generate 15 points with mild fluctuation
    const mockPerf = [
      { date: '2024-01-01', value: 1000 },
      { date: '2024-01-02', value: 1013 },
      { date: '2024-01-03', value: 1001.86 },
      { date: '2024-01-04', value: 1014.88 },
      { date: '2024-01-05', value: 1003.72 },
      { date: '2024-01-06', value: 1016.77 },
      { date: '2024-01-07', value: 1005.59 },
      { date: '2024-01-08', value: 1018.66 },
      { date: '2024-01-09', value: 1007.45 },
      { date: '2024-01-10', value: 1020.55 },
      { date: '2024-01-11', value: 1009.32 },
      { date: '2024-01-12', value: 1022.44 },
      { date: '2024-01-13', value: 1011.19 },
      { date: '2024-01-14', value: 1024.34 },
      { date: '2024-01-15', value: 1013.07 },
    ];

    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'ASSET1',
        type: 'STOCK_BR',
        current_value: 4000,
        total_cost: 3800,
        profit_loss: 200,
        return_percent: 5.26,
      },
      {
        asset_id: '2',
        ticker: 'ASSET2',
        type: 'STOCK_BR',
        current_value: 3500,
        total_cost: 3300,
        profit_loss: 200,
        return_percent: 6.06,
      },
      {
        asset_id: '3',
        ticker: 'ASSET3',
        type: 'STOCK_BR',
        current_value: 2500,
        total_cost: 2400,
        profit_loss: 100,
        return_percent: 4.16,
      },
    ];

    render(
      <RiskConcentrationSection
        positions={mockPositions as any}
        fiPositions={[]}
        treasuryPositions={[]}
        performanceData={mockPerf as any}
        kpiCurrency=""
      />
    );

    expect(screen.getByText('Índice de Sharpe')).toBeInTheDocument();
  });

  it('handles dailyReturns.length < 5 when points have zero value', () => {
    const mockPerf = [
      { date: '2024-01-01', value: 0 },
      { date: '2024-01-02', value: 0 },
      { date: '2024-01-03', value: 0 },
      { date: '2024-01-04', value: 0 },
      { date: '2024-01-05', value: 0 },
      { date: '2024-01-06', value: 0 },
      { date: '2024-01-07', value: 0 },
      { date: '2024-01-08', value: 0 },
      { date: '2024-01-09', value: 0 },
      { date: '2024-01-10', value: 100 },
    ];

    render(
      <RiskConcentrationSection
        positions={[]}
        fiPositions={[]}
        treasuryPositions={[]}
        performanceData={mockPerf as any}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Dados de performance insuficientes para calcular métricas de risco.')).toBeInTheDocument();
  });

  it('handles negative totalContribution, moderate beta, moderate drawdown and concentration tiers', () => {
    // 15 days performance with ~15% drawdown and standard deviation giving beta around 1.0
    // marketVol = 0.012
    const mockPerf = [
      { date: '2024-01-01', value: 1000 },
      { date: '2024-01-02', value: 988 },
      { date: '2024-01-03', value: 1000 },
      { date: '2024-01-04', value: 988 },
      { date: '2024-01-05', value: 1000 },
      { date: '2024-01-06', value: 988 },
      { date: '2024-01-07', value: 1000 },
      { date: '2024-01-08', value: 850 }, // 15% drawdown
      { date: '2024-01-09', value: 860 },
      { date: '2024-01-10', value: 850 },
      { date: '2024-01-11', value: 860 },
      { date: '2024-01-12', value: 850 },
      { date: '2024-01-13', value: 860 },
      { date: '2024-01-14', value: 850 },
      { date: '2024-01-15', value: 860 },
    ];

    // Asset with net negative return
    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'DETRACT1',
        type: 'STOCK_BR',
        current_value: 1000,
        total_cost: 2000,
        profit_loss: -1000,
        return_percent: -50,
      },
      // 7 other assets to make top 3 concentration between 30% and 50%
      ...Array.from({ length: 7 }, (_, i) => ({
        asset_id: `a-${i}`,
        ticker: `ASSET_${i}`,
        type: 'STOCK_BR',
        current_value: 1000,
        total_cost: 1000,
        profit_loss: 0,
        return_percent: 0,
      })),
    ];

    const { rerender } = render(
      <RiskConcentrationSection
        positions={mockPositions as any}
        fiPositions={[]}
        treasuryPositions={[]}
        performanceData={mockPerf as any}
        kpiCurrency="BRL"
      />
    );

    const expandButtons = screen.getAllByText('Ver detalhes e ativos');
    expandButtons.forEach(btn => fireEvent.click(btn));

    expect(screen.getByText('Concentração moderada')).toBeInTheDocument();

    // Now test concentration <= 30% ("Bem diversificada")
    const wellDiversified = Array.from({ length: 12 }, (_, i) => ({
      asset_id: `div-${i}`,
      ticker: `D_${i}`,
      type: 'STOCK_BR',
      current_value: 1000,
      total_cost: 1000,
      profit_loss: 0,
      return_percent: 0,
    }));

    rerender(
      <RiskConcentrationSection
        positions={wellDiversified as any}
        fiPositions={[]}
        treasuryPositions={[]}
        performanceData={mockPerf as any}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Bem diversificada')).toBeInTheDocument();
  });
});
