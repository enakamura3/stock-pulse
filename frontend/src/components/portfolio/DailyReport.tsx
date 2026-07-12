import React, { useState } from 'react';
import { Position, TreasuryPosition } from './types';
import { formatMoney, formatPercentage } from './helpers';

interface DailyReportProps {
  positions: Position[];
  treasuryPositions?: TreasuryPosition[];
  kpiCurrency: string;
}

type SortKey = 'ticker' | 'average_price' | 'previousClose' | 'current_price' | 'daily_change' | 'daily_change_percent' | 'impact';
type SortDir = 'asc' | 'desc';

// Helper: calcula a taxa de câmbio de uma posição para a moeda base
function getExchangeRate(pos: Position): number {
  const price = pos.current_price ?? 0;
  const qty = pos.quantity ?? 0;
  if (price > 0 && qty > 0) {
    return (pos.current_value ?? 0) / (price * qty);
  }
  return 1.0;
}

// Helper: retorna o rótulo de tipo de ativo
function getAssetTypeBadge(pos: Position): { label: string; color: string } | null {
  const type = (pos.type || '').toUpperCase();
  if (type === 'FII') return { label: 'FII', color: 'var(--color-warning, #f59e0b)' };
  if (type === 'ETF') return { label: 'ETF', color: 'var(--color-info, #38bdf8)' };
  if (type === 'BDR') return { label: 'BDR', color: 'var(--color-secondary-text, #a78bfa)' };
  if (type === 'STOCK' || type === 'AÇÃO' || type === 'ACAO') return { label: 'Ação', color: 'var(--color-success, #00e676)' };
  if (type === 'CRYPTO') return { label: 'Crypto', color: '#f97316' };
  return null;
}

