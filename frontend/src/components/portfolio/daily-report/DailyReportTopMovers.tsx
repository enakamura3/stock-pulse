import React from 'react';
import { useRouter } from 'next/navigation';
import { formatMoney, formatPercentage } from '../helpers';
import { EnrichedPosition, AssetTypeBadge } from './types';
import { Position } from '../types';

export interface DailyReportTopMoversProps {
  topRisers: EnrichedPosition[];
  topFallers: EnrichedPosition[];
  kpiCurrency: string;
  getAssetTypeBadge: (pos: Position) => AssetTypeBadge | null;
}

export default function DailyReportTopMovers({
  topRisers,
  topFallers,
  kpiCurrency,
  getAssetTypeBadge,
}: DailyReportTopMoversProps) {
  const router = useRouter();

  const handleNavigate = (ticker: string) => {
    router.push(`/dashboard?ticker=${encodeURIComponent(ticker)}`);
  };

  const handleNavigateKeyDown = (e: React.KeyboardEvent, ticker: string) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleNavigate(ticker);
    }
  };

  return (
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
                  style={{
                    padding: '0.5rem 0.75rem',
                    background: 'var(--color-success-bg)',
                    borderRadius: '8px',
                    borderLeft: '3px solid var(--color-success)',
                    cursor: 'pointer',
                  }}
                  onClick={() => handleNavigate(pos.ticker)}
                  onKeyDown={(e) => handleNavigateKeyDown(e, pos.ticker)}
                  title={`Ver ${pos.ticker} no Monitoramento`}
                >
                  <div className="flex-col" style={{ gap: '2px' }}>
                    <div className="flex-row items-center gap-sm">
                      <span className="font-bold text-accent">{pos.ticker}</span>
                      {badge && (
                        <span
                          style={{
                            fontSize: '0.65rem',
                            fontWeight: 700,
                            color: badge.color,
                            border: `1px solid ${badge.color}`,
                            borderRadius: '4px',
                            padding: '1px 5px',
                            lineHeight: 1.4,
                          }}
                        >
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
                  style={{
                    padding: '0.5rem 0.75rem',
                    background: 'var(--color-danger-bg)',
                    borderRadius: '8px',
                    borderLeft: '3px solid var(--color-danger)',
                    cursor: 'pointer',
                  }}
                  onClick={() => handleNavigate(pos.ticker)}
                  onKeyDown={(e) => handleNavigateKeyDown(e, pos.ticker)}
                  title={`Ver ${pos.ticker} no Monitoramento`}
                >
                  <div className="flex-col" style={{ gap: '2px' }}>
                    <div className="flex-row items-center gap-sm">
                      <span className="font-bold text-accent">{pos.ticker}</span>
                      {badge && (
                        <span
                          style={{
                            fontSize: '0.65rem',
                            fontWeight: 700,
                            color: badge.color,
                            border: `1px solid ${badge.color}`,
                            borderRadius: '4px',
                            padding: '1px 5px',
                            lineHeight: 1.4,
                          }}
                        >
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
  );
}
