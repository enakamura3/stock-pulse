import React from 'react';
import { FixedIncomePosition, TreasuryPosition } from '../types';
import { formatMoney, calculateDailyFixedIncomeRate, calculateEstimatedDailyGain } from '../helpers';

export interface DailyReportFixedIncomeProps {
  fiPositions?: FixedIncomePosition[];
  treasuryPositions?: TreasuryPosition[];
  kpiCurrency: string;
}

export default function DailyReportFixedIncome({
  fiPositions = [],
  treasuryPositions = [],
  kpiCurrency,
}: DailyReportFixedIncomeProps) {
  return (
    <>
      {/* Renda Fixa Privada (CDB, LCI, LCA, Debêntures) */}
      {fiPositions.length > 0 && (
        <div className="card flex-col gap-md w-full">
          <h3 className="card-title">🏛️ Posição Atualizada: Renda Fixa Privada (CDB/LCI/LCA)</h3>
          <p className="text-xs text-secondary">
            Títulos de renda fixa privada são atualizados diariamente com a rentabilidade acumulada de acordo com o indexador contratado.
          </p>
          <div className="table-container flex-col" style={{ overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
            <table className="data-table" style={{ width: '100%', minWidth: '600px' }} aria-label="Posição atualizada de renda fixa privada">
              <caption className="sr-only">Posição atualizada de renda fixa privada</caption>
              <thead>
                <tr>
                  <th scope="col">Instituição & Ativo</th>
                  <th scope="col">Tipo / Taxa</th>
                  <th scope="col" className="text-right">Vencimento</th>
                  <th scope="col" className="text-right">Valor Líquido</th>
                  <th scope="col" className="text-right">Taxa Diária Est.</th>
                  <th scope="col" className="text-right">Ganho Diário Est.</th>
                  <th scope="col" className="text-right">Rent. Acumulada</th>
                </tr>
              </thead>
              <tbody>
                {fiPositions.map(p => {
                  const returnPct = p.net_return_percent ?? 0;
                  const taxa = p.asset.debt_type === 'POS'
                    ? `${p.asset.rate.toFixed(2)}% ${p.asset.indexer}`
                    : `${p.asset.rate.toFixed(2)}% a.a.`;
                  const dailyRatePct = calculateDailyFixedIncomeRate(p.asset.indexer || p.asset.debt_type, p.asset.rate);
                  const dailyGain = calculateEstimatedDailyGain(p.net_value ?? 0, dailyRatePct);

                  return (
                    <tr key={p.asset.id}>
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <div className="flex-col">
                          <span className="font-bold">{p.asset.institution}</span>
                          <span className="text-xs text-secondary">{p.asset.type}</span>
                        </div>
                      </td>
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <div className="flex-row items-center gap-sm">
                          <span
                            style={{
                              background: 'var(--accent-bg)',
                              color: 'var(--accent-color)',
                              border: '1px solid rgba(var(--accent-rgb), 0.3)',
                              fontSize: '0.65rem',
                              padding: '2px 6px',
                              borderRadius: '4px',
                              fontWeight: 700,
                            }}
                          >
                            {p.asset.type}
                          </span>
                          <span className="text-xs">{taxa}</span>
                        </div>
                      </td>
                      <td className="text-right" style={{ whiteSpace: 'nowrap' }}>
                        {new Date(p.asset.maturity_date).toLocaleDateString('pt-BR')}
                      </td>
                      <td className="text-right" style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>
                        {formatMoney(p.net_value, kpiCurrency)}
                      </td>
                      <td className="text-right text-xs text-secondary font-semibold" style={{ whiteSpace: 'nowrap' }}>
                        +{dailyRatePct.toFixed(4)}%/dia
                      </td>
                      <td className="text-right font-bold text-success" style={{ whiteSpace: 'nowrap' }}>
                        +{formatMoney(dailyGain, kpiCurrency)}
                      </td>
                      <td
                        className={`text-right font-bold ${returnPct >= 0 ? 'text-success' : 'text-danger'}`}
                        style={{ whiteSpace: 'nowrap' }}
                      >
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
          <div className="table-container flex-col" style={{ overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
            <table className="data-table" style={{ width: '100%', minWidth: '600px' }} aria-label="Posição atualizada de títulos do tesouro direto">
              <caption className="sr-only">Posição atualizada do tesouro direto</caption>
              <thead>
                <tr>
                  <th scope="col">Título</th>
                  <th scope="col">Tipo</th>
                  <th scope="col" className="text-right">Vencimento</th>
                  <th scope="col" className="text-right">Valor Líquido</th>
                  <th scope="col" className="text-right">Taxa Diária Est.</th>
                  <th scope="col" className="text-right">Ganho Diário Est.</th>
                  <th scope="col" className="text-right">Rent. Acumulada</th>
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
                  const dailyRatePct = calculateDailyFixedIncomeRate(p.treasury_type, p.contracted_rate ?? 0);
                  const dailyGain = calculateEstimatedDailyGain(p.net_value ?? 0, dailyRatePct);

                  return (
                    <tr key={p.transaction_id}>
                      <td style={{ whiteSpace: 'nowrap' }}><span className="font-bold">{p.ticker}</span></td>
                      <td style={{ whiteSpace: 'nowrap' }}>
                        <span style={{ background: bgVar, color: colorVar, border: `1px solid ${colorVar}`, fontSize: '0.65rem', padding: '2px 6px', borderRadius: '4px', fontWeight: 700 }}>
                          {p.treasury_type}
                        </span>
                      </td>
                      <td className="text-right" style={{ whiteSpace: 'nowrap' }}>{new Date(p.maturity_date).toLocaleDateString('pt-BR')}</td>
                      <td className="text-right" style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{formatMoney(p.net_value, kpiCurrency)}</td>
                      <td className="text-right text-xs text-secondary font-semibold" style={{ whiteSpace: 'nowrap' }}>
                        +{dailyRatePct.toFixed(4)}%/dia
                      </td>
                      <td className="text-right font-bold text-success" style={{ whiteSpace: 'nowrap' }}>
                        +{formatMoney(dailyGain, kpiCurrency)}
                      </td>
                      <td className={`text-right font-bold ${returnPct >= 0 ? 'text-success' : 'text-danger'}`} style={{ whiteSpace: 'nowrap' }}>
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
    </>
  );
}
