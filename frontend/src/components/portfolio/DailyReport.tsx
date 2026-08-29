import React, { useState, useEffect, useCallback } from 'react';
import { Position, FixedIncomePosition, TreasuryPosition, CalculatedDividend, MarketBenchmarks } from './types';
import { getMarketStatus } from '@/lib/marketHours';
import { apiFetch } from '@/lib/api';
import MarketBenchmarksBar from './MarketBenchmarksBar';
import { useDailyReportCalculations } from './daily-report/useDailyReportCalculations';
import DailyReportEmptyState from './daily-report/DailyReportEmptyState';
import DailyReportHero from './daily-report/DailyReportHero';
import DailyReportDividendsToday from './daily-report/DailyReportDividendsToday';
import DailyReportTopMovers from './daily-report/DailyReportTopMovers';
import DailyReportTable from './daily-report/DailyReportTable';
import DailyReportFixedIncome from './daily-report/DailyReportFixedIncome';

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

  const calculations = useDailyReportCalculations(
    positions,
    fiPositions,
    treasuryPositions,
    dividends,
    kpiCurrency
  );

  const hasNoData = positions.length === 0 && fiPositions.length === 0 && treasuryPositions.length === 0;
  if (hasNoData) {
    return <DailyReportEmptyState onGoToAssets={onGoToAssets} />;
  }

  return (
    <div className="flex-col gap-xl w-full">
      {/* Card principal Hero: Variação Total Diária */}
      <DailyReportHero
        marketStatus={marketStatus}
        totalDailyChange={calculations.totalDailyChange}
        totalDailyPercent={calculations.totalDailyPercent}
        isDailyPos={calculations.isDailyPos}
        kpiCurrency={kpiCurrency}
        lastUpdateStr={lastUpdateStr}
        isRefreshing={isRefreshing}
        onRefresh={onRefresh ? handleRefresh : undefined}
        totalEstimatedFixedIncomeGain={calculations.totalEstimatedFixedIncomeGain}
      />

      {/* Benchmarks de Mercado */}
      <MarketBenchmarksBar benchmarks={activeBenchmarks} isLoading={isLoadingBenchmarks} />

      {/* Seção Proventos Creditados Hoje */}
      <DailyReportDividendsToday
        todayDividends={calculations.todayDividends}
        totalTodayDividends={calculations.totalTodayDividends}
        kpiCurrency={kpiCurrency}
      />

      {/* Cards: Maiores Altas e Baixas */}
      <DailyReportTopMovers
        topRisers={calculations.topRisers}
        topFallers={calculations.topFallers}
        kpiCurrency={kpiCurrency}
        getAssetTypeBadge={calculations.getAssetTypeBadge}
      />

      {/* Tabela: Resumo Completo de Renda Variável */}
      <DailyReportTable
        positions={positions}
        fiPositions={fiPositions}
        treasuryPositions={treasuryPositions}
        sortedRows={calculations.sortedRows}
        sortKey={calculations.sortKey}
        sortDir={calculations.sortDir}
        onSort={calculations.handleSort}
        kpiCurrency={kpiCurrency}
        getAssetTypeBadge={calculations.getAssetTypeBadge}
      />

      {/* Renda Fixa Privada e Tesouro Direto */}
      <DailyReportFixedIncome
        fiPositions={fiPositions}
        treasuryPositions={treasuryPositions}
        kpiCurrency={kpiCurrency}
      />
    </div>
  );
}
