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
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(230px, 1fr))', gap: '1.25rem' }}>
      <div className="card flex-col justify-center text-left" style={{ padding: '1.25rem 1.5rem' }}>
        <span className="text-secondary text-xs font-semibold flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          <WalletIcon size={16} /> Patrimônio Atual
          {isLoadingTreasury && (
            <span style={{ fontSize: '0.65rem', color: 'var(--text-secondary)', marginLeft: '6px' }}>
              ⟳ Tesouro...
            </span>
          )}
        </span>
        <span className="text-2xl font-bold mt-sm" style={{ color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>
          {formatMoney(currentValue, kpiCurrency)}
        </span>
      </div>
      
      <div className="card flex-col justify-center text-left" style={{ padding: '1.25rem 1.5rem' }}>
        <span className="text-secondary text-xs font-semibold flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          <BankIcon size={16} /> Total Investido
        </span>
        <span className="text-2xl font-bold mt-sm" style={{ color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>
          {formatMoney(totalCost, kpiCurrency)}
        </span>
      </div>

      <div className="card flex-col justify-center text-left" style={{ padding: '1.25rem 1.5rem' }}>
        <span className="text-secondary text-xs font-semibold flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          {isPos ? <TrendingUpIcon size={16} color="var(--color-success)" /> : <TrendingDownIcon size={16} color="var(--color-danger)" />} Lucro / Prejuízo
        </span>
        <div className="flex-col gap-xs mt-sm">
          <span className="text-2xl font-bold" style={{ color: isPos ? 'var(--color-success)' : 'var(--color-danger)', letterSpacing: '-0.02em', wordBreak: 'break-word', lineHeight: 1.1 }}>
            {isPos ? '+' : ''}{formatMoney(profitLoss, kpiCurrency)}
          </span>
          <span style={{ fontSize: '0.9rem', color: isPos ? 'var(--color-success)' : 'var(--color-danger)', opacity: 0.9, fontWeight: 600 }}>
            {formatPercentage(returnPercent)}
          </span>
        </div>
      </div>

      <div className="card flex-col justify-center text-left" style={{ padding: '1.25rem 1.5rem' }}>
        <span className="text-secondary text-xs font-semibold flex-row items-center gap-xs" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          <CoinsIcon size={16} /> Média de Proventos (12m)
        </span>
        <span className="text-2xl font-bold mt-sm" style={{ color: 'var(--accent-color)', letterSpacing: '-0.02em' }}>
          {formatMoney(avgDividends12m, kpiCurrency)}
          <span style={{ fontSize: '0.8rem', opacity: 0.7, fontWeight: 500, marginLeft: '4px' }}>/mês</span>
        </span>
      </div>
    </div>
  );
}
