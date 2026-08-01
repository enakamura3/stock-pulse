import React, { useMemo } from 'react';
import { Position, FixedIncomePosition, TreasuryPosition } from '../types';
import { formatMoney } from '../helpers';
import { isFII, isAcaoOuETF } from './constants';
import { SectionTitle, AnalysisCard, AlertBadge } from './sharedComponents';

interface FundamentalHealthSectionProps {
  positions: Position[];
  fiPositions: FixedIncomePosition[];
  treasuryPositions: TreasuryPosition[];
  kpiCurrency: string;
}

const TOP_N = 8;

export default function FundamentalHealthSection({
  positions,
  fiPositions,
  treasuryPositions,
  kpiCurrency,
}: FundamentalHealthSectionProps) {
  const fundamentals = useMemo(() => {
    // P/VP médio (FIIs e FIAGROs)
    const fiiPositions = positions.filter(p => isFII(p.type) && p.pvp && p.pvp > 0 && p.current_value && p.current_value > 0);
    let avgPVP: number | null = null;
    if (fiiPositions.length > 0) {
      const totalWeight = fiiPositions.reduce((s, p) => s + (p.current_value || 0), 0);
      if (totalWeight > 1e-6) {
        avgPVP = fiiPositions.reduce((s, p) => s + (p.pvp! * (p.current_value || 0)), 0) / totalWeight;
      }
    }

    // P/L médio (Ações e ETFs)
    const stockPositions = positions.filter(p => isAcaoOuETF(p.type) && p.pe && p.pe > 0 && p.current_value && p.current_value > 0);
    let avgPE: number | null = null;
    if (stockPositions.length > 0) {
      const totalWeight = stockPositions.reduce((s, p) => s + (p.current_value || 0), 0);
      if (totalWeight > 1e-6) {
        avgPE = stockPositions.reduce((s, p) => s + (p.pe! * (p.current_value || 0)), 0) / totalWeight;
      }
    }

    // DY médio
    const dyPositions = positions.filter(p => p.dividend_yield && p.dividend_yield > 0 && p.current_value && p.current_value > 0);
    let avgDY: number | null = null;
    if (dyPositions.length > 0) {
      const totalWeight = dyPositions.reduce((s, p) => s + (p.current_value || 0), 0);
      if (totalWeight > 1e-6) {
        avgDY = dyPositions.reduce((s, p) => s + (p.dividend_yield! * (p.current_value || 0)), 0) / totalWeight;
      }
    }

    return { avgPVP, avgPE, avgDY, fiiCount: fiiPositions.length, stockCount: stockPositions.length };
  }, [positions]);

  const rankedPositions = useMemo(() =>
    [...positions]
      .filter(p => p.return_percent !== undefined)
      .sort((a, b) => (b.return_percent || 0) - (a.return_percent || 0)),
    [positions]
  );
  const topPerformers = rankedPositions.slice(0, TOP_N);
  const worstPerformers = [...rankedPositions].reverse().slice(0, TOP_N);

  const maxAbsReturn = Math.max(...[...topPerformers, ...worstPerformers].map(p => Math.abs(p.return_percent || 0)), 1);

  const valuationData = useMemo(() => {
    const withGraham = positions.filter(p => p.graham_value && p.graham_value > 0 && p.current_price && p.current_price > 0);
    const withBazin = positions.filter(p => p.bazin_value && p.bazin_value > 0 && p.current_price && p.current_price > 0);
    
    const grahamItems = withGraham.map(p => {
        const discount = ((p.graham_value! - p.current_price!) / p.graham_value!) * 100;
        return { ticker: p.ticker, discount, graham: p.graham_value, current: p.current_price };
    }).sort((a, b) => b.discount - a.discount);

    const bazinItems = withBazin.map(p => {
        const discount = ((p.bazin_value! - p.current_price!) / p.bazin_value!) * 100;
        return { ticker: p.ticker, discount, bazin: p.bazin_value, current: p.current_price };
    }).sort((a, b) => b.discount - a.discount);

    return { grahamItems, bazinItems };
  }, [positions]);

  const fiLiquidity = useMemo(() => {
    let daily = 0;
    let upTo1Year = 0;
    let upTo3Years = 0;
    let longTerm = 0;

    fiPositions.forEach(p => {
      if (p.days_to_maturity <= 0) {
        daily += p.net_value;
      } else if (p.days_to_maturity <= 365) {
        upTo1Year += p.net_value;
      } else if (p.days_to_maturity <= 1095) {
        upTo3Years += p.net_value;
      } else {
        longTerm += p.net_value;
      }
    });

    treasuryPositions.forEach(p => {
      if (p.days_to_maturity <= 0) {
        daily += p.net_value;
      } else if (p.days_to_maturity <= 365) {
        upTo1Year += p.net_value;
      } else if (p.days_to_maturity <= 1095) {
        upTo3Years += p.net_value;
      } else {
        longTerm += p.net_value;
      }
    });

    return [
      { label: 'Liquidez Diária / Vencido', value: daily, color: '#4ade80' },
      { label: 'Até 1 ano', value: upTo1Year, color: '#60a5fa' },
      { label: '1 a 3 anos', value: upTo3Years, color: '#fbbf24' },
      { label: 'Longo Prazo (> 3 anos)', value: longTerm, color: '#f87171' },
    ].filter(i => i.value > 0);
  }, [fiPositions, treasuryPositions]);

  return (
    <>
      {/* SEÇÃO: Fundamentos da Carteira */}
      <AnalysisCard id="section-fundamentals">
        <SectionTitle
          emoji="🏛️"
          title="Fundamentos da Carteira"
          subtitle="Múltiplos médios ponderados pelo valor de mercado dos seus ativos"
        />

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(min(280px, 100%), 1fr))', gap: '1rem' }}>
          {/* P/VP — FIIs */}
          <div style={{
            background: 'linear-gradient(145deg, rgba(192,132,252,0.06) 0%, rgba(192,132,252,0.01) 100%)',
            border: '1px solid rgba(192,132,252,0.15)',
            borderRadius: '14px',
            padding: '1.25rem',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.75rem' }}>
              <span style={{ fontSize: '1.1rem' }}>🏢</span>
              <span style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)' }}>
                P/VP Médio
              </span>
            </div>
            <div style={{ fontSize: '1.75rem', fontWeight: 800, color: '#c084fc', fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.02em' }}>
              {fundamentals.avgPVP !== null ? fundamentals.avgPVP.toFixed(2) : '—'}
            </div>
            <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.4rem', lineHeight: 1.4 }}>
              {fundamentals.avgPVP !== null ? (
                <>
                  Baseado em <strong style={{ color: 'var(--text-primary)' }}>{fundamentals.fiiCount}</strong> {fundamentals.fiiCount === 1 ? 'FII/FIAGRO' : 'FIIs/FIAGROs'}
                  {fundamentals.avgPVP < 0.95 - 1e-6 && (
                    <span style={{ display: 'block', marginTop: '0.3rem', color: '#4ade80' }}>
                      ✅ Abaixo do VP — carteira com desconto patrimonial
                    </span>
                  )}
                  {fundamentals.avgPVP >= 0.95 - 1e-6 && fundamentals.avgPVP <= 1.05 + 1e-6 && (
                    <span style={{ display: 'block', marginTop: '0.3rem', color: '#fbbf24' }}>
                      ⚠️ Próximo ao VP — avalie com cuidado novas compras
                    </span>
                  )}
                  {fundamentals.avgPVP > 1.05 + 1e-6 && (
                    <span style={{ display: 'block', marginTop: '0.3rem', color: '#f87171' }}>
                      ⚠️ Acima do VP — prêmio sobre o patrimônio
                    </span>
                  )}
                </>
              ) : (
                'Sem FIIs/FIAGROs com P/VP disponível'
              )}
            </div>
          </div>

          {/* P/L — Ações e ETFs */}
          <div style={{
            background: 'linear-gradient(145deg, rgba(96,165,250,0.06) 0%, rgba(96,165,250,0.01) 100%)',
            border: '1px solid rgba(96,165,250,0.15)',
            borderRadius: '14px',
            padding: '1.25rem',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.75rem' }}>
              <span style={{ fontSize: '1.1rem' }}>📊</span>
              <span style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)' }}>
                P/L Médio
              </span>
            </div>
            <div style={{ fontSize: '1.75rem', fontWeight: 800, color: '#60a5fa', fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.02em' }}>
              {fundamentals.avgPE !== null ? fundamentals.avgPE.toFixed(1) + 'x' : '—'}
            </div>
            <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.4rem', lineHeight: 1.4 }}>
              {fundamentals.avgPE !== null ? (
                <>
                  Baseado em <strong style={{ color: 'var(--text-primary)' }}>{fundamentals.stockCount}</strong> {fundamentals.stockCount === 1 ? 'ativo' : 'ativos'} (Ações/ETFs/BDRs)
                  {fundamentals.avgPE < 10 - 1e-6 && (
                    <span style={{ display: 'block', marginTop: '0.3rem', color: '#4ade80' }}>
                      ✅ P/L atrativo — carteira potencialmente subvalorizada
                    </span>
                  )}
                  {fundamentals.avgPE >= 10 - 1e-6 && fundamentals.avgPE <= 18 + 1e-6 && (
                    <span style={{ display: 'block', marginTop: '0.3rem', color: '#fbbf24' }}>
                      💡 P/L na média do mercado brasileiro
                    </span>
                  )}
                  {fundamentals.avgPE > 18 + 1e-6 && (
                    <span style={{ display: 'block', marginTop: '0.3rem', color: '#f87171' }}>
                      ⚠️ P/L elevado — expectativa de crescimento precificada
                    </span>
                  )}
                </>
              ) : (
                'Sem ações/ETFs com P/L disponível'
              )}
            </div>
          </div>

          {/* DY Médio */}
          {fundamentals.avgDY !== null && (
            <div style={{
              background: 'linear-gradient(145deg, rgba(74,222,128,0.06) 0%, rgba(74,222,128,0.01) 100%)',
              border: '1px solid rgba(74,222,128,0.15)',
              borderRadius: '14px',
              padding: '1.25rem',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.75rem' }}>
                <span style={{ fontSize: '1.1rem' }}>💸</span>
                <span style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)' }}>
                  DY Médio
                </span>
              </div>
              <div style={{ fontSize: '1.75rem', fontWeight: 800, color: '#4ade80', fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.02em' }}>
                {fundamentals.avgDY.toFixed(2)}%
              </div>
              <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.4rem', lineHeight: 1.4 }}>
                Dividend Yield médio ponderado dos ativos com pagamento de proventos
              </div>
            </div>
          )}
        </div>
      </AnalysisCard>
    
      {/* Top & Worst Performers */}
      <AnalysisCard>
        <SectionTitle emoji="🏆" title="Top Performers vs Piores" subtitle="Rentabilidade acumulada por ativo na sua carteira" />
        {topPerformers.length === 0 ? (
          <AlertBadge type="info" message="Sem dados de rentabilidade disponíveis." />
        ) : (
          <>
            <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: '#4ade80', marginBottom: '0.5rem' }}>Top {Math.min(TOP_N, topPerformers.length)} melhores</p>
            {topPerformers.map(p => (
              <div key={p.ticker} style={{ display: 'flex', alignItems: 'center', marginBottom: '0.5rem', gap: '0.5rem' }}>
                <span style={{ width: '52px', fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-primary)', flexShrink: 0 }}>{p.ticker}</span>
                <div style={{ flex: 1, background: 'rgba(255,255,255,0.05)', borderRadius: '4px', height: '10px', overflow: 'hidden' }}>
                  <div style={{
                    height: '100%',
                    width: `${((p.return_percent || 0) / maxAbsReturn) * 100}%`,
                    background: 'linear-gradient(90deg, #4ade80, #00e676)',
                    borderRadius: '4px',
                    transition: 'width 0.6s ease',
                  }} />
                </div>
                <span style={{ width: '60px', textAlign: 'right', fontSize: '0.8rem', fontWeight: 700, color: '#4ade80', fontVariantNumeric: 'tabular-nums', flexShrink: 0 }}>
                  +{(p.return_percent || 0).toFixed(1)}%
                </span>
              </div>
            ))}

            <div style={{ borderTop: '1px solid rgba(255,255,255,0.06)', margin: '0.75rem 0' }} />

            <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: '#f87171', marginBottom: '0.5rem' }}>Top {Math.min(TOP_N, worstPerformers.length)} piores</p>
            {worstPerformers.map(p => (
              <div key={p.ticker} style={{ display: 'flex', alignItems: 'center', marginBottom: '0.5rem', gap: '0.5rem' }}>
                <span style={{ width: '52px', fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-primary)', flexShrink: 0 }}>{p.ticker}</span>
                <div style={{ flex: 1, background: 'rgba(255,255,255,0.05)', borderRadius: '4px', height: '10px', overflow: 'hidden' }}>
                  <div style={{
                    height: '100%',
                    width: `${(Math.abs(p.return_percent || 0) / maxAbsReturn) * 100}%`,
                    background: 'linear-gradient(90deg, #f87171, #ef4444)',
                    borderRadius: '4px',
                    transition: 'width 0.6s ease',
                  }} />
                </div>
                <span style={{ width: '60px', textAlign: 'right', fontSize: '0.8rem', fontWeight: 700, color: '#f87171', fontVariantNumeric: 'tabular-nums', flexShrink: 0 }}>
                  {(p.return_percent || 0).toFixed(1)}%
                </span>
              </div>
            ))}
          </>
        )}
      </AnalysisCard>

      {/* Valuation & Margem de Segurança */}
      <AnalysisCard>
        <SectionTitle emoji="⚖️" title="Valuation e Descontos" subtitle="Ativos com maior margem de segurança na carteira" />
        
        {valuationData.grahamItems.length === 0 && valuationData.bazinItems.length === 0 ? (
          <AlertBadge type="info" message="Não há dados suficientes de fundamentos para calcular margem de segurança." />
        ) : (
          <>
             {valuationData.grahamItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>Preço Teto - Graham</p>
                  {valuationData.grahamItems.slice(0, 6).map(item => (
                    <div key={`graham-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.graham!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}

             {valuationData.bazinItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: '0.5rem' }}>Preço Teto - Bazin</p>
                  {valuationData.bazinItems.slice(0, 6).map(item => (
                    <div key={`bazin-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.bazin!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}
          </>
        )}
      </AnalysisCard>

      {/* Liquidez e Renda Fixa */}
      {(fiPositions.length > 0 || treasuryPositions.length > 0) && (
        <AnalysisCard>
          <SectionTitle emoji="💧" title="Liquidez da Renda Fixa" subtitle="Perfil de vencimento dos seus ativos" />
          
          <div style={{ display: 'flex', height: '12px', borderRadius: '6px', overflow: 'hidden', marginBottom: '1rem' }}>
            {fiLiquidity.map((item, i) => (
              <div key={i} style={{ width: `${(item.value / fiLiquidity.reduce((s, x) => s + x.value, 0)) * 100}%`, background: item.color }} title={`${item.label}: ${formatMoney(item.value, kpiCurrency)}`} />
            ))}
          </div>
          
          <div>
            {fiLiquidity.map((item, i) => {
               const totalLiquidity = fiLiquidity.reduce((s, x) => s + x.value, 0);
               const pct = totalLiquidity > 0 ? (item.value / totalLiquidity) * 100 : 0;
               return (
                 <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                   <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                     <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: item.color }} />
                     <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{item.label}</span>
                   </div>
                   <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{pct.toFixed(1)}%</span>
                 </div>
               );
            })}
          </div>
        </AnalysisCard>
      )}
    </>
  );
}
