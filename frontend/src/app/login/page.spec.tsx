import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import LoginPage from './page';
import React from 'react';
import { useAuth } from '@/context/AuthContext';
const replaceMock = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: replaceMock,
  }),
}));

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('redirects to dashboard when already authenticated and not loading', () => {
    (useAuth as any).mockReturnValue({
      login: vi.fn(),
      isAuthenticated: true,
      isLoading: false,
    });

    render(<LoginPage />);
    expect(replaceMock).toHaveBeenCalledWith('/dashboard/portfolio');
  });

  it('renders correctly and submits form', async () => {
    const loginMock = vi.fn().mockResolvedValue(true);
    (useAuth as any).mockReturnValue({
      login: loginMock,
      isAuthenticated: false,
      isLoading: false,
    });

    render(<LoginPage />);
    
    const emailInput = screen.getByLabelText(/E-mail/i);
    const passwordInput = screen.getByLabelText(/Senha/i);
    const submitBtn = screen.getByRole('button', { name: /Entrar no Dashboard/i });

    fireEvent.change(emailInput, { target: { value: 'test@test.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });
    fireEvent.click(submitBtn);

    expect(loginMock).toHaveBeenCalledWith('test@test.com', 'password123');

    // O botão muda de texto no submit e não volta porque simulamos sucesso (o que engatilharia redirecionamento real)
    expect(await screen.findByText(/Verificando.../i)).toBeInTheDocument();
  });

  it('handles login error with fallback message', async () => {
    const loginMock = vi.fn().mockRejectedValue(new Error(''));
    (useAuth as any).mockReturnValue({ login: loginMock });

    render(<LoginPage />);
    
    const emailInput = screen.getByLabelText(/E-mail/i);
    const passwordInput = screen.getByLabelText(/Senha/i);
    const submitBtn = screen.getByRole('button', { name: /Entrar no Dashboard/i });

    fireEvent.change(emailInput, { target: { value: 'test@test.com' } });
    fireEvent.change(passwordInput, { target: { value: 'wrong' } });
    fireEvent.click(submitBtn);

    expect(await screen.findByText('E-mail ou senha incorretos.')).toBeInTheDocument();
  });

  it('does not redirect when loading even if isAuthenticated is true', () => {
    (useAuth as any).mockReturnValue({
      login: vi.fn(),
      isAuthenticated: true,
      isLoading: true,
    });

    render(<LoginPage />);
    expect(replaceMock).not.toHaveBeenCalled();
  });
});
