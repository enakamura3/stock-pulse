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
    <div className="flex-row justify-between items-center mb-md flex-wrap gap-md">
      <div className="flex-row gap-xs items-center" style={{ overflowX: 'auto', paddingBottom: '0.2rem' }}>
        {portfolios.map((p) => {
          const isActive = activePortfolioId === p.id;
          return (
            <button
              key={p.id}
              onClick={() => setActivePortfolioId(p.id)}
              className={`tab-button ${isActive ? 'active' : ''}`}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: '0.78rem',
                padding: '0.4rem 0.75rem',
                borderRadius: '4px',
                border: isActive ? '1px solid var(--accent-color)' : '1px solid var(--panel-border)',
                background: isActive ? 'rgba(238, 96, 24, 0.1)' : 'rgba(255, 255, 255, 0.02)',
                color: isActive ? 'var(--accent-color)' : 'var(--text-secondary)',
                cursor: 'pointer',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '4px',
                fontWeight: isActive ? 600 : 400,
              }}
            >
              {p.is_default ? '⭐ ' : ''}💼 {p.name}{' '}
              <span style={{ fontSize: '0.65rem', opacity: 0.7, marginLeft: '2px' }}>[{p.base_currency}]</span>
            </button>
          );
        })}
        <button
          onClick={() => setShowPortfolioModal(true)}
          className="btn-secondary font-mono"
          style={{
            borderStyle: 'dashed',
            borderColor: 'var(--accent-color)',
            color: 'var(--accent-color)',
            background: 'transparent',
            padding: '0.4rem 0.75rem',
            fontSize: '0.78rem',
            borderRadius: '4px',
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
