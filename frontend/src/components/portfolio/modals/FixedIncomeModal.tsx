import React, { useState, useEffect } from 'react';

export interface FixedIncomeModalProps {
  showFIModal: boolean;
  setShowFIModal: (s: boolean) => void;
  fiInstitution: string;
  setFiInstitution: (s: string) => void;
  fiType: string;
  setFiType: (s: string) => void;
  fiDebtType: string;
  setFiDebtType: (s: string) => void;
  fiIndexer: string;
  setFiIndexer: (s: string) => void;
  fiRate: string | number;
  setFiRate: (s: string | number) => void;
  fiAmount: string | number;
  setFiAmount: (s: string | number) => void;
  fiApplicationDate: string;
  setFiApplicationDate: (s: string) => void;
  fiMaturityDate: string;
  setFiMaturityDate: (s: string) => void;
  isAddingFI: boolean;
  handleAddFixedIncome: (e: React.FormEvent) => void;
}

export default function FixedIncomeModal({
  showFIModal,
  setShowFIModal,
  fiInstitution,
  setFiInstitution,
  fiType,
  setFiType,
  fiDebtType,
  setFiDebtType,
  fiIndexer,
  setFiIndexer,
  fiRate,
  setFiRate,
  fiAmount,
  setFiAmount,
  fiApplicationDate,
  setFiApplicationDate,
  fiMaturityDate,
  setFiMaturityDate,
  isAddingFI,
  handleAddFixedIncome,
}: FixedIncomeModalProps) {
  const [banks, setBanks] = useState<{ name: string; ispb: string }[]>([]);

  useEffect(() => {
    if (showFIModal && banks.length === 0) {
      fetch('https://brasilapi.com.br/api/banks/v1')
        .then((res) => res.json())
        .then((data) => {
          if (Array.isArray(data)) {
            const validBanks = data.filter((b) => b.name).sort((a, b) => a.name.localeCompare(b.name));
            setBanks(validBanks);
          }
        })
        .catch((e) => console.error('Error fetching banks:', e));
    }
  }, [showFIModal, banks.length]);

  if (!showFIModal) return null;

  return (
    <div className="modal-overlay">
      <div className="modal-content" style={{ maxWidth: '460px', maxHeight: '90vh', overflowY: 'auto' }}>
        <div className="modal-header">
          <h2
            className="modal-title"
            style={{
              margin: 0,
              background: 'var(--accent-gradient)',
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
            }}
          >
            🏛️ Nova Aplicação (Renda Fixa)
          </h2>
          <button onClick={() => setShowFIModal(false)} className="btn-close">
            ✕
          </button>
        </div>

        <form onSubmit={handleAddFixedIncome} className="flex-col gap-md">
          <div className="form-group">
            <label className="form-label">Instituição (Banco/Corretora)</label>
            <input
              className="form-input"
              type="text"
              value={fiInstitution}
              onChange={(e) => setFiInstitution(e.target.value)}
              placeholder="Ex: Banco Itaú, XP Investimentos..."
              required
              disabled={isAddingFI}
              list="banks-list"
              autoComplete="off"
            />
            <datalist id="banks-list">
              {banks.map((b) => (
                <option key={b.ispb} value={b.name} />
              ))}
            </datalist>
          </div>

          <div className="flex-row gap-md">
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Tipo de Produto</label>
              <select
                className="form-input"
                value={fiType}
                onChange={(e) => setFiType(e.target.value)}
                disabled={isAddingFI}
              >
                <option value="CDB" style={{ background: '#1c1f24' }}>CDB</option>
                <option value="LCI" style={{ background: '#1c1f24' }}>LCI</option>
                <option value="LCA" style={{ background: '#1c1f24' }}>LCA</option>
                <option value="LC" style={{ background: '#1c1f24' }}>LC</option>
                <option value="TESOURO" style={{ background: '#1c1f24' }}>Tesouro Direto</option>
                <option value="CRI" style={{ background: '#1c1f24' }}>CRI</option>
                <option value="CRA" style={{ background: '#1c1f24' }}>CRA</option>
                <option value="DEBENTURE" style={{ background: '#1c1f24' }}>Debênture</option>
              </select>
            </div>

            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Rentabilidade</label>
              <select
                className="form-input"
                value={fiDebtType}
                onChange={(e) => setFiDebtType(e.target.value)}
                disabled={isAddingFI}
              >
                <option value="POS" style={{ background: '#1c1f24' }}>Pós-Fixado</option>
                <option value="PRE" style={{ background: '#1c1f24' }}>Prefixado</option>
                <option value="HIBRIDO" style={{ background: '#1c1f24' }}>Híbrido (IPCA+)</option>
              </select>
            </div>
          </div>

          <div className="flex-row gap-md">
            {(fiDebtType === 'POS' || fiDebtType === 'HIBRIDO') && (
              <div className="form-group" style={{ flex: 1 }}>
                <label className="form-label">Indexador</label>
                <select
                  className="form-input"
                  value={fiIndexer}
                  onChange={(e) => setFiIndexer(e.target.value)}
                  disabled={isAddingFI}
                >
                  {fiDebtType === 'POS' && (
                    <>
                      <option value="CDI" style={{ background: '#1c1f24' }}>CDI</option>
                      <option value="SELIC" style={{ background: '#1c1f24' }}>Selic</option>
                    </>
                  )}
                  {fiDebtType === 'HIBRIDO' && <option value="IPCA" style={{ background: '#1c1f24' }}>IPCA</option>}
                </select>
              </div>
            )}

            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">
                {fiDebtType === 'POS' ? '% do Indexador' : 'Taxa ao Ano (%)'}
              </label>
              <input
                className="form-input"
                type="number"
                step="any"
                value={fiRate}
                onChange={(e) => setFiRate(e.target.value)}
                placeholder={fiDebtType === 'POS' ? 'Ex: 110' : 'Ex: 12.5'}
                required
                disabled={isAddingFI}
              />
            </div>
          </div>

          <div className="flex-row gap-md">
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Valor Aplicado (R$)</label>
              <input
                className="form-input"
                type="text"
                value={fiAmount}
                onChange={(e) => {
                  let val = e.target.value.replace(/\D/g, '');
                  if (!val) {
                    setFiAmount('');
                    return;
                  }
                  const num = Number(val) / 100;
                  const formatted = num.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
                  setFiAmount(formatted);
                }}
                placeholder="Ex: 5.000,00"
                required
                disabled={isAddingFI}
              />
            </div>
          </div>

          <div className="flex-row gap-md">
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Data de Aplicação</label>
              <input
                className="form-input"
                type="date"
                value={fiApplicationDate}
                onChange={(e) => setFiApplicationDate(e.target.value)}
                required
                disabled={isAddingFI}
              />
            </div>

            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Data de Vencimento</label>
              <input
                className="form-input"
                type="date"
                value={fiMaturityDate}
                onChange={(e) => setFiMaturityDate(e.target.value)}
                required
                disabled={isAddingFI}
              />
            </div>
          </div>

          <div className="flex-row gap-md mt-sm">
            <button
              type="button"
              onClick={() => setShowFIModal(false)}
              className="btn-secondary w-full"
              style={{ padding: '0.75rem' }}
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={isAddingFI}
              className="primary-button w-full"
              style={{ padding: '0.8rem', fontSize: '0.9rem' }}
            >
              {isAddingFI ? 'Cadastrando...' : 'Aplicar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
