import React from 'react';
import { Position } from './types';
import { formatMoney, formatPercentage, formatQuantity } from './helpers';

interface AssetListProps {
  positions: Position[];
  kpiCurrency: string;
  onImportCsv: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onLaunchOperation: () => void;
}

export default function AssetList({ positions, kpiCurrency, onImportCsv, onLaunchOperation }: AssetListProps) {
  return (
    <div className="card flex-col gap-md" style={{ width: '100%' }}>
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

      <div className="table-container flex-col" style={{ width: '100%', overflowX: 'auto' }}>
        {positions.length > 0 ? (
          <table className="data-table" style={{ width: '100%', fontSize: '0.8rem' }}>
            <thead>
              <tr>
                <th style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Ativo</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Qtd</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Preço Médio</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Cotação Atual</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Custo Total</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Valor Atual</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Retorno</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>P. Justo Graham</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>P. Justo Bazin</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>Yield</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>P/L</th>
                <th className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap' }}>P/VP</th>
              </tr>
            </thead>
            <tbody>
              {positions.map((pos) => {
                const isPos = (pos.profit_loss || 0) >= 0;
                return (
                  <tr key={pos.asset_id}>
                    <td title={pos.name} style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', cursor: 'help' }}>
                      <span className="font-bold text-accent">{pos.ticker}</span>
                    </td>
                    <td className="text-right font-semibold" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>{formatQuantity(pos.quantity)}</td>
                    <td className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>{formatMoney(pos.average_price, pos.currency)}</td>
                    <td className="text-right font-semibold" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>{pos.current_price ? formatMoney(pos.current_price, pos.currency) : '--'}</td>
                    <td className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>{formatMoney(pos.total_cost, kpiCurrency)}</td>
                    <td className="text-right font-bold text-primary" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>{pos.current_value ? formatMoney(pos.current_value, kpiCurrency) : '--'}</td>
                    <td className="text-right font-bold" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', color: isPos ? '#00e676' : '#ff3d00' }}>
                      {pos.return_percent !== undefined ? formatPercentage(pos.return_percent) : '--'}
                    </td>
                    <td className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>
                      {pos.graham_value ? (
                        <span className="font-semibold" style={{ color: pos.current_price && pos.current_price < pos.graham_value ? '#00e676' : '#ff3d00' }}>
                          {formatMoney(pos.graham_value, pos.currency)}
                        </span>
                      ) : '--'}
                    </td>
                    <td className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>
                      {pos.bazin_value ? (
                        <span className="font-semibold" style={{ color: pos.current_price && pos.current_price < pos.bazin_value ? '#00e676' : '#ff3d00' }}>
                          {formatMoney(pos.bazin_value, pos.currency)}
                        </span>
                      ) : '--'}
                    </td>
                    <td className="text-right font-semibold text-success" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>{pos.dividend_yield ? `${pos.dividend_yield.toFixed(2)}%` : '--'}</td>
                    <td className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace' }}>{pos.pe ? pos.pe.toFixed(2) : '--'}</td>
                    <td className="text-right" style={{ padding: '0.65rem 0.5rem', whiteSpace: 'nowrap', fontFamily: 'monospace', color: pos.pvp && pos.pvp < 1.0 ? '#00e676' : 'inherit' }}>{pos.pvp ? pos.pvp.toFixed(2) : '--'}</td>
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
