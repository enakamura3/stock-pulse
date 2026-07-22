import React from 'react';
import { Portfolio } from './types';

interface PortfolioTabsProps {
  portfolios: Portfolio[];
  activePortfolioId: string;
  setActivePortfolioId: (id: string) => void;
  setShowPortfolioModal: (show: boolean) => void;
  handleDeletePortfolio: () => void;
  handleExportPortfolio: () => void;
  handleSetDefaultPortfolio: () => void;
}

export default function PortfolioTabs({ 
  portfolios, activePortfolioId, setActivePortfolioId, setShowPortfolioModal, handleDeletePortfolio, handleExportPortfolio, handleSetDefaultPortfolio
}: PortfolioTabsProps) {
  const activeP = portfolios.find(p => p.id === activePortfolioId);

  return (
    <div className="flex-row justify-between items-center mb-lg flex-wrap gap-md">
      <div className="flex-row gap-sm" style={{ overflowX: 'auto', paddingBottom: '0.2rem' }}>
        {portfolios.map((p) => (
          <button
            key={p.id}
            onClick={() => setActivePortfolioId(p.id)}
            className={`tab-button ${activePortfolioId === p.id ? 'active' : ''}`}
            style={{ fontSize: '0.85rem' }}
          >
            {p.is_default ? '⭐ ' : ''}💼 {p.name} <span style={{ fontSize: '0.65rem', opacity: 0.65, marginLeft: '3px' }}>({p.base_currency})</span>
          </button>
        ))}
        <button
          onClick={() => setShowPortfolioModal(true)}
          className="btn-secondary"
          style={{ borderStyle: 'dashed', borderColor: 'var(--accent-color)', color: 'var(--accent-color)', background: 'transparent' }}
        >
          + Criar Carteira
        </button>
      </div>

      {activeP && (
        <div className="flex-row gap-sm items-center">
          {activeP.is_default ? (
            <span
              style={{
                padding: '0.4rem 0.8rem',
                borderRadius: '8px',
                fontSize: '0.8rem',
                background: 'rgba(255, 193, 7, 0.15)',
                color: '#ffc107',
                border: '1px solid rgba(255, 193, 7, 0.3)',
                fontWeight: 600,
                display: 'inline-flex',
                alignItems: 'center',
                gap: '4px'
              }}
              title="Esta é a sua carteira padrão ao fazer login"
            >
              ⭐ Carteira Padrão
            </span>
          ) : (
            <button
              onClick={handleSetDefaultPortfolio}
              className="btn-secondary"
              style={{
                padding: '0.5rem 0.8rem',
                fontSize: '0.8rem',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '4px',
                borderColor: '#ffc107',
                color: '#ffc107',
                background: 'transparent'
              }}
              title="Definir como carteira principal ao fazer login"
            >
              ⭐ Definir como Padrão
            </button>
          )}

          <button
            onClick={handleExportPortfolio}
            className="btn-secondary"
            style={{ padding: '0.5rem', display: 'flex', alignItems: 'center', gap: '5px', fontSize: '0.8rem', background: 'var(--panel-bg)', color: 'var(--text-secondary)' }}
            title="Exportar Carteira (CSV)"
          >
            📥 Exportar Backup
          </button>
          
          {portfolios.length > 1 && (
            <button
              onClick={handleDeletePortfolio}
              className="btn-danger"
              style={{ padding: '0.5rem', display: 'flex', alignItems: 'center', gap: '5px', fontSize: '0.8rem' }}
              title="Excluir carteira atual"
            >
              🗑️ Apagar Carteira
            </button>
          )}
        </div>
      )}
    </div>
  );
}
