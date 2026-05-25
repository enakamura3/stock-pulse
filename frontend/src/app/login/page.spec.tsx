import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import LoginPage from './page';
import React from 'react';
import { useAuth } from '@/context/AuthContext';
import { vi } from 'vitest';

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('LoginPage', () => {
  it('renders correctly and submits form', async () => {
    const loginMock = vi.fn().mockResolvedValue(true);
    (useAuth as any).mockReturnValue({ login: loginMock });

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

  it('handles login error', async () => {
    const loginMock = vi.fn().mockRejectedValue(new Error('Invalid credentials'));
    (useAuth as any).mockReturnValue({ login: loginMock });

    render(<LoginPage />);
    
    const emailInput = screen.getByLabelText(/E-mail/i);
    const passwordInput = screen.getByLabelText(/Senha/i);
    const submitBtn = screen.getByRole('button', { name: /Entrar no Dashboard/i });

    fireEvent.change(emailInput, { target: { value: 'test@test.com' } });
    fireEvent.change(passwordInput, { target: { value: 'wrong' } });
    fireEvent.click(submitBtn);

    expect(await screen.findByText('Invalid credentials')).toBeInTheDocument();
  });
});
