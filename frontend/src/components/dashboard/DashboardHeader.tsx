import React from 'react';
import Link from 'next/link';

interface DashboardHeaderProps {
  userName: string;
  wsConnected: boolean;
  onLogout: () => void;
}

export default function DashboardHeader({ userName, wsConnected, onLogout }: DashboardHeaderProps) {
  return (
    <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', borderBottom: '1px solid var(--panel-border)', paddingBottom: '1.25rem', gap: '1rem' }}>
      <div>
        <h1 style={{ fontSize: '2.3rem', background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', margin: 0, fontWeight: 800, display: 'flex', alignItems: 'center', gap: '10px' }}>
          stock-pulse
          <span style={{ 
            width: 8, 
            height: 8, 
            borderRadius: '50%', 
            backgroundColor: wsConnected ? '#00e676' : '#ff3d00', 
            boxShadow: wsConnected ? '0 0 10px #00e676' : '0 0 10px #ff3d00',
            display: 'inline-block',
            transition: 'all 0.3s ease'
          }} title={wsConnected ? 'Conexão em Tempo Real Ativa' : 'Desconectado da cotação tempo real'} />
        </h1>
        
        {/* Navegação entre telas do Dashboard */}
        <div style={{ display: 'flex', gap: '1.5rem', marginTop: '0.8rem' }}>
          <Link href="/dashboard/portfolio" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
            💼 Minha Carteira
          </Link>
          <Link href="/dashboard" style={{ color: 'var(--accent-color)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 700, borderBottom: '2px solid var(--accent-color)', paddingBottom: '3px', display: 'flex', alignItems: 'center', gap: '5px' }}>
            📊 Monitoramento
          </Link>
          <Link href="/dashboard/alerts" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
            🔔 Meus Alertas
          </Link>
          <Link href="/dashboard/settings" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
            ⚙️ Configurações
          </Link>
        </div>
      </div>
      
      <div style={{ display: 'flex', alignItems: 'center', gap: '1.25rem' }}>
        <div style={{ textAlign: 'right', fontSize: '0.8rem' }}>
          <span style={{ display: 'block', fontWeight: 600 }}>{userName}</span>
          <span style={{ color: 'var(--text-secondary)', fontSize: '0.7rem' }}>Sessão Segura</span>
        </div>
        <button className="primary-button" onClick={onLogout} style={{ padding: '0.5rem 1.25rem', fontSize: '0.85rem' }}>
          Sair
        </button>
      </div>
    </div>
  );
}
