import React, { useState } from 'react';
import { Position, FixedIncomePosition, TreasuryPosition } from './types';
import { formatMoney, formatPercentage } from './helpers';

interface DailyReportProps {
  positions: Position[];
  fiPositions?: FixedIncomePosition[];
  treasuryPositions?: TreasuryPosition[];
  kpiCurrency: string;
}

type SortKey = 'ticker' | 'average_price' | 'previousClose' | 'current_price' | 'daily_change' | 'daily_change_percent' | 'impact';
type SortDir = 'asc' | 'desc';

// Helper: calcula a taxa de câmbio de uma posição para a moeda base
function getExchangeRate(pos: Position, kpiCurrency: string = 'BRL'): number {
  const price = pos.current_price ?? 0;
  const qty = pos.quantity ?? 0;
  const val = pos.current_value ?? 0;
  if (price > 1e-6 && qty > 1e-6 && val > 1e-6) {
    return val / (price * qty);
  }
  if (pos.currency && pos.currency.toUpperCase() === kpiCurrency.toUpperCase()) {
    return 1.0;
  }
  return 0.0;
}

// Helper: retorna o rótulo de tipo de ativo
const getAssetTypeBadge = (pos: Position) => {
  const type = (pos.type || '').toUpperCase();
  if (type === 'FII') return { label: 'FII', color: 'var(--color-warning)' };
  if (type === 'ETF') return { label: 'ETF', color: 'var(--color-info)' };
  if (type === 'BDR') return { label: 'BDR', color: 'var(--color-warning)' };
  if (type === 'STOCK' || type === 'AÇÃO' || type === 'ACAO') return { label: 'Ação', color: 'var(--color-success)' };
  if (type === 'CRYPTO') return { label: 'Crypto', color: 'var(--color-warning)' };
  return null;
};

