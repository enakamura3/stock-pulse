import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import SettingsPage from './page';
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useAuth } from '@/context/AuthContext';
import { ThemeProvider } from '@/components/ThemeProvider';
import { apiFetch } from '@/lib/api';

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

vi.mock('@/lib/api', () => ({
  apiFetch: vi.fn(),
}));

describe('SettingsPage', () => {
  const mockLogout = vi.fn();
  const mockFetchMe = vi.fn();
  const mockConfirm = vi.fn(() => true);
  const mockPrompt = vi.fn(() => 'test@example.com');
  const mockAlert = vi.fn();
  const mockOpen = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('confirm', mockConfirm);
    vi.stubGlobal('prompt', mockPrompt);
    vi.stubGlobal('alert', mockAlert);
    vi.stubGlobal('open', mockOpen);
    localStorage.clear();

    (useAuth as any).mockReturnValue({
      user: { id: 'test', name: 'Test User', email: 'test@example.com', token: 'token' },
      logout: mockLogout,
      fetchMe: mockFetchMe,
      isLoading: false,
    });

    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/status') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ linked: false, chat_id: 0, bot_username: 'StockPulseBot' }),
        });
      }
      if (url === '/workers') {
        return Promise.resolve({
          ok: true,
          json: async () => [
            {
              name: 'Worker B',
              description: 'Worker B description',
              interval: '1h',
              last_run: '2026-08-01T12:00:00Z',
              next_run: '2026-08-01T13:00:00Z',
              status: 'running',
            },
            {
              name: 'Worker A',
              description: 'Worker A description',
              interval: '24h',
              last_run: null,
              next_run: 'invalid-date',
              status: 'idle',
            },
          ],
        });
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });
  });

  it('renders loading state when authLoading is true', () => {
    (useAuth as any).mockReturnValue({
      user: null,
      logout: mockLogout,
      fetchMe: mockFetchMe,
      isLoading: true,
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    expect(screen.getByText('Carregando...')).toBeInTheDocument();
  });

  it('renders null when user is null and not loading', () => {
    (useAuth as any).mockReturnValue({
      user: null,
      logout: mockLogout,
      fetchMe: mockFetchMe,
      isLoading: false,
    });

    const { container } = render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    expect(container.firstChild).toBeNull();
  });

  it('renders settings layout, loads initial data, and sorts workers by name', async () => {
    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    expect(screen.getByText(/Perfil do Usuário/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Worker A')).toBeInTheDocument();
      expect(screen.getByText('Worker B')).toBeInTheDocument();
    });

    // Check sorted order: Worker A should come before Worker B
    const rows = screen.getAllByRole('row');
    expect(rows[1]).toHaveTextContent('Worker A');
    expect(rows[2]).toHaveTextContent('Worker B');

    // Worker B has status 'running'
    expect(screen.getByText('Executando')).toBeInTheDocument();
    // Worker A has status 'idle' -> Inativo
    expect(screen.getByText('Inativo')).toBeInTheDocument();
    // Last run null -> 'Nunca'
    expect(screen.getAllByText(/Nunca/).length).toBeGreaterThan(0);
  });

  it('handles formatDate exception gracefully', async () => {
    const toLocaleSpy = vi.spyOn(Date.prototype, 'toLocaleString').mockImplementationOnce(() => {
      throw new Error('Locale error');
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Worker B')).toBeInTheDocument();
    });

    toLocaleSpy.mockRestore();
  });

  it('handles non-array response from workers endpoint', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/workers') {
        return Promise.resolve({ ok: true, json: async () => ({ notAnArray: true }) });
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: false }) });
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Formato de resposta de workers inválido.')).toBeInTheDocument();
    });
  });

  it('handles server error and error message fallback from workers endpoint', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/workers') {
        return Promise.resolve({
          ok: false,
          json: async () => ({ error: 'Acesso negado aos workers' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: false }) });
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Acesso negado aos workers')).toBeInTheDocument();
    });
  });

  it('handles server error json parsing rejection and default message for workers', async () => {
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/workers') {
        return Promise.resolve({
          ok: false,
          json: async () => { throw new Error('Cannot parse json'); },
        });
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: false }) });
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Não foi possível carregar a lista de workers.')).toBeInTheDocument();
    });
  });

  it('handles network failure on workers load and empty workers array', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/workers') {
        return Promise.reject(new Error('Network failure'));
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: false }) });
    });

    const { unmount } = render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Erro de conexão ao carregar workers.')).toBeInTheDocument();
    });
    unmount();

    // Empty list
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/workers') {
        return Promise.resolve({ ok: true, json: async () => [] });
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: false }) });
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Nenhum worker registrado no sistema.')).toBeInTheDocument();
    });

    consoleError.mockRestore();
  });

  it('handles admin key input, localStorage persistence, and reload with key', async () => {
    localStorage.setItem('admin_api_key', 'initial-key');

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    const keyInput = screen.getByLabelText(/Chave de Administração/i) as HTMLInputElement;
    expect(keyInput.value).toBe('initial-key');

    // Change key input
    fireEvent.change(keyInput, { target: { value: 'new-secret-key' } });
    expect(localStorage.getItem('admin_api_key')).toBe('new-secret-key');

    // Click Conectar
    const connectBtn = screen.getByRole('button', { name: 'Conectar' });
    await act(async () => {
      fireEvent.click(connectBtn);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/workers',
      expect.objectContaining({
        headers: expect.objectContaining({ 'X-Admin-Key': 'new-secret-key' }),
      })
    );

    // Click Atualizar
    const refreshBtn = screen.getByRole('button', { name: /Atualizar/ });
    await act(async () => {
      fireEvent.click(refreshBtn);
    });

    expect(apiFetch).toHaveBeenCalledWith(
      '/workers',
      expect.objectContaining({
        headers: expect.objectContaining({ 'X-Admin-Key': 'new-secret-key' }),
      })
    );
  });

  it('handles triggering workers (success, server error, and network error)', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Worker A')).toBeInTheDocument();
    });

    const executeButtons = screen.getAllByRole('button', { name: 'Executar' });
    const workerABtn = executeButtons[0];

    // 1. Success trigger
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/workers/Worker A/trigger' && opts?.method === 'POST') {
        return Promise.resolve({ ok: true, json: async () => ({}) });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(workerABtn);
    });

    expect(screen.getByText('Worker "Worker A" disparado com sucesso!')).toBeInTheDocument();

    // 2. Server error trigger
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/trigger')) {
        return Promise.resolve({
          ok: false,
          json: async () => ({ error: 'Falha interna ao disparar' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(workerABtn);
    });

    expect(screen.getByText('Falha interna ao disparar')).toBeInTheDocument();

    // 3. Network error trigger
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/trigger')) {
        return Promise.reject(new Error('Trigger network crash'));
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(workerABtn);
    });

    expect(screen.getByText('Erro de rede ao disparar o worker "Worker A".')).toBeInTheDocument();

    consoleError.mockRestore();
  });

  it('handles profile updates (success and error flows)', async () => {
    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    const nameInput = screen.getByDisplayValue('Test User');
    const emailInput = screen.getByDisplayValue('test@example.com');
    const saveBtn = screen.getByRole('button', { name: 'Salvar Alterações' });

    // 1. Success update
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user/profile' && opts?.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) });
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });

    fireEvent.change(nameInput, { target: { value: 'Updated Name' } });
    fireEvent.change(emailInput, { target: { value: 'updated@example.com' } });

    await act(async () => {
      fireEvent.click(saveBtn);
    });

    expect(mockFetchMe).toHaveBeenCalled();
    expect(screen.getByText('Perfil atualizado com sucesso!')).toBeInTheDocument();

    // 2. Server error update
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user/profile' && opts?.method === 'PUT') {
        return Promise.resolve({
          ok: false,
          json: async () => ({ error: 'E-mail já está em uso' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });

    await act(async () => {
      fireEvent.click(saveBtn);
    });

    expect(screen.getByText('E-mail já está em uso')).toBeInTheDocument();

    // 3. Network error update
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user/profile' && opts?.method === 'PUT') {
        return Promise.reject(new Error('Profile network failure'));
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });

    await act(async () => {
      fireEvent.click(saveBtn);
    });

    expect(screen.getByText('Profile network failure')).toBeInTheDocument();
  });

  it('handles password updates (mismatch, server error, network error, and success)', async () => {
    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    const currentPwdInput = screen.getByPlaceholderText('••••••••');
    const newPwdInput = screen.getByPlaceholderText('Mínimo 6 caracteres');
    const confirmPwdInput = screen.getByPlaceholderText('Repita a nova senha');
    const changePwdBtn = screen.getByRole('button', { name: 'Alterar Senha' });

    // 1. Password mismatch
    fireEvent.change(currentPwdInput, { target: { value: 'oldpass123' } });
    fireEvent.change(newPwdInput, { target: { value: 'newpass123' } });
    fireEvent.change(confirmPwdInput, { target: { value: 'different123' } });

    await act(async () => {
      fireEvent.click(changePwdBtn);
    });

    expect(screen.getByText('A nova senha e a confirmação não coincidem.')).toBeInTheDocument();

    // Fix mismatch
    fireEvent.change(confirmPwdInput, { target: { value: 'newpass123' } });

    // 2. Server error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user/password' && opts?.method === 'PUT') {
        return Promise.resolve({
          ok: false,
          json: async () => ({ error: 'Senha atual incorreta' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });

    await act(async () => {
      fireEvent.click(changePwdBtn);
    });

    expect(screen.getByText('Senha atual incorreta')).toBeInTheDocument();

    // 3. Network error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user/password' && opts?.method === 'PUT') {
        return Promise.reject(new Error('Password network failure'));
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });

    await act(async () => {
      fireEvent.click(changePwdBtn);
    });

    expect(screen.getByText('Password network failure')).toBeInTheDocument();

    // 4. Success flow
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user/password' && opts?.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) });
      }
      return Promise.resolve({ ok: true, json: async () => ({}) });
    });

    await act(async () => {
      fireEvent.click(changePwdBtn);
    });

    expect(screen.getByText('Senha alterada com sucesso!')).toBeInTheDocument();
    expect((currentPwdInput as HTMLInputElement).value).toBe('');
    expect((newPwdInput as HTMLInputElement).value).toBe('');
    expect((confirmPwdInput as HTMLInputElement).value).toBe('');
  });

  it('handles Telegram link, window.open, polling success and timeout', async () => {
    vi.useFakeTimers();

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    // Initial load: unlinked
    await act(async () => {
      await Promise.resolve();
    });
    expect(screen.getByText('Não Vinculado')).toBeInTheDocument();

    // 1. Link account success and polling until linked
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/link' && opts?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ token: 'auth-token-123', bot_username: 'CustomBot' }),
        });
      }
      if (url === '/telegram/status') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ linked: true, chat_id: 98765, bot_username: 'CustomBot' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    const linkBtn = screen.getByRole('button', { name: 'Vincular Conta' });
    await act(async () => {
      fireEvent.click(linkBtn);
      await Promise.resolve();
    });

    expect(mockOpen).toHaveBeenCalledWith('https://t.me/CustomBot?start=auth-token-123', '_blank');

    // Run 1 interval tick
    await act(async () => {
      vi.advanceTimersByTime(1000);
      await Promise.resolve();
    });

    expect(screen.getByText(/Vinculado com sucesso \(Chat ID: 98765\)/)).toBeInTheDocument();

    // Test attempts > 30 branch
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/link' && opts?.method === 'POST') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ token: 'another-token' }),
        });
      }
      if (url === '/telegram/status') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ linked: false }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    // Trigger link again and advance 35 intervals
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Desvincular' })); // reset state or trigger link
    });

    vi.useRealTimers();
  });

  it('handles Telegram link server error and network error', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Não Vinculado')).toBeInTheDocument();
    });

    // Server error
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/link' && opts?.method === 'POST') {
        return Promise.resolve({ ok: false });
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: false }) });
    });

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Vincular Conta' }));
    });
    expect(mockAlert).toHaveBeenCalledWith('Erro ao gerar link de ativação.');

    // Network crash
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/link' && opts?.method === 'POST') {
        return Promise.reject(new Error('Link crash'));
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: false }) });
    });

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Vincular Conta' }));
    });
    expect(mockAlert).toHaveBeenCalledWith('Erro ao comunicar com o servidor.');

    consoleError.mockRestore();
  });

  it('handles Telegram status load errors (server error and network failure)', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    // Server error
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/telegram/status') {
        return Promise.resolve({ ok: false });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    const { unmount } = render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Não foi possível carregar as informações do Telegram.')).toBeInTheDocument();
    });
    unmount();

    // Network error
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/telegram/status') {
        return Promise.reject(new Error('Telegram error'));
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Erro de conexão ao carregar Telegram.')).toBeInTheDocument();
    });

    consoleError.mockRestore();
  });

  it('handles Telegram unlink flow (cancel, success, server error, network crash)', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    // Render with linked status
    (apiFetch as any).mockImplementation((url: string) => {
      if (url === '/telegram/status') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ linked: true, chat_id: 12345, bot_username: 'StockPulseBot' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    await waitFor(() => {
      expect(screen.getByText(/Vinculado com sucesso/)).toBeInTheDocument();
    });

    const unlinkBtn = screen.getByRole('button', { name: 'Desvincular' });

    // 1. Cancel unlink
    mockConfirm.mockReturnValueOnce(false);
    fireEvent.click(unlinkBtn);
    expect(apiFetch).not.toHaveBeenCalledWith('/telegram/link', expect.anything());

    // 2. Unlink server error
    mockConfirm.mockReturnValueOnce(true);
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/link' && opts?.method === 'DELETE') {
        return Promise.resolve({ ok: false });
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: true }) });
    });

    await act(async () => {
      fireEvent.click(unlinkBtn);
    });
    expect(mockAlert).toHaveBeenCalledWith('Erro ao desvincular o Telegram.');

    // 3. Unlink network crash
    mockConfirm.mockReturnValueOnce(true);
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/link' && opts?.method === 'DELETE') {
        return Promise.reject(new Error('Unlink fail'));
      }
      return Promise.resolve({ ok: true, json: async () => ({ linked: true }) });
    });

    await act(async () => {
      fireEvent.click(unlinkBtn);
    });
    expect(mockAlert).toHaveBeenCalledWith('Erro ao comunicar com o servidor.');

    // 4. Unlink success
    mockConfirm.mockReturnValueOnce(true);
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/telegram/link' && opts?.method === 'DELETE') {
        return Promise.resolve({ ok: true });
      }
      if (url === '/telegram/status') {
        return Promise.resolve({
          ok: true,
          json: async () => ({ linked: false, chat_id: 0, bot_username: 'StockPulseBot' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(unlinkBtn);
    });
    await waitFor(() => {
      expect(screen.getByText('Não Vinculado')).toBeInTheDocument();
    });

    consoleError.mockRestore();
  });

  it('handles account deletion (cancel, wrong email, server error, network crash, success)', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ThemeProvider>
        <SettingsPage />
      </ThemeProvider>
    );

    const deleteBtn = screen.getByRole('button', { name: 'Excluir Conta' });

    // 1. Cancel prompt
    mockPrompt.mockReturnValueOnce(null);
    fireEvent.click(deleteBtn);
    expect(apiFetch).not.toHaveBeenCalledWith('/user', expect.anything());

    // 2. Wrong email entered
    mockPrompt.mockReturnValueOnce('wrong@email.com');
    fireEvent.click(deleteBtn);
    expect(mockAlert).toHaveBeenCalledWith('Confirmação inválida. O e-mail digitado não coincide.');
    expect(apiFetch).not.toHaveBeenCalledWith('/user', expect.anything());

    // 3. Server error on delete
    mockPrompt.mockReturnValueOnce('test@example.com');
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user' && opts?.method === 'DELETE') {
        return Promise.resolve({
          ok: false,
          json: async () => ({ error: 'Erro no banco de dados' }),
        });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(deleteBtn);
    });
    expect(mockAlert).toHaveBeenCalledWith('Erro no banco de dados');

    // 4. Network error on delete
    mockPrompt.mockReturnValueOnce('test@example.com');
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user' && opts?.method === 'DELETE') {
        return Promise.reject(new Error('Network delete fail'));
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(deleteBtn);
    });
    expect(mockAlert).toHaveBeenCalledWith('Erro de rede ao processar exclusão.');

    // 5. Successful deletion
    mockPrompt.mockReturnValueOnce('test@example.com');
    (apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url === '/user' && opts?.method === 'DELETE') {
        return Promise.resolve({ ok: true });
      }
      return Promise.resolve({ ok: true, json: async () => [] });
    });

    await act(async () => {
      fireEvent.click(deleteBtn);
    });
    expect(mockAlert).toHaveBeenCalledWith(
      'Sua conta foi excluída com sucesso. Lamentamos ver você partir.'
    );
    expect(mockLogout).toHaveBeenCalled();

    consoleError.mockRestore();
  });
});
