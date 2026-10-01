import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import AlertsPage from './page';
import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useAuth } from '@/context/AuthContext';
import { ThemeProvider } from '@/components/ThemeProvider';
import { apiFetch } from '@/lib/api';

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

vi.mock('@/lib/api', () => ({
  apiFetch: vi.fn(),
}));

describe('AlertsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (useAuth as any).mockReturnValue({
      user: { id: 'test', name: 'Test User', token: 'token' },
      logout: vi.fn(),
      isLoading: false,
    });
    const mockConfirm = vi.fn(() => true);
    const mockAlert = vi.fn();
    vi.stubGlobal('confirm', mockConfirm);
    vi.stubGlobal('alert', mockAlert);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders loading session when auth is loading', () => {
    (useAuth as any).mockReturnValue({
      user: null,
      logout: vi.fn(),
      isLoading: true,
    });

    render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );

    expect(screen.getByText('Carregando sua sessão segura...')).toBeInTheDocument();
  });

  it('returns null when user is not authenticated', () => {
    (useAuth as any).mockReturnValue({
      user: null,
      logout: vi.fn(),
      isLoading: false,
    });

    const { container } = render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );

    expect(container.firstChild).toBeNull();
  });

  it('renders empty alerts and handles load errors', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    (apiFetch as any).mockRejectedValueOnce(new Error('Load error'));

    render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText(/Nenhum alerta ativo no momento/i)).toBeInTheDocument();
      expect(screen.getByText(/Nenhum alerta disparado recentemente/i)).toBeInTheDocument();
      expect(screen.getByText(/Nenhum alerta pausado/i)).toBeInTheDocument();
    });
    consoleError.mockRestore();
  });

  it('handles asset search autocomplete, selection, mouse hover and clearing', async () => {
    vi.useFakeTimers();

    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/alerts') {
        return Promise.resolve({ ok: true, json: async () => [] });
      }
      if (url.includes('/assets/search')) {
        return Promise.resolve({
          ok: true,
          json: async () => [
            { symbol: 'PETR4.SA', name: 'Petrobras', exchange: 'BVMF', type: 'Equity' },
          ],
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );

    const searchInput = screen.getByPlaceholderText(/Ticker\.\.\. Ex: VALE3\.SA/i);

    // Enter search text
    fireEvent.change(searchInput, { target: { value: 'PETR' } });

    // Advance debounce
    act(() => {
      vi.advanceTimersByTime(400);
    });

    await act(async () => {
      await Promise.resolve();
    });

    // Dropdown item should appear
    const item = screen.getByText('PETR4.SA');
    expect(item).toBeInTheDocument();

    // Trigger mouse enter and leave on dropdown item
    fireEvent.mouseEnter(item.closest('div')!);
    fireEvent.mouseLeave(item.closest('div')!);

    // Re-focus search input when searchResults exist
    fireEvent.focus(searchInput);

    // Click item to select
    fireEvent.click(item);

    // Selected asset details should appear
    expect(screen.getByText('Empresa Selecionada')).toBeInTheDocument();
    expect(screen.getByText('Petrobras')).toBeInTheDocument();

    // Changing search query after selection clears selectedAsset
    fireEvent.change(searchInput, { target: { value: 'VALE' } });
    expect(screen.queryByText('Empresa Selecionada')).not.toBeInTheDocument();

    // Clearing search query clears results
    fireEvent.change(searchInput, { target: { value: '' } });

    // Test search catch error
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    (apiFetch as any).mockRejectedValueOnce(new Error('Search failed'));
    fireEvent.change(searchInput, { target: { value: 'ERR' } });
    act(() => {
      vi.advanceTimersByTime(400);
    });
    await act(async () => {
      await Promise.resolve();
    });
    consoleError.mockRestore();

    vi.useRealTimers();
  });

  it('handles alert creation validation, server error, and success flow', async () => {
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/alerts' && opts?.method === 'POST') {
        const body = JSON.parse(opts.body);
        if (body.target_price === 999) {
          return Promise.resolve({ ok: false, json: async () => ({ error: 'Preço inválido' }) });
        }
        return Promise.resolve({ ok: true, json: async () => ({ id: 'new-alert' }) });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );

    const submitBtn = screen.getByRole('button', { name: /Ativar Alerta/i });

    // Submit without selected asset
    fireEvent.click(submitBtn);
    // Since button is disabled when !selectedAsset, trigger submit directly on form
    const form = submitBtn.closest('form')!;
    fireEvent.submit(form);
    expect(screen.getByText('⚠️ Por favor, selecione um ativo válido e insira o preço alvo.')).toBeInTheDocument();

    // Select asset via autocomplete
    vi.useFakeTimers();
    (apiFetch as any).mockImplementationOnce((url: string) => {
      return Promise.resolve({
        ok: true,
        json: async () => [{ symbol: 'VALE3.SA', name: 'Vale', exchange: 'BVMF', type: 'Equity' }],
      });
    });

    const searchInput = screen.getByPlaceholderText(/Ticker\.\.\. Ex: VALE3\.SA/i);
    fireEvent.change(searchInput, { target: { value: 'VALE' } });
    act(() => {
      vi.advanceTimersByTime(400);
    });
    await act(async () => {
      await Promise.resolve();
    });

    fireEvent.click(screen.getByText('VALE3.SA'));
    vi.useRealTimers();

    // Change condition to BELOW
    const conditionSelect = screen.getByDisplayValue(/Sobe acima de/);
    fireEvent.change(conditionSelect, { target: { value: 'BELOW' } });

    // Server error test
    const priceInput = screen.getByPlaceholderText('Ex: 38.50');
    fireEvent.change(priceInput, { target: { value: '999' } });
    await act(async () => {
      fireEvent.submit(form);
    });
    expect(screen.getByText('⚠️ Preço inválido')).toBeInTheDocument();

    // Connection failure test
    (apiFetch as any).mockRejectedValueOnce(new Error('Network error'));
    fireEvent.change(priceInput, { target: { value: '60' } });
    await act(async () => {
      fireEvent.submit(form);
    });
    expect(screen.getByText('⚠️ Falha ao conectar com o servidor.')).toBeInTheDocument();

    // Success flow
    fireEvent.change(priceInput, { target: { value: '55.00' } });
    await act(async () => {
      fireEvent.submit(form);
    });
    expect(screen.getByText(/Alerta configurado com sucesso!/i)).toBeInTheDocument();
  });

  it('renders alerts and handles toggle and delete actions and date formatting', async () => {
    const mockAlerts = [
      {
        id: 'alert-1',
        user_id: 'test',
        asset_id: 'asset-1',
        ticker: 'PETR4',
        asset_name: 'Petrobras',
        currency: 'BRL',
        target_price: 38.5,
        condition: 'ABOVE',
        status: 'ACTIVE',
        created_at: '2026-08-01T10:00:00Z',
      },
      {
        id: 'alert-2',
        user_id: 'test',
        asset_id: 'asset-2',
        ticker: 'VALE3',
        asset_name: 'Vale',
        currency: 'BRL',
        target_price: 60.0,
        condition: 'BELOW',
        status: 'TRIGGERED',
        triggered_at: '2026-08-09T14:30:00Z',
        created_at: '2026-08-01T10:00:00Z',
      },
      {
        id: 'alert-3',
        user_id: 'test',
        asset_id: 'asset-3',
        ticker: 'ITUB4',
        asset_name: 'Itaú Unibanco',
        currency: 'BRL',
        target_price: 32.0,
        condition: 'ABOVE',
        status: 'DISABLED',
        created_at: '2026-08-01T10:00:00Z',
      },
      {
        id: 'alert-4',
        user_id: 'test',
        asset_id: 'asset-4',
        ticker: 'BBDC4',
        asset_name: 'Bradesco',
        currency: 'BRL',
        target_price: 15.0,
        condition: 'BELOW',
        status: 'TRIGGERED',
        triggered_at: 'invalid-date-string',
        created_at: '2026-08-01T10:00:00Z',
      },
    ];

    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      console.log('MOCK CALLED URL:', url, 'OPTS:', opts);
      if (url === '/alerts') {
        return Promise.resolve({ ok: true, json: async () => mockAlerts });
      }
      if (url.includes('/toggle') && opts?.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ status: 'DISABLED' }) });
      }
      if (url.includes('/alerts/') && opts?.method === 'DELETE') {
        return Promise.resolve({ ok: true });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('PETR4')).toBeInTheDocument();
      expect(screen.getByText('VALE3')).toBeInTheDocument();
      expect(screen.getByText('ITUB4')).toBeInTheDocument();
      expect(screen.getByText('BBDC4')).toBeInTheDocument();
    });

    // Toggle active alert to disabled
    const pauseBtn = screen.getByRole('button', { name: 'Pausar' });
    await act(async () => {
      fireEvent.click(pauseBtn);
    });

    // Toggle disabled alert to active
    const activateBtn = screen.getAllByRole('button', { name: 'Ativar' })[0];
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ status: 'ACTIVE' }),
    });
    await act(async () => {
      fireEvent.click(activateBtn);
    });

    // Toggle error branch
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    (apiFetch as any).mockRejectedValueOnce(new Error('Toggle error'));
    const toggleBtnForError = screen.getAllByRole('button', { name: 'Ativar' })[0];
    await act(async () => {
      fireEvent.click(toggleBtnForError);
    });
    expect(consoleError).toHaveBeenCalledWith(expect.any(Error));
    consoleError.mockRestore();

    const deleteButtons = screen.getAllByTitle(/Excluir/);
    
    // Delete alert - cancelled
    (global.confirm as any).mockReturnValueOnce(false);
    fireEvent.click(deleteButtons[0]);

    // Delete alert - confirmed success
    await act(async () => {
      fireEvent.click(deleteButtons[0]);
    });
    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/alerts\/alert-/),
      expect.objectContaining({ method: 'DELETE' })
    );

    // Delete alert error branch
    const consoleError2 = vi.spyOn(console, 'error').mockImplementation(() => {});
    (apiFetch as any).mockRejectedValueOnce(new Error('Delete error'));
    const remainingDeleteButtons = screen.getAllByTitle(/Excluir/);
    await act(async () => {
      fireEvent.click(remainingDeleteButtons[0]);
    });
    expect(consoleError2).toHaveBeenCalledWith(expect.any(Error));
    consoleError2.mockRestore();
  });

  it('handles formatDate exception gracefully', async () => {
    const getYearSpy = vi.spyOn(Date.prototype, 'getFullYear').mockImplementationOnce(() => {
      throw new Error('Year crash');
    });

    const mockAlerts = [
      {
        id: 'alert-crash',
        user_id: 'test',
        asset_id: 'asset-crash',
        ticker: 'CRASH3',
        asset_name: 'Crash SA',
        currency: 'BRL',
        target_price: 10.0,
        condition: 'BELOW',
        status: 'TRIGGERED',
        triggered_at: '2026-08-09T14:30:00Z',
        created_at: '2026-08-01T10:00:00Z',
      },
    ];

    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockAlerts,
    });

    render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('CRASH3')).toBeInTheDocument();
    });

    expect(screen.getByText(/Disparou 2026-08-09T14:30:00Z/)).toBeInTheDocument();
    getYearSpy.mockRestore();
  });
});

