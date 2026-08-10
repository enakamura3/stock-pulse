import { render, screen } from '@testing-library/react';
import React from 'react';
import AppSidebar from './AppSidebar';
import { ThemeProvider } from './ThemeProvider';
import userEvent from '@testing-library/user-event';

let mockPathname = '/dashboard/portfolio';

vi.mock('next/navigation', () => ({
  usePathname: () => mockPathname,
}));

describe('AppSidebar Component', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    mockPathname = '/dashboard/portfolio';
  });

  it('renders brand logo, main nav links, portfolio sub-tabs, user name, and logout button', async () => {
    const logoutMock = vi.fn();
    const selectTabMock = vi.fn();

    render(
      <ThemeProvider>
        <AppSidebar
          userName="Maria Souza"
          onLogout={logoutMock}
          activeTab="ativos"
          onSelectTab={selectTabMock}
          wsConnected={true}
        />
      </ThemeProvider>
    );

    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    expect(screen.getByText('Maria Souza')).toBeInTheDocument();

    expect(screen.getByRole('link', { name: /Minha Carteira/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Monitoramento/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Meus Alertas/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Configurações/i })).toBeInTheDocument();

    expect(screen.getByRole('button', { name: /Renda Variável/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Proventos/i })).toBeInTheDocument();

    const subTabBtn = screen.getByRole('button', { name: /Proventos/i });
    await userEvent.click(subTabBtn);
    expect(selectTabMock).toHaveBeenCalledWith('proventos');

    const logoutBtn = screen.getByRole('button', { name: /Sair/i });
    await userEvent.click(logoutBtn);
    expect(logoutMock).toHaveBeenCalledTimes(1);
  });

  it('toggles mobile drawer menu when mobile button is clicked', async () => {
    const user = userEvent.setup();

    const { container } = render(
      <ThemeProvider>
        <AppSidebar userName="Maria Souza" />
      </ThemeProvider>
    );

    const toggleBtn = screen.getByLabelText('Abrir Menu Lateral');
    expect(toggleBtn).toBeInTheDocument();

    await user.click(toggleBtn);
    const aside = container.querySelector('aside');
    expect(aside).toHaveClass('open');
  });

  it('hides portfolio sub-tabs when on non-portfolio route like /dashboard/alerts', () => {
    mockPathname = '/dashboard/alerts';

    render(
      <ThemeProvider>
        <AppSidebar userName="Maria Souza" onSelectTab={vi.fn()} />
      </ThemeProvider>
    );

    expect(screen.queryByRole('button', { name: /Renda Variável/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Proventos/i })).not.toBeInTheDocument();
  });
});
