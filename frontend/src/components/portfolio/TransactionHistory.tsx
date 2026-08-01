import React, { useState, useMemo } from 'react';
import { UnifiedTransaction } from './types';
import { formatMoney } from './helpers';
import {
  TransactionWithBalance,
  PAGE_SIZE,
  formatDateGroupLabel,
} from './transactions/types';
import TransactionFilterBar from './transactions/TransactionFilterBar';
import TransactionGroupList from './transactions/TransactionGroupList';

interface TransactionHistoryProps {
  transactions: UnifiedTransaction[];
  filterTxTicker: string;
  setFilterTxTicker: (t: string) => void;
  handleEditTransaction: (tx: UnifiedTransaction) => void;
  handleDeleteTransaction: (txId: string) => void;
  onLaunchOperation?: () => void;
  kpiCurrency?: string;
}

export default function TransactionHistory({
  transactions,
  filterTxTicker,
  setFilterTxTicker,
  handleEditTransaction,
  handleDeleteTransaction,
  onLaunchOperation,
  kpiCurrency = 'BRL',
}: TransactionHistoryProps) {
  const [filterTxYear, setFilterTxYear] = useState<string>('Todos');
  const [filterTxMonth, setFilterTxMonth] = useState<string>('Todos');
  const [filterTxType, setFilterTxType] = useState<string>('Todos');
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [groupByDate, setGroupByDate] = useState<boolean>(true);

  // Calcula o saldo resultante após cada transação cronológica
  const transactionsWithBalance = useMemo(() => {
    const sorted = [...transactions].sort((a, b) => {
      const dateA = new Date(a.date).getTime();
      const dateB = new Date(b.date).getTime();
      return dateA - dateB;
    });

    const balanceMap: Record<string, { qty: number; invested: number }> = {};

    const enriched = sorted.map((tx) => {
      const assetKey = tx.asset_name;
      if (!balanceMap[assetKey]) {
        balanceMap[assetKey] = { qty: 0, invested: 0 };
      }

      const current = balanceMap[assetKey];
      const isRF = tx.module === 'RF';

      if (isRF) {
        if (tx.type === 'SUBSCRIPTION') {
          current.invested += tx.total_value || 0;
        } else {
          current.invested -= tx.total_value || 0;
        }
      } else {
        if (tx.type === 'BUY') {
          current.qty += tx.quantity || 0;
          current.invested += tx.total_value || 0;
        } else if (tx.type === 'SELL') {
          current.qty -= tx.quantity || 0;
          current.invested -= tx.total_value || 0;
        } else if (tx.type === 'SPLIT' && tx.quantity) {
          current.qty *= tx.quantity;
        } else if (tx.type === 'REVERSE_SPLIT' && tx.quantity) {
          current.qty = Math.floor(current.qty / tx.quantity);
        } else if (tx.type === 'BONUS') {
          current.qty += tx.quantity || 0;
          current.invested += tx.total_value || 0;
        }
      }

      return {
        ...tx,
        resulting_quantity: current.qty,
        resulting_invested: current.invested,
      } as TransactionWithBalance;
    });

    return enriched.reverse();
  }, [transactions]);

  // Filtragem
  const filteredTransactions = useMemo(() => {
    return transactionsWithBalance.filter((tx) => {
      if (filterTxTicker !== '' && tx.asset_name !== filterTxTicker) return false;
      const year = tx.date ? tx.date.substring(0, 4) : '';
      const month = tx.date ? tx.date.substring(5, 7) : '';
      if (filterTxYear !== 'Todos' && year !== filterTxYear) return false;
      if (filterTxMonth !== 'Todos' && month !== filterTxMonth) return false;
      if (filterTxType !== 'Todos') {
        if (filterTxType === 'SUBSCRIPTION') {
          if (tx.module !== 'RF') return false;
        } else {
          if (tx.type !== filterTxType) return false;
        }
      }
      return true;
    });
  }, [transactionsWithBalance, filterTxTicker, filterTxYear, filterTxMonth, filterTxType]);

  // Resumo financeiro
  const summary = useMemo(() => {
    let totalBought = 0;
    let totalSold = 0;
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

  // Paginação
  const totalPages = Math.max(1, Math.ceil(filteredTransactions.length / PAGE_SIZE));
  const safePage = Math.min(currentPage, totalPages);
  const pagedTxs = filteredTransactions.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  // Agrupamento por Data
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

  const tickers = useMemo(
    () => Array.from(new Set(transactions.map((tx) => tx.asset_name))).sort(),
    [transactions]
  );
  const availableYears = useMemo(() => {
    const years = new Set(transactions.map((tx) => (tx.date ? tx.date.substring(0, 4) : '')));
    years.add(String(new Date().getFullYear()));
    return Array.from(years)
      .filter((y) => y !== '')
      .sort((a, b) => b.localeCompare(a));
  }, [transactions]);

  if (transactions.length === 0) {
    return (
      <div className="card flex-col items-center justify-center text-secondary" style={{ padding: '3rem' }}>
        <span className="text-2xl mb-sm">📜</span>
        <p className="m-0">Nenhuma transação registrada nesta carteira.</p>
        {onLaunchOperation && (
          <button onClick={onLaunchOperation} className="primary-button mt-md" style={{ padding: '0.4rem 1rem', fontSize: '0.8rem' }}>
            + Lançar primeira operação
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="card flex-col gap-md" style={{ flex: '1 1 350px' }}>
      {/* ── Filtros ── */}
      <TransactionFilterBar
        filterTxTicker={filterTxTicker}
        setFilterTxTicker={setFilterTxTicker}
        filterType={filterTxType}
        setFilterType={setFilterTxType}
        filterYear={filterTxYear}
        setFilterYear={setFilterTxYear}
        filterMonth={filterTxMonth}
        setFilterMonth={setFilterTxMonth}
        groupByDate={groupByDate}
        setGroupByDate={setGroupByDate}
        availableYears={availableYears}
        availableTickers={tickers}
        totalFilteredCount={filteredTransactions.length}
        onLaunchOperation={onLaunchOperation || (() => {})}
      />

      {/* ── Cards de Resumo ── */}
      {(summary.totalBought > 0 || summary.totalSold > 0) && (
        <div className="flex-row gap-md flex-wrap" style={{ width: '100%' }}>
          {summary.totalBought > 0 && (
            <div
              style={{
                flex: '1 1 200px',
                padding: '0.75rem 1rem',
                background: 'linear-gradient(135deg, rgba(0, 230, 118, 0.08) 0%, rgba(0, 0, 0, 0) 100%)',
                border: '1px solid rgba(0, 230, 118, 0.2)',
                borderRadius: '10px',
                display: 'flex',
                alignItems: 'center',
                gap: '0.75rem',
              }}
            >
              <div style={{ fontSize: '1.5rem' }}>📥</div>
              <div className="flex-col">
                <span className="text-secondary text-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                  Total Comprado:{' '}
                  <strong style={{ color: '#00e676', fontSize: '1.05rem', fontWeight: 700, display: 'block', marginTop: '0.2rem' }}>
                    {formatMoney(summary.totalBought, kpiCurrency)}
                  </strong>
                </span>
              </div>
            </div>
          )}

          {summary.totalSold > 0 && (
            <div
              style={{
                flex: '1 1 200px',
                padding: '0.75rem 1rem',
                background: 'linear-gradient(135deg, rgba(255, 61, 0, 0.08) 0%, rgba(0, 0, 0, 0) 100%)',
                border: '1px solid rgba(255, 61, 0, 0.2)',
                borderRadius: '10px',
                display: 'flex',
                alignItems: 'center',
                gap: '0.75rem',
              }}
            >
              <div style={{ fontSize: '1.5rem' }}>📤</div>
              <div className="flex-col">
                <span className="text-secondary text-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                  Total Vendido:{' '}
                  <strong style={{ color: '#ff3d00', fontSize: '1.05rem', fontWeight: 700, display: 'block', marginTop: '0.2rem' }}>
                    {formatMoney(summary.totalSold, kpiCurrency)}
                  </strong>
                </span>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Lista de Transações ── */}
      <TransactionGroupList
        grouped={grouped}
        pagedTxs={pagedTxs}
        groupByDate={groupByDate}
        kpiCurrency={kpiCurrency}
        handleEditTransaction={handleEditTransaction}
        handleDeleteTransaction={handleDeleteTransaction}
      />

      {/* ── Paginação ── */}
      {totalPages > 1 && (
        <div className="flex-row justify-between items-center mt-md pt-sm" style={{ borderTop: '1px solid var(--panel-border)' }}>
          <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
            Página {safePage} de {totalPages}
          </span>

          <div className="flex-row gap-xs">
            <button
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={safePage === 1}
              className="btn-secondary"
              style={{ padding: '0.3rem 0.7rem', fontSize: '0.8rem', opacity: safePage === 1 ? 0.5 : 1 }}
            >
              ← Anterior
            </button>
            <button
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={safePage === totalPages}
              className="btn-secondary"
              style={{ padding: '0.3rem 0.7rem', fontSize: '0.8rem', opacity: safePage === totalPages ? 0.5 : 1 }}
            >
              Próxima →
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
