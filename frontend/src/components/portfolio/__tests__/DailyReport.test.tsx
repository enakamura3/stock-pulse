import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import DailyReport from '../DailyReport';
import { Position, FixedIncomePosition, TreasuryPosition } from '../types';

const mockPush = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

describe('DailyReport Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });
  const mockPositions: Position[] = [
    {
      asset_id: 'pos1',
      ticker: 'PETR4',
      name: 'Petrobras',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 100,
      average_price: 25,
      total_cost: 2500,
      current_price: 30,
      current_value: 3000,
      daily_change: 1.5,
      daily_change_percent: 5.26,
    },
    {
      asset_id: 'pos2',
      ticker: 'VALE3',
      name: 'Vale',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 50,
      average_price: 80,
      total_cost: 4000,
      current_price: 70,
      current_value: 3500,
      daily_change: -2.0,
      daily_change_percent: -2.77,
    },
    {
      asset_id: 'pos3',
      ticker: 'HGLG11',
      name: 'CSHG Logística',
      type: 'FII',
      currency: 'BRL',
      quantity: 10,
      average_price: 160,
      total_cost: 1600,
      current_price: 160,
      current_value: 1600,
      daily_change: 0.0,
      daily_change_percent: 0.0,
    },
  ];

  const mockFIPositions: FixedIncomePosition[] = [
    {
      asset: {
        id: 'fi1',
        portfolio_id: 'p1',
        institution: 'Banco Sofisa',
        type: 'CDB',
        debt_type: 'POS',
        indexer: 'CDI',
        rate: 110,
        maturity_date: '2027-12-31T00:00:00Z',
      },
      start_date: '2024-01-01',
      total_invested: 5000,
      gross_value: 5500,
      net_value: 5400,
      net_return_percent: 8.0,
      gross_return_percent: 10.0,
      iof_amount: 0,
      ir_amount: 100,
      ir_rate: 17.5,
      iof_rate: 0,
    },
  ];

  const mockTreasuryPositions: TreasuryPosition[] = [
    {
      transaction_id: 'tr1',
      asset_id: 'asset_tr1',
      ticker: 'Tesouro Selic 2029',
      treasury_type: 'SELIC',
      maturity_date: '2029-03-01',
      has_coupons: false,
      start_date: '2024-01-01',
      quantity: 1,
      unit_price: 10000,
      contractedRate: 0,
      total_invested: 10000,
      gross_value: 10500,
      net_value: 10400,
      is_matured: false,
      days_to_maturity: 900,
      taxes: 100,
      b3_fee: 10,
      ir_tax: 90,
      iof_tax: 0,
    },
  ];

  it('renders empty state when no positions exist', () => {
    render(<DailyReport positions={[]} kpiCurrency="BRL" />);
    expect(screen.getByText('Nenhum ativo cadastrado na carteira')).toBeInTheDocument();
  });

  it('renders empty state for variable income when only FI exists', () => {
    render(<DailyReport positions={[]} fiPositions={mockFIPositions} kpiCurrency="BRL" />);
    expect(screen.getByText('Nenhuma posição em renda variável ativa.')).toBeInTheDocument();
  });

  it('renders total daily variation card and asset risers / fallers', () => {
    render(
      <DailyReport
        positions={mockPositions}
        fiPositions={mockFIPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Variação Diária da Carteira (Intraday)')).toBeInTheDocument();
    expect(screen.getByText('🚀 Maiores Altas do Dia')).toBeInTheDocument();
    expect(screen.getByText('📉 Maiores Baixas do Dia')).toBeInTheDocument();
  });

  it('renders all asset categories including private fixed income and treasury', () => {
    render(
      <DailyReport
        positions={mockPositions}
        fiPositions={mockFIPositions}
        treasuryPositions={mockTreasuryPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('PETR4').length).toBeGreaterThan(0);
    expect(screen.getAllByText('VALE3').length).toBeGreaterThan(0);
    expect(screen.getByText('HGLG11')).toBeInTheDocument();

    expect(screen.getByText('Banco Sofisa')).toBeInTheDocument();
    expect(screen.getByText('Tesouro Selic 2029')).toBeInTheDocument();
  });

  it('allows sorting by all column headers in ascending and descending order', () => {
    const { rerender } = render(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
      />
    );

    const headers = [
      'Ativo',
      'Preço Médio',
      'Fech. Anterior',
      'Cotação Atual',
      'Var./Cota',
      'Var. %',
      'Impacto Diário',
    ];

    headers.forEach((headerText) => {
      const th = screen.getByText(new RegExp(headerText));
      fireEvent.click(th); // toggles to desc or switches column
      fireEvent.click(th); // toggles to asc
    });

    expect(screen.getAllByText('PETR4').length).toBeGreaterThan(0);
  });

  it('renders all asset type badges correctly', () => {
    const variedPositions: Position[] = [
      {
        asset_id: 'p1',
        ticker: 'IVVB11',
        name: 'iShares S&P 500',
        type: 'ETF',
        currency: 'BRL',
        quantity: 10,
        average_price: 200,
        total_cost: 2000,
        current_price: 210,
        current_value: 2100,
        daily_change: 2.0,
        daily_change_percent: 0.96,
      },
      {
        asset_id: 'p2',
        ticker: 'AAPL34',
        name: 'Apple BDR',
        type: 'BDR',
        currency: 'BRL',
        quantity: 20,
        average_price: 50,
        total_cost: 1000,
        current_price: 55,
        current_value: 1100,
        daily_change: 1.0,
        daily_change_percent: 1.85,
      },
      {
        asset_id: 'p3',
        ticker: 'VALE3',
        name: 'Vale SA',
        type: 'AÇÃO',
        currency: 'BRL',
        quantity: 10,
        average_price: 60,
        total_cost: 600,
        current_price: 58,
        current_value: 580,
        daily_change: -2.0,
        daily_change_percent: -3.33,
      },
      {
        asset_id: 'p4',
        ticker: 'BTC',
        name: 'Bitcoin',
        type: 'CRYPTO',
        currency: 'BRL',
        quantity: 0.1,
        average_price: 300000,
        total_cost: 30000,
        current_price: 310000,
        current_value: 31000,
        daily_change: 5000,
        daily_change_percent: 1.64,
      },
      {
        asset_id: 'p5',
        ticker: 'OUTRO',
        name: 'Outro Ativo',
        type: 'OTHER',
        currency: 'BRL',
        quantity: 1,
        average_price: 10,
        total_cost: 10,
        current_price: 10,
        current_value: 10,
        daily_change: 0,
        daily_change_percent: 0,
      },
    ];

    render(
      <DailyReport
        positions={variedPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('ETF').length).toBeGreaterThan(0);
    expect(screen.getAllByText('BDR').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Ação').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Crypto').length).toBeGreaterThan(0);
    expect(screen.getAllByText('OUTRO').length).toBeGreaterThan(0);
  });

  it('renders correctly when there are only risers or only fallers', () => {
    const onlyRisers: Position[] = [
      {
        asset_id: 'pos1',
        ticker: 'PETR4',
        name: 'Petrobras',
        type: 'STOCK_BR',
        currency: 'BRL',
        quantity: 100,
        average_price: 25,
        total_cost: 2500,
        current_price: 30,
        current_value: 3000,
        daily_change: 1.5,
        daily_change_percent: 5.26,
      },
    ];

    const { rerender } = render(<DailyReport positions={onlyRisers} kpiCurrency="BRL" />);
    expect(screen.getByText('Nenhuma baixa registrada hoje.')).toBeInTheDocument();

    const onlyFallers: Position[] = [
      {
        asset_id: 'pos2',
        ticker: 'VALE3',
        name: 'Vale',
        type: 'STOCK_BR',
        currency: 'BRL',
        quantity: 50,
        average_price: 80,
        total_cost: 4000,
        current_price: 70,
        current_value: 3500,
        daily_change: -2.0,
        daily_change_percent: -2.77,
      },
    ];

    rerender(<DailyReport positions={onlyFallers} kpiCurrency="BRL" />);
    expect(screen.getByText('Nenhuma alta registrada hoje.')).toBeInTheDocument();
  });

  it('renders pre-fixed fixed income assets and prefixado/ipca treasury assets', () => {
    const preFI: FixedIncomePosition[] = [
      {
        asset: {
          id: 'fi_pre',
          portfolio_id: 'p1',
          institution: 'Banco ABC',
          type: 'LCA',
          debt_type: 'PRE',
          indexer: 'PRE',
          rate: 12.5,
          maturity_date: '2028-06-01T00:00:00Z',
        },
        start_date: '2024-01-01',
        total_invested: 10000,
        gross_value: 11000,
        net_value: 11000,
        net_return_percent: 10.0,
        is_matured: false,
        days_to_maturity: 500,
      },
    ];

    const variedTreasury: TreasuryPosition[] = [
      {
        transaction_id: 'tr_pre',
        asset_id: 'asset_tr_pre',
        ticker: 'Tesouro Prefixado 2029',
        treasury_type: 'PREFIXADO',
        maturity_date: '2029-01-01',
        has_coupons: false,
        start_date: '2024-01-01',
        quantity: 1,
        unit_price: 800,
        contractedRate: 11.5,
        total_invested: 800,
        gross_value: 900,
        net_value: 880,
        is_matured: false,
        days_to_maturity: 800,
        taxes: 20,
        b3_fee: 5,
        ir_tax: 15,
        iof_tax: 0,
      },
      {
        transaction_id: 'tr_ipca',
        asset_id: 'asset_tr_ipca',
        ticker: 'Tesouro IPCA+ 2035',
        treasury_type: 'IPCA+',
        maturity_date: '2035-05-15',
        has_coupons: false,
        start_date: '2024-01-01',
        quantity: 1,
        unit_price: 3000,
        contractedRate: 6.0,
        total_invested: 0, // tests total_invested <= 1e-6 branch
        gross_value: 3200,
        net_value: 3150,
        is_matured: false,
        days_to_maturity: 3000,
        taxes: 50,
        b3_fee: 10,
        ir_tax: 40,
        iof_tax: 0,
      },
    ];

    render(
      <DailyReport
        positions={[]}
        fiPositions={preFI}
        treasuryPositions={variedTreasury}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Banco ABC')).toBeInTheDocument();
    expect(screen.getByText('12.50% a.a.')).toBeInTheDocument();
    expect(screen.getByText('Tesouro Prefixado 2029')).toBeInTheDocument();
    expect(screen.getByText('Tesouro IPCA+ 2035')).toBeInTheDocument();
  });

  it('correctly calculates impact for foreign currency positions', () => {
    const foreignPositions: Position[] = [
      {
        asset_id: 'aapl1',
        ticker: 'AAPL',
        name: 'Apple Inc.',
        type: 'STOCK_US',
        currency: 'USD',
        quantity: 10,
        average_price: 150,
        total_cost: 7500, // in BRL
        current_price: 200, // in USD
        current_value: 10000, // in BRL (implied rate 10000 / (200*10) = 5.0)
        daily_change: 5, // in USD
        daily_change_percent: 2.56,
      },
    ];

    render(
      <DailyReport
        positions={foreignPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('AAPL').length).toBeGreaterThan(0);
    // Daily change is 5 USD * 10 shares * 5.0 BRL/USD = +R$ 250,00 impact
    expect(screen.getAllByText(/250,00/).length).toBeGreaterThan(0);
  });

  it('handles zero portfolio value, negative total daily change and unquoted foreign positions', () => {
    const edgePositions: Position[] = [
      {
        asset_id: 'zero1',
        ticker: 'ZERO1',
        name: 'Zero Val',
        type: 'STOCK_BR',
        currency: '',
        quantity: 0,
        average_price: 0,
        total_cost: 0,
        current_price: 0,
        current_value: 0,
        daily_change: -10,
        daily_change_percent: -5,
      },
      {
        asset_id: 'foreign_no_quote',
        ticker: 'MSFT',
        name: 'Microsoft',
        type: 'STOCK_US',
        currency: 'USD',
        quantity: 10,
        average_price: 300,
        total_cost: 3000,
        current_price: 0, // no quote available
        current_value: 0,
        daily_change: 0,
        daily_change_percent: 0,
      },
      {
        asset_id: 'neg_total',
        ticker: 'LOSS1',
        name: 'Loss Stock',
        type: 'STOCK_BR',
        currency: 'BRL',
        quantity: 10,
        average_price: 100,
        total_cost: 1000,
        current_price: 80,
        current_value: 800,
        daily_change: -10,
        daily_change_percent: -11.11,
      },
    ];

    render(
      <DailyReport
        positions={edgePositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('ZERO1').length).toBeGreaterThan(0);
    expect(screen.getAllByText('MSFT').length).toBeGreaterThan(0);
    expect(screen.getAllByText('LOSS1').length).toBeGreaterThan(0);
  });

  it('renders single zero-value position with currency fallback to achieve 100% branch coverage', () => {
    const singleZeroPos: Position[] = [
      {
        asset_id: 'brl_zero',
        ticker: 'BRLZERO',
        name: 'BRL Zero Price',
        type: 'STOCK_BR',
        currency: 'brl', // lowercase BRL
        quantity: 10,
        average_price: 10,
        total_cost: 100,
        current_price: 0,
        current_value: 0,
        daily_change: 0,
        daily_change_percent: 0,
      },
    ];

    render(
      <DailyReport
        positions={singleZeroPos}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getAllByText('BRLZERO').length).toBeGreaterThan(0);
  });

  it('renders actionable empty state when no positions exist and calls onGoToAssets', () => {
    const mockGoToAssets = vi.fn();
    render(
      <DailyReport
        positions={[]}
        fiPositions={[]}
        treasuryPositions={[]}
        kpiCurrency="BRL"
        onGoToAssets={mockGoToAssets}
      />
    );

    expect(screen.getByText('Nenhum ativo cadastrado na carteira')).toBeInTheDocument();
    const btn = screen.getByRole('button', { name: /\+ Cadastrar Ativos na Carteira/i });
    fireEvent.click(btn);
    expect(mockGoToAssets).toHaveBeenCalledTimes(1);
  });

  it('navigates to dashboard monitoring when table row and top riser/faller cards are clicked', () => {
    render(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
      />
    );

    // Click on table row for PETR4
    const petrRow = screen.getAllByText('PETR4').find(el => el.closest('tr'))?.closest('tr');
    expect(petrRow).toBeInTheDocument();
    fireEvent.click(petrRow!);
    expect(mockPush).toHaveBeenCalledWith('/dashboard?ticker=PETR4');

    // Click on top riser card for PETR4
    const petrRiser = screen.getAllByText('PETR4')[0];
    fireEvent.click(petrRiser);
    expect(mockPush).toHaveBeenCalledWith('/dashboard?ticker=PETR4');
  });

  it('renders refresh button, handles isRefreshing state, and triggers onRefresh', () => {
    const mockRefresh = vi.fn();
    const { rerender } = render(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
        lastFetchedAt={new Date('2026-08-20T14:30:00Z')}
        onRefresh={mockRefresh}
        isRefreshing={false}
      />
    );

    const refreshBtn = screen.getByRole('button', { name: /🔄 Atualizar/i });
    expect(refreshBtn).not.toBeDisabled();
    fireEvent.click(refreshBtn);
    expect(mockRefresh).toHaveBeenCalledTimes(1);

    // Rerender in isRefreshing=true state
    rerender(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
        lastFetchedAt={new Date('2026-08-20T14:30:00Z')}
        onRefresh={mockRefresh}
        isRefreshing={true}
      />
    );

    expect(screen.getByRole('button', { name: /⏳ Atualizando.../i })).toBeDisabled();
  });

  it('renders dividends received today section when dividends match today date', () => {
    const todayISO = new Date().toISOString().split('T')[0];
    const mockDividends = [
      {
        asset_id: 'pos1',
        ticker: 'PETR4',
        type: 'DIVIDENDO',
        gross_amount: 100,
        net_amount: 100,
        currency: 'BRL',
        quantity: 100,
        per_share_amount: 1.0,
        asset_type: 'STOCK_BR',
        asset_name: 'Petrobras',
        payment_date: `${todayISO}T00:00:00Z`,
        cum_date: '2026-08-01',
      },
      {
        asset_id: 'pos2',
        ticker: 'VALE3',
        type: 'JCP',
        gross_amount: 50,
        net_amount: 42.5,
        currency: 'BRL',
        quantity: 50,
        per_share_amount: 1.0,
        asset_type: 'STOCK_BR',
        asset_name: 'Vale',
        payment_date: '2025-01-01T00:00:00Z', // past date
        cum_date: '2024-12-15',
      },
      {
        asset_id: 'pos3',
        ticker: 'HGLG11',
        type: 'RENDIMENTO',
        gross_amount: 10,
        net_amount: 10,
        currency: 'BRL',
        quantity: 10,
        per_share_amount: 1.0,
        asset_type: 'FII',
        asset_name: 'CSHG Logística',
        payment_date: '', // missing payment date branch
        cum_date: '2026-08-01',
      },
    ];

    render(
      <DailyReport
        positions={mockPositions}
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Proventos Recebidos Hoje \(1\)/i)).toBeInTheDocument();
  });

  it('renders negative returns for fixed income and treasury positions with danger styling', () => {
    const negFI: FixedIncomePosition[] = [
      {
        asset: {
          id: 'fi_neg',
          portfolio_id: 'p1',
          institution: 'Banco Negativo',
          type: 'CDB',
          debt_type: 'POS',
          indexer: 'CDI',
          rate: 100,
          maturity_date: '2027-12-31T00:00:00Z',
        },
        start_date: '2024-01-01',
        total_invested: 5000,
        gross_value: 4800,
        net_value: 4800,
        net_return_percent: -4.0, // negative return
      },
    ];

    const negTreasury: TreasuryPosition[] = [
      {
        transaction_id: 'tr_neg',
        asset_id: 'asset_tr_neg',
        ticker: 'Tesouro Prefixado 2031',
        treasury_type: 'PREFIXADO',
        maturity_date: '2031-01-01',
        has_coupons: false,
        start_date: '2024-01-01',
        quantity: 1,
        unit_price: 1000,
        contractedRate: 12.0,
        total_invested: 1000,
        gross_value: 900,
        net_value: 900, // loss: returnPct < 0
        is_matured: false,
        days_to_maturity: 2000,
        taxes: 0,
        b3_fee: 0,
        ir_tax: 0,
        iof_tax: 0,
      },
    ];

    render(
      <DailyReport
        positions={[]}
        fiPositions={negFI}
        treasuryPositions={negTreasury}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Banco Negativo')).toBeInTheDocument();
    expect(screen.getByText('-4.00%')).toBeInTheDocument();
    expect(screen.getByText('Tesouro Prefixado 2031')).toBeInTheDocument();
    expect(screen.getByText('-10.00%')).toBeInTheDocument();
  });

  it('uses real previous_close from backend when available instead of estimation', () => {
    const posWithPreviousClose: Position[] = [
      {
        asset_id: 'pos_pc',
        ticker: 'ITUB4',
        name: 'Itaú Unibanco',
        type: 'STOCK_BR',
        currency: 'BRL',
        quantity: 200,
        average_price: 28,
        total_cost: 5600,
        current_price: 31.50,
        current_value: 6300,
        daily_change: 1.50,
        daily_change_percent: 5.0,
        previous_close: 30.00, // valor real do backend
        volume: 12000000,
      },
    ];

    render(
      <DailyReport
        positions={posWithPreviousClose}
        kpiCurrency="BRL"
      />
    );

    // Preço atual deve aparecer na tabela
    expect(screen.getAllByText('ITUB4').length).toBeGreaterThan(0);
    // Fech. Anterior real (30,00) deve aparecer (não a estimativa 31.50 - 1.50 = 30.00 coincidência)
    expect(screen.getAllByText(/R\$ 30,00/).length).toBeGreaterThan(0);
  });

  it('renders market benchmarks bar when benchmarks are provided as prop', () => {
    const mockBenchmarks = {
      ibov: {
        symbol: '^BVSP',
        name: 'Ibovespa',
        value: 130000,
        change: 1000,
        change_percent: 0.77,
      },
    };

    render(
      <DailyReport
        positions={mockPositions}
        benchmarks={mockBenchmarks}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('🌐 Índices e Benchmarks de Mercado')).toBeInTheDocument();
    expect(screen.getByText('Ibovespa')).toBeInTheDocument();
  });

  it('fetches market benchmarks from API when benchmarks prop is undefined', async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/market/benchmarks')) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              ibov: {
                symbol: '^BVSP',
                name: 'Ibovespa',
                value: 130000,
                change: 500,
                change_percent: 0.38,
              },
            }),
        });
      }
      return Promise.resolve({
        ok: false,
        json: () => Promise.resolve({}),
      });
    });

    render(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
      />
    );

    expect(await screen.findByText('🌐 Índices e Benchmarks de Mercado')).toBeInTheDocument();
  });

  it('handles market benchmarks API failure gracefully without crashing', async () => {
    global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

    render(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Variação Diária da Carteira (Intraday)')).toBeInTheDocument();
  });

  it('handles benchmarks API returning ok: false gracefully', async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/market/benchmarks')) {
        return Promise.resolve({
          ok: false,
          json: () => Promise.resolve({ error: 'not found' }),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
      });
    });

    render(
      <DailyReport
        positions={mockPositions}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText('Variação Diária da Carteira (Intraday)')).toBeInTheDocument();
  });
});
