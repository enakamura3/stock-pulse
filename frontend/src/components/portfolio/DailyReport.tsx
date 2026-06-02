import React from 'react';
import { Position } from './types';
import { formatMoney, formatPercentage } from './helpers';

interface DailyReportProps {
  positions: Position[];
}

export default function DailyReport({ positions }: DailyReportProps) {
  // Sort by daily_change_percent descending
  const sortedByPercent = [...positions].filter(p => p.daily_change_percent !== undefined).sort((a, b) => {
    return (b.daily_change_percent || 0) - (a.daily_change_percent || 0);
  });

  const topRisers = sortedByPercent.slice(0, 5);
  const topFallers = [...sortedByPercent].reverse().slice(0, 5);

  return (
    <div className="flex-col gap-xl w-full">
      <div className="flex-row gap-lg flex-wrap">
        {/* Maiores Altas */}
        <div className="card flex-col gap-md" style={{ flex: '1 1 300px' }}>
          <h3 className="card-title text-success">🚀 Maiores Altas do Dia</h3>
          {topRisers.length > 0 && topRisers[0].daily_change_percent! > 0 ? (
            <div className="flex-col gap-sm">
              {topRisers.filter(p => p.daily_change_percent! > 0).map(pos => (
                <div key={pos.asset_id} className="flex-row justify-between items-center" style={{ padding: '0.5rem', background: 'rgba(255,255,255,0.02)', borderRadius: '8px' }}>
                  <span className="font-bold">{pos.ticker}</span>
                  <div className="flex-col items-end">
                    <span className="text-success font-bold">+{formatPercentage(pos.daily_change_percent!)}</span>
                    <span className="text-xs text-secondary">+{formatMoney(pos.daily_change || 0, pos.currency)}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <span className="text-sm text-secondary">Nenhuma alta registrada hoje.</span>
          )}
        </div>

        {/* Maiores Baixas */}
        <div className="card flex-col gap-md" style={{ flex: '1 1 300px' }}>
          <h3 className="card-title text-danger">📉 Maiores Baixas do Dia</h3>
          {topFallers.length > 0 && topFallers[0].daily_change_percent! < 0 ? (
            <div className="flex-col gap-sm">
              {topFallers.filter(p => p.daily_change_percent! < 0).map(pos => (
                <div key={pos.asset_id} className="flex-row justify-between items-center" style={{ padding: '0.5rem', background: 'rgba(255,255,255,0.02)', borderRadius: '8px' }}>
                  <span className="font-bold">{pos.ticker}</span>
                  <div className="flex-col items-end">
                    <span className="text-danger font-bold">{formatPercentage(pos.daily_change_percent!)}</span>
                    <span className="text-xs text-secondary">{formatMoney(pos.daily_change || 0, pos.currency)}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <span className="text-sm text-secondary">Nenhuma baixa registrada hoje.</span>
          )}
        </div>
      </div>

      <div className="card flex-col gap-md w-full">
        <h3 className="card-title">📊 Resumo Diário Completo</h3>
        <div className="table-container flex-col">
          {positions.length > 0 ? (
            <table className="data-table">
              <thead>
                <tr>
                  <th>Ativo</th>
                  <th className="text-right">Variação (%)</th>
                  <th className="text-right">Variação ($)</th>
                  <th className="text-right">Cotação Atual</th>
                  <th className="text-right">Fechamento Anterior</th>
                </tr>
              </thead>
              <tbody>
                {sortedByPercent.map((pos) => {
                  const percent = pos.daily_change_percent || 0;
                  const absChange = pos.daily_change || 0;
                  const currentPrice = pos.current_price || 0;
                  const previousClose = currentPrice - absChange;
                  const isUp = percent > 0;
                  const isDown = percent < 0;
                  const colorClass = isUp ? 'text-success' : isDown ? 'text-danger' : 'text-secondary';
                  const prefix = isUp ? '+' : '';
                  
                  return (
                    <tr key={pos.asset_id}>
                      <td><span className="font-bold">{pos.ticker}</span></td>
                      <td className={`text-right font-bold ${colorClass}`}>
                        {prefix}{formatPercentage(percent)}
                      </td>
                      <td className={`text-right ${colorClass}`}>
                        {prefix}{formatMoney(absChange, pos.currency)}
                      </td>
                      <td className="text-right" style={{ fontFamily: 'monospace' }}>
                        {formatMoney(currentPrice, pos.currency)}
                      </td>
                      <td className="text-right text-secondary" style={{ fontFamily: 'monospace' }}>
                        {formatMoney(previousClose, pos.currency)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <div className="flex-col items-center justify-center py-xl text-secondary">
              Sem posições ativas.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
