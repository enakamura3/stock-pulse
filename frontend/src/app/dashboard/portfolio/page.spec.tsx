import React from 'react';
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import PortfolioPage from './page';
import { useAuth } from '@/context/AuthContext';
import { usePortfolio } from '@/context/PortfolioContext';
import { apiFetch } from '@/lib/api';

vi.mock('next/navigation', () => ({
  usePathname: () => '/dashboard/portfolio',
  useRouter: () => ({ push: vi.fn() }),
  useSearchParams: () => ({ get: vi.fn().mockReturnValue(null) }),
}));

vi.mock('@/lib/api', () => ({
  apiFetch: vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve([]),
  }),
}));

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

vi.mock('@/context/PortfolioContext', async (importOriginal) => {
  const actual = await importOriginal<any>();
  return {
    ...actual,
    usePortfolio: vi.fn(),
  };
});

// Mock dynamic chart import
vi.mock('@/components/PortfolioChart', () => ({
  default: () => <div data-testid="portfolio-chart">Chart</div>,
}));

vi.mock('@/components/ThemeProvider', () => ({
  useTheme: () => ({ theme: 'dark', toggleTheme: vi.fn(), setTheme: vi.fn() }),
  useThemeOptional: () => ({ theme: 'dark', toggleTheme: vi.fn(), setTheme: vi.fn() }),
  ThemeProvider: ({ children }: any) => <>{children}</>,
}));

