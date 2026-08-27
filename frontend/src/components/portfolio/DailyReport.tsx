import React, { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Position, FixedIncomePosition, TreasuryPosition, CalculatedDividend, MarketBenchmarks } from './types';
import {
  formatMoney,
  formatPercentage,
  calculateDailyFixedIncomeRate,
  calculateEstimatedDailyGain,
  exportDailyReportCSV,
} from './helpers';
import { getMarketStatus } from '@/lib/marketHours';
import { apiFetch } from '@/lib/api';
import MarketBenchmarksBar from './MarketBenchmarksBar';

export interface DailyReportProps {
  positions: Position[];
  fiPositions?: FixedIncomePosition[];
  treasuryPositions?: TreasuryPosition[];
  dividends?: CalculatedDividend[];
  benchmarks?: MarketBenchmarks | null;
  kpiCurrency: string;
  lastFetchedAt?: Date | null;
  onRefresh?: (forceRealtime?: boolean) => Promise<void> | void;
  isRefreshing?: boolean;
  onGoToAssets?: () => void;
}

type SortKey = 'ticker' | 'average_price' | 'previousClose' | 'current_price' | 'daily_change' | 'daily_change_percent' | 'impact';
type SortDir = 'asc' | 'desc';

// Helper: calcula a taxa de câmbio de uma posição para a moeda base
function getExchangeRate(pos: Position, kpiCurrency: string = 'BRL'): number {
  const price = pos.current_price ?? 0;
  const qty = pos.quantity ?? 0;
  const val = pos.current_value ?? 0;
  if (price > 1e-6 && qty > 1e-6 && val > 1e-6) {
    return val / (price * qty);
  }
  if (pos.currency && pos.currency.toUpperCase() === kpiCurrency.toUpperCase()) {
    return 1.0;
  }
  return 0.0;
}

// Helper: retorna o rótulo de tipo de ativo
const getAssetTypeBadge = (pos: Position) => {
  const type = (pos.type || '').toUpperCase();
  if (type === 'FII') return { label: 'FII', color: 'var(--color-warning)' };
  if (type === 'ETF') return { label: 'ETF', color: 'var(--color-info)' };
  if (type === 'BDR') return { label: 'BDR', color: 'var(--color-warning)' };
  if (type === 'STOCK' || type === 'AÇÃO' || type === 'ACAO') return { label: 'Ação', color: 'var(--color-success)' };
  if (type === 'CRYPTO') return { label: 'Crypto', color: 'var(--color-warning)' };
  return null;
};

