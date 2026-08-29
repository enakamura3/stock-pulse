import React from 'react';
import { useRouter } from 'next/navigation';
import { Position, FixedIncomePosition, TreasuryPosition } from '../types';
import { formatMoney, formatPercentage, exportDailyReportCSV } from '../helpers';
import { SortKey, SortDir, EnrichedPosition, AssetTypeBadge } from './types';

export interface DailyReportTableProps {
  positions: Position[];
  fiPositions?: FixedIncomePosition[];
  treasuryPositions?: TreasuryPosition[];
  sortedRows: EnrichedPosition[];
  sortKey: SortKey;
  sortDir: SortDir;
  onSort: (key: SortKey) => void;
  kpiCurrency: string;
  getAssetTypeBadge: (pos: Position) => AssetTypeBadge | null;
}

export default function DailyReportTable({
  positions = [],
  fiPositions = [],
  treasuryPositions = [],
  sortedRows,
  sortKey,
  sortDir,
  onSort,
  kpiCurrency,
  getAssetTypeBadge,
}: DailyReportTableProps) {
  const router = useRouter();

  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return <span style={{ opacity: 0.3, marginLeft: '4px' }}>⇅</span>;
    return <span style={{ marginLeft: '4px' }}>{sortDir === 'asc' ? '↑' : '↓'}</span>;
  };

  const getAriaSort = (key: SortKey): 'ascending' | 'descending' | 'none' => {
    if (sortKey !== key) return 'none';
    return sortDir === 'asc' ? 'ascending' : 'descending';
  };

  const handleSortKeyDown = (e: React.KeyboardEvent, key: SortKey) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onSort(key);
    }
  };

  const handleNavigateKeyDown = (e: React.KeyboardEvent, ticker: string) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      router.push(`/dashboard?ticker=${encodeURIComponent(ticker)}`);
    }
  };

  return (
    <div className="card flex-col gap-md w-full">
      <div className="flex-row justify-between items-center flex-wrap gap-sm">
        <div>
          <h3 className="card-title m-0">📊 Resumo Diário Completo (Renda Variável)</h3>
          <span className="text-xs text-secondary">Clique em um ativo para abrir no Monitoramento</span>
        </div>
        <button
          onClick={() => exportDailyReportCSV(positions, fiPositions, treasuryPositions, kpiCurrency)}
          aria-label="Exportar resumo diário completo em arquivo CSV"
          className="btn-secondary flex-row items-center gap-xs text-xs font-semibold"
          style={{ padding: '0.35rem 0.75rem', borderRadius: '6px' }}
          title="Exportar resumo diário completo em arquivo CSV"
        >
          <span>📥</span> Exportar CSV
        </button>
      </div>
      <div className="table-container flex-col" style={{ overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
        {positions.length > 0 ? (
          <table className="data-table" style={{ width: '100%', minWidth: '650px' }} aria-label="Resumo diário completo de renda variável">
            <caption className="sr-only">Resumo diário de ativos de renda variável</caption>
            <thead>
              <tr>
                <th
                  scope="col"
                  role="columnheader"
                  tabIndex={0}
                  aria-sort={getAriaSort('ticker')}
                  aria-label="Ordenar por ativo"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onSort('ticker')}
                  onKeyDown={(e) => handleSortKeyDown(e, 'ticker')}
                >
                  Ativo {sortIcon('ticker')}
                </th>
                <th
                  scope="col"
                  role="columnheader"
                  tabIndex={0}
                  aria-sort={getAriaSort('average_price')}
                  aria-label="Ordenar por preço médio"
                  className="text-right"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onSort('average_price')}
                  onKeyDown={(e) => handleSortKeyDown(e, 'average_price')}
                >
                  Preço Médio {sortIcon('average_price')}
                </th>
                <th
                  scope="col"
                  role="columnheader"
                  tabIndex={0}
                  aria-sort={getAriaSort('previousClose')}
                  aria-label="Ordenar por fechamento anterior"
                  className="text-right"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onSort('previousClose')}
                  onKeyDown={(e) => handleSortKeyDown(e, 'previousClose')}
                >
                  Fech. Anterior {sortIcon('previousClose')}
                </th>
                <th
                  scope="col"
                  role="columnheader"
                  tabIndex={0}
                  aria-sort={getAriaSort('current_price')}
                  aria-label="Ordenar por cotação atual"
                  className="text-right"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onSort('current_price')}
                  onKeyDown={(e) => handleSortKeyDown(e, 'current_price')}
                >
                  Cotação Atual {sortIcon('current_price')}
                </th>
                <th
                  scope="col"
                  role="columnheader"
                  tabIndex={0}
                  aria-sort={getAriaSort('daily_change')}
                  aria-label="Ordenar por variação nominal por cota"
                  className="text-right"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onSort('daily_change')}
                  onKeyDown={(e) => handleSortKeyDown(e, 'daily_change')}
                >
                  Var./Cota {sortIcon('daily_change')}
                </th>
                <th
                  scope="col"
                  role="columnheader"
                  tabIndex={0}
                  aria-sort={getAriaSort('daily_change_percent')}
                  aria-label="Ordenar por variação percentual"
                  className="text-right"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onSort('daily_change_percent')}
                  onKeyDown={(e) => handleSortKeyDown(e, 'daily_change_percent')}
                >
                  Var. % {sortIcon('daily_change_percent')}
                </th>
                <th
                  scope="col"
                  role="columnheader"
                  tabIndex={0}
                  aria-sort={getAriaSort('impact')}
                  aria-label="Ordenar por impacto diário"
                  className="text-right"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onSort('impact')}
                  onKeyDown={(e) => handleSortKeyDown(e, 'impact')}
                >
                  Impacto Diário {sortIcon('impact')}
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedRows.map(({ pos, percent, absChange, currentPrice, previousClose, impact, portfolioWeight }) => {
                const hasQuote = currentPrice > 1e-6;
                const isUp = percent > 0;
                const isDown = percent < 0;
                const colorClass = isUp ? 'text-success' : isDown ? 'text-danger' : 'text-secondary';
                const prefix = isUp ? '+' : '';
                const avgPrice = pos.average_price ?? 0;
                const prevCloseColor = previousClose >= avgPrice ? 'text-success' : 'text-danger';
                const currentPriceColor = currentPrice >= avgPrice ? 'text-success' : 'text-danger';
                const badge = getAssetTypeBadge(pos);
                const isUSD = pos.currency?.toUpperCase() === 'USD' || pos.type === 'STOCK_US' || pos.type === 'ETF_US';

                return (
                  <tr
                    key={pos.asset_id}
                    tabIndex={0}
                    aria-label={`Ver ${pos.ticker} no Monitoramento`}
                    onClick={() => router.push(`/dashboard?ticker=${encodeURIComponent(pos.ticker)}`)}
                    onKeyDown={(e) => handleNavigateKeyDown(e, pos.ticker)}
                    style={{ cursor: 'pointer', transition: 'background-color 0.15s ease' }}
                    title={`Clique para ver ${pos.ticker} no Monitoramento de Cotações`}
                  >
                    <td style={{ whiteSpace: 'nowrap' }}>
                      <div className="flex-row items-center gap-sm">
                        <span className="font-bold text-accent">{pos.ticker}</span>
                        <span className="text-xs text-secondary" style={{ opacity: 0.6 }}>↗</span>
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
                        {isUSD && kpiCurrency === 'BRL' && pos.fx_rate_to_brl && (
                          <span
                            className="text-xs text-secondary font-mono"
                            style={{ fontSize: '0.65rem', background: 'var(--input-bg)', padding: '1px 4px', borderRadius: '3px' }}
                            title={`Câmbio: US$ 1 = R$ ${pos.fx_rate_to_brl.toFixed(2)}`}
                          >
                            PTAX R$ {pos.fx_rate_to_brl.toFixed(2)}
                          </span>
                        )}
                      </div>
                      <div style={{ width: '100%', background: 'var(--input-bg)', borderRadius: '3px', height: '3px', marginTop: '4px' }}>
                        <div style={{ width: `${Math.min(portfolioWeight, 100)}%`, background: 'var(--accent-color)', borderRadius: '3px', height: '3px' }} />
                      </div>
                      <span className="text-xs text-secondary">{portfolioWeight.toFixed(1)}%</span>
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                      {formatMoney(pos.average_price ?? 0, pos.currency)}
                    </td>
                    <td className={`text-right ${prevCloseColor}`} style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                      {formatMoney(previousClose, pos.currency)}
                    </td>
                    <td className={`text-right ${currentPriceColor}`} style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>
                      {hasQuote ? (
                        formatMoney(currentPrice, pos.currency)
                      ) : (
                        <span className="text-xs text-secondary font-semibold" title="Ativo sem cotação recente disponível">
                          ⚠️ Sem cotação
                        </span>
                      )}
                    </td>
                    <td className={`text-right ${colorClass}`} style={{ whiteSpace: 'nowrap' }}>
                      {hasQuote ? `${prefix}${formatMoney(absChange, pos.currency)}` : '-'}
                    </td>
                    <td className={`text-right font-bold ${colorClass}`} style={{ whiteSpace: 'nowrap' }}>
                      {hasQuote ? formatPercentage(percent) : '-'}
                    </td>
                    <td className={`text-right font-bold ${colorClass}`} style={{ whiteSpace: 'nowrap' }}>
                      {hasQuote ? `${prefix}${formatMoney(impact, kpiCurrency)}` : '-'}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <div className="flex-col items-center justify-center py-xl text-secondary" style={{ gap: '0.75rem' }}>
            <span style={{ fontSize: '2.5rem' }}>📭</span>
            <span className="font-semibold">Nenhuma posição em renda variável ativa.</span>
          </div>
        )}
      </div>
    </div>
  );
}
