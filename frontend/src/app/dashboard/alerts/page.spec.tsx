import { render, screen, waitFor } from '@testing-library/react';
import AlertsPage from './page';
import React from 'react';
import { vi } from 'vitest';
import { useAuth } from '@/context/AuthContext';
import { ThemeProvider } from '@/components/ThemeProvider';

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('AlertsPage', () => {
  beforeEach(() => {
    (useAuth as any).mockReturnValue({
      user: { id: 'test', name: 'Test User', token: 'token' },
      logout: vi.fn(),
      isLoading: false,
    });
    
    // Mock fetch
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue([]),
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders alerts layout', async () => {
    render(
      <ThemeProvider>
        <AlertsPage />
      </ThemeProvider>
    );
    
    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/alerts'), expect.any(Object));
    });
  });

  it('renders active, triggered (with formatDate), and disabled alert cards correctly', async () => {
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
    ];

    (global.fetch as any).mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue(mockAlerts),
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
    });

    // Verify formatDate output for triggered_at
    expect(screen.getByText(/Disparou/i)).toBeInTheDocument();
  });
});