export default function DailyReport({
  positions = [],
  fiPositions = [],
  treasuryPositions = [],
  dividends = [],
  benchmarks,
  kpiCurrency,
  lastFetchedAt,
  onRefresh,
  isRefreshing = false,
  onGoToAssets,
}: DailyReportProps) {
  const router = useRouter();
  const [sortKey, setSortKey] = useState<SortKey>('impact');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const [fetchedBenchmarks, setFetchedBenchmarks] = useState<MarketBenchmarks | null>(null);
  const [isLoadingBenchmarks, setIsLoadingBenchmarks] = useState(false);

  const fetchBenchmarksData = useCallback(() => {
    if (benchmarks !== undefined) return;
    setIsLoadingBenchmarks(true);
    apiFetch('/market/benchmarks')
      .then(res => {
        if (res.ok) return res.json();
        return null;
      })
      .then(data => {
        if (data) {
          setFetchedBenchmarks(data);
        }
      })
      .catch(() => {})
      .finally(() => {
        setIsLoadingBenchmarks(false);
      });
  }, [benchmarks]);

  useEffect(() => {
    fetchBenchmarksData();
  }, [fetchBenchmarksData]);

  const handleRefresh = async (forceRealtime = false) => {
    if (onRefresh) {
      await onRefresh(forceRealtime);
    }
    if (benchmarks === undefined) {
      fetchBenchmarksData();
    }
  };

  const activeBenchmarks = benchmarks !== undefined ? benchmarks : fetchedBenchmarks;

  const refDate = lastFetchedAt ? new Date(lastFetchedAt) : new Date();
  const lastUpdateStr = refDate.toLocaleString('pt-BR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });

  const marketStatus = getMarketStatus(refDate);

  // Proventos creditados hoje (T6)
  const todayStr = new Date().toISOString().split('T')[0];
  const todayDividends = (dividends || []).filter(d => {
    if (!d.payment_date) return false;
    const pDate = d.payment_date.split('T')[0];
    return pDate === todayStr;
  });
  const totalTodayDividends = todayDividends.reduce((acc, d) => acc + (d.net_amount || d.gross_amount || 0), 0);

  // Se não houver posições em nenhuma categoria: Empty State acionável (T4)
  const hasNoData = positions.length === 0 && fiPositions.length === 0 && treasuryPositions.length === 0;
  if (hasNoData) {
    return (
      <div className="card flex-col items-center justify-center text-center w-full" style={{ padding: '3.5rem 1.5rem', gap: '1rem' }}>
        <span style={{ fontSize: '3rem' }}>📊</span>
        <h3 className="m-0" style={{ color: 'var(--text-primary)', fontSize: '1.25rem' }}>Nenhum ativo cadastrado na carteira</h3>
        <p className="text-sm text-secondary m-0" style={{ maxWidth: '440px', lineHeight: 1.6 }}>
          Cadastre ações, fundos imobiliários, renda fixa ou títulos públicos para acompanhar a variação diária consolidada e o impacto no seu patrimônio.
        </p>
        {onGoToAssets && (
          <button
            className="primary-button font-bold mt-sm"
            onClick={onGoToAssets}
            style={{ padding: '0.6rem 1.5rem', fontSize: '0.85rem' }}
          >
            + Cadastrar Ativos na Carteira
          </button>
        )}
      </div>
    );
  }

  // Calcula total diário da carteira em kpiCurrency
  let totalDailyChange = 0;
  let totalPortfolioValue = 0;
  positions.forEach(pos => {
    const rate = getExchangeRate(pos, kpiCurrency);
    totalDailyChange += (pos.daily_change ?? 0) * (pos.quantity ?? 0) * rate;
    totalPortfolioValue += (pos.current_value ?? 0);
  });

  const totalFIPrivateValue = fiPositions.reduce((s, p) => s + (p.net_value ?? 0), 0);
  const totalTreasuryValue = treasuryPositions.reduce((s, p) => s + (p.net_value ?? 0), 0);
  totalPortfolioValue += totalFIPrivateValue + totalTreasuryValue;

  // Rendimento diário estimado para Renda Fixa e Tesouro Direto (PR-6)
  const totalFIDailyGain = fiPositions.reduce((acc, p) => {
    const rate = calculateDailyFixedIncomeRate(p.asset.indexer || p.asset.debt_type, p.asset.rate);
    return acc + calculateEstimatedDailyGain(p.net_value ?? 0, rate);
  }, 0);
  const totalTreasuryDailyGain = treasuryPositions.reduce((acc, p) => {
    const rate = calculateDailyFixedIncomeRate(p.treasury_type, p.contracted_rate ?? 0);
    return acc + calculateEstimatedDailyGain(p.net_value ?? 0, rate);
  }, 0);
  const totalEstimatedFixedIncomeGain = totalFIDailyGain + totalTreasuryDailyGain;

  // Variação % total ponderada = totalDailyChange / (valor_anterior = totalPortfolioValue - totalDailyChange)
  const previousTotalValue = totalPortfolioValue - totalDailyChange;
  const totalDailyPercent = Math.abs(previousTotalValue) > 1e-6
    ? (totalDailyChange / previousTotalValue) * 100
    : 0;

  const isDailyPos = totalDailyChange >= 0;

  // Enriquece todas as posições com campos calculados para o resumo
  const enrichedPositions = positions.map(pos => {
    const percent = pos.daily_change_percent ?? 0;
    const absChange = pos.daily_change ?? 0;
    const currentPrice = pos.current_price ?? 0;
    // Usa previous_close real do backend; caso não disponível, estima via price - change
    const previousClose = (pos.previous_close != null && pos.previous_close > 1e-6)
      ? pos.previous_close
      : currentPrice - absChange;
    const qty = pos.quantity ?? 0;
    const rate = getExchangeRate(pos, kpiCurrency);
    const impact = absChange * qty * rate;
    const portfolioWeight = totalPortfolioValue > 1e-6
      ? ((pos.current_value ?? 0) / totalPortfolioValue) * 100
      : 0;
    return { pos, percent, absChange, currentPrice, previousClose, qty, rate, impact, portfolioWeight };
  });

  // Ordenação interativa
  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const sortedRows = [...enrichedPositions].sort((a, b) => {
    let aVal: number | string = 0;
    let bVal: number | string = 0;
    switch (sortKey) {
      case 'ticker': aVal = a.pos.ticker; bVal = b.pos.ticker; break;
      case 'average_price': aVal = a.pos.average_price ?? 0; bVal = b.pos.average_price ?? 0; break;
      case 'previousClose': aVal = a.previousClose; bVal = b.previousClose; break;
      case 'current_price': aVal = a.currentPrice; bVal = b.currentPrice; break;
      case 'daily_change': aVal = a.absChange; bVal = b.absChange; break;
      case 'daily_change_percent': aVal = a.percent; bVal = b.percent; break;
      case 'impact': aVal = a.impact; bVal = b.impact; break;
    }
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortDir === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    }
    return sortDir === 'asc' ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
  });

  // Top 5 altas e baixas (por %)
  const sortedByPercent = [...enrichedPositions].sort((a, b) => b.percent - a.percent);
  const topRisers = sortedByPercent.filter(r => r.percent > 0).slice(0, 5);
  const topFallers = [...sortedByPercent].reverse().filter(r => r.percent < 0).slice(0, 5);

  // Helper para ícone de ordenação
  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return <span style={{ opacity: 0.3, marginLeft: '4px' }}>⇅</span>;
    return <span style={{ marginLeft: '4px' }}>{sortDir === 'asc' ? '↑' : '↓'}</span>;
  };

  const getAriaSort = (key: SortKey): 'ascending' | 'descending' | 'none' => {
    if (sortKey !== key) return 'none';
    return sortDir === 'asc' ? 'ascending' : 'descending';
  };

  const handleSortKeyDown = (e: React.KeyboardEvent, key: SortKey) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleSort(key);
    }
  };

  const handleNavigateKeyDown = (e: React.KeyboardEvent, ticker: string) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      router.push(`/dashboard?ticker=${encodeURIComponent(ticker)}`);
    }
  };

  return (
    <div className="flex-col gap-xl w-full">

      {/* Card principal Hero: Variação Total Diária (T1, T2, T3, T13-UX) */}
      <div className="card flex-col items-center justify-center text-center w-full" style={{ padding: '1.75rem', gap: '0.4rem', border: '1px solid var(--panel-border)' }}>
        <div className="flex-row items-center gap-sm mb-xs flex-wrap justify-center">
          <span
            style={{
              background: marketStatus.badgeBg,
              color: marketStatus.color,
              border: `1px solid ${marketStatus.color}`,
              fontSize: '0.7rem',
              padding: '0.15rem 0.55rem',
              borderRadius: '12px',
              fontWeight: 700,
              display: 'inline-flex',
              alignItems: 'center',
              gap: '4px',
            }}
            title={marketStatus.description}
          >
            ● {marketStatus.label}
          </span>
          <span className="text-secondary text-xs font-semibold" style={{ textTransform: 'uppercase', letterSpacing: '0.06em' }}>
            Variação Diária da Carteira (Intraday)
          </span>
        </div>

        <span className="text-3xl sm:text-4xl font-extrabold mt-xs" style={{ color: isDailyPos ? 'var(--color-success)' : 'var(--color-danger)', letterSpacing: '-0.02em' }}>
          {isDailyPos ? '🟢 +' : '🔴 '}{formatMoney(totalDailyChange, kpiCurrency)}
        </span>
        <span className="text-lg font-bold" style={{ color: isDailyPos ? 'var(--color-success)' : 'var(--color-danger)' }}>
          ({isDailyPos ? '+' : ''}{totalDailyPercent.toFixed(2)}%)
        </span>

        <div className="flex-row items-center gap-md mt-sm flex-wrap justify-center text-xs text-secondary" aria-live="polite">
          <span>🕐 Cotações em: <strong>{lastUpdateStr}</strong></span>
          {onRefresh && (
            <div className="flex-row items-center gap-xs">
              <button
                onClick={() => handleRefresh(false)}
                disabled={isRefreshing}
                aria-label="Recarregar cotações e resumo do portfólio"
                className="btn-secondary font-bold"
                style={{ padding: '0.25rem 0.65rem', fontSize: '0.75rem', borderRadius: '4px', cursor: isRefreshing ? 'not-allowed' : 'pointer' }}
                title="Recarregar cotações e resumo do portfólio"
              >
                {isRefreshing ? '⏳ Atualizando...' : '🔄 Atualizar'}
              </button>
              <button
                onClick={() => handleRefresh(true)}
                disabled={isRefreshing}
                aria-label="Forçar atualização de cotações em tempo real ignorando cache"
                className="btn-secondary text-xs"
                style={{ padding: '0.25rem 0.5rem', fontSize: '0.7rem', borderRadius: '4px', cursor: isRefreshing ? 'not-allowed' : 'pointer', opacity: 0.85 }}
                title="Forçar atualização de cotações em tempo real (ignora cache)"
              >
                ⚡ Tempo Real
              </button>
            </div>
          )}
        </div>
        {totalEstimatedFixedIncomeGain > 1e-6 && (
          <div className="flex-row items-center gap-sm mt-xs flex-wrap justify-center text-xs" style={{ background: 'var(--card-bg)', padding: '4px 10px', borderRadius: '6px', border: '1px solid var(--panel-border)' }}>
            <span className="text-secondary">🏛️ Rendimento Diário Est. (Renda Fixa + Tesouro):</span>
            <strong className="text-success">+{formatMoney(totalEstimatedFixedIncomeGain, kpiCurrency)}/dia</strong>
          </div>
        )}
        <span className="text-xs text-secondary mt-xs" style={{ opacity: 0.65, fontSize: '0.7rem' }}>
          💡 Cotações de renda variável possuem cache do provedor (TTL 15 min).
        </span>
      </div>

      {/* Benchmarks de Mercado (PR-5) */}
      <MarketBenchmarksBar benchmarks={activeBenchmarks} isLoading={isLoadingBenchmarks} />

      {/* Seção Proventos Creditados Hoje (T6) */}
      {todayDividends.length > 0 && (
        <div className="card flex-col gap-sm w-full" style={{ background: 'var(--color-success-bg)', borderLeft: '4px solid var(--color-success)', padding: '1.25rem' }}>
          <div className="flex-row justify-between items-center flex-wrap gap-sm">
            <div className="flex-col">
              <h4 className="m-0 text-success font-bold flex-row items-center gap-xs" style={{ fontSize: '1rem' }}>
                💰 Proventos Recebidos Hoje ({todayDividends.length})
              </h4>
              <span className="text-xs text-secondary mt-xs">Pagamentos com data de crédito prevista para hoje</span>
            </div>
            <span className="text-xl font-extrabold text-success">
              +{formatMoney(totalTodayDividends, kpiCurrency)}
            </span>
          </div>

          <div className="flex-row gap-sm flex-wrap mt-xs">
            {todayDividends.map((div, idx) => (
              <div
                key={idx}
                className="flex-row items-center gap-sm"
                style={{
                  background: 'var(--card-bg)',
                  padding: '0.4rem 0.75rem',
                  borderRadius: '6px',
                  border: '1px solid var(--panel-border)',
                }}
              >
                <span className="font-bold text-accent">{div.ticker}</span>
                <span className="text-xs text-secondary">{div.type}</span>
                <span className="text-success font-bold text-sm">
                  {formatMoney(div.net_amount || div.gross_amount || 0, kpiCurrency)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Cards: Maiores Altas e Baixas */}
      <div className="flex-row gap-lg flex-wrap">
        {/* Maiores Altas */}
        <div className="card flex-col gap-md" style={{ flex: '1 1 300px' }}>
          <h3 className="card-title text-success">🚀 Maiores Altas do Dia</h3>
          {topRisers.length > 0 ? (
            <div className="flex-col gap-sm">
              {topRisers.map(({ pos, percent, impact, portfolioWeight }) => {
                const badge = getAssetTypeBadge(pos);
                return (
                  <div
                    key={pos.asset_id}
                    role="button"
                    tabIndex={0}
                    aria-label={`Ver ${pos.ticker} no Monitoramento`}
                    className="flex-row justify-between items-center"
                    style={{ padding: '0.5rem 0.75rem', background: 'var(--color-success-bg)', borderRadius: '8px', borderLeft: '3px solid var(--color-success)', cursor: 'pointer' }}
                    onClick={() => router.push(`/dashboard?ticker=${encodeURIComponent(pos.ticker)}`)}
                    onKeyDown={(e) => handleNavigateKeyDown(e, pos.ticker)}
                    title={`Ver ${pos.ticker} no Monitoramento`}
                  >
                    <div className="flex-col" style={{ gap: '2px' }}>
                      <div className="flex-row items-center gap-sm">
                        <span className="font-bold text-accent">{pos.ticker}</span>
                        {badge && (
                          <span style={{ fontSize: '0.65rem', fontWeight: 700, color: badge.color, border: `1px solid ${badge.color}`, borderRadius: '4px', padding: '1px 5px', lineHeight: 1.4 }}>
                            {badge.label}
                          </span>
                        )}
                      </div>
                      <span className="text-xs text-secondary">{portfolioWeight.toFixed(1)}% do portfólio</span>
                    </div>
                    <div className="flex-col items-end">
                      <span className="text-success font-bold">{formatPercentage(percent)}</span>
                      <span className="text-xs text-secondary">Impacto: +{formatMoney(impact, kpiCurrency)}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <span className="text-sm text-secondary">Nenhuma alta registrada hoje.</span>
          )}
        </div>

        {/* Maiores Baixas */}
        <div className="card flex-col gap-md" style={{ flex: '1 1 300px' }}>
          <h3 className="card-title text-danger">📉 Maiores Baixas do Dia</h3>
          {topFallers.length > 0 ? (
            <div className="flex-col gap-sm">
              {topFallers.map(({ pos, percent, impact, portfolioWeight }) => {
                const badge = getAssetTypeBadge(pos);
                return (
                  <div
                    key={pos.asset_id}
                    role="button"
                    tabIndex={0}
                    aria-label={`Ver ${pos.ticker} no Monitoramento`}
                    className="flex-row justify-between items-center"
                    style={{ padding: '0.5rem 0.75rem', background: 'var(--color-danger-bg)', borderRadius: '8px', borderLeft: '3px solid var(--color-danger)', cursor: 'pointer' }}
                    onClick={() => router.push(`/dashboard?ticker=${encodeURIComponent(pos.ticker)}`)}
                    onKeyDown={(e) => handleNavigateKeyDown(e, pos.ticker)}
                    title={`Ver ${pos.ticker} no Monitoramento`}
                  >
                    <div className="flex-col" style={{ gap: '2px' }}>
                      <div className="flex-row items-center gap-sm">
                        <span className="font-bold text-accent">{pos.ticker}</span>
                        {badge && (
                          <span style={{ fontSize: '0.65rem', fontWeight: 700, color: badge.color, border: `1px solid ${badge.color}`, borderRadius: '4px', padding: '1px 5px', lineHeight: 1.4 }}>
                            {badge.label}
                          </span>
                        )}
                      </div>
                      <span className="text-xs text-secondary">{portfolioWeight.toFixed(1)}% do portfólio</span>
                    </div>
                    <div className="flex-col items-end">
                      <span className="text-danger font-bold">{formatPercentage(percent)}</span>
                      <span className="text-xs text-secondary">Impacto: {formatMoney(impact, kpiCurrency)}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <span className="text-sm text-secondary">Nenhuma baixa registrada hoje.</span>
          )}
        </div>
      </div>

      {/* Tabela: Resumo Completo com Linhas Clicáveis e Scroll Horizontal Seguro (T5, T30) */}
      <div className="card flex-col gap-md w-full">
        <div className="flex-row justify-between items-center flex-wrap gap-sm">
          <div>
            <h3 className="card-title m-0">📊 Resumo Diário Completo (Renda Variável)</h3>
            <span className="text-xs text-secondary">Clique em um ativo para abrir no Monitoramento</span>
          </div>
          <button
            onClick={() => exportDailyReportCSV(positions, fiPositions, treasuryPositions, kpiCurrency)}
            aria-label="Exportar resumo diário completo em arquivo CSV"
            className="btn-secondary flex-row items-center gap-xs text-xs font-semibold"
            style={{ padding: '0.35rem 0.75rem', borderRadius: '6px' }}
            title="Exportar resumo diário completo em arquivo CSV"
          >
            <span>📥</span> Exportar CSV
          </button>
        </div>
        <div className="table-container flex-col" style={{ overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
          {positions.length > 0 ? (
            <table className="data-table" style={{ width: '100%', minWidth: '650px' }} aria-label="Resumo diário completo de renda variável">
              <caption className="sr-only">Resumo diário de ativos de renda variável</caption>
              <thead>
                <tr>
                  <th
                    scope="col"
                    role="columnheader"
                    tabIndex={0}
                    aria-sort={getAriaSort('ticker')}
                    aria-label="Ordenar por ativo"
                    style={{ cursor: 'pointer' }}
                    onClick={() => handleSort('ticker')}
                    onKeyDown={(e) => handleSortKeyDown(e, 'ticker')}
                  >
                    Ativo {sortIcon('ticker')}
                  </th>
                  <th
                    scope="col"
                    role="columnheader"
                    tabIndex={0}
                    aria-sort={getAriaSort('average_price')}
                    aria-label="Ordenar por preço médio"
                    className="text-right"
                    style={{ cursor: 'pointer' }}
                    onClick={() => handleSort('average_price')}
                    onKeyDown={(e) => handleSortKeyDown(e, 'average_price')}
                  >
                    Preço Médio {sortIcon('average_price')}
                  </th>
                  <th
                    scope="col"
                    role="columnheader"
                    tabIndex={0}
                    aria-sort={getAriaSort('previousClose')}
                    aria-label="Ordenar por fechamento anterior"
                    className="text-right"
                    style={{ cursor: 'pointer' }}
                    onClick={() => handleSort('previousClose')}
                    onKeyDown={(e) => handleSortKeyDown(e, 'previousClose')}
                  >
                    Fech. Anterior {sortIcon('previousClose')}
                  </th>
                  <th
                    scope="col"
                    role="columnheader"
                    tabIndex={0}
                    aria-sort={getAriaSort('current_price')}
                    aria-label="Ordenar por cotação atual"
                    className="text-right"
                    style={{ cursor: 'pointer' }}
                    onClick={() => handleSort('current_price')}
                    onKeyDown={(e) => handleSortKeyDown(e, 'current_price')}
                  >
                    Cotação Atual {sortIcon('current_price')}
                  </th>
                  <th
                    scope="col"
                    role="columnheader"
                    tabIndex={0}
                    aria-sort={getAriaSort('daily_change')}
                    aria-label="Ordenar por variação nominal por cota"
                    className="text-right"
                    style={{ cursor: 'pointer' }}
                    onClick={() => handleSort('daily_change')}
                    onKeyDown={(e) => handleSortKeyDown(e, 'daily_change')}
                  >
                    Var./Cota {sortIcon('daily_change')}
                  </th>
                  <th
                    scope="col"
                    role="columnheader"
                    tabIndex={0}
                    aria-sort={getAriaSort('daily_change_percent')}
                    aria-label="Ordenar por variação percentual"
                    className="text-right"
                    style={{ cursor: 'pointer' }}
                    onClick={() => handleSort('daily_change_percent')}
                    onKeyDown={(e) => handleSortKeyDown(e, 'daily_change_percent')}
                  >
                    Var. % {sortIcon('daily_change_percent')}
                  </th>
                  <th
                    scope="col"
                    role="columnheader"
                    tabIndex={0}
                    aria-sort={getAriaSort('impact')}
                    aria-label="Ordenar por impacto diário"
                    className="text-right"
                    style={{ cursor: 'pointer' }}
                    onClick={() => handleSort('impact')}
                    onKeyDown={(e) => handleSortKeyDown(e, 'impact')}
                  >
                    Impacto Diário {sortIcon('impact')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {sortedRows.map(({ pos, percent, absChange, currentPrice, previousClose, impact, portfolioWeight }) => {
                  const hasQuote = currentPrice > 1e-6;
                  const isUp = percent > 0;
                  const isDown = percent < 0;
                  const colorClass = isUp ? 'text-success' : isDown ? 'text-danger' : 'text-secondary';
                  const prefix = isUp ? '+' : '';
                  const avgPrice = pos.average_price ?? 0;
                  const prevCloseColor = previousClose >= avgPrice ? 'text-success' : 'text-danger';
                  const currentPriceColor = currentPrice >= avgPrice ? 'text-success' : 'text-danger';
                  const badge = getAssetTypeBadge(pos);
                  const isUSD = pos.currency?.toUpperCase() === 'USD' || pos.type === 'STOCK_US' || pos.type === 'ETF_US';

                  return (
                    <tr
                      key={pos.asset_id}
                      tabIndex={0}
                      aria-label={`Ver ${pos.ticker} no Monitoramento`}
                      onClick={() => router.push(`/dashboard?ticker=${encodeURIComponent(pos.ticker)}`)}
                      onKeyDown={(e) => handleNavigateKeyDown(e, pos.ticker)}
                      style={{ cursor: 'pointer', transition: 'background-color 0.15s ease' }}
                      title={`Clique para ver ${pos.ticker} no Monitoramento de Cotações`}
                    >
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <div className="flex-row items-center gap-sm">
                          <span className="font-bold text-accent">{pos.ticker}</span>
                          <span className="text-xs text-secondary" style={{ opacity: 0.6 }}>↗</span>
                          {badge && (
                            <span style={{ fontSize: '0.65rem', fontWeight: 700, color: badge.color, border: `1px solid ${badge.color}`, borderRadius: '4px', padding: '1px 5px', lineHeight: 1.4 }}>
                              {badge.label}
                            </span>
                          )}
                          {isUSD && kpiCurrency === 'BRL' && pos.fx_rate_to_brl && (
                            <span className="text-xs text-secondary font-mono" style={{ fontSize: '0.65rem', background: 'var(--input-bg)', padding: '1px 4px', borderRadius: '3px' }} title={`Câmbio: US$ 1 = R$ ${pos.fx_rate_to_brl.toFixed(2)}`}>
                              PTAX R$ {pos.fx_rate_to_brl.toFixed(2)}
                            </span>
                          )}
                        </div>
                        <div style={{ width: '100%', background: 'var(--input-bg)', borderRadius: '3px', height: '3px', marginTop: '4px' }}>
                          <div style={{ width: `${Math.min(portfolioWeight, 100)}%`, background: 'var(--accent-color)', borderRadius: '3px', height: '3px' }} />
                        </div>
                        <span className="text-xs text-secondary">{portfolioWeight.toFixed(1)}%</span>
                      </td>
                      <td className="text-right" style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                        {formatMoney(pos.average_price ?? 0, pos.currency)}
                      </td>
                      <td className={`text-right ${prevCloseColor}`} style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                        {formatMoney(previousClose, pos.currency)}
                      </td>
                      <td className={`text-right ${currentPriceColor}`} style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                        {hasQuote ? (
                          formatMoney(currentPrice, pos.currency)
                        ) : (
                          <span className="text-xs text-secondary font-semibold" title="Ativo sem cotação recente disponível">
                            ⚠️ Sem cotação
                          </span>
                        )}
                      </td>
                      <td className={`text-right ${colorClass}`} style={{ whiteSpace: 'nowrap' }}>
                        {hasQuote ? `${prefix}${formatMoney(absChange, pos.currency)}` : '-'}
                      </td>
                      <td className={`text-right font-bold ${colorClass}`} style={{ whiteSpace: 'nowrap' }}>
                        {hasQuote ? formatPercentage(percent) : '-'}
                      </td>
                      <td className={`text-right font-bold ${colorClass}`} style={{ whiteSpace: 'nowrap' }}>
                        {hasQuote ? `${prefix}${formatMoney(impact, kpiCurrency)}` : '-'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <div className="flex-col items-center justify-center py-xl text-secondary" style={{ gap: '0.75rem' }}>
              <span style={{ fontSize: '2.5rem' }}>📭</span>
              <span className="font-semibold">Nenhuma posição em renda variável ativa.</span>
            </div>
          )}
        </div>
      </div>

      {/* Renda Fixa Privada (CDB, LCI, LCA, Debêntures) */}
      {fiPositions.length > 0 && (
        <div className="card flex-col gap-md w-full">
          <h3 className="card-title">🏛️ Posição Atualizada: Renda Fixa Privada (CDB/LCI/LCA)</h3>
          <p className="text-xs text-secondary">
            Títulos de renda fixa privada são atualizados diariamente com a rentabilidade acumulada de acordo com o indexador contratado.
          </p>
          <div className="table-container flex-col" style={{ overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
            <table className="data-table" style={{ width: '100%', minWidth: '600px' }} aria-label="Posição atualizada de renda fixa privada">
              <caption className="sr-only">Posição atualizada de renda fixa privada</caption>
              <thead>
                <tr>
                  <th scope="col">Instituição & Ativo</th>
                  <th scope="col">Tipo / Taxa</th>
                  <th scope="col" className="text-right">Vencimento</th>
                  <th scope="col" className="text-right">Valor Líquido</th>
                  <th scope="col" className="text-right">Taxa Diária Est.</th>
                  <th scope="col" className="text-right">Ganho Diário Est.</th>
                  <th scope="col" className="text-right">Rent. Acumulada</th>
                </tr>
              </thead>
              <tbody>
                {fiPositions.map(p => {
                  const returnPct = p.net_return_percent ?? 0;
                  const taxa = p.asset.debt_type === 'POS'
                    ? `${p.asset.rate.toFixed(2)}% ${p.asset.indexer}`
                    : `${p.asset.rate.toFixed(2)}% a.a.`;
                  const dailyRatePct = calculateDailyFixedIncomeRate(p.asset.indexer || p.asset.debt_type, p.asset.rate);
                  const dailyGain = calculateEstimatedDailyGain(p.net_value ?? 0, dailyRatePct);

                  return (
                    <tr key={p.asset.id}>
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <div className="flex-col">
                          <span className="font-bold">{p.asset.institution}</span>
                          <span className="text-xs text-secondary">{p.asset.type}</span>
                        </div>
                      </td>
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <div className="flex-row items-center gap-sm">
                          <span style={{ background: 'var(--accent-bg)', color: 'var(--accent-color)', border: '1px solid rgba(var(--accent-rgb), 0.3)', fontSize: '0.65rem', padding: '2px 6px', borderRadius: '4px', fontWeight: 700 }}>
                            {p.asset.type}
                          </span>
                          <span className="text-xs">{taxa}</span>
                        </div>
                      </td>
                      <td className="text-right" style={{ whiteSpace: 'nowrap' }}>{new Date(p.asset.maturity_date).toLocaleDateString('pt-BR')}</td>
                      <td className="text-right" style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{formatMoney(p.net_value, kpiCurrency)}</td>
                      <td className="text-right text-xs text-secondary font-semibold" style={{ whiteSpace: 'nowrap' }}>
                        +{dailyRatePct.toFixed(4)}%/dia
                      </td>
                      <td className="text-right font-bold text-success" style={{ whiteSpace: 'nowrap' }}>
                        +{formatMoney(dailyGain, kpiCurrency)}
                      </td>
                      <td className={`text-right font-bold ${returnPct >= 0 ? 'text-success' : 'text-danger'}`} style={{ whiteSpace: 'nowrap' }}>
                        {returnPct >= 0 ? '+' : ''}{returnPct.toFixed(2)}%
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Tesouro Direto */}
      {treasuryPositions.length > 0 && (
        <div className="card flex-col gap-md w-full">
          <h3 className="card-title">🏛️ Posição Atualizada: Tesouro Direto</h3>
          <p className="text-xs text-secondary">
            Títulos do Tesouro Nacional não possuem cotação intraday. Os valores abaixo
            representam a última posição de liquidação líquida disponível.
          </p>
          <div className="table-container flex-col" style={{ overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
            <table className="data-table" style={{ width: '100%', minWidth: '600px' }} aria-label="Posição atualizada de títulos do tesouro direto">
              <caption className="sr-only">Posição atualizada do tesouro direto</caption>
              <thead>
                <tr>
                  <th scope="col">Título</th>
                  <th scope="col">Tipo</th>
                  <th scope="col" className="text-right">Vencimento</th>
                  <th scope="col" className="text-right">Valor Líquido</th>
                  <th scope="col" className="text-right">Taxa Diária Est.</th>
                  <th scope="col" className="text-right">Ganho Diário Est.</th>
                  <th scope="col" className="text-right">Rent. Acumulada</th>
                </tr>
              </thead>
              <tbody>
                {treasuryPositions.map(p => {
                  const returnPct = p.total_invested > 1e-6
                    ? ((p.net_value - p.total_invested) / p.total_invested) * 100
                    : 0;
                  const isSelic = p.treasury_type === 'SELIC';
                  const isPrefix = p.treasury_type === 'PREFIXADO';
                  const colorVar = isSelic ? 'var(--color-success)' : isPrefix ? 'var(--accent-color)' : 'var(--color-warning)';
                  const bgVar = isSelic ? 'var(--color-success-bg)' : isPrefix ? 'var(--accent-bg)' : 'var(--color-warning-bg)';
                  const dailyRatePct = calculateDailyFixedIncomeRate(p.treasury_type, p.contracted_rate ?? 0);
                  const dailyGain = calculateEstimatedDailyGain(p.net_value ?? 0, dailyRatePct);

                  return (
                    <tr key={p.transaction_id}>
                      <td style={{ whiteSpace: 'nowrap' }}><span className="font-bold">{p.ticker}</span></td>
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <span style={{ background: bgVar, color: colorVar, border: `1px solid ${colorVar}`, fontSize: '0.65rem', padding: '2px 6px', borderRadius: '4px', fontWeight: 700 }}>
                          {p.treasury_type}
                        </span>
                      </td>
                      <td className="text-right" style={{ whiteSpace: 'nowrap' }}>{new Date(p.maturity_date).toLocaleDateString('pt-BR')}</td>
                      <td className="text-right" style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{formatMoney(p.net_value, kpiCurrency)}</td>
                      <td className="text-right text-xs text-secondary font-semibold" style={{ whiteSpace: 'nowrap' }}>
                        +{dailyRatePct.toFixed(4)}%/dia
                      </td>
                      <td className="text-right font-bold text-success" style={{ whiteSpace: 'nowrap' }}>
                        +{formatMoney(dailyGain, kpiCurrency)}
                      </td>
                      <td className={`text-right font-bold ${returnPct >= 0 ? 'text-success' : 'text-danger'}`} style={{ whiteSpace: 'nowrap' }}>
                        {returnPct >= 0 ? '+' : ''}{returnPct.toFixed(2)}%
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
