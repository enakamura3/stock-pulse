import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import DashboardPage from './page';
import React from 'react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useAuth } from '@/context/AuthContext';
import { ThemeProvider } from '@/components/ThemeProvider';
import { apiFetch } from '@/lib/api';

const mockGetQueryParam = vi.fn();

vi.mock('next/navigation', () => ({
  useSearchParams: () => ({
    get: mockGetQueryParam,
  }),
  usePathname: () => '/dashboard',
  useRouter: () => ({
    push: vi.fn(),
  }),
}));

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

vi.mock('@/lib/api', () => ({
  apiFetch: vi.fn(),
}));

vi.mock('@/components/dashboard/WatchlistSidebar', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/components/dashboard/WatchlistSidebar')>();
  return {
    ...actual,
    default: (props: any) => (
      <div>
        <actual.default {...props} />
        <button data-testid="force-delete-watchlist" onClick={props.onDeleteActiveWatchlist}>
          Force Delete
        </button>
      </div>
    ),
  };
});

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: any) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn();

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    setTimeout(() => {
      if (this.onopen) this.onopen();
    }, 0);
  }
}

describe('DashboardPage', () => {
  const mockLogout = vi.fn();
  const mockAlert = vi.fn();
  const mockConfirm = vi.fn(() => true);

  const mockWatchlists = [
    {
      id: 'wl-1',
      name: 'Favoritos B3',
      user_id: 'test',
      items: [
        {
          id: 'item-1',
          ticker: 'PETR4',
          name: 'Petrobras PN',
          price: 35.5,
          change: 1.2,
          change_percent: 3.5,
        },
        {
          id: 'item-2',
          ticker: 'VALE3',
          name: 'Vale ON',
          price: 60.0,
          change: -0.5,
          change_percent: -0.8,
        },
      ],
    },
    {
      id: 'wl-2',
      name: 'Internacionais',
      user_id: 'test',
      items: [],
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    MockWebSocket.instances = [];
    vi.stubGlobal('WebSocket', MockWebSocket);
    vi.stubGlobal('alert', mockAlert);
    vi.stubGlobal('confirm', mockConfirm);
    mockGetQueryParam.mockReturnValue(null);

    (useAuth as any).mockReturnValue({
      user: { id: 'test', name: 'Test User', token: 'token' },
      logout: mockLogout,
      isLoading: false,
    });

    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/watchlists') {
        return Promise.resolve({
          ok: true,
          json: async () => mockWatchlists,
        });
      }
      if (url === '/watchlists/wl-1') {
        return Promise.resolve({
          ok: true,
          json: async () => mockWatchlists[0],
        });
      }
      if (url === '/watchlists/wl-2') {
        return Promise.resolve({
          ok: true,
          json: async () => mockWatchlists[1],
        });
      }
      if (url.includes('/quotes/PETR4')) {
        return Promise.resolve({
          ok: true,
          headers: { get: (h: string) => (h === 'X-Cache' ? 'HIT' : null) },
          json: async () => ({
            symbol: 'PETR4',
            name: 'Petrobras PN',
            price: 35.5,
            change: 1.2,
            change_percent: 3.5,
            currency: 'BRL',
          }),
        });
      }
      return Promise.resolve({
        ok: true,
        headers: { get: () => null },
        json: async () => ({}),
      });
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders loading session when authLoading is true', () => {
    (useAuth as any).mockReturnValue({
      user: null,
      logout: mockLogout,
      isLoading: true,
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    expect(screen.getByText('Carregando sua sessão segura...')).toBeInTheDocument();
  });

  it('renders null when user is null and not loading', () => {
    (useAuth as any).mockReturnValue({
      user: null,
      logout: mockLogout,
      isLoading: false,
    });

    const { container } = render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    expect(container.querySelector('.app-layout')).toBeNull();
  });

  it('renders dashboard layout and fetches watchlists', async () => {
    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    expect(screen.getByText('stock-pulse')).toBeInTheDocument();

    await waitFor(() => {
      expect(apiFetch).toHaveBeenCalledWith('/watchlists');
      expect(apiFetch).toHaveBeenCalledWith('/watchlists/wl-1');
    });

    expect(screen.getByText('Favoritos B3')).toBeInTheDocument();
    expect(screen.getByText('VALE3')).toBeInTheDocument();
  });

  it('handles loadWatchlists error and empty list gracefully', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/watchlists') {
        return Promise.reject(new Error('Network error'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    const { unmount } = render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(consoleError).toHaveBeenCalledWith('Erro ao buscar favoritos:', expect.any(Error));
    });
    unmount();

    // Empty list
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/watchlists') {
        return Promise.resolve({ ok: true, json: async () => [] });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(apiFetch).toHaveBeenCalledWith('/watchlists');
    });

    consoleError.mockRestore();
  });

  it('automatically loads quote when ticker parameter is present in URL', async () => {
    mockGetQueryParam.mockImplementation((key: string) => {
      if (key === 'ticker') return 'PETR4';
      return null;
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(mockGetQueryParam).toHaveBeenCalledWith('ticker');
      expect(apiFetch).toHaveBeenCalledWith('/quotes/PETR4');
    });

    // Check active quote card renders
    await waitFor(() => {
      expect(screen.getAllByText('PETR4').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Petrobras PN').length).toBeGreaterThan(0);
    });

    // Refresh quote button click
    const refreshBtn = screen.getByRole('button', { name: /Atualizar/i });
    await act(async () => {
      fireEvent.click(refreshBtn);
    });

    expect(apiFetch).toHaveBeenCalledWith('/quotes/PETR4');
  });

  it('handles quote loading states (miss cache, server error, and network failure)', async () => {
    mockGetQueryParam.mockReturnValue('VALE3');

    // 1. Miss cache status
    (apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/quotes/VALE3')) {
        return Promise.resolve({
          ok: true,
          headers: { get: (h: string) => (h === 'X-Cache' ? 'MISS' : null) },
          json: async () => ({
            symbol: 'VALE3',
            name: 'Vale ON',
            price: 60.0,
            change: -0.5,
            change_percent: -0.8,
            currency: 'BRL',
          }),
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    const { unmount } = render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getAllByText('VALE3').length).toBeGreaterThan(0);
    });
    unmount();

    // 2. Server error on quote load
    (apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/quotes/VALE3')) {
        return Promise.resolve({
          ok: false,
          headers: { get: () => null },
          json: async () => ({ error: 'Ticker não encontrado' }),
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    const { unmount: unmount2 } = render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText(/Ticker não encontrado/)).toBeInTheDocument();
    });
    unmount2();

    // 3. Network failure on quote load
    (apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/quotes/VALE3')) {
        return Promise.reject(new Error('Network error'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText(/Falha ao se comunicar com o servidor/)).toBeInTheDocument();
    });
  });

  it('handles asset autocomplete search, focus, debounce, and selection', async () => {
    vi.useFakeTimers();

    (apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/assets/search?q=PETR')) {
        return Promise.resolve({
          ok: true,
          json: async () => [
            { symbol: 'PETR4', name: 'Petrobras PN', type: 'Equity', exchange: 'BVMF' },
            { symbol: 'PETR3', name: 'Petrobras ON', type: 'Equity', exchange: 'BVMF' },
          ],
        });
      }
      if (url.includes('/quotes/PETR4')) {
        return Promise.resolve({
          ok: true,
          headers: { get: () => 'HIT' },
          json: async () => ({
            symbol: 'PETR4',
            name: 'Petrobras PN',
            price: 35.5,
            change: 1.2,
            change_percent: 3.5,
            currency: 'BRL',
          }),
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    const searchInput = screen.getByPlaceholderText(/Pesquise ativos/i);

    // Empty search query -> dropdown closed
    fireEvent.change(searchInput, { target: { value: '' } });

    // Type query
    fireEvent.change(searchInput, { target: { value: 'PETR' } });

    // Advance debounce
    act(() => {
      vi.advanceTimersByTime(400);
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText('Petrobras ON')).toBeInTheDocument();

    // Focus input to re-show dropdown
    fireEvent.focus(searchInput);

    // Select asset
    fireEvent.click(screen.getByText('PETR4'));

    await act(async () => {
      await Promise.resolve();
    });

    expect(apiFetch).toHaveBeenCalledWith('/quotes/PETR4');
    vi.useRealTimers();
  });

  it('handles asset search network error gracefully', async () => {
    vi.useFakeTimers();
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    (apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/assets/search')) {
        return Promise.reject(new Error('Search failed'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    const searchInput = screen.getByPlaceholderText(/Pesquise ativos/i);
    fireEvent.change(searchInput, { target: { value: 'ERROR' } });

    act(() => {
      vi.advanceTimersByTime(400);
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(consoleError).toHaveBeenCalledWith('Erro na busca de ativos:', expect.any(Error));

    consoleError.mockRestore();
    vi.useRealTimers();
  });

  it('handles watchlist switching, creation, and deletion', async () => {
    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Favoritos B3')).toBeInTheDocument();
      expect(screen.getByText('Internacionais')).toBeInTheDocument();
    });

    // 1. Switch watchlist tab
    fireEvent.click(screen.getByText('Internacionais'));

    await waitFor(() => {
      expect(apiFetch).toHaveBeenCalledWith('/watchlists/wl-2');
    });

    // 2. Create watchlist (submit with empty name returns early)
    const newNameInput = screen.getByPlaceholderText('Nova Lista...');
    const createBtn = screen.getByRole('button', { name: '+ Criar' });

    fireEvent.click(createBtn);
    expect(apiFetch).not.toHaveBeenCalledWith('/watchlists', expect.objectContaining({ method: 'POST' }));

    // Create watchlist with valid name
    const newWatchlistsList = [
      ...mockWatchlists,
      { id: 'wl-3', name: 'Dividendos', user_id: 'test', items: [] },
    ];
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/watchlists' && opts?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ id: 'wl-3', name: 'Dividendos' }),
        });
      }
      if (url === '/watchlists/wl-3') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ id: 'wl-3', name: 'Dividendos', user_id: 'test', items: [] }),
        });
      }
      if (url === '/watchlists') {
        return Promise.resolve({
          ok: true,
          json: async () => newWatchlistsList,
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => mockWatchlists });
    });

    fireEvent.change(newNameInput, { target: { value: 'Dividendos' } });
    await act(async () => {
      fireEvent.click(createBtn);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/watchlists',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ name: 'Dividendos' }),
      })
    );

    // 3. Delete active watchlist - cancelled
    await waitFor(() => {
      expect(screen.getByTitle('Excluir Lista Atual')).toBeInTheDocument();
    });
    const deleteBtn = screen.getByTitle('Excluir Lista Atual');
    mockConfirm.mockReturnValueOnce(false);
    fireEvent.click(deleteBtn);
    expect(apiFetch).not.toHaveBeenCalledWith(expect.stringMatching(/\/watchlists\/wl-/), expect.objectContaining({ method: 'DELETE' }));

    // Delete active watchlist - confirmed
    mockConfirm.mockReturnValueOnce(true);
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/watchlists/wl-') && opts?.method === 'DELETE') {
        return Promise.resolve({ ok: true });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [mockWatchlists[0]] });
    });

    await act(async () => {
      fireEvent.click(deleteBtn);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/watchlists\/wl-/),
      expect.objectContaining({ method: 'DELETE' })
    );
  });

  it('prevents deleting when only one watchlist remains', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/watchlists') {
        return Promise.resolve({
          ok: true,
          json: async () => [mockWatchlists[0]],
        });
      }
      if (url === '/watchlists/wl-1') {
        return Promise.resolve({
          ok: true,
          json: async () => mockWatchlists[0],
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Favoritos B3')).toBeInTheDocument();
    });

    // Use testid button that calls props.onDeleteActiveWatchlist
    const forceDeleteBtn = screen.getByTestId('force-delete-watchlist');
    fireEvent.click(forceDeleteBtn);

    expect(mockAlert).toHaveBeenCalledWith('Você precisa manter pelo menos uma lista de favoritos.');
  });

  it('handles favorite toggling (adding and removing from active watchlist) and sidebar item removal', async () => {
    mockGetQueryParam.mockReturnValue('PETR4');

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getAllByText('PETR4').length).toBeGreaterThan(0);
    });

    // 1. Remove from favorites (activeFavorited is true because PETR4 is in wl-1)
    const favBtn = screen.getByTitle('Remover dos Favoritos');
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/items/PETR4') && opts?.method === 'DELETE') {
        return Promise.resolve({ ok: true });
      }
      if (url === '/watchlists/wl-1') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'wl-1',
            name: 'Favoritos B3',
            items: [mockWatchlists[0].items[1]], // only VALE3 left
          }),
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(favBtn);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/watchlists/wl-1/items/PETR4',
      expect.objectContaining({ method: 'DELETE' })
    );

    // 2. Add back to favorites
    await waitFor(() => {
      expect(screen.getByTitle('Adicionar aos Favoritos')).toBeInTheDocument();
    });

    const addFavBtn = screen.getByTitle('Adicionar aos Favoritos');
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/items') && opts?.method === 'POST') {
        return Promise.resolve({ ok: true });
      }
      if (url === '/watchlists/wl-1') {
        return Promise.resolve({
          ok: true,
          json: async () => mockWatchlists[0],
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(addFavBtn);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/watchlists/wl-1/items',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ ticker: 'PETR4' }),
      })
    );

    // 3. Remove item from sidebar (trash button)
    const removeButtons = screen.getAllByTitle('Remover dos favoritos');
    await act(async () => {
      fireEvent.click(removeButtons[0]);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/watchlists\/wl-1\/items\//),
      expect.objectContaining({ method: 'DELETE' })
    );
  });

  it('handles alert creation modal (open, fill, validation, server error, network crash, success)', async () => {
    mockGetQueryParam.mockReturnValue('PETR4');

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getAllByText('PETR4').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Petrobras PN').length).toBeGreaterThan(0);
    });

    // Open alert modal
    const alertBtn = screen.getByRole('button', { name: /Criar Alerta/i });
    fireEvent.click(alertBtn);

    expect(screen.getByText(/Criar Alerta de Preço/)).toBeInTheDocument();

    const priceInput = screen.getByPlaceholderText('Ex: 38.50');
    const conditionSelect = screen.getByDisplayValue(/Preço sobe acima de/);
    const submitBtn = screen.getByRole('button', { name: 'Salvar Alerta' });

    // Change condition to BELOW
    fireEvent.change(conditionSelect, { target: { value: 'BELOW' } });

    // 1. Server error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/alerts' && opts?.method === 'POST') {
        return Promise.resolve({
          ok: false,
          json: async () => ({ error: 'Limite de alertas excedido' }),
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(submitBtn);
    });

    expect(screen.getByText(/Limite de alertas excedido/)).toBeInTheDocument();

    // 2. Network error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/alerts' && opts?.method === 'POST') {
        return Promise.reject(new Error('Network alert crash'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(submitBtn);
    });

    expect(screen.getByText(/Falha ao se conectar com o servidor/)).toBeInTheDocument();

    // 3. Success
    vi.useFakeTimers();
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/alerts' && opts?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ id: 'new-alert-id' }),
        });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(submitBtn);
    });

    expect(screen.getByText(/Alerta configurado com sucesso!/)).toBeInTheDocument();

    // Fast forward 3000ms to close modal
    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.queryByText(/Criar Alerta de Preço/)).toBeNull();

    vi.useRealTimers();
  });

  it('handles WebSocket messages (price up, price down, flashing, active quote and sidebar updates, error, close)', async () => {
    vi.useFakeTimers();
    mockGetQueryParam.mockReturnValue('PETR4');

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await act(async () => {
      await Promise.resolve();
    });

    expect(MockWebSocket.instances.length).toBeGreaterThan(0);
    const wsInstance = MockWebSocket.instances[0];

    // Trigger open
    act(() => {
      if (wsInstance.onopen) wsInstance.onopen();
    });

    // 1. Send WebSocket quote update (price UP for PETR4)
    act(() => {
      if (wsInstance.onmessage) {
        wsInstance.onmessage({
          data: JSON.stringify({
            type: 'quote',
            data: {
              symbol: 'PETR4',
              name: 'Petrobras PN',
              price: 36.8, // was 35.5 -> UP
              change: 2.5,
              change_percent: 7.2,
              currency: 'BRL',
            },
          }),
        });
      }
    });

    // Advance 1000ms to clear priceFlashing
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    // 2. Send WebSocket quote update (price DOWN for PETR4)
    act(() => {
      if (wsInstance.onmessage) {
        wsInstance.onmessage({
          data: JSON.stringify({
            type: 'quote',
            data: {
              symbol: 'PETR4',
              name: 'Petrobras PN',
              price: 34.0, // was 36.8 -> DOWN
              change: -0.3,
              change_percent: -0.9,
              currency: 'BRL',
            },
          }),
        });
      }
    });

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    // 3. Send WebSocket update for a different symbol in watchlist (VALE3 price UP)
    act(() => {
      if (wsInstance.onmessage) {
        wsInstance.onmessage({
          data: JSON.stringify({
            type: 'quote',
            data: {
              symbol: 'VALE3',
              name: 'Vale ON',
              price: 62.0, // was 60.0 -> UP
              change: 1.5,
              change_percent: 2.5,
              currency: 'BRL',
            },
          }),
        });
      }
    });

    // 4. Send malformed JSON message to test catch branch
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    act(() => {
      if (wsInstance.onmessage) {
        wsInstance.onmessage({ data: 'invalid json' });
      }
    });
    expect(consoleError).toHaveBeenCalledWith('[WS] Erro ao decodificar cotação do WebSocket:', expect.any(Error));
    consoleError.mockRestore();

    // 5. Trigger WebSocket error and close
    act(() => {
      if (wsInstance.onerror) wsInstance.onerror();
    });

    act(() => {
      if (wsInstance.onclose) wsInstance.onclose();
    });

    vi.useRealTimers();
  });

  it('handles error branches for watchlist details, create, delete, toggle favorite, and sidebar removal', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    // 1. loadWatchlistDetails error on mount
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/watchlists') {
        return Promise.resolve({
          ok: true,
          json: async () => [{ id: 'wl-err', name: 'Error WL', items: [] }],
        });
      }
      if (url === '/watchlists/wl-err') {
        return Promise.reject(new Error('Details load failure'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    const { unmount } = render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(consoleError).toHaveBeenCalledWith('Erro ao buscar detalhes da watchlist:', expect.any(Error));
    });
    unmount();

    // 2. handleCreateWatchlist error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/watchlists' && opts?.method === 'POST') {
        return Promise.reject(new Error('Create WL failure'));
      }
      if (url === '/watchlists') {
        return Promise.resolve({ ok: true, json: async () => mockWatchlists });
      }
      if (url === '/watchlists/wl-1') {
        return Promise.resolve({ ok: true, json: async () => mockWatchlists[0] });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Favoritos B3')).toBeInTheDocument();
    });

    const newNameInput = screen.getByPlaceholderText('Nova Lista...');
    fireEvent.change(newNameInput, { target: { value: 'Crash' } });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '+ Criar' }));
    });
    expect(consoleError).toHaveBeenCalledWith(expect.any(Error));

    // 3. handleDeleteActiveWatchlist error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/watchlists/wl-') && opts?.method === 'DELETE') {
        return Promise.reject(new Error('Delete WL failure'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => mockWatchlists });
    });

    mockConfirm.mockReturnValueOnce(true);
    await act(async () => {
      fireEvent.click(screen.getByTitle('Excluir Lista Atual'));
    });
    expect(consoleError).toHaveBeenCalledWith(expect.any(Error));

    // 4. handleRemoveFromSidebar error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/items/') && opts?.method === 'DELETE') {
        return Promise.reject(new Error('Remove item failure'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => mockWatchlists });
    });

    const removeBtns = screen.getAllByTitle('Remover dos favoritos');
    await act(async () => {
      fireEvent.click(removeBtns[0]);
    });
    expect(consoleError).toHaveBeenCalledWith(expect.any(Error));

    // 5. handleToggleFavorite error (with active quote)
    mockGetQueryParam.mockReturnValue('PETR4');
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/quotes/PETR4')) {
        return Promise.resolve({
          ok: true,
          headers: { get: () => 'HIT' },
          json: async () => ({ symbol: 'PETR4', name: 'Petrobras', price: 35.0, change: 0, change_percent: 0, currency: 'BRL' }),
        });
      }
      if (url === '/watchlists/wl-1') {
        return Promise.resolve({ ok: true, json: async () => mockWatchlists[0] });
      }
      if (url.includes('/items')) {
        return Promise.reject(new Error('Toggle favorite failure'));
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => mockWatchlists });
    });

    const { unmount: unmountFav } = render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getAllByText('PETR4').length).toBeGreaterThan(0);
    });

    const favBtn = screen.getByTitle('Remover dos Favoritos');
    await act(async () => {
      fireEvent.click(favBtn);
    });
    expect(consoleError).toHaveBeenCalledWith(expect.any(Error));
    unmountFav();

    consoleError.mockRestore();
  });

  it('handles websocket message when watchlist items is undefined', async () => {
    const watchlistsWithoutItems = [
      {
        id: 'wl-no-items',
        name: 'Empty Items',
        user_id: 'test',
        items: undefined,
      },
    ];

    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/watchlists') {
        return Promise.resolve({ ok: true, json: async () => watchlistsWithoutItems });
      }
      return Promise.resolve({ ok: true, headers: { get: () => null }, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await act(async () => {
      await Promise.resolve();
    });

    const wsInstance = MockWebSocket.instances[0];

    act(() => {
      if (wsInstance.onmessage) {
        wsInstance.onmessage({
          data: JSON.stringify({
            type: 'quote',
            data: {
              symbol: 'PETR4',
              price: 36.0,
            },
          }),
        });
      }
    });
  });
});