export default function DailyReport({ positions, treasuryPositions = [], kpiCurrency }: DailyReportProps) {
  const [sortKey, setSortKey] = useState<SortKey>('daily_change_percent');
  const [sortDir, setSortDir] = useState<SortDir>('desc');

  const now = new Date();
  const lastUpdateStr = now.toLocaleString('pt-BR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });

  // Calcula total diário da carteira em kpiCurrency
  let totalDailyChange = 0;
  let totalPortfolioValue = 0;
  positions.forEach(pos => {
    const rate = getExchangeRate(pos);
    totalDailyChange += (pos.daily_change ?? 0) * (pos.quantity ?? 0) * rate;
    totalPortfolioValue += (pos.current_value ?? 0);
  });

  const totalTreasuryValue = treasuryPositions.reduce((s, p) => s + p.net_value, 0);
  totalPortfolioValue += totalTreasuryValue;

  // Variação % total ponderada = totalDailyChange / (valor_anterior = totalPortfolioValue - totalDailyChange)
  const previousTotalValue = totalPortfolioValue - totalDailyChange;
  const totalDailyPercent = Math.abs(previousTotalValue) > 1e-6
    ? (totalDailyChange / previousTotalValue) * 100
    : 0;

  const isDailyPos = totalDailyChange >= 0;

  // Filtra posições com dado de variação e enriquece com campos calculados
  const enrichedPositions = positions
    .filter(p => p.daily_change_percent !== undefined)
    .map(pos => {
      const percent = pos.daily_change_percent ?? 0;
      const absChange = pos.daily_change ?? 0;
      const currentPrice = pos.current_price ?? 0;
      const previousClose = currentPrice - absChange;
      const qty = pos.quantity ?? 0;
      const rate = getExchangeRate(pos);
      const impact = absChange * qty * rate;
      const portfolioWeight = totalPortfolioValue > 1e-6
        ? ((pos.current_value ?? 0) / totalPortfolioValue) * 100
        : 0;
      return { pos, percent, absChange, currentPrice, previousClose, qty, rate, impact, portfolioWeight };
    });

  // Ordenação interativa
  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const sortedRows = [...enrichedPositions].sort((a, b) => {
    let aVal: number | string = 0;
    let bVal: number | string = 0;
    switch (sortKey) {
      case 'ticker': aVal = a.pos.ticker; bVal = b.pos.ticker; break;
      case 'average_price': aVal = a.pos.average_price ?? 0; bVal = b.pos.average_price ?? 0; break;
      case 'previousClose': aVal = a.previousClose; bVal = b.previousClose; break;
      case 'current_price': aVal = a.currentPrice; bVal = b.currentPrice; break;
      case 'daily_change': aVal = a.absChange; bVal = b.absChange; break;
      case 'daily_change_percent': aVal = a.percent; bVal = b.percent; break;
      case 'impact': aVal = a.impact; bVal = b.impact; break;
    }
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortDir === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    }
    return sortDir === 'asc' ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
  });

  // Top 5 altas e baixas (por %)
  const sortedByPercent = [...enrichedPositions].sort((a, b) => b.percent - a.percent);
  const topRisers = sortedByPercent.filter(r => r.percent > 0).slice(0, 5);
  const topFallers = [...sortedByPercent].reverse().filter(r => r.percent < 0).slice(0, 5);

  // Totais da tabela
  const totalImpact = sortedRows.reduce((acc, r) => acc + r.impact, 0);

  // Helper para ícone de ordenação
  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return <span style={{ opacity: 0.3, marginLeft: '4px' }}>⇅</span>;
    return <span style={{ marginLeft: '4px' }}>{sortDir === 'asc' ? '↑' : '↓'}</span>;
  };

  return (
    <div className="flex-col gap-xl w-full">

      {/* Card principal: Variação Total */}
      {positions.length > 0 && (
        <div className="card flex-col items-center justify-center text-center w-full" style={{ padding: '1.5rem', gap: '0.25rem' }}>
          <span className="text-secondary text-sm font-semibold" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Variação Total Diária da Carteira
          </span>
          <span className="text-3xl font-bold mt-sm" style={{ color: isDailyPos ? '#00e676' : '#ff3d00', letterSpacing: '-0.02em' }}>
            {isDailyPos ? '🟢 +' : '🔴 '}{formatMoney(totalDailyChange, kpiCurrency)}
          </span>
          <span className="text-lg font-semibold" style={{ color: isDailyPos ? '#00e676' : '#ff3d00' }}>
            ({isDailyPos ? '+' : ''}{totalDailyPercent.toFixed(2)}%)
          </span>
          <span className="text-xs text-secondary" style={{ marginTop: '0.5rem' }}>
            🕐 Última atualização: {lastUpdateStr}
          </span>
        </div>
      )}

      {/* Cards: Maiores Altas e Baixas */}
      <div className="flex-row gap-lg flex-wrap">
        {/* Maiores Altas */}
        <div className="card flex-col gap-md" style={{ flex: '1 1 300px' }}>
          <h3 className="card-title text-success">🚀 Maiores Altas do Dia</h3>
          {topRisers.length > 0 ? (
            <div className="flex-col gap-sm">
              {topRisers.map(({ pos, percent, impact, portfolioWeight }) => {
                const badge = getAssetTypeBadge(pos);
                return (
                  <div key={pos.asset_id} className="flex-row justify-between items-center" style={{ padding: '0.5rem 0.75rem', background: 'rgba(0,230,118,0.05)', borderRadius: '8px', borderLeft: '3px solid #00e676' }}>
                    <div className="flex-col" style={{ gap: '2px' }}>
                      <div className="flex-row items-center gap-sm">
                        <span className="font-bold">{pos.ticker}</span>
                        {badge && (
                          <span style={{ fontSize: '0.65rem', fontWeight: 700, color: badge.color, border: `1px solid ${badge.color}`, borderRadius: '4px', padding: '1px 5px', lineHeight: 1.4 }}>
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
                  <div key={pos.asset_id} className="flex-row justify-between items-center" style={{ padding: '0.5rem 0.75rem', background: 'rgba(255,61,0,0.05)', borderRadius: '8px', borderLeft: '3px solid #ff3d00' }}>
                    <div className="flex-col" style={{ gap: '2px' }}>
                      <div className="flex-row items-center gap-sm">
                        <span className="font-bold">{pos.ticker}</span>
                        {badge && (
                          <span style={{ fontSize: '0.65rem', fontWeight: 700, color: badge.color, border: `1px solid ${badge.color}`, borderRadius: '4px', padding: '1px 5px', lineHeight: 1.4 }}>
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

      {/* Tabela: Resumo Completo */}
      <div className="card flex-col gap-md w-full">
        <h3 className="card-title">📊 Resumo Diário Completo</h3>
        <div className="table-container flex-col">
          {positions.length > 0 ? (
            <table className="data-table">
              <thead>
                <tr>
                  <th style={{ cursor: 'pointer' }} onClick={() => handleSort('ticker')}>
                    Ativo {sortIcon('ticker')}
                  </th>
                  <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('average_price')}>
                    Preço Médio {sortIcon('average_price')}
                  </th>
                  <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('previousClose')}>
                    Fech. Anterior {sortIcon('previousClose')}
                  </th>
                  <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('current_price')}>
                    Cotação Atual {sortIcon('current_price')}
                  </th>
                  <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('daily_change')}>
                    Var./Cota {sortIcon('daily_change')}
                  </th>
                  <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('daily_change_percent')}>
                    Var. % {sortIcon('daily_change_percent')}
                  </th>
                  <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('impact')}>
                    Impacto Diário {sortIcon('impact')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {sortedRows.map(({ pos, percent, absChange, currentPrice, previousClose, impact, portfolioWeight }) => {
                  const isUp = percent > 0;
                  const isDown = percent < 0;
                  const colorClass = isUp ? 'text-success' : isDown ? 'text-danger' : 'text-secondary';
                  const prefix = isUp ? '+' : '';
                  const avgPrice = pos.average_price ?? 0;
                  const prevCloseColor = previousClose >= avgPrice ? 'text-success' : 'text-danger';
                  const currentPriceColor = currentPrice >= avgPrice ? 'text-success' : 'text-danger';
                  const badge = getAssetTypeBadge(pos);

                  return (
                    <tr key={pos.asset_id}>
                      <td>
                        <div className="flex-row items-center gap-sm">
                          <span className="font-bold">{pos.ticker}</span>
                          {badge && (
                            <span style={{ fontSize: '0.65rem', fontWeight: 700, color: badge.color, border: `1px solid ${badge.color}`, borderRadius: '4px', padding: '1px 5px', lineHeight: 1.4 }}>
                              {badge.label}
                            </span>
                          )}
                        </div>
                        <div style={{ width: '100%', background: 'rgba(255,255,255,0.06)', borderRadius: '3px', height: '3px', marginTop: '4px' }}>
                          <div style={{ width: `${Math.min(portfolioWeight, 100)}%`, background: 'var(--color-primary, #6366f1)', borderRadius: '3px', height: '3px' }} />
                        </div>
                        <span className="text-xs text-secondary">{portfolioWeight.toFixed(1)}%</span>
                      </td>
                      <td className="text-right" style={{ fontFamily: 'monospace' }}>
                        {formatMoney(pos.average_price ?? 0, pos.currency)}
                      </td>
                      <td className={`text-right ${prevCloseColor}`} style={{ fontFamily: 'monospace' }}>
                        {formatMoney(previousClose, pos.currency)}
                      </td>
                      <td className={`text-right ${currentPriceColor}`} style={{ fontFamily: 'monospace' }}>
                        {formatMoney(currentPrice, pos.currency)}
                      </td>
                      <td className={`text-right ${colorClass}`}>
                        {prefix}{formatMoney(absChange, pos.currency)}
                      </td>
                      <td className={`text-right font-bold ${colorClass}`}>
                        {formatPercentage(percent)}
                      </td>
                      <td className={`text-right font-bold ${colorClass}`}>
                        {prefix}{formatMoney(impact, kpiCurrency)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>

            </table>
          ) : (
            <div className="flex-col items-center justify-center py-xl text-secondary" style={{ gap: '0.75rem' }}>
              <span style={{ fontSize: '2.5rem' }}>📭</span>
              <span className="font-semibold">Nenhuma posição ativa encontrada.</span>
              <span className="text-sm" style={{ opacity: 0.7 }}>Cadastre ativos na aba <strong>Carteira</strong> para visualizar o resumo diário.</span>
            </div>
          )}
        </div>
      </div>

      {treasuryPositions.length > 0 && (
        <div className="card flex-col gap-md w-full">
          <h3 className="card-title">🏛️ Posição Atualizada: Tesouro Direto</h3>
          <p className="text-xs text-secondary">
            Títulos do Tesouro Nacional não possuem cotação intraday. Os valores abaixo
            representam a última posição de liquidação líquida disponível.
          </p>
          <div className="table-container flex-col">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Título</th>
                  <th>Tipo</th>
                  <th className="text-right">Vencimento</th>
                  <th className="text-right">Total Investido</th>
                  <th className="text-right">Valor Líquido</th>
                  <th className="text-right">Rentabilidade Acumulada</th>
                </tr>
              </thead>
              <tbody>
                {treasuryPositions.map(p => {
                  const returnPct = p.total_invested > 1e-6
                    ? ((p.net_value - p.total_invested) / p.total_invested) * 100
                    : 0;
                  const badgeColor = p.treasury_type === 'SELIC' ? '#4caf50'
                    : p.treasury_type === 'PREFIXADO' ? '#2196f3' : '#ff9800';
                  return (
                    <tr key={p.transaction_id}>
                      <td><span className="font-bold">{p.ticker}</span></td>
                      <td>
                        <span style={{ background: `${badgeColor}20`, color: badgeColor, border: `1px solid ${badgeColor}`, fontSize: '0.65rem', padding: '2px 6px', borderRadius: '4px', fontWeight: 700 }}>
                          {p.treasury_type}
                        </span>
                      </td>
                      <td className="text-right">{new Date(p.maturity_date).toLocaleDateString('pt-BR')}</td>
                      <td className="text-right">{formatMoney(p.total_invested, kpiCurrency)}</td>
                      <td className="text-right" style={{ fontWeight: 600 }}>{formatMoney(p.net_value, kpiCurrency)}</td>
                      <td className={`text-right font-bold ${returnPct >= 0 ? 'text-success' : 'text-danger'}`}>
                        {returnPct >= 0 ? '+' : ''}{returnPct.toFixed(2)}%
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
