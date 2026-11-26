import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import AppSidebar from './AppSidebar';
import { ThemeProvider } from './ThemeProvider';
import userEvent from '@testing-library/user-event';

let mockPathname = '/dashboard/portfolio';
const mockPush = vi.fn();
const mockRouter = { push: mockPush };

vi.mock('next/navigation', () => ({
  usePathname: () => mockPathname,
  useRouter: () => mockRouter,
}));

describe('AppSidebar Component', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    mockPathname = '/dashboard/portfolio';
    mockPush.mockClear();
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

  it('closes mobile drawer when clicking backdrop or nav link and shows WS disconnected indicator', async () => {
    const user = userEvent.setup();
    const { container } = render(
      <ThemeProvider>
        <AppSidebar userName="Maria Souza" wsConnected={false} />
      </ThemeProvider>
    );

    expect(screen.getByText('DISC')).toBeInTheDocument();

    const toggleBtn = screen.getByLabelText('Abrir Menu Lateral');
    await user.click(toggleBtn);
    const aside = container.querySelector('aside');
    expect(aside).toHaveClass('open');

    const backdrop = container.querySelector('.sidebar-backdrop') as HTMLElement;
    expect(backdrop).toBeInTheDocument();
    await user.click(backdrop);
    expect(aside).not.toHaveClass('open');

    await user.click(toggleBtn);
    expect(aside).toHaveClass('open');
    await user.click(screen.getByRole('link', { name: /Monitoramento/i }));
    expect(aside).not.toHaveClass('open');
  });

  it('opens and closes keyboard shortcuts modal via button, ?, Escape, overlay, and stopPropagation', async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <AppSidebar userName="Maria Souza" />
      </ThemeProvider>
    );

    expect(screen.queryByRole('dialog', { name: 'Atalhos de Teclado' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Abrir atalhos de teclado/i }));
    const dialog = screen.getByRole('dialog', { name: 'Atalhos de Teclado' });
    expect(dialog).toBeInTheDocument();
    expect(screen.getByText('// Navegação Global')).toBeInTheDocument();
    expect(screen.getByText('Alt + 1')).toBeInTheDocument();
    expect(screen.getByText('Resumo Diário')).toBeInTheDocument();
    expect(screen.getByText('Fechar modais e painéis abertos')).toBeInTheDocument();

    // Clicking inside modal-content should not close
    await user.click(dialog.querySelector('.modal-content') as HTMLElement);
    expect(screen.getByRole('dialog', { name: 'Atalhos de Teclado' })).toBeInTheDocument();

    // Clicking close button closes modal
    await user.click(screen.getByRole('button', { name: 'Fechar atalhos de teclado' }));
    expect(screen.queryByRole('dialog', { name: 'Atalhos de Teclado' })).not.toBeInTheDocument();

    // Press '?' to open, then Escape to close
    fireEvent.keyDown(window, { key: '?', shiftKey: true });
    expect(screen.getByRole('dialog', { name: 'Atalhos de Teclado' })).toBeInTheDocument();

    fireEvent.keyDown(window, { key: 'Escape' });
    expect(screen.queryByRole('dialog', { name: 'Atalhos de Teclado' })).not.toBeInTheDocument();

    // Press '?' to open, click overlay to close
    fireEvent.keyDown(window, { key: '?' });
    const overlay = screen.getByRole('dialog', { name: 'Atalhos de Teclado' });
    await user.click(overlay);
    expect(screen.queryByRole('dialog', { name: 'Atalhos de Teclado' })).not.toBeInTheDocument();
  });

  it('supports Alt+1..4 global navigation and ignores editable elements and modifiers', () => {
    render(
      <ThemeProvider>
        <AppSidebar userName="Maria Souza" />
      </ThemeProvider>
    );

    fireEvent.keyDown(window, { key: '1', altKey: true });
    expect(mockPush).toHaveBeenCalledWith('/dashboard/portfolio');

    fireEvent.keyDown(window, { key: '2', altKey: true });
    expect(mockPush).toHaveBeenCalledWith('/dashboard');

    fireEvent.keyDown(window, { key: '3', altKey: true });
    expect(mockPush).toHaveBeenCalledWith('/dashboard/alerts');

    fireEvent.keyDown(window, { key: '4', altKey: true });
    expect(mockPush).toHaveBeenCalledWith('/dashboard/settings');

    mockPush.mockClear();

    // Unmapped Alt key and Ctrl/Meta modifiers are ignored
    fireEvent.keyDown(window, { key: '9', altKey: true });
    fireEvent.keyDown(window, { key: '1', altKey: true, ctrlKey: true });
    fireEvent.keyDown(window, { key: '?', ctrlKey: true });
    expect(mockPush).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog', { name: 'Atalhos de Teclado' })).not.toBeInTheDocument();

    // Ignore events from input or contenteditable
    const input = document.createElement('input');
    document.body.appendChild(input);
    fireEvent.keyDown(input, { key: '?' });
    fireEvent.keyDown(input, { key: '1', altKey: true });
    expect(mockPush).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog', { name: 'Atalhos de Teclado' })).not.toBeInTheDocument();
    document.body.removeChild(input);

    const editable = document.createElement('div');
    editable.setAttribute('contenteditable', 'true');
    document.body.appendChild(editable);
    fireEvent.keyDown(editable, { key: '?' });
    expect(screen.queryByRole('dialog', { name: 'Atalhos de Teclado' })).not.toBeInTheDocument();
    document.body.removeChild(editable);
  });
});
