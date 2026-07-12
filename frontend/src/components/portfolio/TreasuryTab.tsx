'use client';

import React, { useState, useEffect, useCallback } from 'react';
import dynamic from 'next/dynamic';

const PortfolioChart = dynamic(() => import('@/components/PortfolioChart'), { ssr: false });

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

import { TreasuryPosition, TreasuryPerfPoint } from './types';

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
  positions: TreasuryPosition[];
  isLoadingPositions: boolean;
  onRefresh: () => Promise<void>;
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

export default function TreasuryTab({ portfolioId, positions, isLoadingPositions, onRefresh }: TreasuryTabProps) {
  const [perfData, setPerfData] = useState<TreasuryPerfPoint[]>([]);
  const [isImporting, setIsImporting] = useState(false);
  const [isLoadingPerf, setIsLoadingPerf] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState<NewTreasuryTx>(EMPTY_TX);
  const [editingTxId, setEditingTxId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // ── Fetchers ────────────────────────────────────────────────────────────────

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
    fetchPerformance();
  }, [fetchPerformance]);

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
    setEditingTxId(null);
    setError(null);
    setShowModal(true);
  }

  function closeModal() {
    setShowModal(false);
    setEditingTxId(null);
    setError(null);
  }

  function handleFormChange(field: keyof NewTreasuryTx, value: string | number | boolean) {
    setForm(prev => ({ ...prev, [field]: value }));
  }

  function handleEdit(pos: TreasuryPosition) {
    setEditingTxId(pos.transaction_id);
    setForm({
      ticker: pos.ticker,
      treasury_type: pos.treasury_type,
      maturity_date: new Date(pos.maturity_date).toISOString().split('T')[0],
      has_coupons: pos.has_coupons,
      type: 'SUBSCRIPTION',
      quantity: pos.quantity,
      unit_price: pos.unit_price,
      contracted_rate: pos.contracted_rate,
      transaction_date: new Date(pos.start_date).toISOString().split('T')[0],
    });
    setError(null);
    setShowModal(true);
  }

  async function handleDelete(txId: string) {
    if (!confirm('Deseja realmente excluir esta operação do Tesouro Direto? Isso irá recalcular o FIFO e o histórico da carteira.')) {
      return;
    }
    try {
      const res = await fetch(`${API_URL}/portfolios/${portfolioId}/treasury/transactions/${txId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        await onRefresh();
        fetchPerformance();
      } else {
        const txt = await res.text();
        alert(txt || 'Erro ao excluir a operação.');
      }
    } catch (e) {
      console.error(e);
      alert('Erro de conexão. Tente novamente.');
    }
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

      const method = editingTxId ? 'PUT' : 'POST';
      const url = editingTxId
        ? `${API_URL}/portfolios/${portfolioId}/treasury/transactions/${editingTxId}`
        : `${API_URL}/portfolios/${portfolioId}/treasury/transactions`;

      const res = await fetch(url, {
        method: method,
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        closeModal();
        await onRefresh();
        fetchPerformance();
      } else {
        const txt = await res.text();
        setError(txt || 'Erro ao salvar a operação.');
      }
    } catch {
      setError('Erro de conexão. Tente novamente.');
    } finally {
      setIsSubmitting(false);
    }
  }
  // ── Import / Export handlers ────────────────────────────────────────────────

  async function handleExport() {
    try {
      const res = await fetch(`${API_URL}/portfolios/${portfolioId}/treasury/transactions`, {
        credentials: 'include',
      });
      if (res.ok) {
        const data: NewTreasuryTx[] = await res.json();
        if (!data || data.length === 0) {
          alert('Não há operações para exportar.');
          return;
        }

        const headers = Object.keys(data[0]).join(',');
        const rows = data.map(op => {
          return Object.values(op).map(val => {
            const str = String(val);
            return str.includes(',') ? `"${str}"` : str;
          }).join(',');
        });
        const csvContent = [headers, ...rows].join('\n');

        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `tesouro_direto_${portfolioId}_${new Date().toISOString().split('T')[0]}.csv`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
      } else {
        alert('Erro ao exportar operações.');
      }
    } catch (e) {
      console.error(e);
      alert('Erro de conexão ao exportar.');
    }
  }

  async function handleImport(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      const lines = text.trim().split('\n');
      if (lines.length < 2) {
        alert('Arquivo CSV vazio ou sem dados suficientes.');
        return;
      }

      const headers = lines[0].split(',').map(h => h.trim());
      const operations: NewTreasuryTx[] = [];

      for (let i = 1; i < lines.length; i++) {
        const line = lines[i].trim();
        if (!line) continue;
        const values = line.split(',').map(v => v.trim().replace(/^"|"$/g, ''));
        const op: any = {};
        for (let j = 0; j < headers.length; j++) {
          const header = headers[j];
          let val: any = values[j];
          if (['quantity', 'unit_price', 'contracted_rate'].includes(header)) {
            val = Number(val);
          } else if (header === 'has_coupons') {
            val = val === 'true';
          }
          op[header] = val;
        }
        operations.push(op as NewTreasuryTx);
      }

      setIsImporting(true);
      let successCount = 0;
      let errorCount = 0;

      for (const op of operations) {
        try {
          const res = await fetch(`${API_URL}/portfolios/${portfolioId}/treasury/transactions`, {
            method: 'POST',
            credentials: 'include',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(op),
          });
          if (res.ok) {
            successCount++;
          } else {
            errorCount++;
          }
        } catch {
          errorCount++;
        }
      }

      alert(`Importação concluída: ${successCount} sucesso(s), ${errorCount} erro(s).`);
      await onRefresh();
      fetchPerformance();
    } catch (err) {
      alert('Erro ao processar arquivo CSV. Certifique-se de que é um formato válido.');
    } finally {
      setIsImporting(false);
      e.target.value = ''; // reset
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
      <div className="card flex-col gap-md" style={{ flex: '2 1 600px', minHeight: '380px' }}>
        <div className="flex-row justify-between items-center mb-lg">
          <div>
            <h3 className="card-title">📋 Posições Ativas</h3>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
              {positions.length} título{positions.length !== 1 ? 's' : ''}
            </span>
          </div>
          <div className="flex-row gap-sm">
            <label className="btn-secondary" style={{ padding: '0.45rem 1rem', fontSize: '0.8rem', cursor: 'pointer', margin: 0 }}>
              📥 Importar
              <input type="file" accept=".csv" style={{ display: 'none' }} onChange={handleImport} />
            </label>
            <button
              onClick={handleExport}
              className="btn-secondary"
              style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}
            >
              📤 Exportar
            </button>
            <button
              id="treasury-add-btn"
              onClick={openModal}
              className="primary-button"
              style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}
            >
              + Nova Aplicação
            </button>
          </div>
        </div>

        {(isLoadingPositions || isImporting) ? (
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
          <div className="table-container flex-col" style={{ flex: 1 }}>
            <table className="data-table" style={{ width: '100%' }}>
              <thead>
                <tr>
                  <th>Título</th>
                  <th>Tipo</th>
                  <th className="text-right">Vencimento</th>
                  <th className="text-right">Aplicado</th>
                  <th className="text-right">Bruto</th>
                  <th className="text-right">Líquido</th>
                  <th className="text-right">Retorno Líq.</th>
                  <th className="text-right">IOF</th>
                  <th className="text-right">IR</th>
                  <th className="text-right">Taxa B3</th>
                  <th className="text-center">Status</th>
                  <th className="text-center">Ações</th>
                </tr>
              </thead>
              <tbody>
                {positions.map((pos, i) => {
                  const liqReturn = pos.total_invested > 0
                    ? ((pos.net_value - pos.total_invested) / pos.total_invested) * 100
                    : 0;
                  const isPositive = liqReturn >= 0;
                  return (
                    <tr key={pos.transaction_id}>
                      <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
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
                      <td>
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
                      <td className="text-right" style={{ color: 'var(--text-secondary)' }}>
                        {pos.is_matured
                          ? <span style={{ color: '#f44336', fontWeight: 600 }}>Vencido</span>
                          : `${new Date(pos.maturity_date).toLocaleDateString('pt-BR')} (${pos.days_to_maturity}d)`}
                      </td>
                      <td className="text-right" style={{ fontFamily: 'monospace' }}>{fmt(pos.total_invested)}</td>
                      <td className="text-right" style={{ fontFamily: 'monospace' }}>{fmt(pos.gross_value)}</td>
                      <td className="text-right font-semibold" style={{ fontFamily: 'monospace', color: pos.net_value >= pos.total_invested ? '#4caf50' : '#f44336' }}>
                        {fmt(pos.net_value)}
                      </td>
                      <td className="text-right font-semibold" style={{ fontFamily: 'monospace', color: isPositive ? '#4caf50' : '#f44336' }}>
                        {fmtPct(liqReturn)}
                      </td>
                      <td className="text-right" style={{ fontFamily: 'monospace', color: 'var(--text-secondary)' }}>{fmt(pos.iof_tax)}</td>
                      <td className="text-right" style={{ fontFamily: 'monospace', color: 'var(--text-secondary)' }}>{fmt(pos.ir_tax)}</td>
                      <td className="text-right" style={{ fontFamily: 'monospace', color: 'var(--text-secondary)' }}>{fmt(pos.b3_fee)}</td>
                      <td className="text-center">
                        {pos.is_matured ? (
                          <span style={{ color: '#f44336', fontSize: '0.7rem', fontWeight: 700 }}>● Vencido</span>
                        ) : (
                          <span style={{ color: '#4caf50', fontSize: '0.7rem', fontWeight: 700 }}>● Ativo</span>
                        )}
                      </td>
                      <td className="text-center">
                        <div className="flex-row justify-center gap-xs">
                          <button
                            onClick={() => handleEdit(pos)}
                            title="Editar"
                            style={{
                              background: 'none', border: 'none', cursor: 'pointer', fontSize: '0.9rem', padding: '2px 6px',
                              borderRadius: '4px', color: 'var(--text-secondary)'
                            }}
                            onMouseEnter={e => e.currentTarget.style.color = 'var(--accent-color)'}
                            onMouseLeave={e => e.currentTarget.style.color = 'var(--text-secondary)'}
                          >
                            ✏️
                          </button>
                          <button
                            onClick={() => handleDelete(pos.transaction_id)}
                            title="Apagar"
                            style={{
                              background: 'none', border: 'none', cursor: 'pointer', fontSize: '0.9rem', padding: '2px 6px',
                              borderRadius: '4px', color: 'var(--text-secondary)'
                            }}
                            onMouseEnter={e => e.currentTarget.style.color = '#f44336'}
                            onMouseLeave={e => e.currentTarget.style.color = 'var(--text-secondary)'}
                          >
                            🗑️
                          </button>
                        </div>
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

            <h3 className="card-title mb-lg">
              {editingTxId ? '✏️ Editar Operação — Tesouro Direto' : '🏛️ Registrar Operação — Tesouro Direto'}
            </h3>

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
                  {isSubmitting 
                    ? (editingTxId ? '⏳ Salvando...' : '⏳ Registrando...') 
                    : editingTxId 
                      ? '💾 Salvar Alterações' 
                      : form.type === 'SUBSCRIPTION' ? '📥 Registrar Aplicação' : '📤 Registrar Resgate'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
