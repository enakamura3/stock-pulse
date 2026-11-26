import React from 'react';
import Link from 'next/link';
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
                  className="flex-row justify-between items-center"
                  style={{
                    padding: '0.5rem 0.75rem',
                    background: 'var(--color-success-bg)',
                    borderRadius: '8px',
                    borderLeft: '3px solid var(--color-success)',
                  }}
                >
                  <div className="flex-col" style={{ gap: '2px' }}>
                    <div className="flex-row items-center gap-sm">
                      <Link
                        href={`/dashboard?ticker=${encodeURIComponent(pos.ticker)}`}
                        aria-label={`Ver ${pos.ticker} no Monitoramento`}
                        title={`Ver ${pos.ticker} no Monitoramento`}
                        className="flex-row items-center gap-xs font-bold text-accent"
                        style={{ textDecoration: 'none', cursor: 'pointer' }}
                      >
                        <span>{pos.ticker}</span>
                        <span className="text-xs text-secondary" style={{ opacity: 0.7 }}>↗</span>
                      </Link>
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
                  className="flex-row justify-between items-center"
                  style={{
                    padding: '0.5rem 0.75rem',
                    background: 'var(--color-danger-bg)',
                    borderRadius: '8px',
                    borderLeft: '3px solid var(--color-danger)',
                  }}
                >
                  <div className="flex-col" style={{ gap: '2px' }}>
                    <div className="flex-row items-center gap-sm">
                      <Link
                        href={`/dashboard?ticker=${encodeURIComponent(pos.ticker)}`}
                        aria-label={`Ver ${pos.ticker} no Monitoramento`}
                        title={`Ver ${pos.ticker} no Monitoramento`}
                        className="flex-row items-center gap-xs font-bold text-accent"
                        style={{ textDecoration: 'none', cursor: 'pointer' }}
                      >
                        <span>{pos.ticker}</span>
                        <span className="text-xs text-secondary" style={{ opacity: 0.7 }}>↗</span>
                      </Link>
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
