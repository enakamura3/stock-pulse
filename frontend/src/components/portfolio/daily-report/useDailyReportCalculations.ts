import { useState, useMemo } from 'react';
import { Position, FixedIncomePosition, TreasuryPosition, CalculatedDividend } from '../types';
import { calculateDailyFixedIncomeRate, calculateEstimatedDailyGain } from '../helpers';
import { SortKey, SortDir, EnrichedPosition, AssetTypeBadge, DailyReportCalculations } from './types';

// Helper: calcula a taxa de câmbio de uma posição para a moeda base
export function getExchangeRate(pos: Position, kpiCurrency: string = 'BRL'): number {
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
export function getAssetTypeBadge(pos: Position): AssetTypeBadge | null {
  const type = (pos.type || '').toUpperCase();
  if (type === 'FII') return { label: 'FII', color: 'var(--color-warning)' };
  if (type === 'ETF') return { label: 'ETF', color: 'var(--color-info)' };
  if (type === 'BDR') return { label: 'BDR', color: 'var(--color-warning)' };
  if (type === 'STOCK' || type === 'AÇÃO' || type === 'ACAO') return { label: 'Ação', color: 'var(--color-success)' };
  if (type === 'CRYPTO') return { label: 'Crypto', color: 'var(--color-warning)' };
  return null;
}

export function useDailyReportCalculations(
  positions: Position[] = [],
  fiPositions: FixedIncomePosition[] = [],
  treasuryPositions: TreasuryPosition[] = [],
  dividends: CalculatedDividend[] = [],
  kpiCurrency: string = 'BRL'
): DailyReportCalculations {
  const [sortKey, setSortKey] = useState<SortKey>('impact');
  const [sortDir, setSortDir] = useState<SortDir>('desc');

  // Proventos creditados hoje (T6)
  const todayStr = new Date().toISOString().split('T')[0];
  const todayDividends = useMemo(() => {
    return (dividends || []).filter(d => {
      if (!d.payment_date) return false;
      const pDate = d.payment_date.split('T')[0];
      return pDate === todayStr;
    });
  }, [dividends, todayStr]);

  const totalTodayDividends = useMemo(() => {
    return todayDividends.reduce((acc, d) => acc + (d.net_amount || d.gross_amount || 0), 0);
  }, [todayDividends]);

  // Totais diários da carteira
  const { totalDailyChange, totalPortfolioValue, totalEstimatedFixedIncomeGain } = useMemo(() => {
    let dailyChange = 0;
    let portfolioValue = 0;

    positions.forEach(pos => {
      const rate = getExchangeRate(pos, kpiCurrency);
      dailyChange += (pos.daily_change ?? 0) * (pos.quantity ?? 0) * rate;
      portfolioValue += (pos.current_value ?? 0);
    });

    const totalFIPrivateValue = fiPositions.reduce((s, p) => s + (p.net_value ?? 0), 0);
    const totalTreasuryValue = treasuryPositions.reduce((s, p) => s + (p.net_value ?? 0), 0);
    portfolioValue += totalFIPrivateValue + totalTreasuryValue;

    // Rendimento diário estimado para Renda Fixa e Tesouro Direto
    const totalFIDailyGain = fiPositions.reduce((acc, p) => {
      const rate = calculateDailyFixedIncomeRate(p.asset.indexer || p.asset.debt_type, p.asset.rate);
      return acc + calculateEstimatedDailyGain(p.net_value ?? 0, rate);
    }, 0);
    const totalTreasuryDailyGain = treasuryPositions.reduce((acc, p) => {
      const rate = calculateDailyFixedIncomeRate(p.treasury_type, p.contracted_rate ?? 0);
      return acc + calculateEstimatedDailyGain(p.net_value ?? 0, rate);
    }, 0);

    return {
      totalDailyChange: dailyChange,
      totalPortfolioValue: portfolioValue,
      totalEstimatedFixedIncomeGain: totalFIDailyGain + totalTreasuryDailyGain,
    };
  }, [positions, fiPositions, treasuryPositions, kpiCurrency]);

  const previousTotalValue = totalPortfolioValue - totalDailyChange;
  const totalDailyPercent = Math.abs(previousTotalValue) > 1e-6
    ? (totalDailyChange / previousTotalValue) * 100
    : 0;

  const isDailyPos = totalDailyChange >= 0;

  // Enriquece todas as posições com campos calculados para o resumo
  const enrichedPositions: EnrichedPosition[] = useMemo(() => {
    return positions.map(pos => {
      const percent = pos.daily_change_percent ?? 0;
      const absChange = pos.daily_change ?? 0;
      const currentPrice = pos.current_price ?? 0;
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
  }, [positions, kpiCurrency, totalPortfolioValue]);

  // Ordenação interativa
  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const sortedRows = useMemo(() => {
    return [...enrichedPositions].sort((a, b) => {
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
  }, [enrichedPositions, sortKey, sortDir]);

  // Top 5 altas e baixas (por %)
  const { topRisers, topFallers } = useMemo(() => {
    const sortedByPercent = [...enrichedPositions].sort((a, b) => b.percent - a.percent);
    return {
      topRisers: sortedByPercent.filter(r => r.percent > 0).slice(0, 5),
      topFallers: [...sortedByPercent].reverse().filter(r => r.percent < 0).slice(0, 5),
    };
  }, [enrichedPositions]);

  return {
    totalDailyChange,
    totalPortfolioValue,
    totalDailyPercent,
    isDailyPos,
    totalEstimatedFixedIncomeGain,
    todayDividends,
    totalTodayDividends,
    enrichedPositions,
    sortedRows,
    topRisers,
    topFallers,
    sortKey,
    sortDir,
    handleSort,
    getAssetTypeBadge,
  };
}
