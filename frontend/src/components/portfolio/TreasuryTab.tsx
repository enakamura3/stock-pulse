'use client';

import React, { useState, useEffect, useCallback } from 'react';
import dynamic from 'next/dynamic';

const PortfolioChart = dynamic(() => import('@/components/PortfolioChart'), { ssr: false });

import { TreasuryPosition, TreasuryPerfPoint } from './types';
import { apiFetch } from '@/lib/api';
import { NewTreasuryTx, SortKey, SortDir, fmt } from './treasury/types';
import TreasuryPositionTable from './treasury/TreasuryPositionTable';

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

  const [sortKey, setSortKey] = useState<SortKey>('ticker');
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const fetchPerf = useCallback(async () => {
    if (!portfolioId) return;
    setIsLoadingPerf(true);
    try {
      const res = await apiFetch(`/portfolios/${portfolioId}/treasury/performance`, { cache: 'no-store' });
      if (res.ok) setPerfData(await res.json() || []);
    } catch (e) {
      console.error('Erro ao buscar performance Tesouro:', e);
    } finally {
      setIsLoadingPerf(false);
    }
  }, [portfolioId]);

  useEffect(() => {
    fetchPerf();
  }, [fetchPerf]);

  const openModal = () => {
    setEditingTxId(null);
    setForm(EMPTY_TX);
    setError(null);
    setShowModal(true);
  };

  const closeModal = () => {
    setShowModal(false);
    setEditingTxId(null);
    setForm(EMPTY_TX);
    setError(null);
  };

  const handleFormChange = (field: keyof NewTreasuryTx, value: any) => {
    setForm(f => ({ ...f, [field]: value }));
  };

  const handleOpenRedemption = (pos: TreasuryPosition) => {
    setEditingTxId(null);
    setForm({
      ticker: pos.ticker,
      treasury_type: pos.treasury_type,
      maturity_date: pos.maturity_date ? pos.maturity_date.split('T')[0] : '',
      has_coupons: pos.has_coupons,
      type: 'REDEMPTION',
      quantity: pos.quantity,
      unit_price: pos.current_unit_price || pos.average_unit_price,
      contracted_rate: 0,
      transaction_date: new Date().toISOString().split('T')[0],
    });
    setError(null);
    setShowModal(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.ticker || !form.quantity || !form.unit_price || !form.transaction_date) {
      setError('Preencha todos os campos obrigatórios.');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const payload = {
        ticker: form.ticker.toUpperCase().trim(),
        treasury_type: form.treasury_type,
        maturity_date: form.maturity_date,
        has_coupons: form.has_coupons,
        type: form.type,
        quantity: Number(form.quantity),
        unit_price: Number(form.unit_price),
        contracted_rate: Number(form.contracted_rate),
        transaction_date: form.transaction_date,
      };

      const url = editingTxId
        ? `/portfolios/${portfolioId}/treasury/transactions/${editingTxId}`
        : `/portfolios/${portfolioId}/treasury/transactions`;
      const method = editingTxId ? 'PUT' : 'POST';

      const res = await apiFetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
        cache: 'no-store',
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Erro ao salvar transação.');
      }

      closeModal();
      await onRefresh();
      await fetchPerf();
    } catch (err: any) {
      setError(err.message || 'Erro inesperado.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setIsImporting(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      const res = await apiFetch(`/portfolios/${portfolioId}/treasury/bulk`, {
        method: 'POST',
        body: formData,
      });

      if (res.ok) {
        await onRefresh();
        await fetchPerf();
      } else {
        alert('Erro ao importar arquivo CSV.');
      }
    } catch (err) {
      alert('Erro de conexão ao importar arquivo.');
    } finally {
      setIsImporting(false);
      e.target.value = '';
    }
  };

  const handleExport = async () => {
    try {
      const res = await apiFetch(`/portfolios/${portfolioId}/treasury/export`, {
        method: 'GET',
        cache: 'no-store',
      });

      if (res.ok) {
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `tesouro-direto-${portfolioId}.csv`;
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);
      } else {
        alert('Erro ao exportar posições.');
      }
    } catch (err) {
      alert('Erro ao exportar posições.');
    }
  };

  // Métricas agregadas
  const totalInvested = positions.reduce((acc, p) => acc + p.total_invested, 0);
  const totalGross = positions.reduce((acc, p) => acc + p.gross_value, 0);
  const totalNet = positions.reduce((acc, p) => acc + p.net_value, 0);
  const totalProfitLoss = totalNet - totalInvested;
  const returnPct = totalInvested > 0 ? (totalProfitLoss / totalInvested) * 100 : 0;

  const totalIOF = positions.reduce((acc, p) => acc + (p.iof_tax || 0), 0);
  const totalIR = positions.reduce((acc, p) => acc + (p.ir_tax || 0), 0);
  const totalB3 = positions.reduce((acc, p) => acc + (p.b3_fee || 0), 0);

  const kpis = [
    { label: 'Total Investido', value: fmt(totalInvested), icon: '💰' },
    { label: 'Valor Bruto', value: fmt(totalGross), icon: '📊' },
    {
      label: 'Valor Líquido',
      value: fmt(totalNet),
      icon: '💵',
      sub: `${returnPct >= 0 ? '+' : ''}${returnPct.toFixed(2)}% (${fmt(totalProfitLoss)})`,
      subColor: returnPct >= 0 ? '#4caf50' : '#f44336',
    },
    { label: 'Impostos (IOF + IR)', value: fmt(totalIOF + totalIR), icon: '🏛️', sub: `IOF: ${fmt(totalIOF)} | IR: ${fmt(totalIR)}`, subColor: '#ef5350' },
    { label: 'Taxa B3 Acumulada', value: fmt(totalB3), icon: '🏷️', sub: '0,20% a.a. pró-rata', subColor: '#ff9800' },
  ];

  return (
    <div className="flex-col gap-xl">
      {/* ── KPI Cards ── */}
      <div className="flex-row gap-md flex-wrap">
        {kpis.map((card, idx) => (
          <div
            key={idx}
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
      <TreasuryPositionTable
        positions={positions}
        isLoadingPositions={isLoadingPositions}
        isImporting={isImporting}
        sortKey={sortKey}
        sortDir={sortDir}
        onSort={handleSort}
        onOpenModal={openModal}
        onImport={handleImport}
        onExport={handleExport}
        onRedeem={handleOpenRedemption}
      />

      {/* ── Modal ── */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '520px' }}>
            <div className="modal-header">
              <h3 className="modal-title">
                {editingTxId
                  ? '✏️ Editar Operação de Tesouro Direto'
                  : form.type === 'SUBSCRIPTION'
                  ? '📥 Nova Aplicação — Tesouro Direto'
                  : '📤 Novo Resgate — Tesouro Direto'}
              </h3>
              <button onClick={closeModal} className="btn-close">✕</button>
            </div>

            <form onSubmit={handleSubmit} className="flex-col gap-md">
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

              <div>
                <label className="label-sm">Taxa Contratada (% a.a.)</label>
                <input
                  id="treasury-contracted-rate"
                  type="number"
                  className="input"
                  placeholder={form.treasury_type === 'SELIC' ? 'Ex: 0.15' : form.treasury_type === 'PREFIXADO' ? 'Ex: 13.25' : 'Ex: 6.40'}
                  step="0.01"
                  min="0"
                  value={form.contracted_rate}
                  onChange={e => handleFormChange('contracted_rate', e.target.value)}
                  required
                />
              </div>

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
                Paga cupons semestrais
              </label>

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
                    ? '⏳ Salvando...' 
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
