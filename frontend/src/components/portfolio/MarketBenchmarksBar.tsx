import React from 'react';
import { MarketBenchmarks, BenchmarkItem } from './types';

export interface MarketBenchmarksBarProps {
  benchmarks?: MarketBenchmarks | null;
  isLoading?: boolean;
}

function formatBenchmarkValue(item: BenchmarkItem): string {
  if (!item || typeof item.value !== 'number') return '-';
  if (item.symbol === 'BRL=X') {
    return `R$ ${item.value.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  }
  return `${item.value.toLocaleString('pt-BR', { minimumFractionDigits: 0, maximumFractionDigits: 2 })} pts`;
}

function BenchmarkCard({ item, label, icon }: { item?: BenchmarkItem; label: string; icon: string }) {
  if (!item) return null;
  const isPos = item.change_percent >= 0;
  const colorClass = isPos ? 'text-success' : 'text-danger';
  const bgClass = isPos ? 'var(--color-success-bg)' : 'var(--color-danger-bg)';
  const borderClass = isPos ? 'var(--color-success)' : 'var(--color-danger)';

  return (
    <div
      className="flex-1 min-w-[140px] sm:min-w-[180px] p-sm rounded-lg flex-col gap-xs"
      style={{
        background: 'var(--card-bg)',
        border: '1px solid var(--panel-border)',
        borderRadius: '8px',
        padding: '0.65rem 0.9rem',
      }}
    >
      <div className="flex-row items-center justify-between gap-xs">
        <span className="text-xs font-semibold text-secondary flex-row items-center gap-xs">
          <span>{icon}</span> {label}
        </span>
        <span
          style={{
            fontSize: '0.72rem',
            fontWeight: 700,
            padding: '1px 6px',
            borderRadius: '12px',
            background: bgClass,
            color: borderClass,
            border: `1px solid ${borderClass}`,
            display: 'inline-flex',
            alignItems: 'center',
          }}
        >
          {isPos ? '+' : ''}{item.change_percent.toFixed(2)}%
        </span>
      </div>
      <div className="flex-row items-baseline justify-between mt-xs">
        <span className="font-bold text-sm sm:text-base" style={{ color: 'var(--text-primary)' }}>
          {formatBenchmarkValue(item)}
        </span>
        <span className={`text-xs font-semibold ${colorClass}`}>
          {isPos ? '▲' : '▼'} {Math.abs(item.change).toFixed(2)}
        </span>
      </div>
    </div>
  );
}

export default function MarketBenchmarksBar({ benchmarks, isLoading = false }: MarketBenchmarksBarProps) {
  if (isLoading && !benchmarks) {
    return (
      <div
        className="w-full flex-row gap-md flex-wrap items-center justify-between"
        style={{ opacity: 0.6 }}
      >
        <div className="flex-1 min-w-[160px] p-sm rounded-lg" style={{ background: 'var(--card-bg)', height: '64px', border: '1px solid var(--panel-border)' }} />
        <div className="flex-1 min-w-[160px] p-sm rounded-lg" style={{ background: 'var(--card-bg)', height: '64px', border: '1px solid var(--panel-border)' }} />
        <div className="flex-1 min-w-[160px] p-sm rounded-lg" style={{ background: 'var(--card-bg)', height: '64px', border: '1px solid var(--panel-border)' }} />
        <div className="flex-1 min-w-[160px] p-sm rounded-lg" style={{ background: 'var(--card-bg)', height: '64px', border: '1px solid var(--panel-border)' }} />
      </div>
    );
  }

  if (!benchmarks || (!benchmarks.ibov && !benchmarks.sp500 && !benchmarks.usd_brl && !benchmarks.ifix)) {
    return null;
  }

  return (
    <div className="flex-col gap-xs w-full">
      <div className="flex-row justify-between items-center px-xs">
        <span className="text-xs font-semibold text-secondary" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          🌐 Índices e Benchmarks de Mercado
        </span>
      </div>
      <div className="flex-row gap-md flex-wrap w-full items-stretch">
        <BenchmarkCard item={benchmarks.ibov} label="Ibovespa" icon="🇧🇷" />
        <BenchmarkCard item={benchmarks.sp500} label="S&P 500" icon="🇺🇸" />
        <BenchmarkCard item={benchmarks.usd_brl} label="Dólar" icon="💵" />
        <BenchmarkCard item={benchmarks.ifix} label="IFIX" icon="🏢" />
      </div>
    </div>
  );
}
