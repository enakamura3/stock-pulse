import React, { useState, useMemo } from 'react';
import { UnifiedTransaction } from './types';
import { formatMoney, formatQuantity } from './helpers';

// ─── Constants ───────────────────────────────────────────────────────────────

const MONTHS = [
  { value: '01', label: 'Janeiro' },
  { value: '02', label: 'Fevereiro' },
  { value: '03', label: 'Março' },
  { value: '04', label: 'Abril' },
  { value: '05', label: 'Maio' },
  { value: '06', label: 'Junho' },
  { value: '07', label: 'Julho' },
  { value: '08', label: 'Agosto' },
  { value: '09', label: 'Setembro' },
  { value: '10', label: 'Outubro' },
  { value: '11', label: 'Novembro' },
  { value: '12', label: 'Dezembro' },
];

const TX_TYPES = [
  { value: 'Todos', label: 'Tipo: Todos' },
  { value: 'BUY', label: 'Compra' },
  { value: 'SELL', label: 'Venda' },
  { value: 'BONUS', label: 'Bônus' },
  { value: 'SPLIT', label: 'Split' },
  { value: 'REVERSE_SPLIT', label: 'Agrupamento' },
  { value: 'SUBSCRIPTION', label: 'Aplicação/Resgate' },
];

const PAGE_SIZE = 20;

const SELECT_STYLE: React.CSSProperties = {
  padding: '0.3rem 0.6rem',
  borderRadius: '6px',
  border: '1px solid var(--panel-border)',
  background: '#1E293B',
  color: '#FFFFFF',
  fontSize: '0.8rem',
  outline: 'none',
  cursor: 'pointer',
};

const OPTION_STYLE: React.CSSProperties = { background: '#1c1f24' };

// ─── Types ───────────────────────────────────────────────────────────────────

