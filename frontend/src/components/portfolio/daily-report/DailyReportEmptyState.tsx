import React from 'react';

export interface DailyReportEmptyStateProps {
  onGoToAssets?: () => void;
}

export default function DailyReportEmptyState({ onGoToAssets }: DailyReportEmptyStateProps) {
  return (
    <div className="card flex-col items-center justify-center text-center w-full" style={{ padding: '3.5rem 1.5rem', gap: '1rem' }}>
      <span style={{ fontSize: '3rem' }}>📊</span>
      <h3 className="m-0" style={{ color: 'var(--text-primary)', fontSize: '1.25rem' }}>Nenhum ativo cadastrado na carteira</h3>
      <p className="text-sm text-secondary m-0" style={{ maxWidth: '440px', lineHeight: 1.6 }}>
        Cadastre ações, fundos imobiliários, renda fixa ou títulos públicos para acompanhar a variação diária consolidada e o impacto no seu patrimônio.
      </p>
      {onGoToAssets && (
        <button
          className="primary-button font-bold mt-sm"
          onClick={onGoToAssets}
          style={{ padding: '0.6rem 1.5rem', fontSize: '0.85rem' }}
        >
          + Cadastrar Ativos na Carteira
        </button>
      )}
    </div>
  );
}
