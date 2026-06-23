'use client';

import React, { useState, useEffect, useCallback } from 'react';
import dynamic from 'next/dynamic';

const PortfolioChart = dynamic(() => import('@/components/PortfolioChart'), { ssr: false });

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

// ─── Types ────────────────────────────────────────────────────────────────────

interface TreasuryPosition {
  asset_id: string;
  ticker: string;
  treasury_type: string;  // SELIC, PREFIXADO, IPCA+
  maturity_date: string;
  has_coupons: boolean;
  start_date: string;
  total_invested: number;
  gross_value: number;
  net_value: number;
  is_matured: boolean;
  days_to_maturity: number;
  taxes_calculated: number;  // IOF + IR
  b3_fee: number;
  ir_tax: number;
  iof_tax: number;
}

interface TreasuryPerfPoint {
  date: string;
  value: number;
  total_invested: number;
}

interface NewTreasuryTx {
  ticker: string;
  treasury_type: string;
  maturity_date: string;
  has_coupons: boolean;
  type: 'SUBSCRIPTION' | 'REDEMPTION';
  quantity: number | '';
  unit_price: number | '';
  contracted_rate: number | '';
  transaction_date: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function fmt(value: number, currency = 'BRL'): string {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

function fmtPct(value: number): string {
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(2)}%`;
}

function getTreasuryTypeLabel(t: string): string {
  const map: Record<string, string> = {
    SELIC: '📈 Tesouro Selic',
    PREFIXADO: '🔒 Prefixado',
    'IPCA+': '🏷️ IPCA+',
  };
  return map[t] || t;
}

function getTreasuryTypeBadgeColor(t: string): string {
  const map: Record<string, string> = {
    SELIC: '#4caf50',
    PREFIXADO: '#2196f3',
    'IPCA+': '#ff9800',
  };
  return map[t] || '#9e9e9e';
}

// ─── Component ────────────────────────────────────────────────────────────────

interface TreasuryTabProps {
  portfolioId: string;
}

const EMPTY_TX: NewTreasuryTx = {
  ticker: '',
  treasury_type: 'SELIC',
  maturity_date: '',
  has_coupons: false,
  type: 'SUBSCRIPTION',
  quantity: '',
  unit_price: '',
  contracted_rate: '',
  transaction_date: new Date().toISOString().split('T')[0],
};

export default function TreasuryTab({ portfolioId }: TreasuryTabProps) {
  const [positions, setPositions] = useState<TreasuryPosition[]>([]);
  const [perfData, setPerfData] = useState<TreasuryPerfPoint[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingPerf, setIsLoadingPerf] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState<NewTreasuryTx>(EMPTY_TX);
  const [error, setError] = useState<string | null>(null);

  // ── Fetchers ────────────────────────────────────────────────────────────────

  const fetchPositions = useCallback(async () => {
    if (!portfolioId) return;
    setIsLoading(true);
    try {
      const res = await fetch(`${API_URL}/portfolios/${portfolioId}/treasury/positions`, {
        credentials: 'include',
        cache: 'no-store',
      });
      if (res.ok) {
        const data = await res.json();
        setPositions(data || []);
      }
    } catch (e) {
      console.error('Erro ao buscar posições do Tesouro:', e);
    } finally {
      setIsLoading(false);
    }
  }, [portfolioId]);

  const fetchPerformance = useCallback(async () => {
    if (!portfolioId) return;
    setIsLoadingPerf(true);
    try {
      const res = await fetch(`${API_URL}/portfolios/${portfolioId}/treasury/performance`, {
        credentials: 'include',
        cache: 'no-store',
      });
      if (res.ok) {
        const data = await res.json();
        setPerfData(data || []);
      }
    } catch (e) {
      console.error('Erro ao buscar performance do Tesouro:', e);
    } finally {
      setIsLoadingPerf(false);
    }
  }, [portfolioId]);

  useEffect(() => {
    fetchPositions();
    fetchPerformance();
  }, [fetchPositions, fetchPerformance]);

  // ── KPIs ────────────────────────────────────────────────────────────────────

  const totalInvested = positions.reduce((s, p) => s + p.total_invested, 0);
  const totalGross = positions.reduce((s, p) => s + p.gross_value, 0);
  const totalNet = positions.reduce((s, p) => s + p.net_value, 0);
  const totalTaxes = positions.reduce((s, p) => s + p.taxes_calculated + p.b3_fee, 0);
  const grossReturn = totalInvested > 0 ? ((totalGross - totalInvested) / totalInvested) * 100 : 0;
  const netReturn = totalInvested > 0 ? ((totalNet - totalInvested) / totalInvested) * 100 : 0;

  // ── Form handlers ───────────────────────────────────────────────────────────

  function openModal() {
    setForm(EMPTY_TX);
    setError(null);
    setShowModal(true);
  }

  function closeModal() {
    setShowModal(false);
    setError(null);
  }

  function handleFormChange(field: keyof NewTreasuryTx, value: string | number | boolean) {
    setForm(prev => ({ ...prev, [field]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (!form.ticker.trim()) { setError('Informe o ticker do título (ex: TESOURO SELIC 2027)'); return; }
    if (!form.maturity_date) { setError('Informe a data de vencimento.'); return; }
    if (!form.contracted_rate || Number(form.contracted_rate) < 0) { setError('Informe a taxa contratada (% a.a.).'); return; }
    if (!form.quantity || Number(form.quantity) <= 0) { setError('Informe a quantidade de frações.'); return; }
    if (!form.unit_price || Number(form.unit_price) <= 0) { setError('Informe o preço unitário.'); return; }

    setIsSubmitting(true);
    try {
      const payload = {
        ticker: form.ticker.trim().toUpperCase(),
        treasury_type: form.treasury_type,
        maturity_date: form.maturity_date,
        has_coupons: form.has_coupons,
        type: form.type,
        quantity: Number(form.quantity),
        unit_price: Number(form.unit_price),
        contracted_rate: Number(form.contracted_rate),
        transaction_date: form.transaction_date,
      };

      const res = await fetch(`${API_URL}/portfolios/${portfolioId}/treasury/transactions`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        closeModal();
        fetchPositions();
        fetchPerformance();
      } else {
        const txt = await res.text();
        setError(txt || 'Erro ao registrar a operação.');
      }
    } catch {
      setError('Erro de conexão. Tente novamente.');
    } finally {
      setIsSubmitting(false);
    }
  }

  // ── Render ──────────────────────────────────────────────────────────────────

  return (
    <div className="flex-col gap-xl w-full">

      {/* ── KPI Cards ── */}
      <div className="flex-row gap-md flex-wrap">
        {[
          { label: 'Total Aplicado', value: fmt(totalInvested), icon: '💰', sub: null },
          {
            label: 'Valor Bruto',
            value: fmt(totalGross),
            icon: '📈',
            sub: fmtPct(grossReturn),
            subColor: grossReturn >= 0 ? '#4caf50' : '#f44336',
          },
          {
            label: 'Valor Líquido',
            value: fmt(totalNet),
            icon: '🏦',
            sub: fmtPct(netReturn),
            subColor: netReturn >= 0 ? '#4caf50' : '#f44336',
          },
          { label: 'Total de Impostos', value: fmt(totalTaxes), icon: '🧾', sub: '(IOF + IR + Taxa B3)' },
        ].map(card => (
          <div
            key={card.label}
            className="card"
            style={{ flex: '1 1 180px', minWidth: 160, padding: '1.25rem 1.5rem' }}
          >
            <div style={{ fontSize: '1.4rem', marginBottom: '0.4rem' }}>{card.icon}</div>
            <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)', marginBottom: '0.35rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>{card.label}</div>
            <div style={{ fontSize: '1.2rem', fontWeight: 700, color: 'var(--text-primary)' }}>{card.value}</div>
            {card.sub && (
              <div style={{ fontSize: '0.75rem', color: (card as any).subColor || 'var(--text-secondary)', marginTop: '0.25rem', fontWeight: 600 }}>
                {card.sub}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* ── Performance Chart ── */}
      <div className="card flex-col" style={{ padding: '1.75rem 2rem', minHeight: '320px' }}>
        <div className="flex-row justify-between items-center mb-lg flex-wrap gap-md">
          <div>
            <h3 className="card-title">📊 Evolução do Tesouro Direto</h3>
            <p className="text-xs text-secondary mt-sm">Marcação a Mercado (Preço de Resgate)</p>
          </div>
          <button
            id="treasury-add-btn"
            onClick={openModal}
            className="btn-primary"
            style={{ padding: '0.5rem 1.2rem', fontSize: '0.85rem' }}
          >
            + Registrar Operação
          </button>
        </div>

        {isLoadingPerf ? (
          <div className="flex-row items-center justify-center w-full" style={{ height: '240px' }}>
            <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 30, height: 30 }} />
          </div>
        ) : perfData.length > 0 ? (
          <PortfolioChart data={perfData} />
        ) : (
          <div
            className="flex-col items-center justify-center w-full text-secondary"
            style={{ height: '240px', border: '1px dashed var(--panel-border)', borderRadius: '12px' }}
          >
            <span className="text-2xl mb-sm">🏛️</span>
            <p className="text-sm m-0">Nenhum dado histórico disponível ainda.</p>
            <p className="text-xs m-0 mt-sm">Registre sua primeira aplicação para começar.</p>
          </div>
        )}
      </div>

      {/* ── Positions Table ── */}
      <div className="card flex-col" style={{ padding: '1.75rem 2rem' }}>
        <div className="flex-row justify-between items-center mb-lg">
          <h3 className="card-title">📋 Posições Ativas</h3>
          <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
            {positions.length} título{positions.length !== 1 ? 's' : ''}
          </span>
        </div>

        {isLoading ? (
          <div className="flex-row items-center justify-center" style={{ height: '120px' }}>
            <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 28, height: 28 }} />
          </div>
        ) : positions.length === 0 ? (
          <div
            className="flex-col items-center justify-center text-secondary"
            style={{ height: '120px', border: '1px dashed var(--panel-border)', borderRadius: '10px' }}
          >
            <p className="text-sm m-0">Nenhuma posição ativa de Tesouro Direto.</p>
            <button
              onClick={openModal}
              className="btn-secondary mt-md"
              style={{ fontSize: '0.8rem', padding: '0.4rem 1rem' }}
            >
              + Adicionar primeira aplicação
            </button>
          </div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.82rem' }}>
              <thead>
                <tr style={{ color: 'var(--text-secondary)', borderBottom: '1px solid var(--panel-border)' }}>
                  {['Título', 'Tipo', 'Vencimento', 'Aplicado', 'Bruto', 'Líquido', 'Retorno Líq.', 'IOF', 'IR', 'Taxa B3', 'Status'].map(h => (
                    <th key={h} style={{ textAlign: 'left', padding: '0.5rem 0.75rem', fontWeight: 600, whiteSpace: 'nowrap' }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {positions.map((pos, i) => {
                  const liqReturn = pos.total_invested > 0
                    ? ((pos.net_value - pos.total_invested) / pos.total_invested) * 100
                    : 0;
                  const isPositive = liqReturn >= 0;
                  return (
                    <tr
                      key={pos.asset_id}
                      style={{
                        borderBottom: i < positions.length - 1 ? '1px solid rgba(255,255,255,0.04)' : 'none',
                        transition: 'background 0.15s',
                      }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.03)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                    >
                      <td style={{ padding: '0.65rem 0.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                        {pos.ticker}
                        {pos.has_coupons && (
                          <span
                            title="Paga cupons semestrais"
                            style={{ marginLeft: 6, fontSize: '0.65rem', color: '#ff9800', fontWeight: 400 }}
                          >
                            cupons
                          </span>
                        )}
                      </td>
                      <td style={{ padding: '0.65rem 0.75rem' }}>
                        <span style={{
                          padding: '0.2rem 0.55rem',
                          borderRadius: '12px',
                          fontSize: '0.68rem',
                          fontWeight: 700,
                          background: `${getTreasuryTypeBadgeColor(pos.treasury_type)}22`,
                          color: getTreasuryTypeBadgeColor(pos.treasury_type),
                          border: `1px solid ${getTreasuryTypeBadgeColor(pos.treasury_type)}44`,
                        }}>
                          {pos.treasury_type}
                        </span>
                      </td>
                      <td style={{ padding: '0.65rem 0.75rem', color: 'var(--text-secondary)' }}>
                        {pos.is_matured
                          ? <span style={{ color: '#f44336', fontWeight: 600 }}>Vencido</span>
                          : `${new Date(pos.maturity_date).toLocaleDateString('pt-BR')} (${pos.days_to_maturity}d)`}
                      </td>
                      <td style={{ padding: '0.65rem 0.75rem' }}>{fmt(pos.total_invested)}</td>
                      <td style={{ padding: '0.65rem 0.75rem' }}>{fmt(pos.gross_value)}</td>
                      <td style={{ padding: '0.65rem 0.75rem', fontWeight: 700, color: pos.net_value >= pos.total_invested ? '#4caf50' : '#f44336' }}>
                        {fmt(pos.net_value)}
                      </td>
                      <td style={{ padding: '0.65rem 0.75rem', fontWeight: 700, color: isPositive ? '#4caf50' : '#f44336' }}>
                        {fmtPct(liqReturn)}
                      </td>
                      <td style={{ padding: '0.65rem 0.75rem', color: 'var(--text-secondary)' }}>{fmt(pos.iof_tax)}</td>
                      <td style={{ padding: '0.65rem 0.75rem', color: 'var(--text-secondary)' }}>{fmt(pos.ir_tax)}</td>
                      <td style={{ padding: '0.65rem 0.75rem', color: 'var(--text-secondary)' }}>{fmt(pos.b3_fee)}</td>
                      <td style={{ padding: '0.65rem 0.75rem' }}>
                        {pos.is_matured ? (
                          <span style={{ color: '#f44336', fontSize: '0.7rem', fontWeight: 700 }}>● Vencido</span>
                        ) : (
                          <span style={{ color: '#4caf50', fontSize: '0.7rem', fontWeight: 700 }}>● Ativo</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* ── Disclaimer ── */}
      <div
        style={{
          padding: '0.75rem 1rem',
          borderRadius: '8px',
          background: 'rgba(255, 152, 0, 0.06)',
          border: '1px solid rgba(255, 152, 0, 0.2)',
          fontSize: '0.72rem',
          color: 'var(--text-secondary)',
          lineHeight: 1.6,
        }}
      >
        ⚠️ <strong>Nota:</strong> Os valores de impostos (IOF e IR) e Taxa B3 são calculados com base na <strong>Curva Teórica</strong> (taxa contratada + DU/252). 
        A <strong>Marcação a Mercado</strong> é atualizada diariamente pelo worker com o Preço de Resgate oficial do Tesouro Nacional. 
        Isenção Selic aplicada sobre posições até R$ 10.000,00.
      </div>

      {/* ── Modal ── */}
      {showModal && (
        <div
          className="modal-overlay"
          onClick={e => { if (e.target === e.currentTarget) closeModal(); }}
          style={{
            position: 'fixed', inset: 0, zIndex: 1000,
            background: 'rgba(0,0,0,0.7)', backdropFilter: 'blur(4px)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            padding: '1rem',
          }}
        >
          <div
            className="card"
            style={{ width: '100%', maxWidth: 520, padding: '2rem', position: 'relative' }}
            onClick={e => e.stopPropagation()}
          >
            <button
              onClick={closeModal}
              style={{ position: 'absolute', top: '1rem', right: '1rem', background: 'none', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: '1.25rem' }}
            >
              ✕
            </button>

            <h3 className="card-title mb-lg">🏛️ Registrar Operação — Tesouro Direto</h3>

            <form onSubmit={handleSubmit} className="flex-col gap-md">

              {/* Tipo de Operação */}
              <div className="flex-row gap-sm">
                {(['SUBSCRIPTION', 'REDEMPTION'] as const).map(t => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => handleFormChange('type', t)}
                    style={{
                      flex: 1, padding: '0.6rem', border: '1px solid',
                      borderColor: form.type === t ? 'var(--accent-color)' : 'var(--panel-border)',
                      background: form.type === t ? 'rgba(0,242,254,0.1)' : 'transparent',
                      color: form.type === t ? 'var(--accent-color)' : 'var(--text-secondary)',
                      borderRadius: '6px', cursor: 'pointer', fontWeight: 700, fontSize: '0.8rem',
                    }}
                  >
                    {t === 'SUBSCRIPTION' ? '📥 Aplicação' : '📤 Resgate'}
                  </button>
                ))}
              </div>

              {/* Tipo de Título */}
              <div>
                <label className="label-sm">Tipo de Título</label>
                <select
                  id="treasury-type-select"
                  value={form.treasury_type}
                  onChange={e => handleFormChange('treasury_type', e.target.value)}
                  className="input"
                >
                  <option value="SELIC">Tesouro Selic</option>
                  <option value="PREFIXADO">Tesouro Prefixado</option>
                  <option value="IPCA+">Tesouro IPCA+</option>
                </select>
              </div>

              {/* Ticker */}
              <div>
                <label className="label-sm">Nome / Ticker do Título</label>
                <input
                  id="treasury-ticker-input"
                  type="text"
                  className="input"
                  placeholder="Ex: TESOURO SELIC 2027"
                  value={form.ticker}
                  onChange={e => handleFormChange('ticker', e.target.value)}
                  required
                />
              </div>

              {/* Datas */}
              <div className="flex-row gap-md">
                <div style={{ flex: 1 }}>
                  <label className="label-sm">Data da Operação</label>
                  <input
                    id="treasury-tx-date"
                    type="date"
                    className="input"
                    value={form.transaction_date}
                    onChange={e => handleFormChange('transaction_date', e.target.value)}
                    required
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <label className="label-sm">Data de Vencimento</label>
                  <input
                    id="treasury-maturity-date"
                    type="date"
                    className="input"
                    value={form.maturity_date}
                    onChange={e => handleFormChange('maturity_date', e.target.value)}
                    required
                  />
                </div>
              </div>

              {/* Taxa Contratada */}
              <div>
                <label className="label-sm">
                  Taxa Contratada (% a.a.)
                  {form.treasury_type === 'SELIC' && (
                    <span style={{ fontSize: '0.65rem', color: '#4caf50', marginLeft: 6 }}>
                      spread sobre a Selic
                    </span>
                  )}
                  {form.treasury_type === 'PREFIXADO' && (
                    <span style={{ fontSize: '0.65rem', color: '#2196f3', marginLeft: 6 }}>
                      taxa ao ano
                    </span>
                  )}
                  {form.treasury_type === 'IPCA+' && (
                    <span style={{ fontSize: '0.65rem', color: '#ff9800', marginLeft: 6 }}>
                      spread sobre o IPCA
                    </span>
                  )}
                </label>
                <input
                  id="treasury-contracted-rate"
                  type="number"
                  className="input"
                  placeholder={form.treasury_type === 'SELIC' ? 'Ex: 0.15 (0.15% a.a. acima da Selic)' : form.treasury_type === 'PREFIXADO' ? 'Ex: 13.25 (13.25% a.a.)' : 'Ex: 6.40 (IPCA + 6.40%)'}
                  step="0.01"
                  min="0"
                  value={form.contracted_rate}
                  onChange={e => handleFormChange('contracted_rate', e.target.value)}
                  required
                />
              </div>

              {/* Quantidade e Preço */}
              <div className="flex-row gap-md">
                <div style={{ flex: 1 }}>
                  <label className="label-sm">Quantidade (frações)</label>
                  <input
                    id="treasury-quantity"
                    type="number"
                    className="input"
                    placeholder="Ex: 0.50"
                    step="0.01"
                    min="0.01"
                    value={form.quantity}
                    onChange={e => handleFormChange('quantity', e.target.value)}
                    required
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <label className="label-sm">Preço Unitário (R$)</label>
                  <input
                    id="treasury-unit-price"
                    type="number"
                    className="input"
                    placeholder="Ex: 14523.87"
                    step="0.01"
                    min="0.01"
                    value={form.unit_price}
                    onChange={e => handleFormChange('unit_price', e.target.value)}
                    required
                  />
                </div>
              </div>

              {/* Paga Cupons */}
              <label
                htmlFor="treasury-has-coupons"
                style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', fontSize: '0.82rem', color: 'var(--text-secondary)' }}
              >
                <input
                  id="treasury-has-coupons"
                  type="checkbox"
                  checked={form.has_coupons}
                  onChange={e => handleFormChange('has_coupons', e.target.checked)}
                  style={{ width: 16, height: 16, cursor: 'pointer' }}
                />
                Paga cupons semestrais (Prefixado com Juros Semestrais / IPCA+ com Juros Semestrais)
              </label>

              {/* Total estimado */}
              {form.quantity && form.unit_price && (
                <div
                  style={{
                    padding: '0.75rem 1rem',
                    borderRadius: '8px',
                    background: 'rgba(0,242,254,0.06)',
                    border: '1px solid rgba(0,242,254,0.2)',
                    fontSize: '0.82rem',
                    color: 'var(--text-primary)',
                  }}
                >
                  💡 Total da operação: <strong>{fmt(Number(form.quantity) * Number(form.unit_price))}</strong>
                </div>
              )}

              {error && (
                <div style={{ padding: '0.6rem 0.9rem', borderRadius: '6px', background: 'rgba(244,67,54,0.1)', border: '1px solid rgba(244,67,54,0.3)', color: '#ef9a9a', fontSize: '0.8rem' }}>
                  ⚠️ {error}
                </div>
              )}

              <div className="flex-row gap-sm justify-end mt-sm">
                <button type="button" onClick={closeModal} className="btn-secondary" style={{ fontSize: '0.85rem' }}>
                  Cancelar
                </button>
                <button
                  id="treasury-submit-btn"
                  type="submit"
                  className="btn-primary"
                  disabled={isSubmitting}
                  style={{ fontSize: '0.85rem', minWidth: '130px' }}
                >
                  {isSubmitting ? '⏳ Registrando...' : form.type === 'SUBSCRIPTION' ? '📥 Registrar Aplicação' : '📤 Registrar Resgate'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