interface TransactionWithBalance extends UnifiedTransaction {
  resulting_quantity: number;
  resulting_invested: number;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function formatDateStr(dateStr: string | null | undefined): string {
  if (!dateStr) return 'N/A';
  // Avoid new Date() with timezone issues — directly reformat the ISO string
  return dateStr.substring(0, 10).replace(/-/g, '/');
}

function formatDateGroupLabel(dateStr: string): string {
  const [year, month, day] = dateStr.split('-');
  const monthName = MONTHS.find((m) => m.value === month)?.label ?? month;
  return `${parseInt(day)} de ${monthName} de ${year}`;
}

function getBadge(tx: UnifiedTransaction): { text: string; color: string; bg: string } {
  const isRF = tx.module === 'RF';
  if (isRF) {
    return tx.type === 'SUBSCRIPTION'
      ? { text: 'APLICAÇÃO', color: '#2196F3', bg: 'rgba(33,150,243,0.08)' }
      : { text: 'RESGATE', color: '#FF9800', bg: 'rgba(255,152,0,0.08)' };
  }
  switch (tx.type) {
    case 'BUY':          return { text: 'COMPRA',      color: '#00e676', bg: 'rgba(0,230,118,0.08)' };
    case 'SELL':         return { text: 'VENDA',       color: '#ff3d00', bg: 'rgba(255,61,0,0.08)' };
    case 'BONUS':        return { text: 'BÔNUS',       color: '#00e676', bg: 'rgba(0,230,118,0.08)' };
    case 'SPLIT':        return { text: 'SPLIT',       color: '#00f2fe', bg: 'rgba(0,242,254,0.08)' };
    case 'REVERSE_SPLIT':return { text: 'AGRUPAMENTO', color: '#e040fb', bg: 'rgba(156,39,176,0.08)' };
    default:             return { text: tx.type,       color: '#aaa',    bg: 'rgba(255,255,255,0.05)' };
  }
}

// ─── Props ───────────────────────────────────────────────────────────────────

interface TransactionHistoryProps {
  transactions: UnifiedTransaction[];
  filterTxTicker: string;
  setFilterTxTicker: (t: string) => void;
  handleEditTransaction: (tx: UnifiedTransaction) => void;
  handleDeleteTransaction: (id: string) => void;
  onLaunchOperation?: () => void;
  kpiCurrency: string;
}

// ─── Component ───────────────────────────────────────────────────────────────

export default function TransactionHistory({
  transactions,
  filterTxTicker,
  setFilterTxTicker,
  handleEditTransaction,
  handleDeleteTransaction,
  onLaunchOperation,
  kpiCurrency,
}: TransactionHistoryProps) {
  const [filterTxYear, setFilterTxYear]   = useState<string>('Todos');
  const [filterTxMonth, setFilterTxMonth] = useState<string>('Todos');
  const [filterTxType, setFilterTxType]   = useState<string>('Todos');
  const [currentPage, setCurrentPage]     = useState<number>(1);

  // ── Running balances (oldest → newest, then reversed) ──────────────────────
  const transactionsWithBalance = useMemo<TransactionWithBalance[]>(() => {
    const cumQty: Record<string, number> = {};
    const cumInv: Record<string, number> = {};

    return [...transactions].reverse().map((tx): TransactionWithBalance => {
      if (tx.module === 'RF' && tx.asset_type !== 'TESOURO') {
        return { ...tx, resulting_quantity: 0, resulting_invested: 0 };
      }

      let qty    = cumQty[tx.asset_name] ?? 0;
      let inv    = cumInv[tx.asset_name] ?? 0;
      const txQty   = tx.quantity    ?? 0;
      const txPrice = tx.unit_price  ?? 0;

      if (tx.type === 'BUY' || tx.type === 'SUBSCRIPTION') {
        qty += txQty;
        inv += txQty * txPrice;
      } else if (tx.type === 'BONUS') {
        qty += txQty;
      } else if (tx.type === 'SELL' || tx.type === 'REDEMPTION') {
        const avgCost = qty > 0 ? inv / qty : 0;
        qty -= txQty;
        inv  = qty > 0 ? qty * avgCost : 0;
      } else if (tx.type === 'SPLIT') {
        qty = qty * txQty;
      } else if (tx.type === 'REVERSE_SPLIT') {
        qty = qty / txQty;
      }

      cumQty[tx.asset_name] = qty;
      cumInv[tx.asset_name] = inv;
      return { ...tx, resulting_quantity: qty, resulting_invested: inv };
    }).reverse();
  }, [transactions]);

  // ── Filtering ──────────────────────────────────────────────────────────────
  const filteredTransactions = useMemo(() => {
    return transactionsWithBalance.filter((tx) => {
      if (filterTxTicker !== '' && tx.asset_name !== filterTxTicker) return false;
      const year  = tx.date ? tx.date.substring(0, 4) : '';
      const month = tx.date ? tx.date.substring(5, 7) : '';
      if (filterTxYear  !== 'Todos' && year  !== filterTxYear)  return false;
      if (filterTxMonth !== 'Todos' && month !== filterTxMonth) return false;
      if (filterTxType  !== 'Todos') {
        // RF module: SUBSCRIPTION matches both Aplicação and Resgate
        if (filterTxType === 'SUBSCRIPTION') {
          if (tx.module !== 'RF') return false;
        } else {
          if (tx.type !== filterTxType) return false;
        }
      }
      return true;
    });
  }, [transactionsWithBalance, filterTxTicker, filterTxYear, filterTxMonth, filterTxType]);

  // ── Financial summary ─────────────────────────────────────────────────────
  const summary = useMemo(() => {
    let totalBought = 0;
    let totalSold   = 0;
    filteredTransactions.forEach((tx) => {
      let val = tx.total_value ?? 0;
      if (kpiCurrency && tx.currency && tx.currency !== kpiCurrency) {
        if (tx.exchange_rate && tx.exchange_rate > 0) {
          val = val * tx.exchange_rate;
        }
      }
      if (tx.type === 'BUY' || tx.type === 'BONUS' || (tx.module === 'RF' && tx.type === 'SUBSCRIPTION')) {
        totalBought += val;
      } else if (tx.type === 'SELL' || (tx.module === 'RF' && tx.type !== 'SUBSCRIPTION')) {
        totalSold += val;
      }
    });
    return { totalBought, totalSold };
  }, [filteredTransactions, kpiCurrency]);

  // ── Pagination ────────────────────────────────────────────────────────────
  const totalPages   = Math.max(1, Math.ceil(filteredTransactions.length / PAGE_SIZE));
  const safePage     = Math.min(currentPage, totalPages);
  const pagedTxs     = filteredTransactions.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  // Reset to page 1 whenever filters change
  const handleFilterChange = (setter: (v: string) => void) => (e: React.ChangeEvent<HTMLSelectElement>) => {
    setter(e.target.value);
    setCurrentPage(1);
  };

  // ── Date grouping ─────────────────────────────────────────────────────────
  const grouped = useMemo(() => {
    const groups: { date: string; label: string; txs: TransactionWithBalance[] }[] = [];
    pagedTxs.forEach((tx) => {
      const dateKey = tx.date ? tx.date.substring(0, 10) : 'N/A';
      const last = groups[groups.length - 1];
      if (!last || last.date !== dateKey) {
        groups.push({
          date: dateKey,
          label: dateKey !== 'N/A' ? formatDateGroupLabel(dateKey) : 'Data desconhecida',
          txs: [tx],
        });
      } else {
        last.txs.push(tx);
      }
    });
    return groups;
  }, [pagedTxs]);

  // ── Derived data for selects ───────────────────────────────────────────────
  const tickers       = useMemo(() => Array.from(new Set(transactions.map((tx) => tx.asset_name))).sort(), [transactions]);
  const availableYears = useMemo(
    () =>
      Array.from(new Set(transactions.map((tx) => (tx.date ? tx.date.substring(0, 4) : ''))))
        .filter((y) => y !== '')
        .sort((a, b) => b.localeCompare(a)),
    [transactions]
  );

  // ─────────────────────────────────────────────────────────────────────────
  return (
    <div className="card flex-col gap-md" style={{ flex: '1 1 350px', minHeight: '800px' }}>

      {/* ── Header ────────────────────────────────────────────────────────── */}
      <div className="flex-row justify-between items-center flex-wrap gap-md">
        <div className="flex-col" style={{ gap: '0.1rem' }}>
          <h3 className="card-title" style={{ margin: 0 }}>📜 Histórico de Operações</h3>
          {transactions.length > 0 && (
            <span className="text-secondary text-xs">
              Exibindo <strong>{filteredTransactions.length}</strong> de <strong>{transactions.length}</strong> operações
            </span>
          )}
        </div>

        <div className="flex-row gap-sm flex-wrap items-center">
          {transactions.length > 0 && (
            <>
              <select value={filterTxYear}  onChange={handleFilterChange(setFilterTxYear)}  style={SELECT_STYLE}>
                <option value="Todos" style={OPTION_STYLE}>Ano: Todos</option>
                {availableYears.map((year) => (
                  <option key={year} value={year} style={OPTION_STYLE}>{year}</option>
                ))}
              </select>

              <select value={filterTxMonth} onChange={handleFilterChange(setFilterTxMonth)} style={SELECT_STYLE}>
                <option value="Todos" style={OPTION_STYLE}>Mês: Todos</option>
                {MONTHS.map(({ value, label }) => (
                  <option key={value} value={value} style={OPTION_STYLE}>{label}</option>
                ))}
              </select>

              <select value={filterTxType}  onChange={handleFilterChange(setFilterTxType)}  style={SELECT_STYLE}>
                {TX_TYPES.map(({ value, label }) => (
                  <option key={value} value={value} style={OPTION_STYLE}>{label}</option>
                ))}
              </select>

              <select value={filterTxTicker} onChange={(e) => { setFilterTxTicker(e.target.value); setCurrentPage(1); }} style={SELECT_STYLE}>
                <option value="" style={OPTION_STYLE}>Todos os Ativos</option>
                {tickers.map((ticker) => (
                  <option key={ticker} value={ticker} style={OPTION_STYLE}>{ticker}</option>
                ))}
              </select>
            </>
          )}
          {onLaunchOperation && (
            <button className="primary-button" onClick={onLaunchOperation} style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}>
              + Lançar Operação
            </button>
          )}
        </div>
      </div>

      {/* ── Financial Summary Bar ─────────────────────────────────────────── */}
      {filteredTransactions.length > 0 && (summary.totalBought > 0 || summary.totalSold > 0) && (
        <div className="flex-row gap-md flex-wrap" style={{
          padding: '0.6rem 0.9rem',
          background: 'rgba(255,255,255,0.03)',
          border: '1px solid var(--panel-border)',
          borderRadius: '8px',
          fontSize: '0.8rem',
        }}>
          {summary.totalBought > 0 && (
            <span className="text-secondary">
              💰 Total Comprado:{' '}
              <strong style={{ color: '#00e676' }}>{formatMoney(summary.totalBought, kpiCurrency)}</strong>
            </span>
          )}
          {summary.totalSold > 0 && (
            <span className="text-secondary">
              📤 Total Vendido:{' '}
              <strong style={{ color: '#ff3d00' }}>{formatMoney(summary.totalSold, kpiCurrency)}</strong>
            </span>
          )}
        </div>
      )}

      {/* ── Transaction List ──────────────────────────────────────────────── */}
      <div className="flex-col gap-sm" style={{ overflowY: 'auto', flex: 1, maxHeight: '800px' }}>
        {filteredTransactions.length > 0 ? (
          <>
            {grouped.map((group) => (
              <div key={group.date} className="flex-col" style={{ gap: '0.5rem', marginBottom: '0.8rem' }}>
                {/* Date group separator */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.6rem',
                  padding: '0.3rem 0',
                }}>
                  <span className="text-secondary text-xs" style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>
                    {group.label}
                  </span>
                  <div style={{ flex: 1, height: '1px', background: 'var(--panel-border)' }} />
                  <span className="text-secondary text-xs" style={{ opacity: 0.5, whiteSpace: 'nowrap' }}>
                    {group.txs.length} op.
                  </span>
                </div>

                {/* Transactions list container for this day */}
                <div className="flex-col" style={{ gap: '0.5rem' }}>
                  {group.txs.map((tx) => {
                    const isRF      = tx.module === 'RF';
                    const isSplit   = tx.type === 'SPLIT' || tx.type === 'REVERSE_SPLIT';
                    const isReverse = tx.type === 'REVERSE_SPLIT';
                    const badge     = getBadge(tx);

                    return (
                      <div
                        key={tx.id}
                        style={{
                          padding: '0.7rem 0.9rem',
                          background: 'rgba(255,255,255,0.015)',
                          border: '1px solid var(--panel-border)',
                          borderRadius: '8px',
                          fontSize: '0.8rem',
                          display: 'flex',
                          flexDirection: 'column',
                          gap: '0.35rem',
                        }}
                      >
                        {/* Row 1: icon + badge + ticker + date */}
                        <div className="flex-row items-center justify-between flex-wrap" style={{ gap: '0.4rem' }}>
                          <div className="flex-row items-center" style={{ gap: '0.5rem', flexWrap: 'wrap' }}>
                            <span title={isRF ? (tx.asset_type === 'TESOURO' ? 'Tesouro Direto' : 'Renda Fixa') : 'Renda Variável'} style={{ fontSize: '1.1rem', cursor: 'help' }}>
                              {isRF ? (tx.asset_type === 'TESOURO' ? '🏛️' : '🏦') : '📈'}
                            </span>
                            <span className="badge" style={{ color: badge.color, background: badge.bg }}>
                              {badge.text}
                            </span>
                            <span className="font-bold text-primary">{tx.asset_name}</span>
                            <span className="text-secondary text-xs">{formatDateStr(tx.date)}</span>
                          </div>

                          {/* Action buttons — larger for WCAG accessibility */}
                          <div className="flex-row items-center" style={{ gap: '0.4rem' }}>
                            <button
                              onClick={() => handleEditTransaction(tx)}
                              className="btn-secondary"
                              title="Editar Transação"
                              style={{ padding: '0.5rem 0.65rem', display: 'flex', alignItems: 'center', justifyContent: 'center', minWidth: '36px', minHeight: '36px' }}
                            >
                              ✏️
                            </button>
                            <button
                              onClick={() => handleDeleteTransaction(tx.id)}
                              className="btn-danger"
                              title="Excluir Transação"
                              style={{ padding: '0.5rem 0.65rem', display: 'flex', alignItems: 'center', justifyContent: 'center', minWidth: '36px', minHeight: '36px' }}
                            >
                              🗑️
                            </button>
                          </div>
                        </div>

                        {/* Row 2: financial details */}
                        <div className="flex-row items-center flex-wrap" style={{ gap: '0.5rem', paddingLeft: '0.1rem' }}>
                          {isRF && tx.asset_type !== 'TESOURO' ? (
                            <span className="font-bold">{formatMoney(tx.total_value ?? 0, tx.currency || 'BRL')}</span>
                          ) : !isSplit ? (
                            <>
                              <span className="text-secondary">
                                {formatQuantity(tx.quantity ?? 0)} un. × {formatMoney(tx.unit_price ?? 0, tx.currency || 'BRL')}
                              </span>
                              {kpiCurrency && tx.currency !== kpiCurrency && tx.exchange_rate ? (
                                <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: '0.75rem' }}>
                                  (Câmbio: {tx.exchange_rate.toFixed(4)})
                                </span>
                              ) : null}
                              <span className="text-secondary">=</span>
                              <span className="font-bold">{formatMoney(tx.total_value ?? 0, tx.currency || 'BRL')}</span>
                            </>
                          ) : (
                            <span className="font-bold">
                              Fator: {isReverse
                                ? `${formatQuantity(tx.quantity ?? 0)} para 1`
                                : `1 para ${formatQuantity(tx.quantity ?? 0)}`}
                            </span>
                          )}

                          {(!isRF || tx.asset_type === 'TESOURO') && (
                            <span style={{
                              marginLeft: '0.5rem',
                              borderLeft: '1px solid var(--panel-border)',
                              paddingLeft: '0.75rem',
                              fontSize: '0.75rem',
                              color: 'rgba(255,255,255,0.6)',
                              display: 'flex',
                              gap: '0.4rem',
                              flexWrap: 'wrap',
                            }}>
                              Saldo após:{' '}
                              <strong style={{ color: '#00f2fe' }}>{formatQuantity(tx.resulting_quantity)} un.</strong>
                              <span style={{ opacity: 0.4 }}>|</span>
                              Investido:{' '}
                              <strong style={{ color: '#00e676' }}>{formatMoney(tx.resulting_invested, tx.currency || 'BRL')}</strong>
                            </span>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}

            {/* ── Pagination ──────────────────────────────────────────────── */}
            {totalPages > 1 && (
              <div className="flex-row items-center justify-center gap-sm" style={{ padding: '0.75rem 0', flexWrap: 'wrap' }}>
                <button
                  className="btn-secondary"
                  onClick={() => setCurrentPage(1)}
                  disabled={safePage === 1}
                  style={{ padding: '0.35rem 0.6rem', fontSize: '0.75rem', opacity: safePage === 1 ? 0.4 : 1 }}
                >
                  «
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={safePage === 1}
                  style={{ padding: '0.35rem 0.6rem', fontSize: '0.75rem', opacity: safePage === 1 ? 0.4 : 1 }}
                >
                  ‹
                </button>

                {Array.from({ length: totalPages }, (_, i) => i + 1)
                  .filter((p) => Math.abs(p - safePage) <= 2 || p === 1 || p === totalPages)
                  .reduce<(number | '…')[]>((acc, p, idx, arr) => {
                    if (idx > 0 && (arr[idx - 1] as number) !== p - 1) acc.push('…');
                    acc.push(p);
                    return acc;
                  }, [])
                  .map((item, idx) =>
                    item === '…' ? (
                      <span key={`ellipsis-${idx}`} style={{ color: 'rgba(255,255,255,0.3)', fontSize: '0.75rem' }}>…</span>
                    ) : (
                      <button
                        key={item}
                        className={item === safePage ? 'primary-button' : 'btn-secondary'}
                        onClick={() => setCurrentPage(item as number)}
                        style={{ padding: '0.35rem 0.6rem', fontSize: '0.75rem', minWidth: '32px' }}
                      >
                        {item}
                      </button>
                    )
                  )}

                <button
                  className="btn-secondary"
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={safePage === totalPages}
                  style={{ padding: '0.35rem 0.6rem', fontSize: '0.75rem', opacity: safePage === totalPages ? 0.4 : 1 }}
                >
                  ›
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => setCurrentPage(totalPages)}
                  disabled={safePage === totalPages}
                  style={{ padding: '0.35rem 0.6rem', fontSize: '0.75rem', opacity: safePage === totalPages ? 0.4 : 1 }}
                >
                  »
                </button>

                <span className="text-secondary text-xs" style={{ marginLeft: '0.25rem' }}>
                  Página {safePage} de {totalPages}
                </span>
              </div>
            )}
          </>
        ) : (
          <div className="flex-col items-center justify-center text-secondary" style={{ height: '200px', border: '1px dashed var(--panel-border)', borderRadius: '8px', gap: '0.5rem' }}>
            <span style={{ fontSize: '2rem' }}>📭</span>
            <p className="text-xs text-center m-0">
              {transactions.length === 0
                ? 'Nenhuma transação registrada nesta carteira.'
                : 'Nenhuma transação encontrada com os filtros aplicados.'}
            </p>
            {transactions.length > 0 && (
              <button
                className="btn-secondary"
                style={{ fontSize: '0.75rem', padding: '0.3rem 0.75rem', marginTop: '0.25rem' }}
                onClick={() => { setFilterTxYear('Todos'); setFilterTxMonth('Todos'); setFilterTxType('Todos'); setFilterTxTicker(''); setCurrentPage(1); }}
              >
                Limpar filtros
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
