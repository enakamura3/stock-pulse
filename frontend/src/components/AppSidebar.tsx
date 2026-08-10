'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import ThemeToggle from '@/components/ThemeToggle';
import {
  WalletIcon,
  ChartIcon,
  BellIcon,
  SettingsIcon,
  LogOutIcon,
  MenuIcon,
  XIcon,
  TrendingUpIcon,
  BankIcon,
  ReceiptIcon,
  CoinsIcon,
  MicroscopeIcon,
  CalendarIcon,
} from '@/components/ui/icons';

interface AppSidebarProps {
  userName?: string;
  onLogout?: () => void;
  activeTab?: string;
  onSelectTab?: (tab: any) => void;
  wsConnected?: boolean;
}

export default function AppSidebar({
  userName = 'Investidor',
  onLogout,
  activeTab = 'ativos',
  onSelectTab,
  wsConnected,
}: AppSidebarProps) {
  const pathname = usePathname();
  const [isOpen, setIsOpen] = useState(false);

  const isPortfolioPage = pathname === '/dashboard/portfolio';
  const isDashboardPage = pathname === '/dashboard';
  const isAlertsPage = pathname === '/dashboard/alerts';
  const isSettingsPage = pathname === '/dashboard/settings';

  const mainNavItems = [
    { href: '/dashboard/portfolio', label: 'Minha Carteira', icon: <WalletIcon size={18} />, active: isPortfolioPage },
    { href: '/dashboard', label: 'Monitoramento', icon: <ChartIcon size={18} />, active: isDashboardPage },
    { href: '/dashboard/alerts', label: 'Meus Alertas', icon: <BellIcon size={18} />, active: isAlertsPage },
    { href: '/dashboard/settings', label: 'Configurações', icon: <SettingsIcon size={18} />, active: isSettingsPage },
  ];

  const portfolioSubTabs = [
    { key: 'ativos', label: 'Renda Variável', icon: <TrendingUpIcon size={16} /> },
    { key: 'renda-fixa', label: 'Renda Fixa', icon: <BankIcon size={16} /> },
    { key: 'tesouro', label: 'Tesouro Direto', icon: <BankIcon size={16} /> },
    { key: 'operacoes', label: 'Histórico de Operações', icon: <ReceiptIcon size={16} /> },
    { key: 'proventos', label: 'Proventos', icon: <CoinsIcon size={16} /> },
    { key: 'analise', label: 'Análise da Carteira', icon: <MicroscopeIcon size={16} /> },
    { key: 'diario', label: 'Resumo Diário', icon: <CalendarIcon size={16} /> },
  ];

  return (
    <>
      {/* Botão de Hambúrguer para Mobile */}
      <button
        className="mobile-menu-toggle"
        onClick={() => setIsOpen(!isOpen)}
        aria-label="Abrir Menu Lateral"
        style={{
          display: 'none',
          position: 'fixed',
          top: '1rem',
          left: '1rem',
          zIndex: 'var(--z-drawer)',
          background: 'var(--panel-bg)',
          border: '1px solid var(--panel-border)',
          color: 'var(--text-primary)',
          borderRadius: '8px',
          padding: '0.5rem',
          cursor: 'pointer',
        }}
      >
        {isOpen ? <XIcon size={22} /> : <MenuIcon size={22} />}
      </button>

      {/* Overlay escuro em dispositivos mobile */}
      {isOpen && (
        <div
          className="sidebar-backdrop"
          onClick={() => setIsOpen(false)}
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.6)',
            backdropFilter: 'blur(4px)',
            zIndex: 'calc(var(--z-drawer) - 1)',
          }}
        />
      )}

      {/* Container Principal do AppSidebar */}
      <aside
        className={`app-sidebar ${isOpen ? 'open' : ''}`}
        style={{
          position: 'sticky',
          top: 0,
          height: '100vh',
          width: '260px',
          flexShrink: 0,
          overflowY: 'auto',
          background: 'var(--panel-bg)',
          borderRight: '1px solid var(--panel-border)',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: '1.5rem 1rem',
          backdropFilter: 'blur(16px)',
          transition: 'transform var(--transition-normal)',
          zIndex: 'var(--z-header)',
        }}
      >
        <div className="flex-col gap-lg">
          {/* Header do Logo */}
          <div className="flex-row items-center justify-between px-sm">
            <h1
              style={{
                fontSize: '1.8rem',
                background: 'var(--accent-gradient)',
                WebkitBackgroundClip: 'text',
                WebkitTextFillColor: 'transparent',
                margin: 0,
                fontWeight: 800,
                letterSpacing: '-0.02em',
              }}
            >
              stock-pulse
            </h1>

            {wsConnected !== undefined && (
              <span
                style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '50%',
                  backgroundColor: wsConnected ? 'var(--color-success)' : 'var(--color-danger)',
                  boxShadow: wsConnected ? '0 0 8px var(--color-success)' : 'none',
                }}
                title={wsConnected ? 'WebSocket Conectado' : 'WebSocket Desconectado'}
              />
            )}
          </div>

          {/* Navegação Global Primária */}
          <nav className="flex-col gap-xs">
            <span
              className="text-muted uppercase"
              style={{ fontSize: '0.68rem', fontWeight: 700, letterSpacing: '0.08em', padding: '0 0.5rem 0.25rem' }}
            >
              Navegação
            </span>
            {mainNavItems.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                onClick={() => setIsOpen(false)}
                className={`sidebar-link ${item.active ? 'active' : ''}`}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '10px',
                  padding: '0.6rem 0.75rem',
                  borderRadius: '8px',
                  fontSize: '0.88rem',
                  fontWeight: item.active ? 700 : 500,
                  color: item.active ? 'var(--accent-color)' : 'var(--text-secondary)',
                  background: item.active ? 'rgba(0, 242, 254, 0.08)' : 'transparent',
                  borderLeft: item.active ? '3px solid var(--accent-color)' : '3px solid transparent',
                  textDecoration: 'none',
                  transition: 'all var(--transition-fast)',
                }}
              >
                {item.icon}
                <span>{item.label}</span>
              </Link>
            ))}
          </nav>

          {/* Navegação Secundária da Carteira (Context-Aware) */}
          {isPortfolioPage && onSelectTab && (
            <div className="flex-col gap-xs mt-sm">
              <span
                className="text-muted uppercase"
                style={{ fontSize: '0.68rem', fontWeight: 700, letterSpacing: '0.08em', padding: '0 0.5rem 0.25rem' }}
              >
                Sub-Abas da Carteira
              </span>
              {portfolioSubTabs.map((sub) => {
                const isSubActive = activeTab === sub.key;
                return (
                  <button
                    key={sub.key}
                    onClick={() => {
                      onSelectTab(sub.key);
                      setIsOpen(false);
                    }}
                    className={`sidebar-sublink ${isSubActive ? 'active' : ''}`}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '10px',
                      padding: '0.5rem 0.75rem 0.5rem 1.25rem',
                      borderRadius: '6px',
                      fontSize: '0.82rem',
                      fontWeight: isSubActive ? 600 : 400,
                      color: isSubActive ? 'var(--color-success)' : 'var(--text-secondary)',
                      background: isSubActive ? 'var(--color-success-bg)' : 'transparent',
                      border: 'none',
                      cursor: 'pointer',
                      textAlign: 'left',
                      width: '100%',
                      transition: 'all var(--transition-fast)',
                    }}
                  >
                    {sub.icon}
                    <span>{sub.label}</span>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Rodapé do Sidebar: Tema, Perfil e Sair */}
        <div
          className="flex-col gap-md pt-md"
          style={{ borderTop: '1px solid var(--panel-border)', marginTop: '1.5rem' }}
        >
          <div className="flex-row items-center justify-between px-sm">
            <span className="text-secondary text-xs font-semibold">Tema</span>
            <ThemeToggle />
          </div>

          <div className="flex-row items-center justify-between px-sm">
            <div className="flex-col text-xs">
              <span className="font-semibold text-primary">{userName}</span>
              <span className="text-muted" style={{ fontSize: '0.68rem' }}>Sessão Segura</span>
            </div>

            {onLogout && (
              <button
                onClick={onLogout}
                className="btn-danger"
                style={{ padding: '0.4rem 0.6rem', fontSize: '0.78rem', display: 'inline-flex', alignItems: 'center', gap: '4px' }}
                title="Encerrar Sessão"
              >
                <LogOutIcon size={14} />
                <span>Sair</span>
              </button>
            )}
          </div>
        </div>
      </aside>
    </>
  );
}
