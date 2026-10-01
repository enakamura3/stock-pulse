import React, { useEffect } from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PortfolioProvider, usePortfolio, usePortfolioOptional } from './PortfolioContext';
import { apiFetch } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';

vi.mock('@/lib/api', () => ({
  apiFetch: vi.fn(),
}));

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('PortfolioContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (useAuth as any).mockReturnValue({ user: { id: 'user-1', email: 'user@test.com' } });
    const mockAlert = vi.fn();
    const mockConfirm = vi.fn(() => true);
    window.alert = mockAlert;
    global.alert = mockAlert;
    window.confirm = mockConfirm;
    global.confirm = mockConfirm;
    vi.spyOn(window, 'open').mockImplementation(() => null);
    window.URL.createObjectURL = vi.fn(() => 'blob:mock-url');
    window.URL.revokeObjectURL = vi.fn();
  });

  it('throws an error if usePortfolio is used outside PortfolioProvider', () => {
    const TestConsumer = () => {
      usePortfolio();
      return null;
    };
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    expect(() => render(<TestConsumer />)).toThrow(
      'usePortfolio deve ser usado dentro de um PortfolioProvider'
    );
    consoleSpy.mockRestore();
  });

  it('usePortfolioOptional returns undefined outside provider', () => {
    let contextVal: any = 'initial';
    const TestConsumer = () => {
      contextVal = usePortfolioOptional();
      return null;
    };
    render(<TestConsumer />);
    expect(contextVal).toBeUndefined();
  });

  it('loads portfolios on mount and selects default portfolio', async () => {
    const mockPortfolios = [
      { id: 'port-1', name: 'Geral', base_currency: 'BRL', is_default: false },
      { id: 'port-2', name: 'Dividendos', base_currency: 'USD', is_default: true },
    ];

    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') {
        return Promise.resolve({ ok: true, json: async () => mockPortfolios });
      }
      return Promise.resolve({ ok: true, json: async () => ({ positions: [] }) });
    });

    const TestComponent = () => {
      const { portfolios, activePortfolioId, kpiCurrency } = usePortfolio();
      return (
        <div>
          <div data-testid="count">{portfolios.length}</div>
          <div data-testid="active">{activePortfolioId}</div>
          <div data-testid="currency">{kpiCurrency}</div>
        </div>
      );
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('count')).toHaveTextContent('2');
      expect(screen.getByTestId('active')).toHaveTextContent('port-2');
      expect(screen.getByTestId('currency')).toHaveTextContent('USD');
    });
  });

  it('handles loadPortfolios with no default portfolio and empty response, plus errors', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    // First: load with no default portfolio -> selects first
    (apiFetch as any).mockImplementationOnce((url: string) => {
      return Promise.resolve({
        ok: true,
        json: async () => [{ id: 'p-single', name: 'Unica', base_currency: 'BRL', is_default: false }],
      });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return <div>Active: {context.activePortfolioId}</div>;
    };

    const { rerender } = render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Active: p-single')).toBeInTheDocument();
    });

    // Test error in loadPortfolios
    (apiFetch as any).mockRejectedValueOnce(new Error('Network error'));
    await act(async () => {
      await context.loadPortfolios();
    });
    expect(consoleError).toHaveBeenCalled();

    consoleError.mockRestore();
  });

  it('loads portfolio details, fixed income, treasury and dividends properly', async () => {
    const mockPositions = [
      { asset_id: '1', ticker: 'PETR4', type: 'STOCK_BR', quantity: 100, current_value: 3000, total_cost: 2000 },
      { asset_id: '2', ticker: 'HGLG11', type: 'FII', quantity: 10, current_value: 1000, total_cost: 1000 },
    ];
    const mockTxs = [
      { id: 'tx-1', asset_name: 'PETR4', module: 'RV', type: 'BUY', date: '2024-01-10', total_value: 2000 },
    ];
    const mockFI = [
      { id: 'fi-1', asset_name: 'CDB Inter', net_value: 5000 },
    ];
    const mockTD = [
      { id: 'td-1', title: 'Tesouro Selic', net_value: 3000 },
    ];
    const mockDivs = [
      { asset_id: '1', ticker: 'PETR4', payment_date: '2024-02-15', net_amount: 100 },
    ];
    const mockFIYields = [
      { asset_id: 'fi-1', asset_name: 'CDB Inter', month: '2024-01', net_amount: 50, gross_amount: 60 },
    ];
    const mockTDYields = [
      { asset_id: 'td-1', asset_name: 'Tesouro Selic', month: '2024-01', net_amount: 30, gross_amount: 35 },
    ];

    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') {
        return Promise.resolve({ ok: true, json: async () => [{ id: 'p-1', name: 'Principal' }] });
      }
      if (url === '/portfolios/p-1') {
        return Promise.resolve({ ok: true, json: async () => ({ positions: mockPositions }) });
      }
      if (url === '/portfolios/p-1/history') {
        return Promise.resolve({ ok: true, json: async () => mockTxs });
      }
      if (url === '/portfolios/p-1/fixed-income/positions') {
        return Promise.resolve({ ok: true, json: async () => mockFI });
      }
      if (url === '/portfolios/p-1/treasury/positions') {
        return Promise.resolve({ ok: true, json: async () => mockTD });
      }
      if (url === '/portfolios/p-1/dividends') {
        return Promise.resolve({ ok: true, json: async () => mockDivs });
      }
      if (url === '/portfolios/p-1/fixed-income/monthly-yields') {
        return Promise.resolve({ ok: true, json: async () => mockFIYields });
      }
      if (url === '/portfolios/p-1/treasury/monthly-yields') {
        return Promise.resolve({ ok: true, json: async () => mockTDYields });
      }
      if (url.includes('/performance')) {
        return Promise.resolve({ ok: true, json: async () => [{ date: '2024-01-01', value: 1000 }] });
      }
      return Promise.resolve({ ok: false });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return (
        <div>
          <span data-testid="pos">{context.positions.length}</span>
          <span data-testid="divs">{context.dividends.length}</span>
        </div>
      );
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('pos')).toHaveTextContent('2');
      // 1 equity div + 1 FI yield + 1 TD yield = 3
      expect(screen.getByTestId('divs')).toHaveTextContent('3');
    });

    // Test performance loading with category filter
    act(() => {
      context.setActiveCategoryFilter('Ações (B3)');
    });

    await waitFor(() => {
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('&tickers=PETR4'),
        expect.any(Object)
      );
    });

    // Test performance with category filter where no asset matches
    act(() => {
      context.setActiveCategoryFilter('Cripto');
    });

    await waitFor(() => {
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('&tickers=NONE_FOUND'),
        expect.any(Object)
      );
    });

    // Test chart ticker filter
    act(() => {
      context.setFilterChartTicker('HGLG11');
    });

    await waitFor(() => {
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('&tickers=HGLG11'),
        expect.any(Object)
      );
    });
  });

  it('handles asset search debounce and asset selection', async () => {
    vi.useFakeTimers();

    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') {
        return Promise.resolve({ ok: true, json: async () => [{ id: 'p-1', name: 'Main' }] });
      }
      if (url.includes('/assets/search')) {
        return Promise.resolve({
          ok: true,
          json: async () => [{ symbol: 'AAPL', name: 'Apple Inc' }],
        });
      }
      if (url === '/quotes/AAPL') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ name: 'Apple Inc', currency: 'USD' }),
        });
      }
      if (url === '/quotes/USDBRL=X') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ price: 5.4 }),
        });
      }
      if (url === '/assets/AAPL') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ asset_type: 'STOCK_US' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await act(async () => {
      context.setSearchQuery('AAP');
    });

    act(() => {
      vi.advanceTimersByTime(400);
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(context.searchResults).toEqual([{ symbol: 'AAPL', name: 'Apple Inc' }]);
    expect(context.showDropdown).toBe(true);

    // Now select the asset
    await act(async () => {
      await context.handleSelectAsset('AAPL');
    });

    expect(context.txTicker).toBe('AAPL');
    expect(context.selectedAssetCurrency).toBe('USD');
    expect(context.txExchangeRate).toBe(5.4);
    expect(context.txAssetType).toBe('STOCK_US');

    vi.useRealTimers();
  });

  it('handles asset search clearing and select asset fallbacks', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') {
        return Promise.resolve({ ok: true, json: async () => [{ id: 'p-1' }] });
      }
      if (url === '/quotes/PETR4.SA') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ name: 'Petrobras', currency: 'BRL' }),
        });
      }
      if (url === '/assets/PETR4.SA') {
        return Promise.resolve({ ok: false }); // Asset not in DB -> fallback
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    // Empty search query clears search
    act(() => {
      context.setSearchQuery('');
    });
    expect(context.searchResults).toEqual([]);
    expect(context.showDropdown).toBe(false);

    // Select asset with BRL currency and local fallback
    await act(async () => {
      await context.handleSelectAsset('PETR4.SA');
    });

    expect(context.selectedAssetCurrency).toBe('BRL');
    expect(context.txExchangeRate).toBe(1.0);
    expect(context.txAssetType).toBe('STOCK_BR');
  });

  it('handles Telegram link generation success and error', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') return Promise.resolve({ ok: true, json: async () => [{ id: 'p-1' }] });
      if (url === '/telegram/link') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ bot_username: 'StockBot', token: 'token123' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await act(async () => {
      await context.handleLinkTelegram();
    });

    expect(window.open).toHaveBeenCalledWith('https://t.me/StockBot?start=token123', '_blank');

    // Failure branch
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    await act(async () => {
      await context.handleLinkTelegram();
    });
    expect(window.alert).toHaveBeenCalledWith('Erro ao gerar link do Telegram.');

    // Connection error branch
    (apiFetch as any).mockRejectedValueOnce(new Error('Connection error'));
    await act(async () => {
      await context.handleLinkTelegram();
    });
    expect(window.alert).toHaveBeenCalledWith('Erro ao comunicar com o servidor.');

    // Select asset with USD and rateRes.ok = false fallback
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/quotes/NVDA') return Promise.resolve({ ok: true, json: async () => ({ name: 'Nvidia', currency: 'USD' }) });
      if (url === '/quotes/USDBRL=X') return Promise.resolve({ ok: false });
      return Promise.resolve({ ok: true, json: async () => [] });
    });
    await act(async () => {
      await context.handleSelectAsset('NVDA');
    });
    expect(context.txExchangeRate).toBe(5.25);
  });

  it('handles portfolio creation, deletion, and setting default', async () => {
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/portfolios' && opts?.method === 'POST') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'p-new' }) });
      }
      if (url === '/portfolios') {
        return Promise.resolve({
          ok: true,
          json: async () => [
            { id: 'p-1', name: 'Port 1' },
            { id: 'p-2', name: 'Port 2' },
          ],
        });
      }
      if (url === '/portfolios/p-1' && opts?.method === 'DELETE') {
        return Promise.resolve({ ok: true });
      }
      if (url === '/portfolios/p-1/default' && opts?.method === 'PUT') {
        return Promise.resolve({ ok: true });
      }
      return Promise.resolve({ ok: true, json: async () => ({ positions: [] }) });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => {
      expect(context.portfolios.length).toBe(2);
    });

    // Create portfolio
    act(() => {
      context.setNewPortfolioName('Nova Carteira');
    });
    await act(async () => {
      await context.handleCreatePortfolio({ preventDefault: vi.fn() } as any);
    });
    expect(context.newPortfolioName).toBe('');

    // Set default portfolio
    await act(async () => {
      await context.handleSetDefaultPortfolio();
    });

    // Delete portfolio
    await act(async () => {
      await context.handleDeletePortfolio();
    });
    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/portfolios\/p-/),
      expect.objectContaining({ method: 'DELETE' })
    );
  });

  it('handles add and edit transactions with validation and different types', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') {
        return Promise.resolve({ ok: true, json: async () => [{ id: 'p-1' }] });
      }
      if (url === '/portfolios/p-1') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            positions: [{ asset_id: '1', ticker: 'PETR4', quantity: 100 }],
          }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => {
      expect(context.activePortfolioId).toBe('p-1');
    });

    // Validation failure: missing fields
    await act(async () => {
      await context.handleAddTransaction({ preventDefault: vi.fn() } as any);
    });
    expect(window.alert).toHaveBeenCalledWith('Preencha todos os campos obrigatórios corretamente.');

    // Sell validation failure: selling more than owned
    act(() => {
      context.setTxTicker('PETR4');
      context.setTxType('SELL');
      context.setTxQuantity(150);
      context.setTxUnitPrice('30,00');
    });
    await act(async () => {
      await context.handleAddTransaction({ preventDefault: vi.fn() } as any);
    });
    expect(window.alert).toHaveBeenCalledWith('Saldo insuficiente. Você possui apenas 100 cotas.');

    // Successful BUY transaction
    act(() => {
      context.setTxType('BUY');
      context.setTxQuantity(50);
      context.setTxUnitPrice('30,00');
      context.setTxFee('5,00');
    });
    await act(async () => {
      await context.handleAddTransaction({ preventDefault: vi.fn() } as any);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/portfolios/p-1/transactions',
      expect.objectContaining({ method: 'POST' })
    );

    // Test editing transaction: RV
    act(() => {
      context.handleEditTransaction({
        id: 'tx-1',
        asset_name: 'PETR4',
        asset_type: 'STOCK_BR',
        module: 'RV',
        type: 'BUY',
        quantity: 50,
        unit_price: 30,
        fee: 5,
        exchange_rate: 1,
        total_value: 1500,
        date: '2024-01-01',
      });
    });
    expect(context.editingTxId).toBe('tx-1');
    expect(context.showTxModal).toBe(true);

    // Save edited transaction
    await act(async () => {
      await context.handleAddTransaction({ preventDefault: vi.fn() } as any);
    });
    expect(apiFetch).toHaveBeenCalledWith(
      '/portfolios/p-1/transactions/tx-1',
      expect.objectContaining({ method: 'PUT' })
    );

    // Test editing transaction: RF with TESOURO
    act(() => {
      context.handleEditTransaction({
        id: 'tx-td',
        asset_name: 'Tesouro Selic',
        asset_type: 'TESOURO',
        module: 'RF',
        type: 'SUBSCRIPTION',
        total_value: 1000,
      });
    });
    expect(context.activeTab).toBe('tesouro');

    // Test editing transaction: RF non-TESOURO
    act(() => {
      context.handleEditTransaction({
        id: 'tx-fi',
        asset_name: 'CDB Inter',
        asset_type: 'CDB',
        module: 'RF',
        type: 'SUBSCRIPTION',
        total_value: 1000,
        date: '2024-01-01',
      });
    });
    expect(context.showFIEditModal).toBe(true);
    expect(context.editingTxId).toBe('tx-fi');
  });

  it('handles add and update fixed income transactions', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') return Promise.resolve({ ok: true, json: async () => [{ id: 'p-1' }] });
      if (url.includes('/fixed-income/assets') && !url.includes('/transactions')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'new-fi-asset' }) });
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => expect(context.activePortfolioId).toBe('p-1'));

    // Validation failure: missing fields
    await act(async () => {
      await context.handleAddFixedIncome({ preventDefault: vi.fn() } as any);
    });
    expect(window.alert).toHaveBeenCalledWith('Preencha os campos obrigatórios');

    // Successful add fixed income
    act(() => {
      context.setFiInstitution('Banco Inter');
      context.setFiRate('110');
      context.setFiAmount('5.000,00');
      context.setFiMaturityDate('2026-12-31');
    });

    await act(async () => {
      await context.handleAddFixedIncome({ preventDefault: vi.fn() } as any);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/portfolios/p-1/fixed-income/assets',
      expect.objectContaining({ method: 'POST' })
    );

    // Update FI transaction
    act(() => {
      context.handleEditTransaction({
        id: 'tx-fi-edit',
        asset_name: 'CDB Inter',
        asset_type: 'CDB',
        module: 'RF',
        type: 'SUBSCRIPTION',
        total_value: 2000,
        date: '2024-01-01',
      });
    });

    await act(async () => {
      await context.handleUpdateFITransaction({ preventDefault: vi.fn() } as any);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/portfolios/p-1/fixed-income/transactions/tx-fi-edit',
      expect.objectContaining({ method: 'PUT' })
    );
  });

  it('handles bulk file upload, delete transactions, and export portfolio', async () => {
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/portfolios') return Promise.resolve({ ok: true, json: async () => [{ id: 'p-1' }] });
      if (url.includes('/history')) {
        return Promise.resolve({
          ok: true,
          json: async () => [
            { id: 'tx-1', module: 'RV' },
            { id: 'tx-td', module: 'RF', asset_type: 'TESOURO' },
            { id: 'tx-cdb', module: 'RF', asset_type: 'CDB' },
          ],
        });
      }
      if (url.includes('/transactions/bulk')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ success: 10, errors: [] }),
        });
      }
      if (url.includes('/export')) {
        return Promise.resolve({
          ok: true,
          blob: async () => new Blob(['dummy zip content']),
          headers: {
            get: (h: string) => h === 'Content-Disposition' ? 'attachment; filename="backup-test.zip"' : null,
          },
        });
      }
      return Promise.resolve({ ok: true, json: async () => ({ positions: [] }) });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => expect(context.activePortfolioId).toBe('p-1'));

    // File upload
    const mockFile = new File(['content'], 'test.csv', { type: 'text/csv' });
    const event = {
      target: {
        files: [mockFile],
        value: 'test.csv',
      },
    } as any;

    await act(async () => {
      await context.handleFileUpload(event);
    });
    expect(window.alert).toHaveBeenCalledWith('Importação concluída com sucesso! 10 registros importados.');

    // Delete transaction: RV
    await act(async () => {
      await context.handleDeleteTransaction('tx-1');
    });
    expect(apiFetch).toHaveBeenCalledWith(
      '/portfolios/p-1/transactions/tx-1',
      expect.objectContaining({ method: 'DELETE' })
    );

    // File upload without file
    await act(async () => {
      await context.handleFileUpload({ target: { files: [] } } as any);
    });

    // File upload with errors
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: 5, errors: ['Linha 2 inválida'] }),
    });
    await act(async () => {
      await context.handleFileUpload(event);
    });
    expect(window.alert).toHaveBeenCalledWith(expect.stringContaining('Falhas:'));

    // File upload failure and error
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    await act(async () => {
      await context.handleFileUpload(event);
    });
    expect(window.alert).toHaveBeenCalledWith('Erro ao enviar arquivo.');

    (apiFetch as any).mockRejectedValueOnce(new Error('Network error'));
    await act(async () => {
      await context.handleFileUpload(event);
    });
    expect(window.alert).toHaveBeenCalledWith('Erro de conexão.');

    // Delete transaction: RF Tesouro and RF CDB and cancelled confirm
    await act(async () => {
      await context.handleDeleteTransaction('tx-td');
    });
    expect(apiFetch).toHaveBeenCalledWith(
      '/portfolios/p-1/treasury/transactions/tx-td',
      expect.objectContaining({ method: 'DELETE' })
    );

    await act(async () => {
      await context.handleDeleteTransaction('tx-cdb');
    });
    expect(apiFetch).toHaveBeenCalledWith(
      '/portfolios/p-1/fixed-income/transactions/tx-cdb',
      expect.objectContaining({ method: 'DELETE' })
    );

    // Cancelled delete confirm
    (global.confirm as any).mockReturnValueOnce(false);
    await act(async () => {
      await context.handleDeleteTransaction('tx-td');
    });

    // Export portfolio failure and error and fallback filename
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    await act(async () => {
      await context.handleExportPortfolio();
    });
    expect(window.alert).toHaveBeenCalledWith('Erro ao exportar backup.');

    (apiFetch as any).mockRejectedValueOnce(new Error('Export failed'));
    await act(async () => {
      await context.handleExportPortfolio();
    });
    expect(window.alert).toHaveBeenCalledWith('Erro de conexão ao exportar backup.');

    // Export with default filename
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      blob: async () => new Blob(['dummy']),
      headers: { get: () => null },
    });
    await act(async () => {
      await context.handleExportPortfolio();
    });
  });

  it('covers remaining branches in portfolio, transaction, and fixed income handlers', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/portfolios') return Promise.resolve({ ok: true, json: async () => [{ id: 'p-only' }] });
      return Promise.resolve({ ok: true, json: async () => ({ positions: [] }) });
    });

    let context: any;
    const TestComponent = () => {
      context = usePortfolio();
      return null;
    };

    render(
      <PortfolioProvider>
        <TestComponent />
      </PortfolioProvider>
    );

    await waitFor(() => expect(context.activePortfolioId).toBe('p-only'));

    // handleDeletePortfolio with <= 1 portfolio
    await act(async () => {
      await context.handleDeletePortfolio();
    });
    expect(window.alert).toHaveBeenCalledWith('Você precisa manter pelo menos uma carteira ativa no sistema.');

    // handleSetDefaultPortfolio error branch
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    await act(async () => {
      await context.handleSetDefaultPortfolio();
    });
    expect(window.alert).toHaveBeenCalledWith('Erro ao definir carteira padrão.');

    // handleAddTransaction with SPLIT and BONUS
    act(() => {
      context.setTxTicker('PETR4');
      context.setTxType('SPLIT');
      context.setTxQuantity(2);
    });
    await act(async () => {
      await context.handleAddTransaction({ preventDefault: vi.fn() } as any);
    });

    act(() => {
      context.setTxTicker('PETR4');
      context.setTxType('BONUS');
      context.setTxQuantity(10);
      context.setTxUnitPrice('0,00');
    });
    await act(async () => {
      await context.handleAddTransaction({ preventDefault: vi.fn() } as any);
    });

    // handleAddTransaction failure from server
    (apiFetch as any).mockResolvedValueOnce({
      ok: false,
      json: async () => ({ error: 'Transação inválida no backend' }),
    });
    act(() => {
      context.setTxTicker('PETR4');
      context.setTxType('BUY');
      context.setTxQuantity(10);
      context.setTxUnitPrice('30,00');
    });
    await act(async () => {
      await context.handleAddTransaction({ preventDefault: vi.fn() } as any);
    });
    expect(window.alert).toHaveBeenCalledWith('Transação inválida no backend');

    // handleAddFixedIncome errors
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    act(() => {
      context.setFiInstitution('Inter');
      context.setFiRate('100');
      context.setFiAmount('1000');
    });
    await act(async () => {
      await context.handleAddFixedIncome({ preventDefault: vi.fn() } as any);
    });
    expect(window.alert).toHaveBeenCalledWith('Erro ao salvar aplicação de Renda Fixa.');

    // handleUpdateFITransaction errors
    act(() => {
      context.setEditingTxId('tx-1');
      context.setFiAmount('');
    });
    await act(async () => {
      await context.handleUpdateFITransaction({ preventDefault: vi.fn() } as any);
    });
    expect(window.alert).toHaveBeenCalledWith('Preencha os campos obrigatórios');

    act(() => {
      context.setFiAmount('1000');
      context.setFiApplicationDate('2024-01-01');
      context.setFiMaturityDate('2025-01-01');
    });
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    await act(async () => {
      await context.handleUpdateFITransaction({ preventDefault: vi.fn() } as any);
    });
    expect(window.alert).toHaveBeenCalledWith('Erro ao salvar transação de Renda Fixa.');

    // Export with disposition without quotes
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      blob: async () => new Blob(['dummy']),
      headers: { get: () => 'attachment; filename=plain-name.zip' },
    });
    await act(async () => {
      await context.handleExportPortfolio();
    });

    // Export with disposition with quotes
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      blob: async () => new Blob(['dummy']),
      headers: { get: () => 'attachment; filename="backup-test.zip"' },
    });
    await act(async () => {
      await context.handleExportPortfolio();
    });

    // Test catch block in handleSelectAsset when /assets rejects
    (apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/quotes/')) return Promise.resolve({ ok: true, json: async () => ({ name: 'Test', currency: 'BRL' }) });
      if (url.includes('/assets/')) return Promise.reject(new Error('DB Error'));
      return Promise.resolve({ ok: true, json: async () => [] });
    });
    await act(async () => {
      await context.handleSelectAsset('VALE3.SA');
    });
    expect(context.txAssetType).toBe('STOCK_BR');

    // Error handling on loads
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    (apiFetch as any).mockRejectedValue(new Error('Fetch failed'));
    await act(async () => {
      await context.loadPortfolioDetails('p-only');
      await context.loadDividends('p-only');
      await context.loadPerformance('p-only', 'ALL');
      await context.loadTreasuryPositions('p-only');
    });
    consoleError.mockRestore();
  });
});

