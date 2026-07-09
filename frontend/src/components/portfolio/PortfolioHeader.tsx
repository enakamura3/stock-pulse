import React from 'react';
import Link from 'next/link';

interface PortfolioHeaderProps {
  userName: string;
  onLogout: () => void;
  onLinkTelegram: () => void;
}

export default function PortfolioHeader({ userName, onLogout, onLinkTelegram }: PortfolioHeaderProps) {
  return (
    <div className="flex-row justify-between items-center mb-xl card-header" style={{ flexWrap: 'wrap', gap: '1rem' }}>
      <div>
        <h1 style={{ fontSize: '2.3rem', background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', margin: 0, fontWeight: 800 }}>
          stock-pulse
        </h1>
        <div className="flex-row mt-sm gap-lg">
          <Link href="/dashboard/portfolio" style={{ color: 'var(--accent-color)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 700, borderBottom: '2px solid var(--accent-color)', paddingBottom: '3px', display: 'flex', alignItems: 'center', gap: '5px' }}>
            💼 Minha Carteira
          </Link>
          <Link href="/dashboard" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
            📊 Monitoramento
          </Link>
        </div>
      </div>
      
      <div className="flex-row items-center gap-lg">
        <button 
          className="btn-secondary" 
          onClick={onLinkTelegram} 
          style={{ padding: '0.4rem 0.8rem', fontSize: '0.8rem', display: 'flex', alignItems: 'center', gap: '5px' }}
          title="Vincular Telegram"
        >
          📱 Telegram
        </button>
        <div className="text-right text-xs">
          <span className="font-semibold" style={{ display: 'block' }}>{userName}</span>
          <span className="text-secondary" style={{ fontSize: '0.7rem' }}>Sessão Segura</span>
        </div>
        <button 
          className="btn-secondary" 
          onClick={onLogout} 
          style={{ padding: '0.4rem 0.8rem', fontSize: '0.8rem', display: 'flex', alignItems: 'center', gap: '5px' }}
          title="Sair"
        >
          🚪 Sair
        </button>
      </div>
    </div>
  );
}
