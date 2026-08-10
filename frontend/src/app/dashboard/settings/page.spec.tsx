import { render, screen, waitFor } from '@testing-library/react';
import SettingsPage from './page';
import React from 'react';
import { vi } from 'vitest';
import { useAuth } from '@/context/AuthContext';
import { ThemeProvider } from '@/components/ThemeProvider';

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('SettingsPage', () => {
  beforeEach(() => {
    (useAuth as any).mockReturnValue({
      user: { id: 'test', name: 'Test User', email: 'test@example.com', token: 'token' },
      logout: vi.fn(),
      updateUser: vi.fn(),
      deleteAccount: vi.fn(),
      isLoading: false,
    });

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue([]),
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders settings layout without crashing', async () => {
    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    expect(screen.getByText(/Perfil do Usuário/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/workers'), expect.any(Object));
    });
  });
});
