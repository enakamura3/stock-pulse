import React from 'react';

export interface NewPortfolioModalProps {
  showPortfolioModal: boolean;
  setShowPortfolioModal: (s: boolean) => void;
  newPortfolioName: string;
  setNewPortfolioName: (s: string) => void;
  newPortfolioCurrency: string;
  setNewPortfolioCurrency: (s: string) => void;
  isCreatingPortfolio: boolean;
  handleCreatePortfolio: (e: React.FormEvent) => void;
}

export default function NewPortfolioModal({
  showPortfolioModal,
  setShowPortfolioModal,
  newPortfolioName,
  setNewPortfolioName,
  newPortfolioCurrency,
  setNewPortfolioCurrency,
  isCreatingPortfolio,
  handleCreatePortfolio,
}: NewPortfolioModalProps) {
  if (!showPortfolioModal) return null;

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h3
          className="modal-title mb-lg"
          style={{
            background: 'var(--accent-gradient)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
          }}
        >
          💼 Nova Carteira
        </h3>

        <form onSubmit={handleCreatePortfolio} className="flex-col gap-lg">
          <div className="form-group">
            <label className="form-label">Nome da Carteira</label>
            <input
              className="form-input"
              type="text"
              value={newPortfolioName}
              onChange={(e) => setNewPortfolioName(e.target.value)}
              placeholder="Ex: Minha Aposentadoria, Ações B3..."
              required
              disabled={isCreatingPortfolio}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Moeda Base</label>
            <select
              value={newPortfolioCurrency}
              onChange={(e) => setNewPortfolioCurrency(e.target.value)}
              disabled={isCreatingPortfolio}
              className="form-input"
            >
              <option value="BRL" style={{ background: '#1c1f24' }}>
                BRL (R$) - Real Brasileiro
              </option>
              <option value="USD" style={{ background: '#1c1f24' }}>
                USD ($) - Dólar Americano
              </option>
            </select>
          </div>

          <div className="flex-row gap-md mt-sm">
            <button
              type="button"
              onClick={() => setShowPortfolioModal(false)}
              className="btn-secondary w-full"
              style={{ padding: '0.75rem' }}
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={isCreatingPortfolio}
              className="primary-button w-full"
              style={{ padding: '0.75rem' }}
            >
              {isCreatingPortfolio ? 'Criando...' : 'Salvar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
