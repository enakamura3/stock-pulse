import React, { useState, useMemo, useEffect } from 'react';
import { UnifiedTransaction } from './types';
import { formatMoney } from './helpers';
import {
  TransactionWithBalance,
  PAGE_SIZE,
  formatDateGroupLabel,
  getMacroAssetCategory,
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
  const currentYear = new Date().getFullYear().toString();
  const currentMonth = String(new Date().getMonth() + 1).padStart(2, '0');
  const [filterTxYear, setFilterTxYear] = useState<string>(currentYear);
  const [filterTxMonth, setFilterTxMonth] = useState<string>(currentMonth);
  const [filterTxType, setFilterTxType] = useState<string>('Todos');
  const [filterTxCategory, setFilterTxCategory] = useState<string>('Todos');
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [groupByDate, setGroupByDate] = useState<boolean>(true);

  // Reset de paginação sempre que qualquer filtro for alterado
  useEffect(() => {
    setCurrentPage(1);
  }, [filterTxTicker, filterTxYear, filterTxMonth, filterTxType, filterTxCategory]);

  // Ajusta inteligentemente os filtros iniciais para priorizar o mês/ano corrente se houver transações
  useEffect(() => {
    if (transactions.length === 0) return;
    const now = new Date();
    const curY = now.getFullYear().toString();
    const curM = String(now.getMonth() + 1).padStart(2, '0');

    const hasCurMonth = transactions.some((tx) => {
      if (!tx.date) return false;
      return tx.date.substring(0, 4) === curY && tx.date.substring(5, 7) === curM;
    });

    if (hasCurMonth) {
      setFilterTxYear(curY);
      setFilterTxMonth(curM);
    } else {
      const hasCurYear = transactions.some((tx) => tx.date && tx.date.substring(0, 4) === curY);
      if (hasCurYear) {
        setFilterTxYear(curY);
        setFilterTxMonth('Todos');
      } else {
        setFilterTxYear('Todos');
        setFilterTxMonth('Todos');
      }
    }
  }, [transactions]);

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

  // Transações filtradas por período, ticker e tipo (sem filtro de categoria)
  const baseFilteredTransactions = useMemo(() => {
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

  // Transações finais exibidas na tabela (aplicando filtro de categoria)
  const filteredTransactions = useMemo(() => {
    if (filterTxCategory === 'Todos') return baseFilteredTransactions;
    return baseFilteredTransactions.filter((tx) => {
      const cat = getMacroAssetCategory(tx);
      return cat.id === filterTxCategory;
    });
  }, [baseFilteredTransactions, filterTxCategory]);

  // Resumo financeiro e distribuição detalhada de aquisições por classe e por ativo
  const summary = useMemo(() => {
    let totalBought = 0;
    let totalSold = 0;
    const catMap: Record<
      string,
      {
        id: string;
        name: string;
        emoji: string;
        color: string;
        total: number;
        assetMap: Record<string, number>;
      }
    > = {};

    baseFilteredTransactions.forEach((tx) => {
      let val = tx.total_value ?? 0;
      if (kpiCurrency && tx.currency && tx.currency !== kpiCurrency) {
        if (tx.exchange_rate && tx.exchange_rate > 0) {
          val = val * tx.exchange_rate;
        }
      }

      const isBuy = tx.type === 'BUY' || tx.type === 'BONUS' || (tx.module === 'RF' && tx.type === 'SUBSCRIPTION');
      const isSell = tx.type === 'SELL' || (tx.module === 'RF' && tx.type !== 'SUBSCRIPTION');

      if (isBuy) {
        totalBought += val;
        const cat = getMacroAssetCategory(tx);
        if (!catMap[cat.id]) {
          catMap[cat.id] = { ...cat, total: 0, assetMap: {} };
        }
        catMap[cat.id].total += val;

        const ticker = tx.asset_name;
        if (!catMap[cat.id].assetMap[ticker]) {
          catMap[cat.id].assetMap[ticker] = 0;
        }
        catMap[cat.id].assetMap[ticker] += val;
      } else if (isSell) {
        totalSold += val;
      }
    });

    const netContribution = totalBought - totalSold;
    const categoryBreakdown = Object.values(catMap)
      .map((cat) => {
        const assets = Object.entries(cat.assetMap)
          .map(([ticker, assetTotal]) => ({
            ticker,
            total: assetTotal,
            categoryPercentage: cat.total > 0 ? (assetTotal / cat.total) * 100 : 0,
            totalPercentage: totalBought > 0 ? (assetTotal / totalBought) * 100 : 0,
          }))
          .sort((a, b) => b.total - a.total);

        return {
          id: cat.id,
          name: cat.name,
          emoji: cat.emoji,
          color: cat.color,
          total: cat.total,
          percentage: totalBought > 0 ? (cat.total / totalBought) * 100 : 0,
          assets,
        };
      })
      .sort((a, b) => b.total - a.total);

    return { totalBought, totalSold, netContribution, categoryBreakdown };
  }, [baseFilteredTransactions, kpiCurrency]);

  const hasActiveFilters =
    filterTxTicker !== '' ||
    filterTxType !== 'Todos' ||
    filterTxCategory !== 'Todos' ||
    filterTxYear !== 'Todos' ||
    filterTxMonth !== 'Todos';

  const handleClearFilters = () => {
    setFilterTxTicker('');
    setFilterTxType('Todos');
    setFilterTxCategory('Todos');
    setFilterTxYear('Todos');
    setFilterTxMonth('Todos');
  };

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
        filterCategory={filterTxCategory}
        setFilterCategory={setFilterTxCategory}
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
        onClearFilters={handleClearFilters}
        hasActiveFilters={hasActiveFilters}
      />

      {/* ── Painel de Resumo do Período Filtrado ── */}
      <div
        className="card flex-col gap-lg"
        style={{
          background: 'rgba(255, 255, 255, 0.02)',
          border: '1px solid var(--panel-border)',
          borderRadius: '12px',
          padding: '1.25rem 1.5rem',
        }}
      >
        {/* KPIs Principais */}
        <div className="flex-row gap-lg flex-wrap items-center justify-between" style={{ paddingBottom: '0.5rem' }}>
          <div className="flex-row items-center gap-md">
            <span style={{ fontSize: '1.5rem' }}>📥</span>
            <div className="flex-col" style={{ gap: '0.15rem' }}>
              <span className="text-secondary text-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                Total Comprado
              </span>
              <strong style={{ color: '#00e676', fontSize: '1.1rem', fontWeight: 700 }}>
                {formatMoney(summary.totalBought, kpiCurrency)}
              </strong>
            </div>
          </div>

          <div className="flex-row items-center gap-md">
            <span style={{ fontSize: '1.5rem' }}>📤</span>
            <div className="flex-col" style={{ gap: '0.15rem' }}>
              <span className="text-secondary text-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                Total Vendido
              </span>
              <strong style={{ color: '#ff3d00', fontSize: '1.1rem', fontWeight: 700 }}>
                {formatMoney(summary.totalSold, kpiCurrency)}
              </strong>
            </div>
          </div>

          <div className="flex-row items-center gap-md">
            <span style={{ fontSize: '1.5rem' }}>💰</span>
            <div className="flex-col" style={{ gap: '0.15rem' }}>
              <span className="text-secondary text-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                Aporte Líquido
              </span>
              <strong
                style={{
                  color: summary.netContribution >= 0 ? '#00f2fe' : '#ffc107',
                  fontSize: '1.1rem',
                  fontWeight: 700,
                }}
              >
                {formatMoney(summary.netContribution, kpiCurrency)}
              </strong>
            </div>
          </div>
        </div>

        {/* Breakdown de Aquisições no Período */}
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.06)', paddingTop: '1rem' }}>
          <div className="flex-row justify-between items-center mb-sm">
            <span className="text-xs font-bold text-secondary" style={{ textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              📊 Aquisições por Classe de Ativo (Clique para filtrar)
            </span>
          </div>

          {summary.categoryBreakdown.length > 0 ? (
            <div className="flex-row flex-wrap gap-md mt-xs">
              {summary.categoryBreakdown.map((cat) => {
                const isSelected = filterTxCategory === cat.id;
                return (
                  <div
                    key={cat.id}
                    onClick={() => setFilterTxCategory(isSelected ? 'Todos' : cat.id)}
                    style={{
                      flex: '1 1 210px',
                      padding: '0.85rem 1rem',
                      borderRadius: '10px',
                      background: isSelected ? 'rgba(0, 242, 254, 0.12)' : 'rgba(255, 255, 255, 0.03)',
                      border: isSelected ? `1.5px solid ${cat.color}` : '1px solid var(--panel-border)',
                      cursor: 'pointer',
                      transition: 'all 0.2s ease',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: '0.4rem',
                    }}
                    onMouseEnter={(e) => {
                      if (!isSelected) e.currentTarget.style.background = 'rgba(255, 255, 255, 0.06)';
                    }}
                    onMouseLeave={(e) => {
                      if (!isSelected) e.currentTarget.style.background = 'rgba(255, 255, 255, 0.03)';
                    }}
                  >
                    <div className="flex-row justify-between items-center" style={{ fontSize: '0.8rem' }}>
                      <span className="font-bold text-primary">
                        {cat.emoji} {cat.name}
                      </span>
                      <span style={{ color: cat.color, fontWeight: 700 }}>
                        {cat.percentage.toFixed(1)}%
                      </span>
                    </div>

                    <div className="text-xs text-secondary font-semibold" style={{ fontSize: '0.82rem' }}>
                      {formatMoney(cat.total, kpiCurrency)}
                    </div>

                    {/* Barra de Progresso */}
                    <div style={{ width: '100%', height: '5px', background: 'rgba(255,255,255,0.1)', borderRadius: '3px', overflow: 'hidden', marginTop: '0.1rem', marginBottom: '0.25rem' }}>
                      <div
                        style={{
                          width: `${Math.min(100, Math.max(2, cat.percentage))}%`,
                          height: '100%',
                          background: cat.color,
                          borderRadius: '3px',
                          transition: 'width 0.4s ease',
                        }}
                      />
                    </div>

                    {/* Detalhamento dos Ativos que Compõem a Classe */}
                    {cat.assets.length > 0 && (
                      <div
                        className="flex-col gap-xs mt-xs pt-xs"
                        style={{ borderTop: '1px dashed rgba(255,255,255,0.1)', paddingTop: '0.4rem' }}
                      >
                        {cat.assets.map((ast) => (
                          <div
                            key={ast.ticker}
                            className="flex-row justify-between items-center text-xs"
                            style={{
                              fontSize: '0.74rem',
                              padding: '0.25rem 0.4rem',
                              background: 'rgba(255, 255, 255, 0.02)',
                              borderRadius: '4px',
                            }}
                          >
                            <span className="font-semibold text-primary">
                              • {ast.ticker}
                            </span>
                            <div className="flex-row items-center gap-xs">
                              <span className="text-secondary" style={{ fontSize: '0.72rem' }}>
                                {formatMoney(ast.total, kpiCurrency)}
                              </span>
                              <span style={{ color: cat.color, fontWeight: 700, fontSize: '0.7rem' }}>
                                ({ast.categoryPercentage.toFixed(1)}%)
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-xs text-secondary italic text-center py-xs">
              Nenhuma aquisição no período filtrado.
            </div>
          )}
        </div>
      </div>

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
