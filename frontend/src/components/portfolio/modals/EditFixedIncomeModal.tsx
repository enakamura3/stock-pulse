import React from 'react';

export interface EditFixedIncomeModalProps {
  showFIEditModal: boolean;
  setShowFIEditModal: (s: boolean) => void;
  fiEditTxAssetName: string;
  fiTxType: string;
  setFiTxType: (s: string) => void;
  fiAmount: string | number;
  setFiAmount: (s: string | number) => void;
  fiApplicationDate: string;
  setFiApplicationDate: (s: string) => void;
  fiMaturityDate: string;
  setFiMaturityDate: (s: string) => void;
  isAddingFI: boolean;
  handleUpdateFITransaction: (e: React.FormEvent) => void;
}

export default function EditFixedIncomeModal({
  showFIEditModal,
  setShowFIEditModal,
  fiEditTxAssetName,
  fiTxType,
  setFiTxType,
  fiAmount,
  setFiAmount,
  fiApplicationDate,
  setFiApplicationDate,
  fiMaturityDate,
  setFiMaturityDate,
  isAddingFI,
  handleUpdateFITransaction,
}: EditFixedIncomeModalProps) {
  if (!showFIEditModal) return null;

  return (
    <div className="modal-overlay">
      <div className="modal-content" style={{ maxWidth: '400px' }}>
        <h2 className="modal-title mb-lg">Editar Operação RF</h2>
        <form onSubmit={handleUpdateFITransaction} className="flex-col gap-md">
          <div className="form-group">
            <label className="form-label">Ativo (Somente Leitura)</label>
            <input
              className="form-input"
              type="text"
              value={fiEditTxAssetName}
              disabled
              style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--text-secondary)' }}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Tipo de Operação</label>
            <div className="flex-row flex-wrap gap-sm">
              <button
                type="button"
                onClick={() => setFiTxType('SUBSCRIPTION')}
                disabled={isAddingFI}
                className="flex-row justify-center items-center font-bold text-sm"
                style={{
                  flex: 1,
                  padding: '0.6rem',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  border: fiTxType === 'SUBSCRIPTION' ? '1px solid #00e676' : '1px solid var(--panel-border)',
                  background: fiTxType === 'SUBSCRIPTION' ? 'rgba(0, 230, 118, 0.08)' : 'transparent',
                  color: fiTxType === 'SUBSCRIPTION' ? '#00e676' : 'var(--text-secondary)',
                }}
              >
                🟢 APLICAÇÃO
              </button>
              <button
                type="button"
                onClick={() => setFiTxType('REDEMPTION')}
                disabled={isAddingFI}
                className="flex-row justify-center items-center font-bold text-sm"
                style={{
                  flex: 1,
                  padding: '0.6rem',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  border: fiTxType === 'REDEMPTION' ? '1px solid #ff3d00' : '1px solid var(--panel-border)',
                  background: fiTxType === 'REDEMPTION' ? 'rgba(255, 61, 0, 0.08)' : 'transparent',
                  color: fiTxType === 'REDEMPTION' ? '#ff3d00' : 'var(--text-secondary)',
                }}
              >
                🔴 RESGATE
              </button>
            </div>
          </div>

          <div className="form-group">
            <label className="form-label">Valor (R$)</label>
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
              placeholder="Ex: 1000.00"
              required
              disabled={isAddingFI}
            />
          </div>

          <div className="flex-row gap-md">
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Data da Operação</label>
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
                disabled={isAddingFI}
              />
            </div>
          </div>

          <div className="flex-row gap-md mt-sm">
            <button
              type="button"
              onClick={() => setShowFIEditModal(false)}
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
              {isAddingFI ? 'Salvando...' : 'Salvar Operação'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
