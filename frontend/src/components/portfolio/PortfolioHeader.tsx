import React from 'react';
import Link from 'next/link';
import ThemeToggle from '@/components/ThemeToggle';

interface PortfolioHeaderProps {
  userName: string;
  onLogout: () => void;
  onLinkTelegram?: () => void;
}

export default function PortfolioHeader({ userName, onLogout }: PortfolioHeaderProps) {
  return (
    <div
      className="flex-row justify-between items-center mb-lg card-header"
      style={{
        flexWrap: 'wrap',
        gap: '1rem',
        paddingBottom: '0.85rem',
        borderBottom: '1px solid var(--panel-border)',
      }}
    >
      <div>
        <div className="flex-row items-center gap-xs">
          <span className="font-mono text-muted" style={{ fontSize: '1.1rem', color: 'var(--accent-color)' }}>&gt;</span>
          <h1
            className="font-mono"
            style={{
              fontSize: '1.6rem',
              color: 'var(--text-primary)',
              margin: 0,
              fontWeight: 700,
              letterSpacing: '-0.02em',
            }}
          >
            stock-pulse
          </h1>
        </div>
        <div className="flex-row mt-xs gap-md" style={{ fontFamily: 'var(--font-mono)' }}>
          <Link
            href="/dashboard/portfolio"
            style={{
              color: 'var(--accent-color)',
              textDecoration: 'none',
              fontSize: '0.78rem',
              fontWeight: 600,
              borderBottom: '2px solid var(--accent-color)',
              paddingBottom: '2px',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            [CARTEIRA] Minha Carteira
          </Link>
          <Link
            href="/dashboard"
            style={{
              color: 'var(--text-secondary)',
              textDecoration: 'none',
              fontSize: '0.78rem',
              fontWeight: 500,
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            [MONITOR] Monitoramento
          </Link>
          <Link
            href="/dashboard/alerts"
            style={{
              color: 'var(--text-secondary)',
              textDecoration: 'none',
              fontSize: '0.78rem',
              fontWeight: 500,
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            [ALERTAS] Meus Alertas
          </Link>
          <Link
            href="/dashboard/settings"
            style={{
              color: 'var(--text-secondary)',
              textDecoration: 'none',
              fontSize: '0.78rem',
              fontWeight: 500,
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            [CONFIG] Configurações
          </Link>
        </div>
      </div>

      <div className="flex-row items-center gap-md">
        <ThemeToggle />
        <div className="text-right font-mono" style={{ fontSize: '0.75rem' }}>
          <span className="font-semibold text-primary" style={{ display: 'block', fontSize: '0.8rem' }}>{userName}</span>
          <span className="text-muted" style={{ fontSize: '0.68rem' }}>Sessão Segura</span>
        </div>
        <button
          className="btn-danger"
          onClick={onLogout}
          style={{ padding: '0.35rem 0.75rem', fontSize: '0.78rem', display: 'flex', alignItems: 'center', gap: '5px' }}
          title="Sair"
        >
          Sair
        </button>
      </div>
    </div>
  );
}
