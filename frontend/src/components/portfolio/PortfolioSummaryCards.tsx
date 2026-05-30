import React from 'react';
import { formatMoney, formatPercentage } from './helpers';

interface PortfolioSummaryCardsProps {
  totalCost: number;
  currentValue: number;
  profitLoss: number;
  returnPercent: number;
  kpiCurrency: string;
}

export default function PortfolioSummaryCards({
  totalCost, currentValue, profitLoss, returnPercent, kpiCurrency
}: PortfolioSummaryCardsProps) {
  const isPos = profitLoss >= 0;

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(230px, 1fr))', gap: '1.25rem' }}>
      <div className="card flex-col justify-center text-left">
        <span className="text-secondary text-xs font-semibold" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          Patrimônio Atual
        </span>
        <span className="text-2xl font-bold mt-sm" style={{ color: '#fff', letterSpacing: '-0.02em' }}>
          {formatMoney(currentValue, kpiCurrency)}
        </span>
      </div>
      
      <div className="card flex-col justify-center text-left">
        <span className="text-secondary text-xs font-semibold" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          Total Investido
        </span>
        <span className="text-2xl font-bold mt-sm" style={{ color: '#fff', letterSpacing: '-0.02em' }}>
          {formatMoney(totalCost, kpiCurrency)}
        </span>
      </div>

      <div className="card flex-col justify-center text-left">
        <span className="text-secondary text-xs font-semibold" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          Lucro / Prejuízo
        </span>
        <span className="text-2xl font-bold mt-sm" style={{ color: isPos ? '#00e676' : '#ff3d00', letterSpacing: '-0.02em' }}>
          {isPos ? '▲' : '▼'} {formatMoney(profitLoss, kpiCurrency)}
        </span>
      </div>

      <div className="card flex-col justify-center text-left" style={{ position: 'relative' }}>
        <span className="text-secondary text-xs font-semibold" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          Rentabilidade
        </span>
        <span className="text-2xl font-bold mt-sm" style={{ color: isPos ? '#00e676' : '#ff3d00', letterSpacing: '-0.02em' }}>
          {formatPercentage(returnPercent)}
        </span>
        {totalCost > 0 && profitLoss > 0 && (
          <span className="pulse-dot" style={{ position: 'absolute', top: '15px', right: '15px', width: '8px', height: '8px', background: '#00e676', borderRadius: '50%' }}></span>
        )}
      </div>
    </div>
  );
}
