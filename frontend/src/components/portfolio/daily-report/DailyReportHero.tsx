import React from 'react';
import { formatMoney } from '../helpers';
import { MarketStatus } from '@/lib/marketHours';

export interface DailyReportHeroProps {
  marketStatus: MarketStatus;
  totalDailyChange: number;
  totalDailyPercent: number;
  isDailyPos: boolean;
  kpiCurrency: string;
  lastUpdateStr: string;
  isRefreshing?: boolean;
  onRefresh?: (forceRealtime?: boolean) => Promise<void> | void;
  totalEstimatedFixedIncomeGain: number;
}

export default function DailyReportHero({
  marketStatus,
  totalDailyChange,
  totalDailyPercent,
  isDailyPos,
  kpiCurrency,
  lastUpdateStr,
  isRefreshing = false,
  onRefresh,
  totalEstimatedFixedIncomeGain,
}: DailyReportHeroProps) {
  return (
    <div
      className="card flex-col items-center justify-center text-center w-full"
      style={{ padding: '1.75rem', gap: '0.4rem', border: '1px solid var(--panel-border)' }}
    >
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
        <span
          className="text-secondary text-xs font-semibold"
          style={{ textTransform: 'uppercase', letterSpacing: '0.06em' }}
        >
          Variação Diária da Carteira (Intraday)
        </span>
      </div>

      <span
        className="text-3xl sm:text-4xl font-extrabold mt-xs"
        style={{
          color: isDailyPos ? 'var(--color-success)' : 'var(--color-danger)',
          letterSpacing: '-0.02em',
        }}
      >
        {isDailyPos ? '🟢 +' : '🔴 '}{formatMoney(totalDailyChange, kpiCurrency)}
      </span>
      <span
        className="text-lg font-bold"
        style={{ color: isDailyPos ? 'var(--color-success)' : 'var(--color-danger)' }}
      >
        ({isDailyPos ? '+' : ''}{totalDailyPercent.toFixed(2)}%)
      </span>

      <div
        className="flex-row items-center gap-md mt-sm flex-wrap justify-center text-xs text-secondary"
        aria-live="polite"
      >
        <span>🕐 Cotações em: <strong>{lastUpdateStr}</strong></span>
        {onRefresh && (
          <div className="flex-row items-center gap-xs">
            <button
              onClick={() => onRefresh(false)}
              disabled={isRefreshing}
              aria-label="Recarregar cotações e resumo do portfólio"
              className="btn-secondary font-bold"
              style={{
                padding: '0.25rem 0.65rem',
                fontSize: '0.75rem',
                borderRadius: '4px',
                cursor: isRefreshing ? 'not-allowed' : 'pointer',
              }}
              title="Recarregar cotações e resumo do portfólio"
            >
              {isRefreshing ? '⏳ Atualizando...' : '🔄 Atualizar'}
            </button>
            <button
              onClick={() => onRefresh(true)}
              disabled={isRefreshing}
              aria-label="Forçar atualização de cotações em tempo real ignorando cache"
              className="btn-secondary text-xs"
              style={{
                padding: '0.25rem 0.5rem',
                fontSize: '0.7rem',
                borderRadius: '4px',
                cursor: isRefreshing ? 'not-allowed' : 'pointer',
                opacity: 0.85,
              }}
              title="Forçar atualização de cotações em tempo real (ignora cache)"
            >
              ⚡ Tempo Real
            </button>
          </div>
        )}
      </div>
      {totalEstimatedFixedIncomeGain > 1e-6 && (
        <div
          className="flex-row items-center gap-sm mt-xs flex-wrap justify-center text-xs"
          style={{
            background: 'var(--card-bg)',
            padding: '4px 10px',
            borderRadius: '6px',
            border: '1px solid var(--panel-border)',
          }}
        >
          <span className="text-secondary">🏛️ Rendimento Diário Est. (Renda Fixa + Tesouro):</span>
          <strong className="text-success">+{formatMoney(totalEstimatedFixedIncomeGain, kpiCurrency)}/dia</strong>
        </div>
      )}
      <span className="text-xs text-secondary mt-xs" style={{ opacity: 0.65, fontSize: '0.7rem' }}>
        💡 Cotações de renda variável possuem cache do provedor (TTL 15 min).
      </span>
    </div>
  );
}
