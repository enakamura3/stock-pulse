'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
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

      {/* Container Principal do AppSidebar (IDE Terminal Workbench) */}
      <aside
        className={`app-sidebar ${isOpen ? 'open' : ''}`}
        style={{
          position: 'sticky',
          top: 0,
          height: 'calc(100vh - 24px)',
          width: '260px',
          flexShrink: 0,
          overflowY: 'auto',
          background: 'var(--panel-bg)',
          borderRight: '1px solid var(--panel-border)',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: '1.25rem 0.85rem',
          transition: 'transform var(--transition-normal)',
          zIndex: 'var(--z-header)',
        }}
      >
        <div className="flex-col gap-md">
          {/* Terminal Title Bar & Brand */}
          <div className="flex-col gap-xs px-xs pb-sm" style={{ borderBottom: '1px solid var(--panel-border)' }}>
            <div className="flex-row items-center justify-between">
              <span className="font-mono text-muted text-xs" style={{ letterSpacing: '0.06em' }}>[TERMINAL]</span>
              {wsConnected !== undefined && (
                <div className="flex-row items-center gap-xs font-mono" style={{ fontSize: '0.65rem' }}>
                  <span
                    style={{
                      width: '6px',
                      height: '6px',
                      borderRadius: '50%',
                      backgroundColor: wsConnected ? 'var(--color-success)' : 'var(--color-danger)',
                      boxShadow: wsConnected ? '0 0 6px var(--color-success)' : 'none',
                    }}
                  />
                  <span style={{ color: wsConnected ? 'var(--color-success)' : 'var(--color-danger)' }}>
                    {wsConnected ? 'LIVE' : 'DISC'}
                  </span>
                </div>
              )}
            </div>

            <div className="flex-row items-center gap-xs mt-xs">
              <span className="font-mono text-muted" style={{ fontSize: '1rem', color: 'var(--accent-color)' }}>&gt;</span>
              <h1
                className="font-mono"
                style={{
                  fontSize: '1.35rem',
                  color: 'var(--text-primary)',
                  margin: 0,
                  fontWeight: 700,
                  letterSpacing: '-0.03em',
                }}
              >
                stock-pulse
              </h1>
            </div>
          </div>

          {/* Navegação Global Primária */}
          <nav className="flex-col gap-xs">
            <span
              className="text-muted font-mono uppercase"
              style={{ fontSize: '0.65rem', fontWeight: 600, letterSpacing: '0.08em', padding: '0 0.5rem 0.25rem' }}
            >
              {'// WORKBENCH'}
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
                  gap: '8px',
                  padding: '0.5rem 0.65rem',
                  borderRadius: '4px',
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.82rem',
                  fontWeight: item.active ? 600 : 400,
                  color: item.active ? 'var(--accent-color)' : 'var(--text-secondary)',
                  background: item.active ? 'rgba(238, 96, 24, 0.08)' : 'transparent',
                  border: item.active ? '1px solid rgba(238, 96, 24, 0.25)' : '1px solid transparent',
                  textDecoration: 'none',
                  transition: 'all 0.15s ease',
                }}
              >
                <span style={{ opacity: item.active ? 1 : 0.7 }}>{item.icon}</span>
                <span>{item.label}</span>
              </Link>
            ))}
          </nav>

          {/* Navegação Secundária da Carteira (Context-Aware Tree View) */}
          {isPortfolioPage && onSelectTab && (
            <div className="flex-col gap-xs mt-xs pt-xs" style={{ borderTop: '1px dashed var(--panel-border)' }}>
              <span
                className="text-muted font-mono uppercase"
                style={{ fontSize: '0.65rem', fontWeight: 600, letterSpacing: '0.08em', padding: '0 0.5rem 0.25rem' }}
              >
                {'// SUB-MÓDULOS'}
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
                      gap: '8px',
                      padding: '0.42rem 0.65rem 0.42rem 1rem',
                      borderRadius: '3px',
                      fontFamily: 'var(--font-mono)',
                      fontSize: '0.78rem',
                      color: isSubActive ? 'var(--accent-color)' : 'var(--text-secondary)',
                      background: isSubActive ? 'rgba(238, 96, 24, 0.06)' : 'transparent',
                      borderLeft: isSubActive ? '2px solid var(--accent-color)' : '2px solid transparent',
                      borderTop: 'none',
                      borderRight: 'none',
                      borderBottom: 'none',
                      cursor: 'pointer',
                      textAlign: 'left',
                      width: '100%',
                      transition: 'all 0.15s ease',
                    }}
                  >
                    <span style={{ opacity: isSubActive ? 1 : 0.6 }}>{sub.icon}</span>
                    <span>{sub.label}</span>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Rodapé do Sidebar: Terminal Mode, Perfil e Sair */}
        <div
          className="flex-col gap-sm pt-sm"
          style={{ borderTop: '1px solid var(--panel-border)', marginTop: '1rem' }}
        >
          <div className="flex-row items-center justify-between px-xs">
            <span className="font-mono text-muted text-xs">ENV</span>
            <span className="badge badge-neutral" style={{ fontSize: '0.65rem', padding: '0.15rem 0.45rem' }}>
              TERM // PROD
            </span>
          </div>

          <div className="flex-row items-center justify-between px-xs pt-xs" style={{ borderTop: '1px dashed var(--panel-border)' }}>
            <div className="flex-col">
              <span className="font-mono text-xs font-semibold text-primary">{userName}</span>
              <span className="font-mono text-muted" style={{ fontSize: '0.65rem' }}>Sessão Segura</span>
            </div>

            {onLogout && (
              <button
                onClick={onLogout}
                className="btn-danger"
                style={{ padding: '0.35rem 0.6rem', fontSize: '0.74rem' }}
                title="Encerrar Sessão"
              >
                <LogOutIcon size={13} />
                <span>Sair</span>
              </button>
            )}
          </div>
        </div>
      </aside>
    </>
  );
}
