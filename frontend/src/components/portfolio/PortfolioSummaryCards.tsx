import React from 'react';
import { formatMoney, formatPercentage } from './helpers';
import { WalletIcon, TrendingUpIcon, TrendingDownIcon, CoinsIcon, BankIcon } from '@/components/ui/icons';

interface PortfolioSummaryCardsProps {
  totalCost: number;
  currentValue: number;
  profitLoss: number;
  returnPercent: number;
  avgDividends12m: number;
  kpiCurrency: string;
  isLoadingTreasury?: boolean;
}

export default function PortfolioSummaryCards({
  totalCost, currentValue, profitLoss, returnPercent, avgDividends12m, kpiCurrency, isLoadingTreasury
}: PortfolioSummaryCardsProps) {
  const isPos = profitLoss >= 0;

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(230px, 1fr))', gap: '1rem' }}>
      <div className="card flex-col justify-center text-left" style={{ padding: '1rem 1.25rem', position: 'relative' }}>
        <div className="flex-row items-center justify-between">
          <span className="text-secondary font-mono text-xs flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.04em' }}>
            <WalletIcon size={14} /> Patrimônio Atual
          </span>
          {isLoadingTreasury && (
            <span className="badge badge-neutral" style={{ fontSize: '0.65rem' }}>
              ⟳ Tesouro...
            </span>
          )}
        </div>
        <span className="text-2xl font-bold font-mono mt-xs" style={{ color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>
          {formatMoney(currentValue, kpiCurrency)}
        </span>
      </div>

      <div className="card flex-col justify-center text-left" style={{ padding: '1rem 1.25rem' }}>
        <span className="text-secondary font-mono text-xs flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.04em' }}>
          <BankIcon size={14} /> Total Investido
        </span>
        <span className="text-2xl font-bold font-mono mt-xs" style={{ color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>
          {formatMoney(totalCost, kpiCurrency)}
        </span>
      </div>

      <div className="card flex-col justify-center text-left" style={{ padding: '1rem 1.25rem' }}>
        <span className="text-secondary font-mono text-xs flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.04em' }}>
          {isPos ? <TrendingUpIcon size={14} color="var(--color-success)" /> : <TrendingDownIcon size={14} color="var(--color-danger)" />} Lucro / Prejuízo
        </span>
        <div className="flex-col gap-xs mt-xs">
          <span className="text-2xl font-bold font-mono" style={{ color: isPos ? 'var(--color-success)' : 'var(--color-danger)', fontVariantNumeric: 'tabular-nums', wordBreak: 'break-word', lineHeight: 1.1 }}>
            {isPos ? '+' : ''}{formatMoney(profitLoss, kpiCurrency)}
          </span>
          <span className="font-mono" style={{ fontSize: '0.82rem', color: isPos ? 'var(--color-success)' : 'var(--color-danger)', fontWeight: 600 }}>
            {formatPercentage(returnPercent)}
          </span>
        </div>
      </div>

      <div className="card flex-col justify-center text-left" style={{ padding: '1rem 1.25rem' }}>
        <span className="text-secondary font-mono text-xs flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.04em' }}>
          <CoinsIcon size={14} /> Média de Proventos (12m)
        </span>
        <span className="text-2xl font-bold font-mono mt-xs" style={{ color: 'var(--accent-color)', fontVariantNumeric: 'tabular-nums' }}>
          {formatMoney(avgDividends12m, kpiCurrency)}
          <span style={{ fontSize: '0.75rem', opacity: 0.7, fontWeight: 500, marginLeft: '4px' }}>/mês</span>
        </span>
      </div>
    </div>
  );
}
