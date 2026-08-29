import React from 'react';
import { CalculatedDividend } from '../types';
import { formatMoney } from '../helpers';

export interface DailyReportDividendsTodayProps {
  todayDividends: CalculatedDividend[];
  totalTodayDividends: number;
  kpiCurrency: string;
}

export default function DailyReportDividendsToday({
  todayDividends,
  totalTodayDividends,
  kpiCurrency,
}: DailyReportDividendsTodayProps) {
  if (todayDividends.length === 0) return null;

  return (
    <div
      className="card flex-col gap-sm w-full"
      style={{
        background: 'var(--color-success-bg)',
        borderLeft: '4px solid var(--color-success)',
        padding: '1.25rem',
      }}
    >
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
  );
}
