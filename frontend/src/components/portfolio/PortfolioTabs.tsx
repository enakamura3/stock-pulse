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
    <div className="flex-row justify-between items-center mb-md flex-wrap gap-xs" style={{ borderBottom: '1px solid var(--panel-border)', paddingBottom: '0.4rem' }}>
      <div className="ide-tabstrip" style={{ flex: '1 1 auto', borderBottom: 'none' }}>
        {portfolios.map((p) => {
          const isActive = activePortfolioId === p.id;
          return (
            <button
              key={p.id}
              onClick={() => setActivePortfolioId(p.id)}
              className={`ide-tab ${isActive ? 'active' : ''}`}
            >
              <span className="text-accent" style={{ fontSize: '0.75rem', marginRight: '2px', opacity: isActive ? 1 : 0.6 }}>▸</span>
              {p.is_default ? <span title="Padrão" className="text-warning" style={{ fontSize: '0.7rem' }}>★ </span> : ''}
              <span>{p.name}</span>{' '}
              <span style={{ fontSize: '0.65rem', opacity: 0.6, marginLeft: '2px' }}>[{p.base_currency}]</span>
            </button>
          );
        })}
        <button
          onClick={() => setShowPortfolioModal(true)}
          className="ide-tab"
          style={{
            borderStyle: 'dashed',
            color: 'var(--accent-color)',
            background: 'transparent',
          }}
        >
          + Criar Carteira
        </button>
      </div>

      {activeP && (
        <div className="flex-row gap-xs items-center font-mono">
          {activeP.is_default ? (
            <span
              className="badge"
              style={{
                padding: '0.35rem 0.65rem',
                borderRadius: '4px',
                fontSize: '0.74rem',
                background: 'var(--color-warning-bg)',
                color: 'var(--color-warning)',
                border: '1px solid rgba(var(--warning-rgb), 0.3)',
                fontWeight: 600,
                display: 'inline-flex',
                alignItems: 'center',
                gap: '4px',
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
                padding: '0.35rem 0.65rem',
                fontSize: '0.74rem',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '4px',
                borderColor: 'var(--color-warning)',
                color: 'var(--color-warning)',
                background: 'transparent',
              }}
              title="Definir como carteira principal ao fazer login"
            >
              ⭐ Definir como Padrão
            </button>
          )}

          <button
            onClick={handleExportPortfolio}
            className="btn-secondary"
            style={{
              padding: '0.35rem 0.65rem',
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              fontSize: '0.74rem',
            }}
            title="Exportar Carteira (CSV)"
          >
            📥 Exportar Backup
          </button>

          {portfolios.length > 1 && (
            <button
              onClick={handleDeletePortfolio}
              className="btn-danger"
              style={{
                padding: '0.35rem 0.65rem',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
                fontSize: '0.74rem',
              }}
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
