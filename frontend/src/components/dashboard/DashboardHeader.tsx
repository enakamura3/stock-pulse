import React from 'react';
import Link from 'next/link';
import ThemeToggle from '../ThemeToggle';

interface DashboardHeaderProps {
  userName: string;
  wsConnected: boolean;
  onLogout: () => void;
}

export default function DashboardHeader({ userName, wsConnected, onLogout }: DashboardHeaderProps) {
  return (
    <div
      style={{
        display: 'flex',
        flexFlow: 'row wrap',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: '1.5rem',
        borderBottom: '1px solid var(--panel-border)',
        paddingBottom: '1rem',
        gap: '1rem',
      }}
    >
      <div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span className="font-mono text-muted" style={{ fontSize: '1.1rem', color: 'var(--accent-color)' }}>&gt;</span>
          <h1
            className="font-mono"
            style={{
              fontSize: '1.6rem',
              color: 'var(--text-primary)',
              margin: 0,
              fontWeight: 700,
              letterSpacing: '-0.02em',
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
            }}
          >
            stock-pulse
            <span
              style={{
                width: 7,
                height: 7,
                borderRadius: '50%',
                backgroundColor: wsConnected ? 'var(--color-success)' : 'var(--color-danger)',
                boxShadow: wsConnected ? '0 0 8px rgba(var(--success-rgb), 0.6)' : 'none',
                display: 'inline-block',
                transition: 'all 0.2s ease',
              }}
              title={wsConnected ? 'Conexão em Tempo Real Ativa' : 'Desconectado da cotação tempo real'}
            />
          </h1>
        </div>

        {/* Navegação entre telas do Dashboard */}
        <div style={{ display: 'flex', gap: '1.25rem', marginTop: '0.6rem', fontFamily: 'var(--font-mono)' }}>
          <Link
            href="/dashboard/portfolio"
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
            [CARTEIRA] Minha Carteira
          </Link>
          <Link
            href="/dashboard"
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

      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        <ThemeToggle />
        <div style={{ textAlign: 'right', fontFamily: 'var(--font-mono)' }}>
          <span style={{ display: 'block', fontWeight: 600, fontSize: '0.8rem', color: 'var(--text-primary)' }}>{userName}</span>
          <span style={{ color: 'var(--text-muted)', fontSize: '0.68rem' }}>Sessão Segura</span>
        </div>
        <button
          className="btn-danger"
          onClick={onLogout}
          style={{ padding: '0.35rem 0.75rem', fontSize: '0.78rem' }}
        >
          Sair
        </button>
      </div>
    </div>
  );
}
