import { render, screen, waitFor } from '@testing-library/react';
import DashboardPage from './page';
import React from 'react';
import { vi } from 'vitest';
import { useAuth } from '@/context/AuthContext';

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('DashboardPage', () => {
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

  it('renders dashboard layout and fetches watchlists', async () => {
    render(<DashboardPage />);
    
    expect(screen.getByText('StockPulse')).toBeInTheDocument();
    
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/watchlists'), expect.any(Object));
    });
  });
});
