import React, { useMemo, useState, useEffect } from 'react';
import { BarChart, Bar, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { useTheme } from '@/components/ThemeProvider';
import { Position, CalculatedDividend } from '../types';

import { formatMoney } from '../helpers';
import { DIVIDENDS_COLORS } from './constants';
import { SectionTitle, AnalysisCard, StatPill, AlertBadge, ProgressBar } from './sharedComponents';

interface PassiveIncomeSectionProps {
  positions: Position[];
  dividends: CalculatedDividend[];
  kpiCurrency: string;
}

const MONTHS_FOR_YIELD = 12;

export default function PassiveIncomeSection({
  positions,
  dividends,
  kpiCurrency,
}: PassiveIncomeSectionProps) {
  let isLight = false;
  try {
    const { theme } = useTheme();
    isLight = theme === 'light';
  } catch (e) {
    // Fallback gracioso quando renderizado fora do ThemeProvider
  }
  const strokeColor = isLight ? 'rgba(0, 0, 0, 0.5)' : 'rgba(255, 255, 255, 0.4)';
  const gridColor = isLight ? 'rgba(0, 0, 0, 0.06)' : 'rgba(255, 255, 255, 0.05)';
  const [monthlyGoal, setMonthlyGoal] = useState<number>(0);
  const [goalInput, setGoalInput] = useState<string>('');
  const [editingGoal, setEditingGoal] = useState<boolean>(false);

  const [monthlyContribution, setMonthlyContribution] = useState<number>(1000);
  const [contributionInput, setContributionInput] = useState<string>('1.000,00');
  const [editingContribution, setEditingContribution] = useState<boolean>(false);

  useEffect(() => {
    const savedGoal = localStorage.getItem('stockpulse_monthly_goal');
    if (savedGoal) {
      const parsed = parseFloat(savedGoal);
      if (!isNaN(parsed)) {
        setMonthlyGoal(parsed);
        setGoalInput(parsed.toLocaleString('pt-BR', { minimumFractionDigits: 2 }));
      }
    }

    const savedContrib = localStorage.getItem('stockpulse_monthly_contribution');
    if (savedContrib) {
      const parsed = parseFloat(savedContrib);
      if (!isNaN(parsed)) {
        setMonthlyContribution(parsed);
        setContributionInput(parsed.toLocaleString('pt-BR', { minimumFractionDigits: 2 }));
      }
    }
  }, []);

  const saveGoal = () => {
    const raw = goalInput.replace(/\./g, '').replace(',', '.');
    const parsed = parseFloat(raw);
    if (!isNaN(parsed) && parsed >= 0) {
      setMonthlyGoal(parsed);
      localStorage.setItem('stockpulse_monthly_goal', String(parsed));
    }
    setEditingGoal(false);
  };

  const saveContribution = () => {
    const raw = contributionInput.replace(/\./g, '').replace(',', '.');
    const parsed = parseFloat(raw);
    if (!isNaN(parsed) && parsed >= 0) {
      setMonthlyContribution(parsed);
      localStorage.setItem('stockpulse_monthly_contribution', String(parsed));
    }
    setEditingContribution(false);
  };

  const upcomingDividends = useMemo(() => {
    return dividends.filter(div => {
      if (!div.payment_date || div.payment_date.startsWith('0001')) return true;
      const today = new Date();
      today.setHours(0, 0, 0, 0);
      const [year, month, day] = div.payment_date.split('T')[0].split('-');
      const payDate = new Date(parseInt(year), parseInt(month) - 1, parseInt(day));
      return payDate > today;
    });
  }, [dividends]);

  const dividendSeasonality = useMemo(() => {
    const today = new Date();
    const currentYear = today.getFullYear();
    const currentMonth = today.getMonth();

    const getDividendMonthKey = (div: CalculatedDividend): string | null => {
      let dateStr = div.payment_date;
      if (!dateStr || dateStr.startsWith('0001')) {
        dateStr = div.cum_date;
      }
      if (!dateStr || dateStr.startsWith('0001')) {
        return null;
      }
      const parts = dateStr.split('T')[0].split('-');
      if (parts.length >= 2) {
        return `${parts[0]}-${parts[1]}`;
      }
      return null;
    };

    const isPaidVal = (div: CalculatedDividend) => {
      if (!div.payment_date || div.payment_date.startsWith('0001')) return false;
      const [y, mm, dd] = div.payment_date.split('T')[0].split('-');
      const payDate = new Date(parseInt(y), parseInt(mm) - 1, parseInt(dd));
      const t = new Date();
      t.setHours(0, 0, 0, 0);
      return payDate <= t;
    };

    const start = new Date(currentYear, currentMonth - 11, 1);
    let end = new Date(currentYear, currentMonth, 1);

    dividends.forEach(div => {
      if (!isPaidVal(div)) {
        const key = getDividendMonthKey(div);
        if (key) {
          const [y, m] = key.split('-').map(Number);
          const divDate = new Date(y, m - 1, 1);
          if (divDate > end) {
            end = divDate;
          }
        }
      }
    });

    const maxFuture = new Date(currentYear, currentMonth + 11, 1);
    if (end > maxFuture) {
      end = maxFuture;
    }

    const monthsList: { year: number; month: number; key: string; label: string; isCurrent: boolean }[] = [];
    let current = new Date(start);
    const monthLabels = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];

    while (current <= end) {
      const y = current.getFullYear();
      const m = current.getMonth();
      const key = `${y}-${String(m + 1).padStart(2, '0')}`;
      const label = `${monthLabels[m]}/${String(y).slice(-2)}`;
      const isCurr = y === currentYear && m === currentMonth;

      monthsList.push({
        year: y,
        month: m,
        key,
        label,
        isCurrent: isCurr,
      });

      current.setMonth(current.getMonth() + 1);
    }

    const monthlyData = monthsList.map(m => {
      let pastValue = 0;
      let futureValue = 0;

      dividends.forEach(div => {
        const key = getDividendMonthKey(div);
        if (key === m.key) {
          const amount = div.net_amount || div.gross_amount || 0;
          if (isPaidVal(div)) {
            pastValue += amount;
          } else {
            futureValue += amount;
          }
        }
      });

      return {
        monthLabel: m.label,
        isCurrent: m.isCurrent,
        pastValue,
        futureValue,
        totalValue: pastValue + futureValue,
      };
    });

    const maxVal = Math.max(...monthlyData.map(m => m.totalValue), 0);

    return monthlyData.map(m => {
      const pctPast = maxVal > 0 ? (m.pastValue / maxVal) * 100 : 0;
      const pctFuture = maxVal > 0 ? (m.futureValue / maxVal) * 100 : 0;
      return {
        ...m,
        pctPast,
        pctFuture,
      };
    });
  }, [dividends]);

  const dividendsMonthly = useMemo(() => {
    const now = new Date();
    const twelveMonthsAgo = new Date(now);
    twelveMonthsAgo.setMonth(now.getMonth() - 12);

    const grouped: Record<string, { label: string; rawDate: Date; nacionais: number; internacionais: number; rendaFixa: number }> = {};

    dividends.forEach(div => {
      const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.cum_date;
      if (!dateStr) return;
      const date = new Date(dateStr);
      if (date < twelveMonthsAgo) return;

      const monthKey = dateStr.substring(0, 7);
      if (!grouped[monthKey]) {
        const [yearStr, monthStr] = monthKey.split('-');
        grouped[monthKey] = {
          label: monthKey,
          rawDate: new Date(parseInt(yearStr), parseInt(monthStr) - 1, 1),
          nacionais: 0,
          internacionais: 0,
          rendaFixa: 0,
        };
      }

      if (div.is_accrued) {
        grouped[monthKey].rendaFixa += div.net_amount;
      } else if (div.original_net_amount !== undefined && div.original_net_amount > 0) {
        grouped[monthKey].internacionais += div.net_amount;
      } else {
        grouped[monthKey].nacionais += div.net_amount;
      }
    });

    return Object.values(grouped)
      .sort((a, b) => a.rawDate.getTime() - b.rawDate.getTime())
      .map(item => ({
        name: item.rawDate.toLocaleDateString('pt-BR', { month: 'short', year: 'numeric' }).toUpperCase(),
        'Nacionais (R$)': Number(item.nacionais.toFixed(2)),
        'Internacionais (R$)': Number(item.internacionais.toFixed(2)),
        'Renda Fixa (R$)': Number(item.rendaFixa.toFixed(2)),
      }));
  }, [dividends]);

  const incomeKPIs = useMemo(() => {
    const now = new Date();
    const twelveMonthsAgo = new Date(now);
    twelveMonthsAgo.setMonth(now.getMonth() - 12);

    const recentDividends = dividends.filter(d => {
      if (d.is_accrued) return false;
      const dateStr = (d.payment_date && !d.payment_date.startsWith('0001')) ? d.payment_date : d.cum_date;
      if (!dateStr) return false;
      return new Date(dateStr) >= twelveMonthsAgo;
    });

    const totalDiv12m = recentDividends.reduce((s, d) => s + d.net_amount, 0);
    const totalEquityValue = positions.reduce((s, p) => s + (p.current_value || 0), 0);
    const totalCost = positions.reduce((s, p) => s + p.total_cost, 0);

    const dy = totalEquityValue > 1e-6 ? (totalDiv12m / totalEquityValue) * 100 : 0;
    const yoc = totalCost > 1e-6 ? (totalDiv12m / totalCost) * 100 : 0;

    return {
      totalDiv12m,
      dy: Number(dy.toFixed(2)),
      yoc: Number(yoc.toFixed(2)),
      totalEquityValue,
    };
  }, [dividends, positions]);

  const avgMonthly = incomeKPIs.totalDiv12m / MONTHS_FOR_YIELD;

  // Projeção do Efeito Bola de Neve (Juros Compostos)
  const snowballProjection = useMemo(() => {
    const monthlyDY = (incomeKPIs.dy > 0 ? incomeKPIs.dy : 8) / 12 / 100;
    const startValue = incomeKPIs.totalEquityValue;

    const simulateMonths = (targetMonths: number) => {
      let portfolio = startValue;
      for (let m = 0; m < targetMonths; m++) {
        const monthIncome = portfolio * monthlyDY;
        portfolio += monthIncome + monthlyContribution;
      }
      const projectedIncome = portfolio * monthlyDY;
      return { portfolio, projectedIncome };
    };

    const targetYears = [1, 3, 5, 10];
    const targets = targetYears.map(years => {
      const res = simulateMonths(years * 12);
      return {
        label: `${years} ano${years > 1 ? 's' : ''}`,
        years,
        projectedPortfolio: res.portfolio,
        projectedMonthlyIncome: res.projectedIncome,
      };
    });

    // Série temporal mensal para o gráfico (60 meses = 5 anos)
    const timeline: { label: string; portfolio: number; monthlyIncome: number }[] = [];
    let runningPortfolio = startValue;
    for (let m = 1; m <= 60; m++) {
      const mIncome = runningPortfolio * monthlyDY;
      runningPortfolio += mIncome + monthlyContribution;
      if (m % 6 === 0) {
        const yearLabel = (m / 12).toFixed(1).replace('.0', '');
        timeline.push({
          label: `${yearLabel}a`,
          portfolio: Number(runningPortfolio.toFixed(2)),
          monthlyIncome: Number((runningPortfolio * monthlyDY).toFixed(2)),
        });
      }
    }

    return {
      targets,
      timeline,
      monthlyDY,
    };
  }, [incomeKPIs.dy, incomeKPIs.totalEquityValue, monthlyContribution]);

  const DividendBarTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload || payload.length === 0) return null;
    const total = payload.reduce((sum: number, entry: any) => sum + entry.value, 0);
    return (
      <div style={{
        background: 'rgba(15, 23, 42, 0.95)',
        border: '1px solid rgba(255,255,255,0.1)',
        padding: '1rem',
        borderRadius: '10px',
        boxShadow: '0 8px 32px rgba(0,0,0,0.45)',
        backdropFilter: 'blur(12px)',
      }}>
        <p style={{ margin: '0 0 0.5rem 0', fontWeight: 700, color: '#fff' }}>{label}</p>
        {payload.map((entry: any, i: number) => (
          <p key={i} style={{ margin: '0.25rem 0', fontSize: '0.85rem', color: entry.color, display: 'flex', justifyContent: 'space-between', gap: '1rem' }}>
            <span>{entry.name}:</span>
            <span style={{ fontWeight: 700 }}>R$ {entry.value.toFixed(2)}</span>
          </p>
        ))}
        <div style={{ marginTop: '0.5rem', paddingTop: '0.5rem', borderTop: '1px solid rgba(255,255,255,0.1)', display: 'flex', justifyContent: 'space-between', fontSize: '0.9rem', fontWeight: 700, color: '#fff' }}>
          <span>Total:</span>
          <span>R$ {total.toFixed(2)}</span>
        </div>
      </div>
    );
  };

  const ProjectionTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload || payload.length === 0) return null;
    const item = payload[0].payload;
    return (
      <div style={{
        background: 'rgba(15, 23, 42, 0.95)',
        border: '1px solid rgba(255,255,255,0.1)',
        padding: '0.85rem 1rem',
        borderRadius: '10px',
        boxShadow: '0 8px 32px rgba(0,0,0,0.45)',
        backdropFilter: 'blur(12px)',
      }}>
        <p style={{ margin: '0 0 0.4rem 0', fontWeight: 700, color: '#fff', fontSize: '0.85rem' }}>Projeção em {label}</p>
        <p style={{ margin: '0.2rem 0', fontSize: '0.8rem', color: '#4ade80', display: 'flex', justifyContent: 'space-between', gap: '1rem' }}>
          <span>Renda Mensal Projetada:</span>
          <span style={{ fontWeight: 700 }}>{formatMoney(item.monthlyIncome, 'BRL')}</span>
        </p>
        <p style={{ margin: '0.2rem 0', fontSize: '0.78rem', color: 'var(--text-secondary)', display: 'flex', justifyContent: 'space-between', gap: '1rem' }}>
          <span>Patrimônio Acumulado:</span>
          <span style={{ fontWeight: 600 }}>{formatMoney(item.portfolio, 'BRL')}</span>
        </p>
      </div>
    );
  };

  return (
    <>
      {/* SEÇÃO: Geração de Renda (Proventos) */}
      <AnalysisCard id="section-income">
        <SectionTitle
          emoji="💰"
          title="Geração de Renda"
          subtitle="Histórico mensal de proventos e indicadores de rendimento"
        />

        {/* KPI pills */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '0.75rem', marginBottom: '1.25rem' }}>
          <StatPill label="Dividend Yield (DY)" value={`${incomeKPIs.dy.toFixed(2)}% a.a.`} color="#4ade80" />
          <StatPill label="Yield on Cost (YOC)" value={`${incomeKPIs.yoc.toFixed(2)}% a.a.`} color="#00f2fe" />
          <StatPill label="Total 12 meses" value={formatMoney(incomeKPIs.totalDiv12m, 'BRL')} color="#fbbf24" />
        </div>

        {/* Bar chart */}
        {dividendsMonthly.length > 0 ? (
          <ResponsiveContainer width="100%" height={280}>
            <BarChart data={dividendsMonthly} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
              <XAxis
                dataKey="name"
                stroke="rgba(255,255,255,0.4)"
                fontSize={11}
                tickMargin={10}
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                stroke="rgba(255,255,255,0.4)"
                fontSize={11}
                tickFormatter={(v) => `R$ ${v}`}
                axisLine={false}
                tickLine={false}
              />
              <Tooltip content={<DividendBarTooltip />} cursor={{ fill: 'rgba(255,255,255,0.02)' }} />
              <Legend wrapperStyle={{ paddingTop: '16px', fontSize: '0.75rem' }} />
              <Bar dataKey="Nacionais (R$)" stackId="a" fill={DIVIDENDS_COLORS.nacionais} radius={[0, 0, 4, 4]} barSize={36} />
              <Bar dataKey="Internacionais (R$)" stackId="a" fill={DIVIDENDS_COLORS.internacionais} radius={[0, 0, 0, 0]} barSize={36} />
              <Bar dataKey="Renda Fixa (R$)" stackId="a" fill={DIVIDENDS_COLORS.rendaFixa} radius={[4, 4, 0, 0]} barSize={36} />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <div style={{ height: '220px', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', border: '1px dashed var(--panel-border)', borderRadius: '12px', color: 'var(--text-secondary)' }}>
            <span style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>💸</span>
            <p style={{ fontSize: '0.85rem', margin: 0 }}>Nenhum provento registrado nos últimos 12 meses.</p>
          </div>
        )}
      </AnalysisCard>

      {/* Meta Mensal de Renda Passiva */}
      <AnalysisCard>
        <SectionTitle emoji="🎯" title="Cobertura de Renda Passiva" subtitle="Meta mensal vs média real dos últimos 12 meses" />
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1rem' }}>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.3rem' }}>Meta Mensal</div>
            {editingGoal ? (
              <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                <input
                  type="text"
                  value={goalInput}
                  onChange={e => setGoalInput(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && saveGoal()}
                  placeholder="Ex: 1.000,00"
                  autoFocus
                  style={{
                    background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.15)',
                    borderRadius: '8px', padding: '0.4rem 0.6rem', color: 'var(--text-primary)',
                    fontSize: '0.9rem', width: '130px', outline: 'none', fontVariantNumeric: 'tabular-nums',
                  }}
                />
                <button onClick={saveGoal} style={{ background: 'rgba(74,222,128,0.15)', border: '1px solid rgba(74,222,128,0.3)', borderRadius: '8px', padding: '0.4rem 0.75rem', color: '#4ade80', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 700 }}>Salvar</button>
              </div>
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                <span style={{ fontSize: '1.3rem', fontWeight: 700, color: monthlyGoal > 0 ? 'var(--text-primary)' : 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                  {monthlyGoal > 0 ? formatMoney(monthlyGoal, 'BRL') : 'Não definida'}
                </span>
                <button onClick={() => setEditingGoal(true)} style={{ background: 'transparent', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '6px', padding: '0.2rem 0.5rem', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: '0.72rem' }}>
                  ✏️ {monthlyGoal > 0 ? 'Editar' : 'Definir'}
                </button>
              </div>
            )}
          </div>
          <div style={{ textAlign: 'right' }}>
            <div style={{ fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.3rem' }}>Média Real (12m)</div>
            <div style={{ fontSize: '1.3rem', fontWeight: 700, color: '#4ade80', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(avgMonthly, 'BRL')}</div>
          </div>
        </div>

        {monthlyGoal > 0 && (
          <>
            {(() => {
              const coverage = Math.min((avgMonthly / monthlyGoal) * 100, 100);
              const barColor = coverage >= 100 ? '#4ade80' : coverage >= 60 ? '#fbbf24' : '#f87171';
              return (
                <>
                  <ProgressBar
                    value={avgMonthly}
                    max={monthlyGoal}
                    color={barColor}
                    sublabel={`${coverage.toFixed(1)}% da meta atingida`}
                  />
                  {coverage >= 100 ? (
                    <AlertBadge type="success" message="🎉 Parabéns! Sua renda passiva já cobre 100% da sua meta mensal!" />
                  ) : (
                    <AlertBadge type="info" message={`Faltam ${formatMoney(monthlyGoal - avgMonthly, 'BRL')}/mês para atingir sua meta. Continue investindo!`} />
                  )}
                </>
              );
            })()}
          </>
        )}

        {monthlyGoal === 0 && (
          <AlertBadge type="info" message="Defina uma meta mensal de renda passiva para acompanhar seu progresso." />
        )}
      </AnalysisCard>

      {/* ❄️ SEÇÃO: Efeito Bola de Neve & Projeção de Renda */}
      <AnalysisCard id="section-snowball">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '0.5rem', marginBottom: '1rem' }}>
          <SectionTitle
            emoji="❄️"
            title="Efeito Bola de Neve & Projeção de Renda"
            subtitle="Simulação de juros compostos considerando reinvestimento de proventos e aportes mensais"
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>Aporte Mensal Estimado:</span>
            {editingContribution ? (
              <div style={{ display: 'flex', gap: '0.4rem', alignItems: 'center' }}>
                <input
                  type="text"
                  value={contributionInput}
                  onChange={e => setContributionInput(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && saveContribution()}
                  placeholder="1.000,00"
                  autoFocus
                  style={{
                    background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(0,242,254,0.4)',
                    borderRadius: '6px', padding: '0.25rem 0.5rem', color: '#00f2fe',
                    fontSize: '0.8rem', width: '100px', textAlign: 'right', fontWeight: 700,
                  }}
                />
                <button onClick={saveContribution} style={{ background: 'rgba(0,242,254,0.15)', border: '1px solid rgba(0,242,254,0.3)', borderRadius: '6px', padding: '0.25rem 0.5rem', color: '#00f2fe', cursor: 'pointer', fontSize: '0.72rem', fontWeight: 700 }}>OK</button>
              </div>
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <span style={{ fontSize: '0.9rem', fontWeight: 700, color: '#00f2fe', fontVariantNumeric: 'tabular-nums' }}>
                  {formatMoney(monthlyContribution, 'BRL')}
                </span>
                <button onClick={() => setEditingContribution(true)} style={{ background: 'transparent', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '6px', padding: '0.2rem 0.5rem', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: '0.72rem' }}>
                  ✏️ Alterar
                </button>
              </div>
            )}
          </div>
        </div>

        {/* 4 StatPills: Projeção de Renda Mensal em 1, 3, 5, 10 anos */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '0.75rem', marginBottom: '1.25rem' }}>
          {snowballProjection.targets.map(target => (
            <StatPill
              key={target.years}
              label={`Renda em ${target.label}`}
              value={`${formatMoney(target.projectedMonthlyIncome, 'BRL')}/mês`}
              color={target.years === 5 ? '#4ade80' : target.years === 10 ? '#00f2fe' : '#fbbf24'}
            />
          ))}
        </div>

        {/* AreaChart: Evolução da Renda Mensal em 5 anos */}
        <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)', marginBottom: '0.75rem' }}>
          Evolução da Renda Mensal Projetada (Próximos 5 Anos)
        </p>
        <ResponsiveContainer width="100%" height={220}>
          <AreaChart data={snowballProjection.timeline} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
            <defs>
              <linearGradient id="colorIncome" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#4ade80" stopOpacity={0.4} />
                <stop offset="95%" stopColor="#4ade80" stopOpacity={0.0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke={gridColor} vertical={false} />
            <XAxis
              dataKey="label"
              stroke={strokeColor}
              fontSize={11}
              axisLine={false}
              tickLine={false}
            />
            <YAxis
              stroke={strokeColor}
              fontSize={11}
              tickFormatter={(v) => `R$ ${v}`}
              axisLine={false}
              tickLine={false}
            />
            <Tooltip content={<ProjectionTooltip />} />
            <Area
              type="monotone"
              dataKey="monthlyIncome"
              name="Renda Mensal Projetada"
              stroke="#4ade80"
              strokeWidth={2.5}
              fillOpacity={1}
              fill="url(#colorIncome)"
            />
          </AreaChart>
        </ResponsiveContainer>

        <AlertBadge
          type="info"
          message={`❄️ Efeito Bola de Neve: Com aportes mensais de ${formatMoney(monthlyContribution, 'BRL')} e DY médio de ${incomeKPIs.dy > 0 ? incomeKPIs.dy.toFixed(2) : '8.00'}% a.a., sua renda passiva mensal poderá saltar de ${formatMoney(avgMonthly, 'BRL')} para ${formatMoney(snowballProjection.targets[2].projectedMonthlyIncome, 'BRL')} em 5 anos devido ao reinvestimento composto dos proventos.`}
        />
      </AnalysisCard>

      {/* Sazonalidade de Proventos */}
      <AnalysisCard>
        <SectionTitle emoji="🗓️" title="Sazonalidade de Proventos" subtitle="Mapa de calor do fluxo de caixa (12 meses + próximos)" />
        <div style={{ display: 'flex', alignItems: 'flex-end', height: '140px', gap: '8px', marginTop: '1rem', borderBottom: '1px solid rgba(255,255,255,0.1)', paddingBottom: '0.5rem' }}>
          {dividendSeasonality.map((item, i) => (
             <div key={i} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', gap: '4px' }}>
                <span style={{ fontSize: '0.65rem', color: 'var(--text-secondary)', fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
                   {item.totalValue > 0 ? (item.totalValue >= 1000 ? `${(item.totalValue/1000).toFixed(1).replace('.0','')}k` : Math.round(item.totalValue)) : ''}
                </span>
                <div 
                   title={`${item.monthLabel}: ${formatMoney(item.totalValue, 'BRL')} ${item.futureValue > 0 ? `(Provisionado: ${formatMoney(item.futureValue, 'BRL')})` : ''}`}
                   style={{ 
                     width: '100%', 
                     display: 'flex',
                     flexDirection: 'column',
                     justifyContent: 'flex-end',
                     height: `${Math.max(item.pctPast + item.pctFuture, 1)}%`, 
                     minHeight: '4px',
                     borderRadius: '4px 4px 0 0',
                     overflow: 'hidden'
                   }} 
                >
                   {item.pctFuture > 0 && (
                      <div style={{ 
                          width: '100%', 
                          height: `${(item.pctFuture / (item.pctPast + item.pctFuture)) * 100}%`, 
                          background: 'var(--color-warning)'
                      }} />
                   )}
                   {item.pctPast > 0 && (
                      <div style={{ 
                          width: '100%', 
                          height: `${(item.pctPast / (item.pctPast + item.pctFuture)) * 100}%`, 
                          background: item.isCurrent ? 'var(--color-success)' : 'var(--accent-color)' 
                      }} />
                   )}
                </div>
             </div>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem', padding: '0 2px' }}>
           {dividendSeasonality.map((item, i) => (
              <div key={i} style={{ flex: 1, textAlign: 'center' }}>
                 <span style={{ fontSize: '0.65rem', color: item.isCurrent ? 'var(--color-success)' : 'var(--text-secondary)', fontWeight: item.isCurrent ? 700 : 400 }}>
                    {item.monthLabel}
                 </span>
              </div>
           ))}
        </div>
        
        {upcomingDividends.length > 0 && (
           <div style={{ marginTop: '1.25rem', padding: '0.75rem', background: 'var(--color-success-bg)', borderRadius: '8px', border: '1px solid rgba(var(--success-rgb), 0.2)' }}>
              <p style={{ fontSize: '0.75rem', color: 'var(--color-success)', fontWeight: 600, margin: 0, marginBottom: '0.2rem' }}>💰 Proventos a Receber</p>
              <p style={{ fontSize: '1.1rem', color: 'var(--text-primary)', fontWeight: 700, margin: 0, fontVariantNumeric: 'tabular-nums' }}>
                 {formatMoney(upcomingDividends.reduce((s, d) => s + d.net_amount, 0), 'BRL')}
              </p>
           </div>
        )}
      </AnalysisCard>
    </>
  );
}

