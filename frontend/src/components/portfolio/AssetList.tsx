import React, { useState } from 'react';
import { Position } from './types';
import { formatMoney, formatPercentage, formatQuantity } from './helpers';
import { TrendingUpIcon, ReceiptIcon, WalletIcon } from '@/components/ui/icons';

interface AssetListProps {
  positions: Position[];
  kpiCurrency: string;
  onImportCsv: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onLaunchOperation: () => void;
}

export default function AssetList({ positions, kpiCurrency, onImportCsv, onLaunchOperation }: AssetListProps) {
  type SortKey = 'ticker' | 'quantity' | 'average_price' | 'current_price' | 'total_cost' | 'current_value' | 'return_percent' | 'graham_value' | 'bazin_value' | 'dividend_yield' | 'pe' | 'pvp';
  type SortDir = 'asc' | 'desc';

  const [sortKey, setSortKey] = useState<SortKey>('ticker');
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const sortedPositions = [...positions].sort((a, b) => {
    let aVal: number | string = 0;
    let bVal: number | string = 0;
    switch (sortKey) {
      case 'ticker': aVal = a.ticker || ''; bVal = b.ticker || ''; break;
      case 'quantity': aVal = a.quantity ?? 0; bVal = b.quantity ?? 0; break;
      case 'average_price': aVal = a.average_price ?? 0; bVal = b.average_price ?? 0; break;
      case 'current_price': aVal = a.current_price ?? 0; bVal = b.current_price ?? 0; break;
      case 'total_cost': aVal = a.total_cost ?? 0; bVal = b.total_cost ?? 0; break;
      case 'current_value': aVal = a.current_value ?? 0; bVal = b.current_value ?? 0; break;
      case 'return_percent': aVal = a.return_percent ?? 0; bVal = b.return_percent ?? 0; break;
      case 'graham_value': aVal = a.graham_value ?? 0; bVal = b.graham_value ?? 0; break;
      case 'bazin_value': aVal = a.bazin_value ?? 0; bVal = b.bazin_value ?? 0; break;
      case 'dividend_yield': aVal = a.dividend_yield ?? 0; bVal = b.dividend_yield ?? 0; break;
      case 'pe': aVal = a.pe ?? 0; bVal = b.pe ?? 0; break;
      case 'pvp': aVal = a.pvp ?? 0; bVal = b.pvp ?? 0; break;
    }
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortDir === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    }
    return sortDir === 'asc' ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
  });

  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return <span style={{ opacity: 0.3, marginLeft: '4px' }}>⇅</span>;
    return <span style={{ marginLeft: '4px' }}>{sortDir === 'asc' ? '↑' : '↓'}</span>;
  };

  return (
    <div className="card flex-col gap-0" style={{ width: '100%', padding: 0, overflow: 'hidden' }}>
      {/* Terminal Title Bar */}
      <div className="terminal-bar">
        <div className="flex-row items-center gap-xs font-mono">
          <span className="text-accent" style={{ fontSize: '0.8rem', fontWeight: 'bold' }}>▸</span>
          <span style={{ color: 'var(--text-secondary)' }}>portfolio/positions.grid</span>
        </div>
        <span className="badge badge-neutral font-mono" style={{ fontSize: '0.65rem' }}>
          [ {positions.length} POSIÇÕES ]
        </span>
      </div>

      <div style={{ padding: '1rem 1.25rem' }}>
        <div className="flex-row justify-between items-center mb-md">
          <h3 className="card-title font-mono" style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.95rem' }}>
            <span className="text-muted" style={{ opacity: 0.5 }}>{'//'}</span>
            <TrendingUpIcon size={16} /> <span>Posições Ativas</span>
          </h3>
          <div className="flex-row gap-sm">
            <label className="btn-secondary" style={{ padding: '0.4rem 0.8rem', fontSize: '0.78rem', display: 'inline-flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }}>
              <ReceiptIcon size={13} /> Importar CSV
              <input 
                type="file" accept=".csv,.txt" style={{ display: 'none' }}
                onClick={(e) => { (e.target as HTMLInputElement).value = ''; }}
                onChange={onImportCsv} 
              />
            </label>
            <button className="primary-button" onClick={onLaunchOperation} style={{ padding: '0.4rem 0.85rem', fontSize: '0.78rem' }}>
              + Lançar Operação
            </button>
          </div>
        </div>

        <div className="table-container flex-col" style={{ width: '100%', overflowX: 'auto' }}>
          {positions.length > 0 ? (
            <table className="data-table" style={{ width: '100%', fontSize: '0.78rem' }}>
              <thead>
                <tr>
                  <th className="table-line-num">#</th>
                  <th style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('ticker')}>Ativo {sortIcon('ticker')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('quantity')}>Qtd {sortIcon('quantity')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('average_price')}>Preço Médio {sortIcon('average_price')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('current_price')}>Cotação Atual {sortIcon('current_price')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('total_cost')}>Custo Total {sortIcon('total_cost')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('current_value')}>Valor Atual {sortIcon('current_value')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('return_percent')}>Retorno {sortIcon('return_percent')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('graham_value')}>P. Justo Graham {sortIcon('graham_value')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('bazin_value')}>P. Justo Bazin {sortIcon('bazin_value')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('dividend_yield')}>Yield {sortIcon('dividend_yield')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('pe')}>P/L {sortIcon('pe')}</th>
                  <th className="text-right" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'pointer' }} onClick={() => handleSort('pvp')}>P/VP {sortIcon('pvp')}</th>
                </tr>
              </thead>
              <tbody>
                {sortedPositions.map((pos, idx) => {
                  const isPos = (pos.profit_loss || 0) >= 0;
                  return (
                    <tr key={pos.asset_id || pos.ticker}>
                      <td className="table-line-num">{String(idx + 1).padStart(2, '0')}</td>
                      <td title={pos.name} style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', cursor: 'help' }}>
                        <span className="font-bold font-mono text-accent">{pos.ticker}</span>
                      </td>
                      <td className="text-right font-semibold font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{formatQuantity(pos.quantity)}</td>
                      <td className="text-right font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(pos.average_price, pos.currency)}</td>
                      <td className="text-right font-semibold font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{pos.current_price ? formatMoney(pos.current_price, pos.currency) : '--'}</td>
                      <td className="text-right font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(pos.total_cost, kpiCurrency)}</td>
                      <td className="text-right font-bold text-primary font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{pos.current_value ? formatMoney(pos.current_value, kpiCurrency) : '--'}</td>
                      <td className="text-right font-bold font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums', color: isPos ? 'var(--color-success)' : 'var(--color-danger)' }}>
                        {pos.return_percent !== undefined ? formatPercentage(pos.return_percent) : '--'}
                      </td>
                      <td className="text-right font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>
                        {pos.graham_value ? (
                          <span className="font-semibold" style={{ color: pos.current_price && pos.current_price < pos.graham_value ? 'var(--color-success)' : 'var(--color-danger)' }}>
                            {formatMoney(pos.graham_value, pos.currency)}
                          </span>
                        ) : '--'}
                      </td>
                      <td className="text-right font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>
                        {pos.bazin_value ? (
                          <span className="font-semibold" style={{ color: pos.current_price && pos.current_price < pos.bazin_value ? 'var(--color-success)' : 'var(--color-danger)' }}>
                            {formatMoney(pos.bazin_value, pos.currency)}
                          </span>
                        ) : '--'}
                      </td>
                      <td className="text-right font-semibold text-success font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{pos.dividend_yield ? `${pos.dividend_yield.toFixed(2)}%` : '--'}</td>
                      <td className="text-right font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{pos.pe ? pos.pe.toFixed(2) : '--'}</td>
                      <td className="text-right font-mono" style={{ padding: '0.55rem 0.5rem', whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums', color: pos.pvp && pos.pvp < 1.0 ? 'var(--color-success)' : 'inherit' }}>{pos.pvp ? pos.pvp.toFixed(2) : '--'}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <div className="flex-col items-center justify-center text-secondary font-mono" style={{ height: '200px' }}>
              <WalletIcon size={28} />
              <p className="text-xs mt-sm text-muted">Esta carteira ainda não possui ativos ativos.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
