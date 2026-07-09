import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { AuthProvider, useAuth } from './AuthContext';
import React from 'react';

// Mock de fetch global
global.fetch = vi.fn();

// Mock do useRouter do Next.js
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    prefetch: vi.fn(),
  })
}));

// Componente de Teste Consumidor
const TestComponent = () => {
  const { user, isAuthenticated, isLoading, login, register, logout } = useAuth();
  const [error, setError] = React.useState('');
  
  const handleLogin = async () => {
    try { await login('test@test.com', '123'); }
    catch(e: any) { setError(e.message); }
  };

  const handleRegister = async () => {
    try { await register('Test', 'test@test.com', '123'); }
    catch(e: any) { setError(e.message); }
  };
  
  if (isLoading) return <div data-testid="loading">Loading...</div>;
  
  return (
    <div>
      <div data-testid="status">{isAuthenticated ? 'Auth' : 'Not Auth'}</div>
      <div data-testid="username">{user?.name || 'No User'}</div>
      <div data-testid="error">{error}</div>
      <button onClick={handleLogin} data-testid="btn-login">Login</button>
      <button onClick={handleRegister} data-testid="btn-register">Register</button>
      <button onClick={() => logout()} data-testid="btn-logout">Logout</button>
    </div>
  );
};

describe('AuthContext', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    // Previne que window.location altere durante testes (evita erros do jsdom)
    delete (window as any).location;
    window.location = { href: '' } as any;
  });

  it('deve falhar e exibir loading falso se /auth/me falhar (Sem Sessão Inicial)', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({ error: 'Error' })
    });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    expect(screen.getByTestId('loading')).toBeInTheDocument();
    
    await waitFor(() => {
      expect(screen.getByTestId('status')).toHaveTextContent('Not Auth');
    });
  });

  it('deve carregar sessão com sucesso se /auth/me retornar user', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: '1', name: 'Onigiri', email: 'test@test.com' })
    });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('status')).toHaveTextContent('Auth');
      expect(screen.getByTestId('username')).toHaveTextContent('Onigiri');
    });
  });

  it('deve realizar refresh token silencioso se /auth/me der 401', async () => {
    // 1º chamada: /me falha (401)
    (global.fetch as any).mockResolvedValueOnce({ ok: false, status: 401 });
    // 2º chamada: /refresh sucesso
    (global.fetch as any).mockResolvedValueOnce({ ok: true });
    // 3º chamada: /me sucesso com dados recuperados
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: '1', name: 'RefreshUser' })
    });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(3);
      expect(screen.getByTestId('username')).toHaveTextContent('RefreshUser');
    });
  });

  it('deve falhar o refresh token silencioso e retornar null', async () => {
    // 1º chamada: /me falha (401)
    (global.fetch as any).mockResolvedValueOnce({ ok: false, status: 401 });
    // 2º chamada: /refresh falha
    (global.fetch as any).mockResolvedValueOnce({ ok: false, status: 401 });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledTimes(2);
      expect(screen.getByTestId('status')).toHaveTextContent('Not Auth');
    });
  });

  it('deve efetuar login via button', async () => {
    // Estado inicial sem logar
    (global.fetch as any).mockResolvedValueOnce({ ok: false, status: 400 });
    
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('Not Auth'));

    // Configura Mock do Login
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: '1', name: 'LoginUser' })
    });

    await userEvent.click(screen.getByTestId('btn-login'));
    
    await waitFor(() => {
      expect(window.location.href).toBe('/dashboard/portfolio');
    });
  });

  it('deve repassar o erro se a chamada de login falhar no catch', async () => {
    (global.fetch as any).mockResolvedValueOnce({ ok: false, status: 401 });
    (global.fetch as any).mockResolvedValueOnce({ ok: false, status: 401 }); // mock refresh
    
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('Not Auth'));

    // Reseta mock e configura falha de login
    vi.resetAllMocks();
    (global.fetch as any).mockRejectedValueOnce(new Error('Network error'));

    await userEvent.click(screen.getByTestId('btn-login'));
    
    await waitFor(() => {
      expect(screen.getByTestId('error')).toHaveTextContent('Network error');
    });
  });

  it('deve efetuar registro via button', async () => {
    // Estado inicial
    (global.fetch as any).mockResolvedValue({ ok: false, status: 400 });
    
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('Not Auth'));

    // Reseta mock para o fluxo de registro
    vi.resetAllMocks();
    
    // 1º: Mock do Register
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({ message: 'Created' })
    });

    // 2º: Mock do Login automático após Register
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: '1', name: 'Test' })
    });

    await userEvent.click(screen.getByTestId('btn-register'));
    
    await waitFor(() => {
      expect(window.location.href).toBe('/dashboard/portfolio');
    });
  });

  it('deve repassar o erro se a chamada de register falhar', async () => {
    // 1º mock para initAuth
    (global.fetch as any).mockResolvedValueOnce({ ok: false, status: 400 });
    
    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('Not Auth'));

    // Reseta e mocka register erro
    vi.resetAllMocks();
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({ error: 'E-mail em uso' })
    });

    await userEvent.click(screen.getByTestId('btn-register'));

    await waitFor(() => {
      expect(screen.getByTestId('error')).toHaveTextContent('E-mail em uso');
    });
  });

  it('deve efetuar logout', async () => {
    // Inicializa logado
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: '1', name: 'LoginUser' })
    });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('Auth'));

    // Mock Logout endpoint
    (global.fetch as any).mockResolvedValueOnce({ ok: true });

    await userEvent.click(screen.getByTestId('btn-logout'));

    await waitFor(() => {
      expect(window.location.href).toBe('/login');
    });
  });
  
  it('lança erro se useAuth for usado fora do provider', () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    
    const BrokenComponent = () => {
      useAuth();
      return null;
    };
    
    expect(() => render(<BrokenComponent />)).toThrow('useAuth deve ser usado dentro de um AuthProvider');
    
    errorSpy.mockRestore();
  });

  it('deve logar erro no console se a chamada de logout no servidor falhar', async () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    // Inicializa logado
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ id: '1', name: 'LoginUser' })
    });

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    );

    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('Auth'));

    // Mock Logout endpoint to fail
    (global.fetch as any).mockRejectedValueOnce(new Error('Logout failed on server'));

    await userEvent.click(screen.getByTestId('btn-logout'));

    await waitFor(() => {
      expect(errorSpy).toHaveBeenCalledWith('Erro ao efetuar logout no servidor:', expect.any(Error));
      expect(window.location.href).toBe('/login'); // Mesmo com erro, tem que forçar o logout visualmente
    });
    
    errorSpy.mockRestore();
  });
});
