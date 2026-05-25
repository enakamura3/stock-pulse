import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import RegisterPage from './page';
import React from 'react';
import { useAuth } from '@/context/AuthContext';
import { vi } from 'vitest';

vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('RegisterPage', () => {
  it('renders correctly and submits form', async () => {
    const registerMock = vi.fn().mockResolvedValue(true);
    (useAuth as any).mockReturnValue({ register: registerMock });

    render(<RegisterPage />);
    
    const nameInput = screen.getByLabelText(/Nome Completo/i);
    const emailInput = screen.getByLabelText(/E-mail/i);
    const passwordInput = screen.getByLabelText(/Senha/i);
    const submitBtn = screen.getByRole('button', { name: /Criar Minha Conta/i });

    fireEvent.change(nameInput, { target: { value: 'Test User' } });
    fireEvent.change(emailInput, { target: { value: 'test@test.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });
    fireEvent.click(submitBtn);

    expect(registerMock).toHaveBeenCalledWith('Test User', 'test@test.com', 'password123');
    
    expect(await screen.findByText(/Criar Minha Conta/i)).toBeInTheDocument();
  });

  it('validates password length', async () => {
    const registerMock = vi.fn();
    (useAuth as any).mockReturnValue({ register: registerMock });

    render(<RegisterPage />);
    
    const passwordInput = screen.getByLabelText(/Senha/i);
    const submitBtn = screen.getByRole('button', { name: /Criar Minha Conta/i });

    fireEvent.change(passwordInput, { target: { value: '123' } });
    fireEvent.click(submitBtn);

    expect(registerMock).not.toHaveBeenCalled();
    expect(await screen.findByText('A senha deve conter pelo menos 6 caracteres.')).toBeInTheDocument();
  });

  it('handles register error', async () => {
    const registerMock = vi.fn().mockRejectedValue(new Error('Email already in use'));
    (useAuth as any).mockReturnValue({ register: registerMock });

    render(<RegisterPage />);
    
    const nameInput = screen.getByLabelText(/Nome Completo/i);
    const emailInput = screen.getByLabelText(/E-mail/i);
    const passwordInput = screen.getByLabelText(/Senha/i);
    const submitBtn = screen.getByRole('button', { name: /Criar Minha Conta/i });

    fireEvent.change(nameInput, { target: { value: 'Test' } });
    fireEvent.change(emailInput, { target: { value: 'test@test.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });
    fireEvent.click(submitBtn);

    expect(await screen.findByText('Email already in use')).toBeInTheDocument();
  });
});
