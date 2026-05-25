import React, { useState, useEffect } from 'react';
import { SearchResult } from './types';

interface ModalsProps {
  // Portfolio Modal
  showPortfolioModal: boolean;
  setShowPortfolioModal: (s: boolean) => void;
  newPortfolioName: string;
  setNewPortfolioName: (s: string) => void;
  newPortfolioCurrency: string;
  setNewPortfolioCurrency: (s: string) => void;
  isCreatingPortfolio: boolean;
  handleCreatePortfolio: (e: React.FormEvent) => void;

  // Transaction Modal
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
  txExchangeRate: string | number;
  setTxExchangeRate: (r: string | number) => void;
  txExecutedAt: string;
  setTxExecutedAt: (d: string) => void;
  selectedAssetCurrency: string;
  kpiCurrency: string;
  handleAddTransaction: (e: React.FormEvent) => void;

  // Fixed Income Modal
  showFIModal?: boolean;
  setShowFIModal?: (s: boolean) => void;
  showFIEditModal?: boolean;
  setShowFIEditModal?: (s: boolean) => void;
  fiEditTxAssetName?: string;
  setFiEditTxAssetName?: (s: string) => void;
  handleUpdateFITransaction?: (e: React.FormEvent) => void;
  fiInstitution?: string;
  setFiInstitution?: (s: string) => void;
  fiType?: string;
  setFiType?: (s: string) => void;
  fiDebtType?: string;
  setFiDebtType?: (s: string) => void;
  fiIndexer?: string;
  setFiIndexer?: (s: string) => void;
  fiRate?: string | number;
  setFiRate?: (s: string | number) => void;
  fiTxType?: string;
  setFiTxType?: (s: string) => void;
  fiAmount?: string | number;
  setFiAmount?: (s: string | number) => void;
  fiApplicationDate?: string;
  setFiApplicationDate?: (s: string) => void;
  fiMaturityDate?: string;
  setFiMaturityDate?: (s: string) => void;
  isAddingFI?: boolean;
  handleAddFixedIncome?: (e: React.FormEvent) => void;
}

