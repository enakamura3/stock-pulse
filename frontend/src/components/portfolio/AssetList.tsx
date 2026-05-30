import React from 'react';
import { Position } from './types';
import { formatMoney, formatPercentage } from './helpers';

interface AssetListProps {
  positions: Position[];
  kpiCurrency: string;
  onImportCsv: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onLaunchOperation: () => void;
}

export default function AssetList({ positions, kpiCurrency, onImportCsv, onLaunchOperation }: AssetListProps) {
  return (
    <div className="card flex-col gap-md" style={{ flex: '2 1 600px', minHeight: '380px' }}>
      <div className="flex-row justify-between items-center mb-lg">
        <h3 className="card-title">📦 Posições Ativas</h3>
        <div className="flex-row gap-sm">
          <label className="btn-secondary" style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}>
            📥 Importar CSV
            <input 
              type="file" accept=".csv,.txt" style={{ display: 'none' }}
              onClick={(e) => { (e.target as HTMLInputElement).value = ''; }}
              onChange={onImportCsv} 
            />
          </label>
          <button className="primary-button" onClick={onLaunchOperation} style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}>
            + Lançar Operação
          </button>
        </div>
      </div>

      <div className="table-container flex-col" style={{ flex: 1 }}>
        {positions.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>Ativo</th>
                <th className="text-right">Qtd</th>
                <th className="text-right">Preço Médio</th>
                <th className="text-right">Cotação Atual</th>
                <th className="text-right">Custo Total</th>
                <th className="text-right">Valor Atual</th>
                <th className="text-right">Retorno</th>
                <th className="text-right">P. Justo Graham</th>
                <th className="text-right">P. Justo Bazin</th>
                <th className="text-right">Yield</th>
                <th className="text-right">P/L</th>
                <th className="text-right">P/VP</th>
              </tr>
            </thead>
            <tbody>
              {positions.map((pos) => {
                const isPos = (pos.profit_loss || 0) >= 0;
                return (
                  <tr key={pos.asset_id}>
                    <td>
                      <span className="font-bold text-accent block">{pos.ticker}</span>
                      <span className="text-xs text-secondary block" style={{ maxWidth: '140px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {pos.name}
                      </span>
                    </td>
                    <td className="text-right font-semibold" style={{ fontFamily: 'monospace' }}>{pos.quantity}</td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{formatMoney(pos.average_price, pos.currency)}</td>
                    <td className="text-right font-semibold" style={{ fontFamily: 'monospace' }}>{pos.current_price ? formatMoney(pos.current_price, pos.currency) : '--'}</td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{formatMoney(pos.total_cost, kpiCurrency)}</td>
                    <td className="text-right font-bold text-primary" style={{ fontFamily: 'monospace' }}>{pos.current_value ? formatMoney(pos.current_value, kpiCurrency) : '--'}</td>
                    <td className="text-right font-bold" style={{ color: isPos ? '#00e676' : '#ff3d00' }}>
                      {pos.return_percent !== undefined ? formatPercentage(pos.return_percent) : '--'}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>
                      {pos.graham_value ? (
                        <span className="font-semibold" style={{ color: pos.current_price && pos.current_price < pos.graham_value ? '#00e676' : '#ff3d00' }}>
                          {formatMoney(pos.graham_value, pos.currency)}
                        </span>
                      ) : '--'}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>
                      {pos.bazin_value ? (
                        <span className="font-semibold" style={{ color: pos.current_price && pos.current_price < pos.bazin_value ? '#00e676' : '#ff3d00' }}>
                          {formatMoney(pos.bazin_value, pos.currency)}
                        </span>
                      ) : '--'}
                    </td>
                    <td className="text-right font-semibold text-success" style={{ fontFamily: 'monospace' }}>{pos.dividend_yield ? `${pos.dividend_yield.toFixed(2)}%` : '--'}</td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{pos.pe ? pos.pe.toFixed(2) : '--'}</td>
                    <td className="text-right" style={{ fontFamily: 'monospace', color: pos.pvp && pos.pvp < 1.0 ? '#00e676' : 'inherit' }}>{pos.pvp ? pos.pvp.toFixed(2) : '--'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <div className="flex-col items-center justify-center text-secondary" style={{ height: '240px' }}>
            <span className="text-2xl mb-sm">📁</span>
            <p className="text-sm">Esta carteira ainda não possui ativos ativos.</p>
          </div>
        )}
      </div>
    </div>
  );
}
