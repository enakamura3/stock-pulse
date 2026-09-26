import React from 'react';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import PortfolioPage from './page';
import { useAuth } from '@/context/AuthContext';
import { usePortfolio } from '@/context/PortfolioContext';

vi.mock('next/navigation', () => ({
  usePathname: () => '/dashboard/portfolio',
  useRouter: () => ({ push: vi.fn() }),
  useSearchParams: () => ({ get: vi.fn().mockReturnValue(null) }),
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

  it('auto-hides filter pills on operacoes, proventos, analise, and diario tabs', () => {
    const tabs = ['operacoes', 'proventos', 'analise', 'diario'] as const;

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
});
