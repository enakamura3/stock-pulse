import React from 'react';
import { Portfolio } from './types';

interface PortfolioTabsProps {
  portfolios: Portfolio[];
  activePortfolioId: string;
  setActivePortfolioId: (id: string) => void;
  setShowPortfolioModal: (show: boolean) => void;
  handleDeletePortfolio: () => void;
  handleExportPortfolio: () => void;
}

export default function PortfolioTabs({ 
  portfolios, activePortfolioId, setActivePortfolioId, setShowPortfolioModal, handleDeletePortfolio, handleExportPortfolio
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
            💼 {p.name} <span style={{ fontSize: '0.65rem', opacity: 0.65, marginLeft: '3px' }}>({p.base_currency})</span>
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
              style={{ background: 'none', border: 'none', padding: '0.5rem' }}
              title="Excluir carteira atual"
            >
              🗑️ Excluir Carteira
            </button>
          )}
        </div>
      )}
    </div>
  );
}
