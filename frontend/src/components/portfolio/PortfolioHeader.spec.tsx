import { render, screen } from '@testing-library/react';
import React from 'react';
import PortfolioHeader from './PortfolioHeader';
import { ThemeProvider } from '@/components/ThemeProvider';
import userEvent from '@testing-library/user-event';

describe('PortfolioHeader', () => {
  it('renders user name, title, navigation links, theme toggle, and logout button', async () => {
    const user = userEvent.setup();
    const logoutMock = vi.fn();

    render(
      <ThemeProvider>
        <PortfolioHeader userName="Carlos Silva" onLogout={logoutMock} />
      </ThemeProvider>
    );

    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    expect(screen.getByText('Carlos Silva')).toBeInTheDocument();
    expect(screen.getByText('Sessão Segura')).toBeInTheDocument();

    expect(screen.getByRole('link', { name: /Minha Carteira/i })).toHaveAttribute('href', '/dashboard/portfolio');
    expect(screen.getByRole('link', { name: /Monitoramento/i })).toHaveAttribute('href', '/dashboard');
    expect(screen.getByRole('link', { name: /Meus Alertas/i })).toHaveAttribute('href', '/dashboard/alerts');
    expect(screen.getByRole('link', { name: /Configurações/i })).toHaveAttribute('href', '/dashboard/settings');

    const logoutBtn = screen.getByRole('button', { name: /Sair/i });
    await user.click(logoutBtn);
    expect(logoutMock).toHaveBeenCalledTimes(1);
  });
});
