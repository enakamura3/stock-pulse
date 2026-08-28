import React from 'react';
import { SearchResult } from '../types';

export interface TransactionModalProps {
  showTxModal: boolean;
  setShowTxModal: (s: boolean) => void;
  editingTxId: string | null;
  setEditingTxId: (id: string | null) => void;
  txTicker: string;
  searchQuery: string;
  setSearchQuery: (s: string) => void;
  isSearching: boolean;
  showDropdown: boolean;
  searchResults: SearchResult[];
  handleSelectAsset: (s: string) => void;
  isAddingTx: boolean;
  txType: 'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS';
  setTxType: (t: 'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS') => void;
  txQuantity: string | number;
  setTxQuantity: (q: string | number) => void;
  txUnitPrice: string | number;
  setTxUnitPrice: (p: string | number) => void;
  txFee: string | number;
  setTxFee: (f: string | number) => void;
  txExchangeRate: string | number;
  setTxExchangeRate: (r: string | number) => void;
  txExecutedAt: string;
  setTxExecutedAt: (d: string) => void;
  selectedAssetCurrency: string;
  kpiCurrency: string;
  handleAddTransaction: (e: React.FormEvent) => void;
}

export default function TransactionModal({
  showTxModal,
  setShowTxModal,
  editingTxId,
  setEditingTxId,
  txTicker,
  searchQuery,
  setSearchQuery,
  isSearching,
  showDropdown,
  searchResults,
  handleSelectAsset,
  isAddingTx,
  txType,
  setTxType,
  txQuantity,
  setTxQuantity,
  txUnitPrice,
  setTxUnitPrice,
  txFee,
  setTxFee,
  txExchangeRate,
  setTxExchangeRate,
  txExecutedAt,
  setTxExecutedAt,
  selectedAssetCurrency,
  kpiCurrency,
  handleAddTransaction,
}: TransactionModalProps) {
  if (!showTxModal) return null;

  const parsedQty = parseFloat(txQuantity.toString()) || 0;
  const parsedPrice = parseFloat(txUnitPrice.toString()) || 0;
  const parsedFee = parseFloat(txFee.toString()) || 0;

  const grossTotal = parsedQty * parsedPrice;
  const netTotal = txType === 'SELL' ? grossTotal - parsedFee : grossTotal + parsedFee;

  const currencySymbol = selectedAssetCurrency === 'USD' ? '$' : 'R$';

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
            {editingTxId ? '✏️ Editar Transação' : '➕ Nova Transação'}
          </h2>
          <button
            onClick={() => {
              setShowTxModal(false);
              setEditingTxId(null);
            }}
            className="btn-close"
          >
            ✕
          </button>
        </div>

        <form onSubmit={handleAddTransaction} className="flex-col gap-md">
          <div className="form-group" style={{ position: 'relative' }}>
            <label className="form-label">Ativo / Ticker</label>
            {editingTxId ? (
              <input
                className="form-input"
                type="text"
                value={txTicker}
                readOnly
                disabled
                style={{ textTransform: 'uppercase', opacity: 0.6 }}
              />
            ) : (
              <input
                className="form-input"
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Pesquise o ticker (Ex: PETR4, AAPL, IVV)..."
                required
                disabled={isAddingTx}
                autoComplete="off"
              />
            )}
            {isSearching && (
              <span
                className="loading-spinner"
                style={{ position: 'absolute', right: '15px', top: '55%', borderTopColor: 'var(--accent-color)' }}
              />
            )}

            {showDropdown && searchResults.length > 0 && (
              <div
                className="card"
                style={{
                  position: 'absolute',
                  top: '100%',
                  left: 0,
                  width: '100%',
                  marginTop: '0.4rem',
                  zIndex: 101,
                  padding: '0.4rem',
                  maxHeight: '200px',
                  overflowY: 'auto',
                  boxShadow: '0 16px 40px rgba(0,0,0,0.7)',
                }}
              >
                {searchResults.map((item) => (
                  <div
                    key={item.symbol}
                    onClick={() => handleSelectAsset(item.symbol)}
                    className="flex-row justify-between items-center sidebar-link"
                    style={{ padding: '0.55rem 0.8rem', borderRadius: '6px', cursor: 'pointer' }}
                  >
                    <div>
                      <span className="font-bold text-accent" style={{ marginRight: '0.6rem' }}>
                        {item.symbol}
                      </span>
                      <span className="text-sm opacity-80">{item.name}</span>
                    </div>
                    <span className="badge badge-neutral text-accent" style={{ background: 'var(--accent-bg)' }}>
                      {item.exchange}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="form-group">
            <label className="form-label">Operação</label>
            <div className="flex-row flex-wrap gap-sm">
              <button
                type="button"
                onClick={() => setTxType('BUY')}
                disabled={isAddingTx}
                className="flex-row justify-center items-center font-bold text-sm"
                style={{
                  flex: 1,
                  padding: '0.6rem',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  border: txType === 'BUY' ? '1px solid var(--color-success)' : '1px solid var(--panel-border)',
                  background: txType === 'BUY' ? 'var(--color-success-bg)' : 'transparent',
                  color: txType === 'BUY' ? 'var(--color-success)' : 'var(--text-secondary)',
                }}
              >
                🟢 COMPRA
              </button>
              <button
                type="button"
                onClick={() => setTxType('SELL')}
                disabled={isAddingTx}
                className="flex-row justify-center items-center font-bold text-sm"
                style={{
                  flex: 1,
                  padding: '0.6rem',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  border: txType === 'SELL' ? '1px solid var(--color-danger)' : '1px solid var(--panel-border)',
                  background: txType === 'SELL' ? 'var(--color-danger-bg)' : 'transparent',
                  color: txType === 'SELL' ? 'var(--color-danger)' : 'var(--text-secondary)',
                }}
              >
                🔴 VENDA
              </button>
              <button
                type="button"
                onClick={() => {
                  setTxType('SPLIT');
                  setTxUnitPrice(0);
                  setTxFee(0);
                }}
                disabled={isAddingTx}
                className="flex-row justify-center items-center font-bold text-sm"
                style={{
                  flex: 1,
                  padding: '0.6rem',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  border: txType === 'SPLIT' ? '1px solid var(--accent-color)' : '1px solid var(--panel-border)',
                  background: txType === 'SPLIT' ? 'var(--accent-bg)' : 'transparent',
                  color: txType === 'SPLIT' ? 'var(--accent-color)' : 'var(--text-secondary)',
                }}
              >
                ✂️ SPLIT
              </button>
              <button
                type="button"
                onClick={() => {
                  setTxType('REVERSE_SPLIT');
                  setTxUnitPrice(0);
                  setTxFee(0);
                }}
                disabled={isAddingTx}
                className="flex-row justify-center items-center font-bold text-sm"
                style={{
                  flex: 1,
                  padding: '0.6rem',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  border: txType === 'REVERSE_SPLIT' ? '1px solid #e040fb' : '1px solid var(--panel-border)',
                  background: txType === 'REVERSE_SPLIT' ? 'rgba(156, 39, 176, 0.08)' : 'transparent',
                  color: txType === 'REVERSE_SPLIT' ? '#e040fb' : 'var(--text-secondary)',
                }}
              >
                🗜️ AGRUP.
              </button>
            </div>
          </div>

          <div className="flex-row flex-wrap gap-md">
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">
                {txType === 'SPLIT' || txType === 'REVERSE_SPLIT' ? 'Fator / Multiplicador' : 'Quantidade'}
              </label>
              <input
                className="form-input"
                type="number"
                step="any"
                value={txQuantity}
                onChange={(e) => setTxQuantity(e.target.value)}
                placeholder={txType === 'SPLIT' || txType === 'REVERSE_SPLIT' ? 'Ex: 10' : '0'}
                required
                disabled={isAddingTx}
              />
              {(txType === 'SPLIT' || txType === 'REVERSE_SPLIT') && (
                <span className="text-xs text-secondary mt-sm block">
                  {txType === 'SPLIT'
                    ? 'Ex: Desdobramento 1 para 10 = Fator 10.'
                    : 'Ex: Agrupamento 10 para 1 = Fator 10.'}
                </span>
              )}
            </div>

            {txType !== 'SPLIT' && txType !== 'REVERSE_SPLIT' && (
              <div className="form-group" style={{ flex: 1 }}>
                <label className="form-label">Preço Unitário ({selectedAssetCurrency})</label>
                <input
                  className="form-input"
                  type="number"
                  step="any"
                  value={txUnitPrice}
                  onChange={(e) => setTxUnitPrice(e.target.value)}
                  placeholder="0.00"
                  required
                  disabled={isAddingTx}
                />
              </div>
            )}
          </div>

          {txType !== 'SPLIT' && txType !== 'REVERSE_SPLIT' && txType !== 'BONUS' && (
            <div className="form-group">
              <label className="form-label">Corretagem / Taxas ({selectedAssetCurrency})</label>
              <input
                className="form-input"
                type="number"
                step="any"
                min="0"
                value={txFee}
                onChange={(e) => setTxFee(e.target.value)}
                placeholder="0.00"
                disabled={isAddingTx}
              />
            </div>
          )}

          {selectedAssetCurrency && kpiCurrency && selectedAssetCurrency !== kpiCurrency && (
            <div className="form-group">
              <label className="form-label text-warning" style={{ color: '#ffc107' }}>
                Taxa Cambial {selectedAssetCurrency}{kpiCurrency}
              </label>
              <input
                className="form-input"
                type="number"
                step="any"
                value={txExchangeRate}
                onChange={(e) => setTxExchangeRate(e.target.value)}
                placeholder="Ex: 5.2500"
                disabled={isAddingTx}
                style={{ borderColor: 'rgba(255, 193, 7, 0.4)' }}
              />
              <span className="text-xs text-secondary mt-sm block">
                Se deixado em branco, o sistema buscará a taxa automaticamente.
              </span>
            </div>
          )}

          <div className="form-group">
            <label className="form-label">Data de Execução</label>
            <input
              className="form-input"
              type="date"
              value={txExecutedAt}
              onChange={(e) => setTxExecutedAt(e.target.value)}
              required
              disabled={isAddingTx}
            />
          </div>

          {txType !== 'SPLIT' && txType !== 'REVERSE_SPLIT' && grossTotal > 0 && (
            <div
              className="card"
              style={{
                background: 'rgba(255, 255, 255, 0.03)',
                padding: '0.75rem 1rem',
                borderRadius: '8px',
                border: '1px solid var(--panel-border)',
              }}
            >
              <div className="flex-row justify-between text-xs text-secondary">
                <span>Valor dos Ativos:</span>
                <span>{currencySymbol} {grossTotal.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
              </div>
              {parsedFee > 0 && (
                <div className="flex-row justify-between text-xs text-secondary mt-xs">
                  <span>Taxas / Corretagem:</span>
                  <span>{currencySymbol} {parsedFee.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
                </div>
              )}
              <div className="flex-row justify-between text-sm font-bold mt-xs" style={{ borderTop: '1px solid rgba(255,255,255,0.1)', paddingTop: '0.4rem', color: 'var(--text-primary)' }}>
                <span>Total Operação:</span>
                <span style={{ color: txType === 'BUY' ? '#00e676' : '#ff3d00' }}>
                  {currencySymbol} {netTotal.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                </span>
              </div>
            </div>
          )}

          <div className="flex-row gap-md mt-sm">
            <button
              type="button"
              onClick={() => setShowTxModal(false)}
              className="btn-secondary w-full"
              style={{ padding: '0.75rem' }}
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={isAddingTx}
              className="primary-button w-full"
              style={{ padding: '0.8rem', fontSize: '0.9rem' }}
            >
              {isAddingTx ? 'Registrando...' : editingTxId ? 'Salvar Alterações' : 'Lançar'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
