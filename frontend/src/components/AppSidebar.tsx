'use client';

import React, { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import KeyboardShortcutsModal from './KeyboardShortcutsModal';
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
  ShieldCheckIcon,
  ReceiptIcon,
  CoinsIcon,
  MicroscopeIcon,
  CalendarIcon,
  ChevronsLeftIcon,
  ChevronsRightIcon,
} from '@/components/ui/icons';

const SIDEBAR_COLLAPSED_KEY = 'stock_pulse_sidebar_collapsed';

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
  const router = useRouter();
  const [isOpen, setIsOpen] = useState(false);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [isCollapsed, setIsCollapsed] = useState(false);

  useEffect(() => {
    if (localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true') {
      setIsCollapsed(true);
    }
  }, []);

  const toggleCollapsed = useCallback(() => {
    setIsCollapsed(prev => {
      const next = !prev;
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(next));
      return next;
    });
  }, []);

  const collapsed = isCollapsed && !isOpen;

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setShowShortcuts(false);
        return;
      }

      const target = e.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.tagName === 'SELECT' ||
          target.isContentEditable ||
          target.getAttribute?.('contenteditable') === 'true' ||
          target.closest?.('[contenteditable="true"]'))
      ) {
        return;
      }

      if ((e.ctrlKey || e.metaKey) && !e.altKey && e.key.toLowerCase() === 'b') {
        e.preventDefault();
        toggleCollapsed();
        return;
      }

      if (e.altKey && !e.ctrlKey && !e.metaKey) {
        const altNavMap: Record<string, string> = {
          '1': '/dashboard/portfolio',
          '2': '/dashboard',
          '3': '/dashboard/alerts',
          '4': '/dashboard/settings',
        };
        const dest = altNavMap[e.key];
        if (dest) {
          e.preventDefault();
          setIsOpen(false);
          setShowShortcuts(false);
          router.push(dest);
        }
        return;
      }

      if (e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        setShowShortcuts(prev => !prev);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [router, toggleCollapsed]);

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
    { key: 'tesouro', label: 'Tesouro Direto', icon: <ShieldCheckIcon size={16} /> },
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
        className={`app-sidebar ${isOpen ? 'open' : ''} ${collapsed ? 'collapsed' : ''}`}
        style={{
          position: 'sticky',
          top: 0,
          height: '100vh',
          width: collapsed ? '64px' : '260px',
          flexShrink: 0,
          overflowY: 'auto',
          overflowX: 'hidden',
          background: 'var(--panel-bg)',
          borderRight: '1px solid var(--panel-border)',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: collapsed ? '1rem 0.45rem' : '1.25rem 0.85rem',
          transition: 'width 0.2s ease, padding 0.2s ease, transform var(--transition-normal)',
          zIndex: 'var(--z-header)',
        }}
      >
        <div className="flex-col gap-md">
          {/* Terminal Title Bar & Brand */}
          <div className="flex-col gap-xs px-xs pb-sm" style={{ borderBottom: '1px solid var(--panel-border)' }}>
            {collapsed ? (
              <div className="flex-col items-center gap-xs">
                <button
                  type="button"
                  onClick={toggleCollapsed}
                  aria-label="Expandir barra lateral"
                  title="Expandir barra lateral (Ctrl+B)"
                  className="btn-secondary hide-mobile"
                  style={{ padding: '0.35rem', justifyContent: 'center' }}
                >
                  <ChevronsRightIcon size={16} />
                </button>
                {wsConnected !== undefined && (
                  <span
                    title={wsConnected ? 'WebSocket Conectado (LIVE)' : 'WebSocket Desconectado (DISC)'}
                    style={{
                      width: '7px',
                      height: '7px',
                      borderRadius: '50%',
                      backgroundColor: wsConnected ? 'var(--color-success)' : 'var(--color-danger)',
                      boxShadow: wsConnected ? '0 0 6px var(--color-success)' : 'none',
                    }}
                  />
                )}
              </div>
            ) : (
              <>
                <div className="flex-row items-center justify-between">
                  <span className="font-mono text-muted text-xs" style={{ letterSpacing: '0.06em' }}>[TERMINAL]</span>
                  <div className="flex-row items-center gap-xs">
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
                    <button
                      type="button"
                      onClick={toggleCollapsed}
                      aria-label="Recolher barra lateral"
                      title="Recolher barra lateral (Ctrl+B)"
                      className="btn-secondary hide-mobile"
                      style={{ padding: '0.2rem 0.3rem' }}
                    >
                      <ChevronsLeftIcon size={15} />
                    </button>
                  </div>
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
              </>
            )}
          </div>

          {/* Navegação Global Primária */}
          <nav className="flex-col gap-xs">
            {!collapsed && (
              <span
                className="text-muted font-mono uppercase"
                style={{ fontSize: '0.65rem', fontWeight: 600, letterSpacing: '0.08em', padding: '0 0.5rem 0.25rem' }}
              >
                {'// WORKBENCH'}
              </span>
            )}
            {mainNavItems.map((item, idx) => (
              <Link
                key={item.href}
                href={item.href}
                aria-label={item.label}
                title={collapsed ? `${item.label} (Alt+${idx + 1})` : undefined}
                onClick={() => setIsOpen(false)}
                className={`sidebar-link ${item.active ? 'active' : ''}`}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: collapsed ? 'center' : 'flex-start',
                  gap: '8px',
                  padding: collapsed ? '0.55rem 0' : '0.5rem 0.65rem',
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
                {!collapsed && (
                  <>
                    <span>{item.label}</span>
                    <span
                      className="text-muted font-mono hide-mobile"
                      style={{ marginLeft: 'auto', fontSize: '0.65rem', opacity: item.active ? 0.85 : 0.4 }}
                    >
                      Alt+{idx + 1}
                    </span>
                  </>
                )}
              </Link>
            ))}
          </nav>

          {/* Navegação Secundária da Carteira (Context-Aware Tree View) */}
          {isPortfolioPage && onSelectTab && (
            <div className="flex-col gap-xs mt-xs pt-xs" style={{ borderTop: '1px dashed var(--panel-border)' }}>
              {!collapsed && (
                <span
                  className="text-muted font-mono uppercase"
                  style={{ fontSize: '0.65rem', fontWeight: 600, letterSpacing: '0.08em', padding: '0 0.5rem 0.25rem' }}
                >
                  {'// SUB-MÓDULOS'}
                </span>
              )}
              {portfolioSubTabs.map((sub, idx) => {
                const isSubActive = activeTab === sub.key;
                return (
                  <button
                    key={sub.key}
                    aria-label={sub.label}
                    title={collapsed ? `${sub.label} [${idx + 1}]` : undefined}
                    onClick={() => {
                      onSelectTab(sub.key);
                      setIsOpen(false);
                    }}
                    className={`sidebar-sublink ${isSubActive ? 'active' : ''}`}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: collapsed ? 'center' : 'flex-start',
                      gap: '8px',
                      padding: collapsed ? '0.5rem 0' : '0.42rem 0.65rem 0.42rem 0.85rem',
                      fontFamily: 'var(--font-mono)',
                      fontSize: '0.78rem',
                      color: isSubActive ? 'var(--accent-color)' : 'var(--text-secondary)',
                      background: isSubActive ? 'var(--accent-bg)' : 'transparent',
                      borderLeft: isSubActive ? '2px solid var(--accent-color)' : '2px solid transparent',
                      borderTop: 'none',
                      borderRight: 'none',
                      borderBottom: 'none',
                      cursor: 'pointer',
                      textAlign: 'left',
                      width: '100%',
                    }}
                  >
                    {!collapsed && (
                      <span className="text-muted font-mono" style={{ fontSize: '0.68rem', opacity: isSubActive ? 1 : 0.45 }}>[{idx + 1}]</span>
                    )}
                    <span style={{ opacity: isSubActive ? 1 : 0.6 }}>{sub.icon}</span>
                    {!collapsed && <span>{sub.label}</span>}
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Rodapé do Sidebar: Atalhos de Teclado, Perfil e Sair */}
        <div
          className="flex-col gap-sm pt-sm"
          style={{ borderTop: '1px solid var(--panel-border)', marginTop: '1rem' }}
        >
          <button
            type="button"
            onClick={() => setShowShortcuts(true)}
            aria-label="Abrir atalhos de teclado"
            title="Atalhos de Teclado (?)"
            className="btn-secondary font-mono flex-row items-center"
            style={{
              padding: collapsed ? '0.35rem 0' : '0.35rem 0.6rem',
              fontSize: '0.72rem',
              width: '100%',
              justifyContent: collapsed ? 'center' : 'space-between',
            }}
          >
            {!collapsed && <span>Atalhos de Teclado</span>}
            <span className="text-accent font-bold">[?]</span>
          </button>

          <div
            className="flex-row items-center pt-xs"
            style={{
              borderTop: '1px dashed var(--panel-border)',
              justifyContent: collapsed ? 'center' : 'space-between',
              paddingLeft: collapsed ? 0 : '0.25rem',
              paddingRight: collapsed ? 0 : '0.25rem',
            }}
          >
            {collapsed ? (
              onLogout && (
                <button
                  onClick={onLogout}
                  aria-label="Sair"
                  className="btn-danger"
                  style={{ padding: '0.35rem 0.5rem', fontSize: '0.74rem' }}
                  title={`${userName} — Encerrar Sessão`}
                >
                  <LogOutIcon size={15} />
                </button>
              )
            ) : (
              <>
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
              </>
            )}
          </div>
        </div>
      </aside>

      <KeyboardShortcutsModal isOpen={showShortcuts} onClose={() => setShowShortcuts(false)} />
    </>
  );
}