export default function Modals(props: ModalsProps) {
  const [banks, setBanks] = useState<{name: string, ispb: string}[]>([]);

  useEffect(() => {
    // Only fetch banks if the FI modal is opened to save resources
    if (props.showFIModal && banks.length === 0) {
      fetch('https://brasilapi.com.br/api/banks/v1')
        .then(res => res.json())
        .then(data => {
          if (Array.isArray(data)) {
            // Sort alphabetically and filter out null names
            const validBanks = data.filter(b => b.name).sort((a, b) => a.name.localeCompare(b.name));
            setBanks(validBanks);
          }
        })
        .catch(e => console.error("Error fetching banks:", e));
    }
  }, [props.showFIModal, banks.length]);

  return (
    <>
      {props.showPortfolioModal && (
        <div className="modal-overlay">
          <div className="modal-content">
            <h3 className="modal-title mb-lg" style={{ background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
              💼 Nova Carteira
            </h3>
            
            <form onSubmit={props.handleCreatePortfolio} className="flex-col gap-lg">
              <div className="form-group">
                <label className="form-label">Nome da Carteira</label>
                <input
                  className="form-input"
                  type="text"
                  value={props.newPortfolioName}
                  onChange={(e) => props.setNewPortfolioName(e.target.value)}
                  placeholder="Ex: Minha Aposentadoria, Ações B3..."
                  required
                  disabled={props.isCreatingPortfolio}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Moeda Base</label>
                <select
                  value={props.newPortfolioCurrency}
                  onChange={(e) => props.setNewPortfolioCurrency(e.target.value)}
                  disabled={props.isCreatingPortfolio}
                  className="form-input"
                >
                  <option value="BRL" style={{ background: '#1c1f24' }}>BRL (R$) - Real Brasileiro</option>
                  <option value="USD" style={{ background: '#1c1f24' }}>USD ($) - Dólar Americano</option>
                </select>
              </div>

              <div className="flex-row gap-md mt-sm">
                <button type="button" onClick={() => props.setShowPortfolioModal(false)} className="btn-secondary w-full" style={{ padding: '0.75rem' }}>
                  Cancelar
                </button>
                <button type="submit" disabled={props.isCreatingPortfolio} className="primary-button w-full" style={{ padding: '0.75rem' }}>
                  {props.isCreatingPortfolio ? 'Criando...' : 'Salvar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {props.showTxModal && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '460px', maxHeight: '90vh', overflowY: 'auto' }}>
            <div className="modal-header">
              <h2 className="modal-title" style={{ margin: 0, background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
                {props.editingTxId ? '✏️ Editar Transação' : '➕ Nova Transação'}
              </h2>
              <button
                onClick={() => {
                  props.setShowTxModal(false);
                  props.setEditingTxId(null);
                }}
                className="btn-close"
              >
                ✕
              </button>
            </div>
            
            <form onSubmit={props.handleAddTransaction} className="flex-col gap-md">
              <div className="form-group" style={{ position: 'relative' }}>
                <label className="form-label">Ativo / Ticker</label>
                {props.editingTxId ? (
                  <input
                    className="form-input" type="text" value={props.txTicker} readOnly disabled
                    style={{ textTransform: 'uppercase', opacity: 0.6 }}
                  />
                ) : (
                  <input
                    className="form-input" type="text" value={props.searchQuery}
                    onChange={(e) => props.setSearchQuery(e.target.value)}
                    placeholder="Pesquise o ticker (Ex: PETR4, AAPL, IVV)..."
                    required disabled={props.isAddingTx} autoComplete="off"
                  />
                )}
                {props.isSearching && (
                  <span className="loading-spinner" style={{ position: 'absolute', right: '15px', top: '55%', borderTopColor: 'var(--accent-color)' }}></span>
                )}

                {props.showDropdown && props.searchResults.length > 0 && (
                  <div className="card" style={{
                    position: 'absolute', top: '100%', left: 0, width: '100%', marginTop: '0.4rem', zIndex: 101, padding: '0.4rem',
                    maxHeight: '200px', overflowY: 'auto', boxShadow: '0 16px 40px rgba(0,0,0,0.7)'
                  }}>
                    {props.searchResults.map((item) => (
                      <div
                        key={item.symbol}
                        onClick={() => props.handleSelectAsset(item.symbol)}
                        className="flex-row justify-between items-center"
                        style={{ padding: '0.55rem 0.8rem', borderRadius: '6px', cursor: 'pointer' }}
                        onMouseEnter={(e) => e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.04)'}
                        onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
                      >
                        <div>
                          <span className="font-bold text-accent" style={{ marginRight: '0.6rem' }}>{item.symbol}</span>
                          <span className="text-sm opacity-80">{item.name}</span>
                        </div>
                        <span className="badge badge-neutral text-accent" style={{ background: 'rgba(0, 242, 254, 0.08)' }}>{item.exchange}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="form-group">
                <label className="form-label">Operação</label>
                <div className="flex-row flex-wrap gap-sm">
                  <button
                    type="button" onClick={() => props.setTxType('BUY')} disabled={props.isAddingTx}
                    className="flex-row justify-center items-center font-bold text-sm"
                    style={{ flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', border: props.txType === 'BUY' ? '1px solid #00e676' : '1px solid var(--panel-border)', background: props.txType === 'BUY' ? 'rgba(0, 230, 118, 0.08)' : 'transparent', color: props.txType === 'BUY' ? '#00e676' : 'var(--text-secondary)' }}
                  >
                    🟢 COMPRA
                  </button>
                  <button
                    type="button" onClick={() => props.setTxType('SELL')} disabled={props.isAddingTx}
                    className="flex-row justify-center items-center font-bold text-sm"
                    style={{ flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', border: props.txType === 'SELL' ? '1px solid #ff3d00' : '1px solid var(--panel-border)', background: props.txType === 'SELL' ? 'rgba(255, 61, 0, 0.08)' : 'transparent', color: props.txType === 'SELL' ? '#ff3d00' : 'var(--text-secondary)' }}
                  >
                    🔴 VENDA
                  </button>
                  <button
                    type="button" onClick={() => { props.setTxType('SPLIT'); props.setTxUnitPrice(0); }} disabled={props.isAddingTx}
                    className="flex-row justify-center items-center font-bold text-sm"
                    style={{ flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', border: props.txType === 'SPLIT' ? '1px solid #00f2fe' : '1px solid var(--panel-border)', background: props.txType === 'SPLIT' ? 'rgba(0, 242, 254, 0.08)' : 'transparent', color: props.txType === 'SPLIT' ? '#00f2fe' : 'var(--text-secondary)' }}
                  >
                    ✂️ SPLIT
                  </button>
                  <button
                    type="button" onClick={() => { props.setTxType('REVERSE_SPLIT'); props.setTxUnitPrice(0); }} disabled={props.isAddingTx}
                    className="flex-row justify-center items-center font-bold text-sm"
                    style={{ flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', border: props.txType === 'REVERSE_SPLIT' ? '1px solid #e040fb' : '1px solid var(--panel-border)', background: props.txType === 'REVERSE_SPLIT' ? 'rgba(156, 39, 176, 0.08)' : 'transparent', color: props.txType === 'REVERSE_SPLIT' ? '#e040fb' : 'var(--text-secondary)' }}
                  >
                    🗜️ AGRUP.
                  </button>
                </div>
              </div>

              <div className="flex-row flex-wrap gap-md">
                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">
                    {(props.txType === 'SPLIT' || props.txType === 'REVERSE_SPLIT') ? 'Fator / Multiplicador' : 'Quantidade'}
                  </label>
                  <input
                    className="form-input" type="number" step="any"
                    value={props.txQuantity} onChange={(e) => props.setTxQuantity(e.target.value)}
                    placeholder={(props.txType === 'SPLIT' || props.txType === 'REVERSE_SPLIT') ? "Ex: 10" : "0"}
                    required disabled={props.isAddingTx}
                  />
                  {(props.txType === 'SPLIT' || props.txType === 'REVERSE_SPLIT') && (
                    <span className="text-xs text-secondary mt-sm block">
                      {props.txType === 'SPLIT' ? 'Ex: Desdobramento 1 para 10 = Fator 10.' : 'Ex: Agrupamento 10 para 1 = Fator 10.'}
                    </span>
                  )}
                </div>

                {props.txType !== 'SPLIT' && props.txType !== 'REVERSE_SPLIT' && (
                  <div className="form-group" style={{ flex: 1 }}>
                    <label className="form-label">Preço Unitário ({props.selectedAssetCurrency})</label>
                    <input
                      className="form-input" type="number" step="any"
                      value={props.txUnitPrice} onChange={(e) => props.setTxUnitPrice(e.target.value)}
                      placeholder="0.00" required disabled={props.isAddingTx}
                    />
                  </div>
                )}
              </div>

              {props.selectedAssetCurrency && props.kpiCurrency && props.selectedAssetCurrency !== props.kpiCurrency && (
                <div className="form-group">
                  <label className="form-label text-warning" style={{ color: '#ffc107' }}>Taxa Cambial {props.selectedAssetCurrency}{props.kpiCurrency}</label>
                  <input
                    className="form-input" type="number" step="any"
                    value={props.txExchangeRate} onChange={(e) => props.setTxExchangeRate(e.target.value)}
                    placeholder="Ex: 5.2500" disabled={props.isAddingTx}
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
                  className="form-input" type="date"
                  value={props.txExecutedAt} onChange={(e) => props.setTxExecutedAt(e.target.value)}
                  required disabled={props.isAddingTx}
                />
              </div>

              <div className="flex-row gap-md mt-sm">
                <button type="button" onClick={() => props.setShowTxModal(false)} className="btn-secondary w-full" style={{ padding: '0.75rem' }}>
                  Cancelar
                </button>
                <button type="submit" disabled={props.isAddingTx} className="primary-button w-full" style={{ padding: '0.8rem', fontSize: '0.9rem' }}>
                  {props.isAddingTx ? 'Registrando...' : (props.editingTxId ? 'Salvar Alterações' : 'Lançar')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {props.showFIModal && props.setShowFIModal && props.handleAddFixedIncome && props.setFiInstitution && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '460px', maxHeight: '90vh', overflowY: 'auto' }}>
            <div className="modal-header">
              <h2 className="modal-title" style={{ margin: 0, background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
                🏛️ Nova Aplicação (Renda Fixa)
              </h2>
              <button onClick={() => props.setShowFIModal!(false)} className="btn-close">✕</button>
            </div>
            
            <form onSubmit={props.handleAddFixedIncome} className="flex-col gap-md">
              <div className="form-group">
                <label className="form-label">Instituição (Banco/Corretora)</label>
                <input
                  className="form-input" type="text" value={props.fiInstitution}
                  onChange={(e) => props.setFiInstitution!(e.target.value)}
                  placeholder="Ex: Banco Itaú, XP Investimentos..."
                  required disabled={props.isAddingFI}
                  list="banks-list"
                  autoComplete="off"
                />
                <datalist id="banks-list">
                  {banks.map(b => (
                    <option key={b.ispb} value={b.name} />
                  ))}
                </datalist>
              </div>

              <div className="flex-row gap-md">
                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">Tipo de Produto</label>
                  <select
                    className="form-input" value={props.fiType} onChange={(e) => props.setFiType!(e.target.value)} disabled={props.isAddingFI}
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
                    className="form-input" value={props.fiDebtType} onChange={(e) => props.setFiDebtType!(e.target.value)} disabled={props.isAddingFI}
                  >
                    <option value="POS" style={{ background: '#1c1f24' }}>Pós-Fixado</option>
                    <option value="PRE" style={{ background: '#1c1f24' }}>Prefixado</option>
                    <option value="HIBRIDO" style={{ background: '#1c1f24' }}>Híbrido (IPCA+)</option>
                  </select>
                </div>
              </div>

              <div className="flex-row gap-md">
                {(props.fiDebtType === 'POS' || props.fiDebtType === 'HIBRIDO') && (
                  <div className="form-group" style={{ flex: 1 }}>
                    <label className="form-label">Indexador</label>
                    <select
                      className="form-input" value={props.fiIndexer} onChange={(e) => props.setFiIndexer!(e.target.value)} disabled={props.isAddingFI}
                    >
                      {props.fiDebtType === 'POS' && (
                        <>
                          <option value="CDI" style={{ background: '#1c1f24' }}>CDI</option>
                          <option value="SELIC" style={{ background: '#1c1f24' }}>Selic</option>
                        </>
                      )}
                      {props.fiDebtType === 'HIBRIDO' && <option value="IPCA" style={{ background: '#1c1f24' }}>IPCA</option>}
                    </select>
                  </div>
                )}

                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">
                    {props.fiDebtType === 'POS' ? '% do Indexador' : 'Taxa ao Ano (%)'}
                  </label>
                  <input
                    className="form-input" type="number" step="any"
                    value={props.fiRate} onChange={(e) => props.setFiRate!(e.target.value)}
                    placeholder={props.fiDebtType === 'POS' ? "Ex: 110" : "Ex: 12.5"}
                    required disabled={props.isAddingFI}
                  />
                </div>
              </div>

              <div className="flex-row gap-md">
                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">Valor Aplicado (R$)</label>
                  <input
                    className="form-input" type="text"
                    value={props.fiAmount} 
                    onChange={(e) => {
                      let val = e.target.value.replace(/\D/g, "");
                      if (!val) {
                        props.setFiAmount!('');
                        return;
                      }
                      const num = Number(val) / 100;
                      const formatted = num.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
                      props.setFiAmount!(formatted);
                    }}
                    placeholder="Ex: 5.000,00" required disabled={props.isAddingFI}
                  />
                </div>
              </div>

              <div className="flex-row gap-md">
                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">Data de Aplicação</label>
                  <input
                    className="form-input" type="date"
                    value={props.fiApplicationDate} onChange={(e) => props.setFiApplicationDate!(e.target.value)}
                    required disabled={props.isAddingFI}
                  />
                </div>

                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">Data de Vencimento</label>
                  <input
                    className="form-input" type="date"
                    value={props.fiMaturityDate} onChange={(e) => props.setFiMaturityDate!(e.target.value)}
                    required disabled={props.isAddingFI}
                  />
                </div>
              </div>

              <div className="flex-row gap-md mt-sm">
                <button type="button" onClick={() => props.setShowFIModal!(false)} className="btn-secondary w-full" style={{ padding: '0.75rem' }}>
                  Cancelar
                </button>
                <button type="submit" disabled={props.isAddingFI} className="primary-button w-full" style={{ padding: '0.8rem', fontSize: '0.9rem' }}>
                  {props.isAddingFI ? 'Cadastrando...' : 'Aplicar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Fixed Income Transaction Modal */}
      {props.showFIEditModal && props.setShowFIEditModal && props.handleUpdateFITransaction && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '400px' }}>
            <h2 className="modal-title mb-lg">Editar Operação RF</h2>
            <form onSubmit={props.handleUpdateFITransaction} className="flex-col gap-md">
              <div className="form-group">
                <label className="form-label">Ativo (Somente Leitura)</label>
                <input
                  className="form-input" type="text"
                  value={props.fiEditTxAssetName} disabled
                  style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--text-secondary)' }}
                />
              </div>

              <div className="form-group">
                <label className="form-label">Tipo de Operação</label>
                <div className="flex-row flex-wrap gap-sm">
                  <button
                    type="button"
                    onClick={() => props.setFiTxType!('SUBSCRIPTION')}
                    disabled={props.isAddingFI}
                    className="flex-row justify-center items-center font-bold text-sm"
                    style={{ flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', border: props.fiTxType === 'SUBSCRIPTION' ? '1px solid #00e676' : '1px solid var(--panel-border)', background: props.fiTxType === 'SUBSCRIPTION' ? 'rgba(0, 230, 118, 0.08)' : 'transparent', color: props.fiTxType === 'SUBSCRIPTION' ? '#00e676' : 'var(--text-secondary)' }}
                  >
                    🟢 APLICAÇÃO
                  </button>
                  <button
                    type="button"
                    onClick={() => props.setFiTxType!('REDEMPTION')}
                    disabled={props.isAddingFI}
                    className="flex-row justify-center items-center font-bold text-sm"
                    style={{ flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', border: props.fiTxType === 'REDEMPTION' ? '1px solid #ff3d00' : '1px solid var(--panel-border)', background: props.fiTxType === 'REDEMPTION' ? 'rgba(255, 61, 0, 0.08)' : 'transparent', color: props.fiTxType === 'REDEMPTION' ? '#ff3d00' : 'var(--text-secondary)' }}
                  >
                    🔴 RESGATE
                  </button>
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Valor (R$)</label>
                <input
                  className="form-input" type="text"
                  value={props.fiAmount} 
                  onChange={(e) => {
                    let val = e.target.value.replace(/\D/g, "");
                    if (!val) {
                      props.setFiAmount!('');
                      return;
                    }
                    const num = Number(val) / 100;
                    const formatted = num.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
                    props.setFiAmount!(formatted);
                  }}
                  placeholder="Ex: 1000.00" required disabled={props.isAddingFI}
                />
              </div>

              <div className="flex-row gap-md">
                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">Data da Operação</label>
                  <input
                    className="form-input" type="date"
                    value={props.fiApplicationDate} onChange={(e) => props.setFiApplicationDate!(e.target.value)}
                    required disabled={props.isAddingFI}
                  />
                </div>

                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">Data de Vencimento</label>
                  <input
                    className="form-input" type="date"
                    value={props.fiMaturityDate} onChange={(e) => props.setFiMaturityDate!(e.target.value)}
                    disabled={props.isAddingFI}
                  />
                </div>
              </div>

              <div className="flex-row gap-md mt-sm">
                <button type="button" onClick={() => props.setShowFIEditModal!(false)} className="btn-secondary w-full" style={{ padding: '0.75rem' }}>
                  Cancelar
                </button>
                <button type="submit" disabled={props.isAddingFI} className="primary-button w-full" style={{ padding: '0.8rem', fontSize: '0.9rem' }}>
                  {props.isAddingFI ? 'Salvando...' : 'Salvar Operação'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
