import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import DashboardHeader from './DashboardHeader';
import { ThemeProvider } from '../ThemeProvider';

describe('DashboardHeader Component', () => {
  it('renders correctly with connected websocket and handles logout click', () => {
    const onLogout = vi.fn();
    render(
      <ThemeProvider>
        <DashboardHeader
          userName="Alice Bob"
          wsConnected={true}
          onLogout={onLogout}
        />
      </ThemeProvider>
    );

    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    expect(screen.getByText('Alice Bob')).toBeInTheDocument();
    expect(screen.getByText('Sessão Segura')).toBeInTheDocument();
    expect(screen.getByTitle('Conexão em Tempo Real Ativa')).toBeInTheDocument();

    const logoutBtn = screen.getByRole('button', { name: /sair/i });
    fireEvent.click(logoutBtn);
    expect(onLogout).toHaveBeenCalledTimes(1);
  });

  it('renders disconnected indicator when wsConnected is false', () => {
    render(
      <ThemeProvider>
        <DashboardHeader
          userName="Charlie"
          wsConnected={false}
          onLogout={vi.fn()}
        />
      </ThemeProvider>
    );

    expect(screen.getByTitle('Desconectado da cotação tempo real')).toBeInTheDocument();
    expect(screen.getByText('Charlie')).toBeInTheDocument();
  });
});