describe('PortfolioPage Contextual Filters', () => {
  const mockSetActiveTab = vi.fn();
  const mockSetActiveCategoryFilter = vi.fn();
  const mockSetFilterChartTicker = vi.fn();
  const mockSetPeriod = vi.fn();

  const basePortfolioMock = {
    portfolios: [{ id: 'p1', name: 'Carteira Principal', is_default: true, base_currency: 'BRL' }],
    activePortfolioId: 'p1',
    setActivePortfolioId: vi.fn(),
    kpiCurrency: 'BRL',
    positions: [],
    fiPositions: [],
    treasuryPositions: [],
    transactions: [],
    performanceData: [],
    dividends: [],
    isLoadingPortfolios: false,
    isLoadingDetails: false,
    isLoadingPerformance: false,
    isLoadingDividends: false,
    isLoadingTreasury: false,
    activeTab: 'ativos',
    setActiveTab: mockSetActiveTab,
    activeCategoryFilter: 'Todas',
    setActiveCategoryFilter: mockSetActiveCategoryFilter,
    filterTxTicker: '',
    setFilterTxTicker: vi.fn(),
    filterChartTicker: 'Todos',
    setFilterChartTicker: mockSetFilterChartTicker,
    filterDivYear: 'Todos',
    setFilterDivYear: vi.fn(),
    filterDivMonth: 'Todos',
    setFilterDivMonth: vi.fn(),
    period: 'ALL',
    setPeriod: mockSetPeriod,
    setShowPortfolioModal: vi.fn(),
    setShowTxModal: vi.fn(),
    setShowFIModal: vi.fn(),
    setEditingTxId: vi.fn(),
    handleDeletePortfolio: vi.fn(),
    handleSetDefaultPortfolio: vi.fn(),
    handleEditTransaction: vi.fn(),
    handleDeleteTransaction: vi.fn(),
    handleFileUpload: vi.fn(),
    handleExportPortfolio: vi.fn(),
    loadTreasuryPositions: vi.fn(),
    lastFetchedAt: new Date(),
    loadPortfolioDetails: vi.fn(),
    loadDividends: vi.fn(),
    loadPerformance: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (useAuth as any).mockReturnValue({
      user: { id: 'u1', name: 'Carlos Investidor' },
      logout: vi.fn(),
      isLoading: false,
    });
  });

  it('renders loading state when auth or portfolios are loading', () => {
    (useAuth as any).mockReturnValue({ user: null, isLoading: true });
    (usePortfolio as any).mockReturnValue({ ...basePortfolioMock, isLoadingPortfolios: true });

    render(<PortfolioPage />);
    expect(screen.getByText(/Carregando dados financeiros seguros/i)).toBeInTheDocument();
  });

  it('auto-hides filter pills when user owns only 1 equity category in ativos tab', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      positions: [
        { ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
        { ticker: 'VALE3', type: 'STOCK_BR', total_cost: 2000, current_value: 2100 },
      ],
    });

    render(<PortfolioPage />);
    expect(screen.queryByTestId('contextual-filter-pills')).not.toBeInTheDocument();
  });

  it('dynamically displays filter pills when user owns multiple equity categories in ativos tab', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      positions: [
        { ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
        { ticker: 'HGLG11', type: 'FII', total_cost: 2000, current_value: 2100 },
        { ticker: 'AAPL34', type: 'BDR', total_cost: 1500, current_value: 1600 },
      ],
    });

    render(<PortfolioPage />);
    const filterBar = screen.getByTestId('contextual-filter-pills');
    expect(within(filterBar).getByRole('button', { name: 'Todas' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'Ações (B3)' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'FIIs' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'BDRs' })).toBeInTheDocument();
    // Must NOT display fixed income or treasury pills
    expect(within(filterBar).queryByRole('button', { name: 'Renda Fixa' })).not.toBeInTheDocument();
    expect(within(filterBar).queryByRole('button', { name: 'Tesouro Direto' })).not.toBeInTheDocument();
  });

  it('allows clicking an equity filter pill to select the category', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      positions: [
        { ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
        { ticker: 'HGLG11', type: 'FII', total_cost: 2000, current_value: 2100 },
      ],
    });

    render(<PortfolioPage />);
    const filterBar = screen.getByTestId('contextual-filter-pills');
    const fiiBtn = within(filterBar).getByRole('button', { name: 'FIIs' });
    fireEvent.click(fiiBtn);
    expect(mockSetActiveCategoryFilter).toHaveBeenCalledWith('FIIs');
  });

  it('auto-hides filter pills on operacoes, analise, and diario tabs', () => {
    const tabs = ['operacoes', 'analise', 'diario'] as const;

    for (const tab of tabs) {
      (usePortfolio as any).mockReturnValue({
        ...basePortfolioMock,
        activeTab: tab,
        positions: [
          { ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
          { ticker: 'HGLG11', type: 'FII', total_cost: 2000, current_value: 2100 },
        ],
        fiPositions: [
          { asset: { id: '1', type: 'CDB', institution: 'Inter', rate: 12.0, debt_type: 'PRE', indexer: 'PRE' }, total_invested: 1000, net_value: 1100 },
        ],
      });

      const { unmount } = render(<PortfolioPage />);
      expect(screen.queryByTestId('contextual-filter-pills')).not.toBeInTheDocument();
      unmount();
    }
  });

  it('dynamically displays asset categories on proventos tab when multiple categories exist and filters by category', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'proventos',
      activeCategoryFilter: 'FIIs',
      dividends: [
        { id: '1', ticker: 'PETR4', asset_type: 'STOCK_BR', cum_date: '2026-05-01', payment_date: '2026-05-10', gross_amount: 100, net_amount: 100 },
        { id: '2', ticker: 'HGLG11', asset_type: 'FII', cum_date: '2026-06-01', payment_date: '2026-06-15', gross_amount: 50, net_amount: 50 },
      ],
    });

    render(<PortfolioPage />);
    const filterBar = screen.getByTestId('contextual-filter-pills');
    expect(within(filterBar).getByRole('button', { name: 'Todas' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'Ações (B3)' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'FIIs' })).toBeInTheDocument();

    const acoesBtn = within(filterBar).getByRole('button', { name: 'Ações (B3)' });
    fireEvent.click(acoesBtn);
    expect(mockSetActiveCategoryFilter).toHaveBeenCalledWith('Ações (B3)');
  });

  it('auto-hides proventos filter pills when user has only 1 dividend category', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'proventos',
      dividends: [
        { id: '1', ticker: 'PETR4', asset_type: 'STOCK_BR', cum_date: '2026-05-01', payment_date: '2026-05-10', gross_amount: 100, net_amount: 100 },
        { id: '2', ticker: 'VALE3', asset_type: 'STOCK_BR', cum_date: '2026-06-01', payment_date: '2026-06-15', gross_amount: 200, net_amount: 200 },
      ],
    });

    render(<PortfolioPage />);
    expect(screen.queryByTestId('contextual-filter-pills')).not.toBeInTheDocument();
  });

  it('dynamically displays fixed income categories on renda-fixa tab when multiple types exist', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'renda-fixa',
      fiPositions: [
        { asset: { id: '1', type: 'CDB', institution: 'Inter' }, total_invested: 1000, net_value: 1100 },
        { asset: { id: '2', type: 'LCI', institution: 'XP' }, total_invested: 2000, net_value: 2150 },
        { asset: { id: '3', type: 'DEBENTURE', institution: 'Vale' }, total_invested: 3000, net_value: 3200 },
      ],
    });

    render(<PortfolioPage />);
    const filterBar = screen.getByTestId('contextual-filter-pills');
    expect(within(filterBar).getByRole('button', { name: 'Todas' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'CDB' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'LCI' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'Debêntures' })).toBeInTheDocument();
    // Must NOT display equity pills
    expect(within(filterBar).queryByRole('button', { name: 'Ações (B3)' })).not.toBeInTheDocument();
  });

  it('auto-hides fixed income filter pills when user has only 1 fixed income type', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'renda-fixa',
      fiPositions: [
        { asset: { id: '1', type: 'CDB', institution: 'Inter' }, total_invested: 1000, net_value: 1100 },
        { asset: { id: '2', type: 'CDB', institution: 'Nubank' }, total_invested: 2000, net_value: 2150 },
      ],
    });

    render(<PortfolioPage />);
    expect(screen.queryByTestId('contextual-filter-pills')).not.toBeInTheDocument();
  });

  it('dynamically displays treasury types on tesouro tab when multiple types exist', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'tesouro',
      treasuryPositions: [
        { transaction_id: '1', ticker: 'Tesouro Selic 2029', treasury_type: 'SELIC', total_invested: 1000, net_value: 1050 },
        { transaction_id: '2', ticker: 'Tesouro IPCA+ 2035', treasury_type: 'IPCA+', total_invested: 2000, net_value: 2100 },
      ],
    });

    render(<PortfolioPage />);
    const filterBar = screen.getByTestId('contextual-filter-pills');
    expect(within(filterBar).getByRole('button', { name: 'Todas' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'Tesouro Selic' })).toBeInTheDocument();
    expect(within(filterBar).getByRole('button', { name: 'IPCA+' })).toBeInTheDocument();
  });

  it('calculates total portfolio summary cards regardless of active category filter', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      activeCategoryFilter: 'FIIs',
      positions: [
        { ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
        { ticker: 'HGLG11', type: 'FII', total_cost: 2000, current_value: 2100 },
      ],
      fiPositions: [
        { asset: { id: '1', type: 'CDB', institution: 'Inter' }, total_invested: 3000, net_value: 3300 },
      ],
      treasuryPositions: [
        { transaction_id: '1', ticker: 'Tesouro Selic', treasury_type: 'SELIC', total_invested: 4000, net_value: 4400 },
      ],
    });

    render(<PortfolioPage />);
    // Total invested = 1000 + 2000 + 3000 + 4000 = 10,000
    // Total current = 1200 + 2100 + 3300 + 4400 = 11,000
    expect(screen.getByText(/R\$\s*10\.000,00/i)).toBeInTheDocument();
    expect(screen.getByText(/R\$\s*11\.000,00/i)).toBeInTheDocument();
  });

  it('returns null when user is null and not loading', () => {
    (useAuth as any).mockReturnValue({ user: null, isLoading: false });
    (usePortfolio as any).mockReturnValue({ ...basePortfolioMock, isLoadingPortfolios: false });
    const { container } = render(<PortfolioPage />);
    expect(container.firstChild).toBeNull();
  });

  it('handles chart ticker selector, period buttons, and performance states', async () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      positions: [
        { asset_id: '1', ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
      ],
      isLoadingPerformance: true,
    });

    const { rerender } = render(<PortfolioPage />);
    // Select ticker
    const select = screen.getByRole('combobox');
    fireEvent.change(select, { target: { value: 'PETR4' } });
    expect(mockSetFilterChartTicker).toHaveBeenCalledWith('PETR4');

    // Click period button
    const period1M = screen.getByRole('button', { name: '1M' });
    fireEvent.click(period1M);
    expect(mockSetPeriod).toHaveBeenCalledWith('1M');

    // Performance data loaded
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      positions: [
        { asset_id: '1', ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
      ],
      isLoadingPerformance: false,
      performanceData: [{ date: '2024-01-01', value: 1000, total_invested: 1000 }],
    });
    rerender(<PortfolioPage />);
    expect(await screen.findByTestId('portfolio-chart')).toBeInTheDocument();
  });

  it('triggers transaction modal on AssetList launch operation', () => {
    const mockSetShowTxModal = vi.fn();
    const mockSetEditingTxId = vi.fn();

    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      positions: [
        { asset_id: '1', ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
      ],
      setShowTxModal: mockSetShowTxModal,
      setEditingTxId: mockSetEditingTxId,
    });

    render(<PortfolioPage />);
    const launchBtn = screen.getByRole('button', { name: /\+ Lançar Operação/i });
    fireEvent.click(launchBtn);
    expect(mockSetEditingTxId).toHaveBeenCalledWith(null);
    expect(mockSetShowTxModal).toHaveBeenCalledWith(true);
  });

  it('handles DailyReport onRefresh (normal and force realtime) and onGoToAssets', async () => {
    const mockLoadPortfolioDetails = vi.fn();
    const mockLoadDividends = vi.fn();
    const mockLoadPerformance = vi.fn();

    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'diario',
      positions: [
        { asset_id: '1', ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200, average_price: 20, current_price: 24 },
      ],
      loadPortfolioDetails: mockLoadPortfolioDetails,
      loadDividends: mockLoadDividends,
      loadPerformance: mockLoadPerformance,
    });

    const { rerender } = render(<PortfolioPage />);
    
    // Normal refresh
    const refreshBtn = screen.getByRole('button', { name: /Recarregar cotações e resumo do portfólio/i });
    fireEvent.click(refreshBtn);

    await waitFor(() => {
      expect(mockLoadPortfolioDetails).toHaveBeenCalledWith('p1');
      expect(mockLoadDividends).toHaveBeenCalledWith('p1');
      expect(mockLoadPerformance).toHaveBeenCalledWith('p1', 'ALL');
    });

    // Realtime refresh with successful apiFetch
    (apiFetch as any).mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });
    const realtimeBtn = screen.getByRole('button', { name: /Forçar atualização de cotações em tempo real ignorando cache/i });
    fireEvent.click(realtimeBtn);

    await waitFor(() => {
      expect(apiFetch).toHaveBeenCalledWith('/market/quotes/invalidate', { method: 'POST' });
      expect(mockLoadPortfolioDetails).toHaveBeenCalledTimes(2);
    });

    // Realtime refresh with apiFetch error handled gracefully
    (apiFetch as any).mockRejectedValueOnce(new Error('Network error'));
    fireEvent.click(realtimeBtn);

    await waitFor(() => {
      expect(mockLoadPortfolioDetails).toHaveBeenCalledTimes(3);
    });

    // Empty state onGoToAssets
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'diario',
      positions: [],
      fiPositions: [],
      treasuryPositions: [],
    });
    rerender(<PortfolioPage />);

    const goAssetsBtn = screen.getByRole('button', { name: /\+ Cadastrar Ativos na Carteira/i });
    fireEvent.click(goAssetsBtn);
    expect(mockSetActiveTab).toHaveBeenCalledWith('ativos');
  });

  it('handles FixedIncomeTab launch and TreasuryTab refresh', async () => {
    const mockSetShowFIModal = vi.fn();
    const mockLoadTreasuryPositions = vi.fn();

    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'renda-fixa',
      setShowFIModal: mockSetShowFIModal,
      loadTreasuryPositions: mockLoadTreasuryPositions,
    });

    const { rerender } = render(<PortfolioPage />);
    const newFiBtn = await screen.findByRole('button', { name: /\+ Nova Aplicação/i });
    fireEvent.click(newFiBtn);
    expect(mockSetShowFIModal).toHaveBeenCalledWith(true);

    // Treasury tab
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'tesouro',
      treasuryPositions: [],
      loadTreasuryPositions: mockLoadTreasuryPositions,
    });
    rerender(<PortfolioPage />);
    expect(screen.getByText(/Nenhuma posição ativa de Tesouro Direto/i)).toBeInTheDocument();

    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
    if (fileInput) {
      const file = new File(['csv content'], 'treasury.csv', { type: 'text/csv' });
      fireEvent.change(fileInput, { target: { files: [file] } });
      await waitFor(() => {
        expect(mockLoadTreasuryPositions).toHaveBeenCalledWith('p1');
      });
    }
  });

  it('renders loading spinner when isLoadingDetails is true', () => {
    (useAuth as any).mockReturnValue({
      user: { id: 'u1', name: 'Tester' },
      logout: vi.fn(),
      isLoading: false,
    });
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      isLoadingDetails: true,
    });

    render(<PortfolioPage />);
    expect(document.querySelector('.loading-spinner')).toBeInTheDocument();
  });

  it('calculates 12m dividends correctly handling payment_date fallback to cum_date and empty dates', () => {
    const recentDate = new Date().toISOString();
    const mockDividends = [
      { id: 'd1', payment_date: recentDate, total_value: 120 },
      { id: 'd2', payment_date: '0001-01-01T00:00:00Z', cum_date: recentDate, net_amount: 60 },
      { id: 'd3', payment_date: '2015-01-01T00:00:00Z', cum_date: '2015-01-01', total_value: 50 },
      { id: 'd4', payment_date: '', cum_date: '' },
      { id: 'd5', payment_date: recentDate, total_value: undefined, net_amount: undefined },
    ];

    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      positions: [
        { asset_id: '1', ticker: 'VALE3', type: 'STOCK_BR', total_cost: 100, current_value: undefined },
      ],
      dividends: mockDividends,
      filterDivYear: '2026',
      filterDivMonth: '09',
    });

    render(<PortfolioPage />);
    // Average dividends per month: (120 + 60) / 12 = 15
    expect(screen.getByText('R$ 15,00')).toBeInTheDocument();
  });

  it('renders and formats dynamic contextual category filters for Tesouro and Renda Fixa', () => {
    // Tesouro tab labels
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'tesouro',
      treasuryPositions: [
        { treasury_type: 'SELIC', total_invested: 100, gross_value: 110, net_value: 108 },
        { treasury_type: 'PREFIXADO', total_invested: 100, gross_value: 110, net_value: 108 },
        { treasury_type: 'IPCA+', total_invested: 100, gross_value: 110, net_value: 108 },
        { treasury_type: 'OUTRO', total_invested: 100, gross_value: 110, net_value: 108 },
      ],
    });

    const { rerender } = render(<PortfolioPage />);
    let pillsContainer = screen.getByTestId('contextual-filter-pills');
    expect(within(pillsContainer).getByRole('button', { name: 'Tesouro Selic' })).toBeInTheDocument();
    expect(within(pillsContainer).getByRole('button', { name: 'Prefixado' })).toBeInTheDocument();
    expect(within(pillsContainer).getByRole('button', { name: 'IPCA+' })).toBeInTheDocument();
    expect(within(pillsContainer).getByRole('button', { name: 'OUTRO' })).toBeInTheDocument();

    fireEvent.click(within(pillsContainer).getByRole('button', { name: 'Tesouro Selic' }));
    expect(mockSetActiveCategoryFilter).toHaveBeenCalledWith('SELIC');

    // Test IPCA alias mapping and filtering on tesouro tab
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'tesouro',
      activeCategoryFilter: 'SELIC',
      treasuryPositions: [
        { treasury_type: 'SELIC', total_invested: 100, gross_value: 110, net_value: 108 },
        { treasury_type: 'IPCA', total_invested: 100, gross_value: 110, net_value: 108 },
      ],
    });
    rerender(<PortfolioPage />);
    pillsContainer = screen.getByTestId('contextual-filter-pills');
    expect(within(pillsContainer).getByRole('button', { name: 'IPCA+' })).toBeInTheDocument();

    // Renda Fixa tab labels
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'renda-fixa',
      fiPositions: [
        { asset: { type: 'DEBENTURE' }, total_invested: 200, net_value: 210 },
        { asset: { type: 'CDB' }, total_invested: 300, net_value: 320 },
      ],
    });
    rerender(<PortfolioPage />);
    const fiPillsContainer = screen.getByTestId('contextual-filter-pills');
    expect(within(fiPillsContainer).getByRole('button', { name: 'Debêntures' })).toBeInTheDocument();
    expect(within(fiPillsContainer).getByRole('button', { name: 'CDB' })).toBeInTheDocument();
  });

  it('resets category filter when selected category is no longer present in dynamicCategories', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeCategoryFilter: 'CategoriaInexistente',
      positions: [
        { type: 'STOCK_BR', total_cost: 1000, current_value: 1100 },
        { type: 'FII', total_cost: 1000, current_value: 1050 },
      ],
    });

    render(<PortfolioPage />);
    expect(mockSetActiveCategoryFilter).toHaveBeenCalledWith('Todas');
  });

  it('renders default user name when user name is undefined', () => {
    (useAuth as any).mockReturnValue({
      user: { id: 'u1' },
      logout: vi.fn(),
      isLoading: false,
    });
    (usePortfolio as any).mockReturnValue(basePortfolioMock);

    render(<PortfolioPage />);
    expect(screen.getByText('Investidor')).toBeInTheDocument();
  });

  it('renders equity KPI summary cards reflecting filtered positions on ativos tab', () => {
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      activeCategoryFilter: 'FIIs',
      positions: [
        { ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 1200 },
        { ticker: 'HGLG11', type: 'FII', total_cost: 2000, current_value: 2300 },
        { asset_id: 'aid-fii', type: 'FII', total_cost: 500, current_value: 600 },
      ],
      dividends: [
        { ticker: 'PETR4', net_amount: 50 },
        { ticker: 'HGLG11', net_amount: 120 },
        { asset_id: 'aid-fii', total_value: 40 },
      ],
    });

    const { rerender } = render(<PortfolioPage />);
    let kpiCards = screen.getByTestId('equity-kpi-cards');
    expect(within(kpiCards).getByText(/Total Investido/i)).toBeInTheDocument();
    expect(within(kpiCards).getByText(/Patrimônio Atual/i)).toBeInTheDocument();
    expect(within(kpiCards).getByText(/Lucro \/ Prejuízo/i)).toBeInTheDocument();
    expect(within(kpiCards).getByText(/Proventos Recebidos/i)).toBeInTheDocument();
    expect(within(kpiCards).getByText(/Ativos em Carteira/i)).toBeInTheDocument();

    // FIIs has 2 assets (HGLG11 + aid-fii) with total_cost 2500, current_value 2900, profit 400 (+16.00%)
    expect(within(kpiCards).getByText('2')).toBeInTheDocument();
    expect(within(kpiCards).getByText('FIIs')).toBeInTheDocument();
    expect(within(kpiCards).getByText('R$ 2.500,00')).toBeInTheDocument();
    expect(within(kpiCards).getByText('R$ 2.900,00')).toBeInTheDocument();
    expect(within(kpiCards).getByText('R$ 160,00')).toBeInTheDocument();

    // Test negative profit and "Todas" category filter
    (usePortfolio as any).mockReturnValue({
      ...basePortfolioMock,
      activeTab: 'ativos',
      activeCategoryFilter: 'Todas',
      positions: [
        { ticker: 'PETR4', type: 'STOCK_BR', total_cost: 1000, current_value: 800 },
      ],
      dividends: [],
    });
    rerender(<PortfolioPage />);
    kpiCards = screen.getByTestId('equity-kpi-cards');
    expect(within(kpiCards).getByText('Todas as categorias')).toBeInTheDocument();
    expect(within(kpiCards).getByText('-20.00% (-R$ 200,00)')).toBeInTheDocument();
  });
});

