import { render, screen, waitFor } from '@testing-library/react';
import DashboardPage from './page';
import React from 'react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useAuth } from '@/context/AuthContext';
import { ThemeProvider } from '@/components/ThemeProvider';

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

describe('DashboardPage', () => {
  beforeEach(() => {
    mockGetQueryParam.mockReturnValue(null);
    (useAuth as any).mockReturnValue({
      user: { id: 'test', name: 'Test User', token: 'token' },
      logout: vi.fn(),
      isLoading: false,
    });
    
    // Mock fetch
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: vi.fn().mockReturnValue('HIT') },
      json: vi.fn().mockResolvedValue([]),
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders dashboard layout and fetches watchlists', async () => {
    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );
    
    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/watchlists'), expect.any(Object));
    });
  });

  it('automatically loads quote when ticker parameter is present in URL', async () => {
    mockGetQueryParam.mockImplementation((key: string) => {
      if (key === 'ticker') return 'PETR4';
      return null;
    });

    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/quotes/PETR4')) {
        return Promise.resolve({
          ok: true,
          headers: { get: vi.fn().mockReturnValue('HIT') },
          json: () => Promise.resolve({
            symbol: 'PETR4',
            price: 35.5,
            change: 1.2,
            change_percent: 3.5,
          }),
        });
      }
      return Promise.resolve({
        ok: true,
        headers: { get: vi.fn().mockReturnValue(null) },
        json: () => Promise.resolve([]),
      });
    });

    render(
      <ThemeProvider>
        <DashboardPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/quotes/PETR4'), expect.any(Object));
    });
  });
});
