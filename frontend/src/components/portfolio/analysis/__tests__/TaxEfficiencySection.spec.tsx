import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import TaxEfficiencySection from '../TaxEfficiencySection';

describe('TaxEfficiencySection', () => {
  it('renders correctly with exempt and taxable assets, JCP, FI and Treasury taxes', () => {
    const mockPositions = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        type: 'STOCK_BR',
        current_value: 10000,
      },
      {
        asset_id: '2',
        ticker: 'AAPL34',
        type: 'BDR',
        current_value: 5000,
      },
    ];

    const mockFiPositions = [
      {
        id: 'fi-1',
        net_value: 4000,
        asset: { type: 'LCI' },
        ir_amount: 0,
        iof_amount: 0,
      },
      {
        id: 'fi-2',
        net_value: 6000,
        type: 'CDB',
        ir_amount: 600,
        iof_amount: 50,
      },
    ];

    const mockTreasuryPositions = [
      {
        transaction_id: 'td-1',
        ticker: 'TESOURO SELIC',
        net_value: 5000,
        ir_tax: 500,
        iof_tax: 20,
        b3_fee: 15,
      },
    ];

    const mockDividends = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        type: 'Dividendo',
        net_amount: 100,
        is_accrued: false,
      },
      {
        asset_id: '1',
        ticker: 'PETR4',
        type: 'Juros sobre Capital Próprio',
        net_amount: 85,
        is_accrued: false,
      },
      {
        asset_id: 'fi-2',
        ticker: 'CDB',
        type: 'Rendimento',
        net_amount: 50,
        is_accrued: true,
      },
    ];

    render(
      <TaxEfficiencySection
        positions={mockPositions as any}
        dividends={mockDividends as any}
        fiPositions={mockFiPositions as any}
        treasuryPositions={mockTreasuryPositions as any}
        kpiCurrency="BRL"
      />
    );

    // Expand all KPI scorecards to render details
    const expandButtons = screen.getAllByText('Ver detalhes e ativos');
    expandButtons.forEach(btn => fireEvent.click(btn));

    expect(screen.getByText(/Eficiência Tributária/)).toBeInTheDocument();
    expect(screen.getByText('Patrimônio Isento de IR')).toBeInTheDocument();
    expect(screen.getByText('Proventos Isentos')).toBeInTheDocument();
    expect(screen.getByText('Impostos Retidos (RF/TD)')).toBeInTheDocument();
    expect(screen.getByText(/Excelente escolha de alocação/)).toBeInTheDocument();
    expect(screen.getByText(/Taxa de Custódia B3/)).toBeInTheDocument();
    expect(screen.getByText(/IOF Retido/)).toBeInTheDocument();
  });

  it('handles empty portfolio and moderate/danger alert branches', () => {
    // Empty data: totalPortfolio = 0, totalProventos = 0
    const { rerender } = render(
      <TaxEfficiencySection
        positions={[]}
        dividends={[]}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency=""
      />
    );

    expect(screen.getAllByText('0.0%')[0]).toBeInTheDocument();

    // Rerender with moderate isentoPct (e.g. 30%), low proventosIsentosPct (< 70%), and totalIRRetidoRF > 1000
    const moderatePositions = [
      {
        asset_id: '1',
        ticker: 'FII1',
        type: 'FII',
        current_value: 3000,
      },
      {
        asset_id: '2',
        ticker: 'BDR1',
        type: 'BDR',
        current_value: 7000,
      },
    ];

    const highTaxFi = [
      {
        id: 'fi-1',
        net_value: 0,
        asset: { type: 'CDB' },
        ir_amount: 1500,
        iof_amount: 0,
      },
    ];

    const jcpDividends = [
      {
        asset_id: '2',
        ticker: 'BDR1',
        type: 'JCP',
        net_amount: 1000,
        is_accrued: false,
      },
    ];

    rerender(
      <TaxEfficiencySection
        positions={moderatePositions as any}
        dividends={jcpDividends as any}
        fiPositions={highTaxFi as any}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    const expandButtons = screen.getAllByText('Ver detalhes e ativos');
    expandButtons.forEach(btn => fireEvent.click(btn));

    expect(screen.getAllByText('30.0%')[0]).toBeInTheDocument();
    expect(screen.getByText(/IR estimado retido na fonte/)).toBeInTheDocument();
  });

  it('handles low isentoPct (< 25%)', () => {
    const lowIsentoPositions = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        type: 'STOCK_BR',
        current_value: 1000,
      },
      {
        asset_id: '2',
        ticker: 'BDR1',
        type: 'BDR',
        current_value: 9000,
      },
    ];

    render(
      <TaxEfficiencySection
        positions={lowIsentoPositions as any}
        dividends={[]}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('10.0%')).toBeInTheDocument();
  });
});
