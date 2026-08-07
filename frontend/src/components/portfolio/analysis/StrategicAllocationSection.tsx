import React, { useState, useEffect, useMemo } from 'react';
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from 'recharts';
import { Position, FixedIncomePosition, TreasuryPosition } from '../types';
import { formatMoney, getAssetCategory } from '../helpers';
import { ALLOCATION_COLORS, CATEGORY_COLORS, EXPOSURE_COLORS, isExposicaoGlobal } from './constants';
import { SectionTitle, AnalysisCard, ProgressBar, AlertBadge, StatPill } from './sharedComponents';

interface StrategicAllocationSectionProps {
  positions: Position[];
  fiPositions: FixedIncomePosition[];
  treasuryPositions: TreasuryPosition[];
  kpiCurrency: string;
}

export default function StrategicAllocationSection({
  positions,
  fiPositions,
  treasuryPositions,
  kpiCurrency,
}: StrategicAllocationSectionProps) {
  const [targetAlloc, setTargetAlloc] = useState<Record<string, number>>({});
  const [isEditing, setIsEditing] = useState(false);
  const [contributionInput, setContributionInput] = useState<string>('1000');

  useEffect(() => {
    try {
      const saved = localStorage.getItem('stockpulse_target_allocation');
      if (saved) {
        setTargetAlloc(JSON.parse(saved));
      }
    } catch (e) {
      console.error('Erro ao carregar metas de alocação:', e);
    }
  }, []);

  const saveTargetAlloc = (newTargets: Record<string, number>) => {
    setTargetAlloc(newTargets);
    try {
      localStorage.setItem('stockpulse_target_allocation', JSON.stringify(newTargets));
    } catch (e) {
      console.error('Erro ao salvar metas de alocação:', e);
    }
  };

  const totalPortfolioValue = useMemo(() => {
    const eq = positions.reduce((s, p) => s + (p.current_value || 0), 0);
    const fi = fiPositions.reduce((s, p) => s + p.net_value, 0);
    const td = treasuryPositions.reduce((s, p) => s + p.net_value, 0);
    return eq + fi + td;
  }, [positions, fiPositions, treasuryPositions]);

  // 1a. Classe de Ativo: Renda Fixa vs Renda Variável
  const assetClassData = useMemo(() => {
    const rv = positions.reduce((s, p) => s + (p.current_value || 0), 0);
    const rf = fiPositions.reduce((s, p) => s + p.net_value, 0);
    const td = treasuryPositions.reduce((s, p) => s + p.net_value, 0);
    const totalRF = rf + td;
    if (rv + totalRF < 1e-6) return [];
    return [
      { name: 'Renda Variável', value: rv, pct: (rv / (rv + totalRF)) * 100 },
      { name: 'Renda Fixa', value: totalRF, pct: (totalRF / (rv + totalRF)) * 100 },
    ].filter(d => d.value > 1e-6);
  }, [positions, fiPositions, treasuryPositions]);

  // 1a-extra. Detalhamento por categoria
  const categoryBreakdown = useMemo(() => {
    const map: Record<string, number> = {};
    positions.forEach(p => {
      const cat = getAssetCategory(p.type);
      map[cat] = (map[cat] || 0) + (p.current_value || 0);
    });
    const fiTotal = fiPositions.reduce((s, p) => s + p.net_value, 0);
    if (fiTotal > 1e-6) map['Renda Fixa'] = (map['Renda Fixa'] || 0) + fiTotal;
    const tdTotal = treasuryPositions.reduce((s, p) => s + p.net_value, 0);
    if (tdTotal > 1e-6) map['Tesouro Direto'] = (map['Tesouro Direto'] || 0) + tdTotal;
    return Object.entries(map)
      .map(([name, value]) => ({
        name,
        value,
        pct: totalPortfolioValue > 1e-6 ? (value / totalPortfolioValue) * 100 : 0,
      }))
      .filter(d => d.value > 1e-6)
      .sort((a, b) => b.value - a.value);
  }, [positions, fiPositions, treasuryPositions, totalPortfolioValue]);

  // Rebalanceamento: Cálculo dos Gaps
  const rebalanceMetrics = useMemo(() => {
    const categories = new Set<string>([
      ...categoryBreakdown.map(c => c.name),
      ...Object.keys(targetAlloc),
    ]);

    const items = Array.from(categories).map(catName => {
      const catObj = categoryBreakdown.find(c => c.name === catName);
      const currentPct = catObj ? catObj.pct : 0;
      const currentValue = catObj ? catObj.value : 0;
      const targetPct = targetAlloc[catName] || 0;
      const gapPct = targetPct - currentPct;
      const gapMoney = (gapPct / 100) * totalPortfolioValue;

      return {
        name: catName,
        currentPct,
        currentValue,
        targetPct,
        gapPct,
        gapMoney,
      };
    }).sort((a, b) => b.targetPct - a.targetPct || b.currentPct - a.currentPct);

    const totalTargetSum = Object.values(targetAlloc).reduce((s, v) => s + v, 0);

    return {
      items,
      totalTargetSum,
    };
  }, [categoryBreakdown, targetAlloc, totalPortfolioValue]);

  // Sugestão de Aporte Proporcional aos Gaps
  const contributionSuggestions = useMemo(() => {
    const contributionVal = parseFloat(contributionInput.replace(',', '.')) || 0;
    if (contributionVal <= 0) return [];

    const subAllocated = rebalanceMetrics.items.filter(i => i.gapPct > 0);
    const sumGapPct = subAllocated.reduce((s, i) => s + i.gapPct, 0);

    if (sumGapPct < 1e-6) return [];

    return subAllocated.map(item => {
      const suggestedAmount = (item.gapPct / sumGapPct) * contributionVal;
      return {
        name: item.name,
        amount: Number(suggestedAmount.toFixed(2)),
        gapPct: item.gapPct,
      };
    }).sort((a, b) => b.amount - a.amount);
  }, [rebalanceMetrics, contributionInput]);

  // 1b. Exposição Cambial e Geográfica
  const geoExposureData = useMemo(() => {
    const globalVal = positions
      .filter(p => isExposicaoGlobal(p.type))
      .reduce((s, p) => s + (p.current_value || 0), 0);

    const localEquity = positions
      .filter(p => !isExposicaoGlobal(p.type))
      .reduce((s, p) => s + (p.current_value || 0), 0);

    const fi = fiPositions.reduce((s, p) => s + p.net_value, 0);
    const td = treasuryPositions.reduce((s, p) => s + p.net_value, 0);
    const localTotal = localEquity + fi + td;

    if (localTotal + globalVal < 1e-6) return [];
    const total = localTotal + globalVal;
    return [
      { name: '🇧🇷 Risco Local', value: localTotal, pct: (localTotal / total) * 100 },
      { name: '🌍 Exposição Global', value: globalVal, pct: (globalVal / total) * 100 },
    ].filter(d => d.value > 1e-6);
  }, [positions, fiPositions, treasuryPositions]);

  const PieTooltip = ({ active, payload }: any) => {
    if (!active || !payload || payload.length === 0) return null;
    const d = payload[0].payload;
    return (
      <div style={{
        background: 'rgba(15, 23, 42, 0.95)',
        border: '1px solid rgba(255,255,255,0.1)',
        padding: '0.75rem 1rem',
        borderRadius: '10px',
        boxShadow: '0 8px 32px rgba(0,0,0,0.45)',
        backdropFilter: 'blur(12px)',
      }}>
        <p style={{ margin: '0 0 0.3rem 0', fontWeight: 700, color: '#fff', fontSize: '0.85rem' }}>{d.name}</p>
        <p style={{ margin: 0, fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
          {formatMoney(d.value, kpiCurrency)} ({d.pct.toFixed(1)}%)
        </p>
      </div>
    );
  };

  const renderPieLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, pct }: any) => {
    if (pct < 3) return null;
    const RADIAN = Math.PI / 180;
    const radius = innerRadius + (outerRadius - innerRadius) * 0.5;
    const x = cx + radius * Math.cos(-midAngle * RADIAN);
    const y = cy + radius * Math.sin(-midAngle * RADIAN);
    return (
      <text x={x} y={y} fill="#fff" textAnchor="middle" dominantBaseline="central" fontSize="0.7rem" fontWeight={700}>
        {pct.toFixed(0)}%
      </text>
    );
  };

  return (
    <AnalysisCard id="section-allocation">
      <SectionTitle
        emoji="🎯"
        title="Alocação Estratégica"
        subtitle="Distribuição do patrimônio por classe de ativo e exposição geográfica"
      />

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(min(340px, 100%), 1fr))', gap: '1.5rem' }}>
        {/* Donut: Classe de Ativo */}
        <div>
          <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)', marginBottom: '0.75rem', textAlign: 'center' }}>
            Classe de Ativo
          </p>
          {assetClassData.length > 0 ? (
            <ResponsiveContainer width="100%" height={220}>
              <PieChart>
                <Pie
                  data={assetClassData}
                  cx="50%"
                  cy="50%"
                  innerRadius={55}
                  outerRadius={90}
                  paddingAngle={3}
                  dataKey="value"
                  stroke="none"
                  label={renderPieLabel}
                  labelLine={false}
                  animationDuration={800}
                  animationEasing="ease-out"
                >
                  {assetClassData.map((entry, i) => (
                    <Cell key={i} fill={ALLOCATION_COLORS[entry.name] || '#94a3b8'} />
                  ))}
                </Pie>
                <Tooltip content={<PieTooltip />} />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div style={{ height: '220px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
              Sem dados
            </div>
          )}
          {/* Legend */}
          <div style={{ display: 'flex', justifyContent: 'center', gap: '1.25rem', marginTop: '0.25rem' }}>
            {assetClassData.map(d => (
              <div key={d.name} style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <div style={{ width: '10px', height: '10px', borderRadius: '3px', background: ALLOCATION_COLORS[d.name] || '#94a3b8' }} />
                <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>{d.name}</span>
                <span style={{ fontSize: '0.75rem', fontWeight: 700, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>{d.pct.toFixed(1)}%</span>
              </div>
            ))}
          </div>
        </div>

        {/* Donut: Exposição Geográfica */}
        <div>
          <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)', marginBottom: '0.75rem', textAlign: 'center' }}>
            Exposição Cambial e Geográfica
          </p>
          {geoExposureData.length > 0 ? (
            <ResponsiveContainer width="100%" height={220}>
              <PieChart>
                <Pie
                  data={geoExposureData}
                  cx="50%"
                  cy="50%"
                  innerRadius={55}
                  outerRadius={90}
                  paddingAngle={3}
                  dataKey="value"
                  stroke="none"
                  label={renderPieLabel}
                  labelLine={false}
                  animationDuration={800}
                  animationEasing="ease-out"
                >
                  <Cell fill={EXPOSURE_COLORS.local} />
                  <Cell fill={EXPOSURE_COLORS.global} />
                </Pie>
                <Tooltip content={<PieTooltip />} />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div style={{ height: '220px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
              Sem dados
            </div>
          )}
          {/* Legend */}
          <div style={{ display: 'flex', justifyContent: 'center', gap: '1.25rem', marginTop: '0.25rem' }}>
            {geoExposureData.map((d, i) => (
              <div key={d.name} style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <div style={{ width: '10px', height: '10px', borderRadius: '3px', background: i === 0 ? EXPOSURE_COLORS.local : EXPOSURE_COLORS.global }} />
                <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>{d.name}</span>
                <span style={{ fontSize: '0.75rem', fontWeight: 700, color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>{d.pct.toFixed(1)}%</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Detalhamento por categoria — minibar */}
      {categoryBreakdown.length > 0 && (
        <div style={{ marginTop: '1.25rem', padding: '1rem', background: 'rgba(255,255,255,0.02)', borderRadius: '12px', border: '1px solid rgba(255,255,255,0.04)' }}>
          <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)', marginBottom: '0.75rem' }}>
            Detalhamento por Categoria
          </p>
          {categoryBreakdown.map(cat => (
            <div key={cat.name} style={{ marginBottom: '0.6rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.2rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                  <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: CATEGORY_COLORS[cat.name] || '#94a3b8' }} />
                  <span style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)' }}>{cat.name}</span>
                </div>
                <span style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                  {cat.pct.toFixed(1)}% · {formatMoney(cat.value, kpiCurrency)}
                </span>
              </div>
              <div style={{ background: 'rgba(255,255,255,0.06)', borderRadius: '4px', height: '6px', overflow: 'hidden' }}>
                <div style={{
                  height: '100%',
                  width: `${cat.pct}%`,
                  background: CATEGORY_COLORS[cat.name] || '#94a3b8',
                  borderRadius: '4px',
                  transition: 'width 0.6s cubic-bezier(0.4,0,0.2,1)',
                }} />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* ⚖️ Seção de Rebalanceamento Inteligente */}
      <div style={{ marginTop: '1.5rem', paddingTop: '1.25rem', borderTop: '1px solid rgba(255,255,255,0.06)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem', flexWrap: 'wrap', gap: '0.5rem' }}>
          <div>
            <h4 style={{ fontSize: '0.9rem', fontWeight: 700, color: 'var(--text-primary)', margin: 0, display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
              <span>⚖️</span> Rebalanceamento Inteligente
            </h4>
            <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', margin: '0.2rem 0 0 0' }}>
              Compare a alocação atual com a sua meta desejada e calcule a distribuição do próximo aporte
            </p>
          </div>
          <button
            onClick={() => setIsEditing(!isEditing)}
            style={{
              background: isEditing ? 'rgba(0,242,254,0.15)' : 'rgba(255,255,255,0.06)',
              border: `1px solid ${isEditing ? 'rgba(0,242,254,0.4)' : 'rgba(255,255,255,0.1)'}`,
              borderRadius: '8px',
              padding: '0.4rem 0.85rem',
              fontSize: '0.75rem',
              fontWeight: 600,
              color: isEditing ? '#00f2fe' : 'var(--text-secondary)',
              cursor: 'pointer',
              transition: 'all 0.2s ease',
            }}
          >
            {isEditing ? '✓ Salvar Metas' : '🎯 Definir % Alvo'}
          </button>
        </div>

        {/* Painel de Configuração de Metas (%) */}
        {isEditing && (
          <div style={{ padding: '1rem', background: 'rgba(0,242,254,0.03)', borderRadius: '12px', border: '1px solid rgba(0,242,254,0.15)', marginBottom: '1rem' }}>
            <p style={{ fontSize: '0.75rem', color: 'var(--text-primary)', fontWeight: 600, marginBottom: '0.75rem' }}>
              Defina o percentual alvo desejado para cada categoria (a soma recomendada é 100%):
            </p>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '0.75rem' }}>
              {categoryBreakdown.map(cat => (
                <div key={cat.name} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem', background: 'rgba(255,255,255,0.03)', padding: '0.5rem 0.75rem', borderRadius: '8px' }}>
                  <span style={{ fontSize: '0.78rem', color: 'var(--text-primary)', fontWeight: 600 }}>{cat.name}</span>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                    <input
                      type="number"
                      min="0"
                      max="100"
                      step="1"
                      value={targetAlloc[cat.name] ?? ''}
                      onChange={(e) => {
                        const val = parseFloat(e.target.value) || 0;
                        saveTargetAlloc({ ...targetAlloc, [cat.name]: val });
                      }}
                      placeholder="0"
                      style={{
                        width: '60px',
                        background: 'rgba(15,23,42,0.8)',
                        border: '1px solid rgba(255,255,255,0.15)',
                        borderRadius: '6px',
                        padding: '0.25rem 0.4rem',
                        fontSize: '0.8rem',
                        color: '#fff',
                        textAlign: 'right',
                        fontVariantNumeric: 'tabular-nums',
                      }}
                    />
                    <span style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>%</span>
                  </div>
                </div>
              ))}
            </div>
            {Math.abs(rebalanceMetrics.totalTargetSum - 100) > 0.1 && (
              <div style={{ fontSize: '0.72rem', color: '#fbbf24', marginTop: '0.75rem', display: 'flex', alignItems: 'center', gap: '0.3rem' }}>
                <span>⚠️</span> Soma atual das metas: <strong>{rebalanceMetrics.totalTargetSum.toFixed(1)}%</strong> (recomendado exatamente 100%)
              </div>
            )}
          </div>
        )}

        {/* Tabela de Gaps de Rebalanceamento */}
        {rebalanceMetrics.items.length > 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem', marginBottom: '1.25rem' }}>
            {rebalanceMetrics.items.map(item => {
              const isSub = item.gapPct > 0.5;
              const isSuper = item.gapPct < -0.5;
              const statusColor = isSub ? '#4ade80' : isSuper ? '#f87171' : 'var(--text-secondary)';
              const statusText = isSub ? 'Subalocado (Aportar)' : isSuper ? 'Superalocado' : 'Em Meta';

              return (
                <div key={item.name} style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.04)', borderRadius: '10px', padding: '0.75rem 1rem' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                      <span style={{ fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-primary)' }}>{item.name}</span>
                      <span style={{ fontSize: '0.68rem', padding: '0.15rem 0.4rem', borderRadius: '4px', background: `${statusColor}15`, color: statusColor, fontWeight: 600 }}>
                        {statusText}
                      </span>
                    </div>
                    <div style={{ fontSize: '0.78rem', fontVariantNumeric: 'tabular-nums', display: 'flex', gap: '0.8rem' }}>
                      <span style={{ color: 'var(--text-secondary)' }}>Atual: <strong style={{ color: 'var(--text-primary)' }}>{item.currentPct.toFixed(1)}%</strong></span>
                      <span style={{ color: 'var(--text-secondary)' }}>Alvo: <strong style={{ color: '#00f2fe' }}>{item.targetPct.toFixed(1)}%</strong></span>
                      <span style={{ fontWeight: 700, color: statusColor }}>
                        {item.gapPct >= 0 ? '+' : ''}{item.gapPct.toFixed(1)}% ({item.gapMoney >= 0 ? '+' : ''}{formatMoney(item.gapMoney, kpiCurrency)})
                      </span>
                    </div>
                  </div>
                  <div style={{ background: 'rgba(255,255,255,0.06)', borderRadius: '4px', height: '5px', overflow: 'hidden' }}>
                    <div style={{
                      height: '100%',
                      width: `${Math.min(100, Math.max(0, item.currentPct))}%`,
                      background: statusColor,
                      borderRadius: '4px',
                      transition: 'width 0.4s ease',
                    }} />
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Simulador de Aporte Proporcional */}
        <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px dashed rgba(255,255,255,0.08)', borderRadius: '12px', padding: '1rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.75rem', marginBottom: '0.75rem' }}>
            <span style={{ fontSize: '0.78rem', fontWeight: 600, color: 'var(--text-primary)', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
              <span>💡</span> Sugestão de Distribuição do Próximo Aporte
            </span>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
              <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>Valor do Aporte:</span>
              <input
                type="text"
                value={contributionInput}
                onChange={(e) => setContributionInput(e.target.value)}
                placeholder="1000"
                style={{
                  width: '90px',
                  background: 'rgba(15,23,42,0.8)',
                  border: '1px solid rgba(0,242,254,0.3)',
                  borderRadius: '6px',
                  padding: '0.25rem 0.5rem',
                  fontSize: '0.8rem',
                  fontWeight: 700,
                  color: '#00f2fe',
                  textAlign: 'right',
                }}
              />
            </div>
          </div>

          {contributionSuggestions.length > 0 ? (
            <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
              {contributionSuggestions.map(sug => (
                <StatPill
                  key={sug.name}
                  label={`Comprar em ${sug.name}`}
                  value={formatMoney(sug.amount, kpiCurrency)}
                  color="#4ade80"
                />
              ))}
            </div>
          ) : (
            <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', textAlign: 'center', padding: '0.5rem 0' }}>
              {rebalanceMetrics.items.every(i => i.targetPct === 0)
                ? 'Clique em "🎯 Definir % Alvo" para configurar a alocação desejada por categoria.'
                : 'Sua carteira está perfeitamente equilibrada em relação às metas definidas.'}
            </div>
          )}
        </div>
      </div>
    </AnalysisCard>
  );
}