export default function DailyReport({ positions, fiPositions = [], treasuryPositions = [], kpiCurrency }: DailyReportProps) {
  const [sortKey, setSortKey] = useState<SortKey>('impact');
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
    const rate = getExchangeRate(pos, kpiCurrency);
    totalDailyChange += (pos.daily_change ?? 0) * (pos.quantity ?? 0) * rate;
    totalPortfolioValue += (pos.current_value ?? 0);
  });

  const totalFIPrivateValue = fiPositions.reduce((s, p) => s + (p.net_value ?? 0), 0);
  const totalTreasuryValue = treasuryPositions.reduce((s, p) => s + (p.net_value ?? 0), 0);
  totalPortfolioValue += totalFIPrivateValue + totalTreasuryValue;

  // Variação % total ponderada = totalDailyChange / (valor_anterior = totalPortfolioValue - totalDailyChange)
  const previousTotalValue = totalPortfolioValue - totalDailyChange;
  const totalDailyPercent = Math.abs(previousTotalValue) > 1e-6
    ? (totalDailyChange / previousTotalValue) * 100
    : 0;

  const isDailyPos = totalDailyChange >= 0;

  // Enriquece todas as posições com campos calculados para o resumo
  const enrichedPositions = positions.map(pos => {
    const percent = pos.daily_change_percent ?? 0;
    const absChange = pos.daily_change ?? 0;
    const currentPrice = pos.current_price ?? 0;
    const previousClose = currentPrice - absChange;
    const qty = pos.quantity ?? 0;
    const rate = getExchangeRate(pos, kpiCurrency);
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

  // Helper para ícone de ordenação
  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return <span style={{ opacity: 0.3, marginLeft: '4px' }}>⇅</span>;
    return <span style={{ marginLeft: '4px' }}>{sortDir === 'asc' ? '↑' : '↓'}</span>;
  };

  return (
    <div className="flex-col gap-xl w-full">

      {/* Card principal: Variação Total */}
      {(positions.length > 0 || fiPositions.length > 0 || treasuryPositions.length > 0) && (
        <div className="card flex-col items-center justify-center text-center w-full" style={{ padding: '1.5rem', gap: '0.25rem' }}>
          <span className="text-secondary text-sm font-semibold" style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Variação Total Diária da Carteira
          </span>
          <span className="text-3xl font-bold mt-sm" style={{ color: isDailyPos ? 'var(--color-success)' : 'var(--color-danger)', letterSpacing: '-0.02em' }}>
            {isDailyPos ? '🟢 +' : '🔴 '}{formatMoney(totalDailyChange, kpiCurrency)}
          </span>
          <span className="text-lg font-semibold" style={{ color: isDailyPos ? 'var(--color-success)' : 'var(--color-danger)' }}>
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
                  <div key={pos.asset_id} className="flex-row justify-between items-center" style={{ padding: '0.5rem 0.75rem', background: 'var(--color-success-bg)', borderRadius: '8px', borderLeft: '3px solid var(--color-success)' }}>
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
                  <div key={pos.asset_id} className="flex-row justify-between items-center" style={{ padding: '0.5rem 0.75rem', background: 'var(--color-danger-bg)', borderRadius: '8px', borderLeft: '3px solid var(--color-danger)' }}>
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
        <h3 className="card-title">📊 Resumo Diário Completo (Renda Variável)</h3>
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
                        <div style={{ width: '100%', background: 'var(--input-bg)', borderRadius: '3px', height: '3px', marginTop: '4px' }}>
                          <div style={{ width: `${Math.min(portfolioWeight, 100)}%`, background: 'var(--accent-color)', borderRadius: '3px', height: '3px' }} />
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

      {/* Renda Fixa Privada (CDB, LCI, LCA, Debêntures) */}
      {fiPositions.length > 0 && (
        <div className="card flex-col gap-md w-full">
          <h3 className="card-title">🏛️ Posição Atualizada: Renda Fixa Privada (CDB/LCI/LCA)</h3>
          <p className="text-xs text-secondary">
            Títulos de renda fixa privada são atualizados diariamente com a rentabilidade acumulada de acordo com o indexador contratado.
          </p>
          <div className="table-container flex-col">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Instituição & Ativo</th>
                  <th>Tipo / Taxa</th>
                  <th className="text-right">Vencimento</th>
                  <th className="text-right">Total Investido</th>
                  <th className="text-right">Valor Líquido</th>
                  <th className="text-right">Rentabilidade Acumulada</th>
                </tr>
              </thead>
              <tbody>
                {fiPositions.map(p => {
                  const returnPct = p.net_return_percent ?? 0;
                  const taxa = p.asset.debt_type === 'POS'
                    ? `${p.asset.rate.toFixed(2)}% ${p.asset.indexer}`
                    : `${p.asset.rate.toFixed(2)}% a.a.`;

                  return (
                    <tr key={p.asset.id}>
                      <td>
                        <div className="flex-col">
                          <span className="font-bold">{p.asset.institution}</span>
                          <span className="text-xs text-secondary">{p.asset.type}</span>
                        </div>
                      </td>
                      <td>
                        <div className="flex-row items-center gap-sm">
                          <span style={{ background: 'var(--accent-bg)', color: 'var(--accent-color)', border: '1px solid rgba(var(--accent-rgb), 0.3)', fontSize: '0.65rem', padding: '2px 6px', borderRadius: '4px', fontWeight: 700 }}>
                            {p.asset.type}
                          </span>
                          <span className="text-xs">{taxa}</span>
                        </div>
                      </td>
                      <td className="text-right">{new Date(p.asset.maturity_date).toLocaleDateString('pt-BR')}</td>
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

      {/* Tesouro Direto */}
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
                  const isSelic = p.treasury_type === 'SELIC';
                  const isPrefix = p.treasury_type === 'PREFIXADO';
                  const colorVar = isSelic ? 'var(--color-success)' : isPrefix ? 'var(--accent-color)' : 'var(--color-warning)';
                  const bgVar = isSelic ? 'var(--color-success-bg)' : isPrefix ? 'var(--accent-bg)' : 'var(--color-warning-bg)';
                  return (
                    <tr key={p.transaction_id}>
                      <td><span className="font-bold">{p.ticker}</span></td>
                      <td>
                        <span style={{ background: bgVar, color: colorVar, border: `1px solid ${colorVar}`, fontSize: '0.65rem', padding: '2px 6px', borderRadius: '4px', fontWeight: 700 }}>
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

