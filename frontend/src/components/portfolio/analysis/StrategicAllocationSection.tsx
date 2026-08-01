import React, { useMemo } from 'react';
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from 'recharts';
import { Position, FixedIncomePosition, TreasuryPosition } from '../types';
import { formatMoney, getAssetCategory } from '../helpers';
import { ALLOCATION_COLORS, CATEGORY_COLORS, EXPOSURE_COLORS, isExposicaoGlobal } from './constants';
import { SectionTitle, AnalysisCard } from './sharedComponents';

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
    </AnalysisCard>
  );
}
