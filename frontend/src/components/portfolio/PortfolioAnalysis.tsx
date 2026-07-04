'use client';

import React, { useMemo, useState, useEffect } from 'react';
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
  Legend,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  BarChart,
  Bar,
} from 'recharts';
import { Position, CalculatedDividend, FixedIncomePosition, PerformancePoint } from './types';
import { formatMoney, getAssetCategory } from './helpers';

// ─── Types ───────────────────────────────────────────────────────────────────

interface PortfolioAnalysisProps {
  positions: Position[];
  dividends: CalculatedDividend[];
  fiPositions: FixedIncomePosition[];
  performanceData: PerformancePoint[];
  kpiCurrency: string;
}

interface BenchmarkPoint {
  label: string;
  portfolio: number;
  cdi: number;
  ipca: number;
  ifix: number;
  sp500: number;
}

// ─── Reusable Micro-Components ───────────────────────────────────────────────

function SectionTitle({ emoji, title, subtitle }: { emoji: string; title: string; subtitle?: string }) {
  return (
    <div className="mb-md">
      <h3 className="font-bold" style={{ fontSize: '1rem', color: 'var(--text-primary)' }}>
        {emoji} {title}
      </h3>
      {subtitle && <p className="text-secondary" style={{ fontSize: '0.78rem', marginTop: '0.2rem' }}>{subtitle}</p>}
    </div>
  );
}


function ProgressBar({
  value, max, color = '#00f2fe', label, sublabel,
}: {
  value: number; max: number; color?: string; label?: string; sublabel?: string;
}) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  return (
    <div style={{ marginBottom: '0.75rem' }}>
      {(label || sublabel) && (
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.3rem' }}>
          <span style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)' }}>{label}</span>
          <span style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>{sublabel}</span>
        </div>
      )}
      <div style={{ background: 'rgba(255,255,255,0.06)', borderRadius: '6px', height: '8px', overflow: 'hidden' }}>
        <div style={{
          height: '100%',
          width: `${pct}%`,
          background: color,
          borderRadius: '6px',
          transition: 'width 0.6s cubic-bezier(0.4,0,0.2,1)',
          boxShadow: `0 0 6px ${color}55`,
        }} />
      </div>
    </div>
  );
}

function AlertBadge({ type, message }: { type: 'warning' | 'info' | 'success'; message: string }) {
  const colors = {
    warning: { bg: 'rgba(251,191,36,0.1)', border: 'rgba(251,191,36,0.3)', text: '#fbbf24' },
    info:    { bg: 'rgba(96,165,250,0.1)',  border: 'rgba(96,165,250,0.3)',  text: '#60a5fa' },
    success: { bg: 'rgba(74,222,128,0.1)',  border: 'rgba(74,222,128,0.3)',  text: '#4ade80' },
  };
  const c = colors[type];
  return (
    <div style={{
      background: c.bg, border: `1px solid ${c.border}`, borderRadius: '10px',
      padding: '0.5rem 0.75rem', marginTop: '0.75rem', fontSize: '0.78rem',
      color: c.text, lineHeight: 1.4,
    }}>
      {message}
    </div>
  );
}

function AnalysisCard({ children, style, id }: { children: React.ReactNode; style?: React.CSSProperties; id?: string }) {
  return (
    <div id={id} style={{
      background: 'linear-gradient(145deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0.01) 100%)',
      border: '1px solid rgba(255,255,255,0.06)',
      borderRadius: '16px',
      padding: '1.5rem',
      boxShadow: '0 4px 24px rgba(0,0,0,0.25)',
      ...style,
    }}>
      {children}
    </div>
  );
}

function AssetRiskDetailRow({
  ticker,
  subText,
  valueText,
  valueColor,
  barPct,
  barColor,
}: {
  ticker: string;
  subText: string;
  valueText: string;
  valueColor?: string;
  barPct?: number;
  barColor?: string;
}) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', margin: '0.25rem 0' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <span style={{ fontWeight: 700, color: 'var(--text-primary)' }}>{ticker}</span>
          <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)', marginLeft: '0.4rem' }}>{subText}</span>
        </div>
        <span style={{ fontWeight: 600, color: valueColor || 'var(--text-primary)', fontVariantNumeric: 'tabular-nums', fontSize: '0.78rem' }}>
          {valueText}
        </span>
      </div>
      {barPct !== undefined && (
        <div style={{ background: 'rgba(255,255,255,0.04)', borderRadius: '4px', height: '4px', overflow: 'hidden' }}>
          <div style={{
            height: '100%',
            width: `${barPct}%`,
            background: barColor || '#00f2fe',
            borderRadius: '4px',
            transition: 'width 0.4s ease'
          }} />
        </div>
      )}
    </div>
  );
}

function KPIScorecard({
  label,
  value,
  subtitle,
  description,
  color,
  icon,
  alertLevel,
  children,
}: {
  label: string;
  value: string;
  subtitle?: string;
  description?: string;
  color: string;
  icon: string;
  alertLevel?: 'safe' | 'moderate' | 'danger';
  children?: React.ReactNode;
}) {
  const [expanded, setExpanded] = useState(false);
  const [isHovered, setIsHovered] = useState(false);

  const alertBorder = alertLevel === 'danger'
    ? 'rgba(248,113,113,0.4)'
    : alertLevel === 'moderate'
      ? 'rgba(251,191,36,0.4)'
      : 'rgba(74,222,128,0.2)';

  const alertGlow = alertLevel === 'danger'
    ? '0 0 20px rgba(248,113,113,0.15)'
    : alertLevel === 'moderate'
      ? '0 0 20px rgba(251,191,36,0.1)'
      : '0 0 20px rgba(74,222,128,0.08)';

  return (
    <div
      onClick={() => children && setExpanded(!expanded)}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      style={{
        flex: '1 1 280px',
        background: 'linear-gradient(145deg, rgba(255,255,255,0.04) 0%, rgba(255,255,255,0.01) 100%)',
        border: `1px solid ${alertBorder}`,
        borderRadius: '14px',
        padding: '1.25rem',
        boxShadow: isHovered && children ? '0 8px 30px rgba(0,0,0,0.35)' : alertGlow,
        transform: isHovered && children ? 'translateY(-2px)' : 'none',
        transition: 'transform 0.2s ease, box-shadow 0.3s ease',
        cursor: children ? 'pointer' : 'default',
        position: 'relative',
      }}
    >
      <div style={{ fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--text-secondary)', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
        <span style={{ fontSize: '1rem' }}>{icon}</span>
        {label}
      </div>
      <div style={{ fontSize: '1.75rem', fontWeight: 800, color, fontVariantNumeric: 'tabular-nums', letterSpacing: '-0.02em', lineHeight: 1.1 }}>
        {value}
      </div>
      {subtitle && (
        <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.4rem', lineHeight: 1.3 }}>
          {subtitle}
        </div>
      )}
      {description && (
        <div style={{
          fontSize: '0.75rem',
          color: 'var(--text-secondary)',
          marginTop: '0.75rem',
          paddingTop: '0.75rem',
          borderTop: '1px solid rgba(255,255,255,0.08)',
          lineHeight: 1.45,
        }}>
          {description}
        </div>
      )}

      {children && (
        <div style={{
          marginTop: '0.75rem',
          paddingTop: '0.75rem',
          borderTop: '1px dashed rgba(255,255,255,0.08)',
          fontSize: '0.72rem',
          color: color,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          gap: '0.3rem',
          opacity: isHovered ? 1 : 0.8,
          transition: 'opacity 0.2s ease',
          fontWeight: 600,
        }}>
          <span>{expanded ? 'Ocultar detalhes' : 'Ver detalhes e ativos'}</span>
          <span style={{
            display: 'inline-block',
            transition: 'transform 0.2s ease',
            transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
          }}>▼</span>
        </div>
      )}

      {expanded && children && (
        <div
          onClick={(e) => e.stopPropagation()}
          style={{
            marginTop: '1rem',
            paddingTop: '1rem',
            borderTop: '1px solid rgba(255,255,255,0.1)',
            display: 'flex',
            flexDirection: 'column',
            gap: '0.75rem',
            cursor: 'default',
          }}
        >
          {children}
        </div>
      )}
    </div>
  );
}

function StatPill({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div style={{
      background: `${color}10`,
      border: `1px solid ${color}25`,
      borderRadius: '12px',
      padding: '0.75rem 1rem',
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
    }}>
      <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ fontSize: '1.05rem', fontWeight: 700, color, fontVariantNumeric: 'tabular-nums' }}>{value}</span>
    </div>
  );
}

// ─── Custom Tooltip (shared) ─────────────────────────────────────────────────

function ChartTooltipShell({ active, payload, label, formatter }: any) {
  if (!active || !payload || payload.length === 0) return null;
  return (
    <div style={{
      background: 'rgba(15, 23, 42, 0.95)',
      border: '1px solid rgba(255,255,255,0.1)',
      padding: '0.85rem 1rem',
      borderRadius: '10px',
      boxShadow: '0 8px 32px rgba(0,0,0,0.45)',
      backdropFilter: 'blur(12px)',
      maxWidth: '260px',
    }}>
      <p style={{ margin: '0 0 0.4rem 0', fontWeight: 700, color: '#fff', fontSize: '0.85rem' }}>{label}</p>
      {payload.map((entry: any, i: number) => (
        <p key={i} style={{ margin: '0.2rem 0', fontSize: '0.78rem', color: entry.color, display: 'flex', justifyContent: 'space-between', gap: '1rem' }}>
          <span>{entry.name}:</span>
          <span style={{ fontWeight: 700, fontVariantNumeric: 'tabular-nums' }}>
            {formatter ? formatter(entry.value) : entry.value}
          </span>
        </p>
      ))}
    </div>
  );
}

// ─── Donut Center Label ──────────────────────────────────────────────────────

function DonutCenterLabel({ viewBox, title, value }: { viewBox?: any; title: string; value: string }) {
  if (!viewBox) return null;
  const { cx, cy } = viewBox;
  return (
    <g>
      <text x={cx} y={cy - 8} textAnchor="middle" fill="var(--text-secondary)" fontSize="0.65rem" fontWeight={600} style={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        {title}
      </text>
      <text x={cx} y={cy + 14} textAnchor="middle" fill="var(--text-primary)" fontSize="1.15rem" fontWeight={800}>
        {value}
      </text>
    </g>
  );
}

// ─── Color Palettes ──────────────────────────────────────────────────────────

const ALLOCATION_COLORS: Record<string, string> = {
  'Renda Variável': '#60a5fa',
  'Renda Fixa': '#fbbf24',
};

const CATEGORY_COLORS: Record<string, string> = {
  'Ações (B3)': '#60a5fa',
  'FIIs': '#c084fc',
  'FIAGROs': '#a78bfa',
  'ETFs Nacionais': '#34d399',
  'BDRs': '#f87171',
  'Ações EUA': '#38bdf8',
  'ETF Internacional': '#2dd4bf',
  'Cripto': '#fb923c',
  'Renda Fixa': '#fbbf24',
  'Desconhecido': '#94a3b8',
};

const EXPOSURE_COLORS = {
  local: '#4ade80',
  global: '#60a5fa',
};

const BENCHMARK_COLORS = {
  portfolio: '#00f2fe',
  cdi: '#fbbf24',
  ipca: '#f87171',
  ifix: '#c084fc',
  sp500: '#34d399',
};

const DIVIDENDS_COLORS = {
  nacionais: '#00e676',
  internacionais: '#00f2fe',
  rendaFixa: '#FFB300',
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

const MONTHS_LABEL = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];

const isRendaVariavel = (type: string) =>
  ['STOCK_BR', 'FII', 'FIAGRO', 'ETF_BR', 'BDR', 'STOCK_US', 'ETF_US', 'CRYPTO'].includes(type);

const isExposicaoGlobal = (type: string) =>
  ['BDR', 'STOCK_US', 'ETF_US'].includes(type);

const isFII = (type: string) =>
  ['FII', 'FIAGRO'].includes(type);

const isAcaoOuETF = (type: string) =>
  ['STOCK_BR', 'STOCK_US', 'ETF_BR', 'ETF_US', 'BDR'].includes(type);

// ─── Main Component ─────────────────────────────────────────────────────────


const MONTHS_FOR_YIELD = 12;
const TOP_N = 8;

export default function PortfolioAnalysis({
  positions,
  dividends,
  fiPositions,
  performanceData,
  kpiCurrency,
}: PortfolioAnalysisProps) {
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
        dateStr = div.ex_date;
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

  const [monthlyGoal, setMonthlyGoal] = useState<number>(0);
  const [goalInput, setGoalInput] = useState<string>('');
  const [editingGoal, setEditingGoal] = useState<boolean>(false);

  useEffect(() => {
    const saved = localStorage.getItem('stockpulse_monthly_goal');
    if (saved) {
      const parsed = parseFloat(saved);
      if (!isNaN(parsed)) {
        setMonthlyGoal(parsed);
        setGoalInput(parsed.toLocaleString('pt-BR', { minimumFractionDigits: 2 }));
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


  // ═══════════════════════════════════════════════════════════════════════════
  // SECTION 1: Alocação Estratégica
  // ═══════════════════════════════════════════════════════════════════════════

  const totalPortfolioValue = useMemo(() => {
    const eq = positions.reduce((s, p) => s + (p.current_value || 0), 0);
    const fi = fiPositions.reduce((s, p) => s + p.net_value, 0);
    return eq + fi;
  }, [positions, fiPositions]);

  // 1a. Classe de Ativo: Renda Fixa vs Renda Variável
  const assetClassData = useMemo(() => {
    const rv = positions.reduce((s, p) => s + (p.current_value || 0), 0);
    const rf = fiPositions.reduce((s, p) => s + p.net_value, 0);
    if (rv + rf < 1e-6) return [];
    return [
      { name: 'Renda Variável', value: rv, pct: (rv / (rv + rf)) * 100 },
      { name: 'Renda Fixa', value: rf, pct: (rf / (rv + rf)) * 100 },
    ].filter(d => d.value > 1e-6);
  }, [positions, fiPositions]);

  // 1a-extra. Detalhamento por categoria
  const categoryBreakdown = useMemo(() => {
    const map: Record<string, number> = {};
    positions.forEach(p => {
      const cat = getAssetCategory(p.type);
      map[cat] = (map[cat] || 0) + (p.current_value || 0);
    });
    const fiTotal = fiPositions.reduce((s, p) => s + p.net_value, 0);
    if (fiTotal > 1e-6) map['Renda Fixa'] = (map['Renda Fixa'] || 0) + fiTotal;
    return Object.entries(map)
      .map(([name, value]) => ({
        name,
        value,
        pct: totalPortfolioValue > 1e-6 ? (value / totalPortfolioValue) * 100 : 0,
      }))
      .filter(d => d.value > 1e-6)
      .sort((a, b) => b.value - a.value);
  }, [positions, fiPositions, totalPortfolioValue]);

  // 1b. Exposição Cambial e Geográfica
  const geoExposureData = useMemo(() => {
    const globalVal = positions
      .filter(p => isExposicaoGlobal(p.type))
      .reduce((s, p) => s + (p.current_value || 0), 0);

    const localEquity = positions
      .filter(p => !isExposicaoGlobal(p.type))
      .reduce((s, p) => s + (p.current_value || 0), 0);

    const fi = fiPositions.reduce((s, p) => s + p.net_value, 0);
    const localTotal = localEquity + fi;

    if (localTotal + globalVal < 1e-6) return [];
    const total = localTotal + globalVal;
    return [
      { name: '🇧🇷 Risco Local', value: localTotal, pct: (localTotal / total) * 100 },
      { name: '🌍 Exposição Global', value: globalVal, pct: (globalVal / total) * 100 },
    ].filter(d => d.value > 1e-6);
  }, [positions, fiPositions]);

  // ═══════════════════════════════════════════════════════════════════════════
  // SECTION 2: Comparação com Benchmarks (simulated)
  // ═══════════════════════════════════════════════════════════════════════════

  const benchmarkData = useMemo((): BenchmarkPoint[] => {
    if (!performanceData || performanceData.length === 0) return [];

    const firstValue = performanceData[0].value;
    const firstInvested = performanceData[0].total_invested;
    if (firstInvested < 1e-6) return [];

    // CDI ~1.05%/month → daily ~0.034%
    // IPCA ~0.5%/month → daily ~0.016%
    // IFIX ~0.7%/month → daily ~0.023%
    // S&P 500 ~0.9%/month → daily ~0.029%
    const dailyRates = {
      cdi: 0.00034,
      ipca: 0.00016,
      ifix: 0.00023,
      sp500: 0.00029,
    };

    // Sample at most ~60 data points for rendering performance
    const step = Math.max(1, Math.floor(performanceData.length / 60));
    const sampled = performanceData.filter((_, i) => i === 0 || i === performanceData.length - 1 || i % step === 0);

    return sampled.map((point, i) => {
      const portfolioReturn = firstInvested > 1e-6
        ? ((point.value - firstInvested) / firstInvested) * 100
        : 0;

      const dayIndex = Math.round((new Date(point.date).getTime() - new Date(performanceData[0].date).getTime()) / (1000 * 60 * 60 * 24));

      const d = new Date(point.date);
      const label = d.toLocaleDateString('pt-BR', { month: 'short', year: '2-digit' });

      return {
        label,
        portfolio: Number(portfolioReturn.toFixed(2)),
        cdi: Number(((Math.pow(1 + dailyRates.cdi, dayIndex) - 1) * 100).toFixed(2)),
        ipca: Number(((Math.pow(1 + dailyRates.ipca, dayIndex) - 1) * 100).toFixed(2)),
        ifix: Number(((Math.pow(1 + dailyRates.ifix, dayIndex) - 1) * 100).toFixed(2)),
        sp500: Number(((Math.pow(1 + dailyRates.sp500, dayIndex) - 1) * 100).toFixed(2)),
      };
    });
  }, [performanceData]);

  // ═══════════════════════════════════════════════════════════════════════════
  // SECTION 3: Termômetro de Risco (Scorecards)
  // ═══════════════════════════════════════════════════════════════════════════

  const riskMetrics = useMemo(() => {
    if (!performanceData || performanceData.length < 10) {
      return { sharpe: null, beta: null, maxDrawdown: null };
    }

    // Calculate daily returns
    const dailyReturns: number[] = [];
    for (let i = 1; i < performanceData.length; i++) {
      const prev = performanceData[i - 1].value;
      const curr = performanceData[i].value;
      if (prev > 1e-6) {
        dailyReturns.push((curr - prev) / prev);
      }
    }

    if (dailyReturns.length < 5) {
      return { sharpe: null, beta: null, maxDrawdown: null };
    }

    // Mean return
    const meanReturn = dailyReturns.reduce((s, r) => s + r, 0) / dailyReturns.length;

    // Std deviation
    const variance = dailyReturns.reduce((s, r) => s + Math.pow(r - meanReturn, 2), 0) / (dailyReturns.length - 1);
    const stdDev = Math.sqrt(variance);

    // Sharpe Ratio (annualized, risk-free ~ CDI 13.25% a.a. → daily 0.05%)
    const riskFreeDaily = 0.0005;
    const annualizedExcess = (meanReturn - riskFreeDaily) * 252;
    const annualizedVol = stdDev * Math.sqrt(252);
    const sharpe = annualizedVol > 1e-8 ? annualizedExcess / annualizedVol : 0;

    // Beta (against benchmark proxy — CDI ~0.034% daily growth)
    const benchDailyReturn = 0.00034;
    const covNumerator = dailyReturns.reduce((s, r) => s + (r - meanReturn) * (benchDailyReturn - benchDailyReturn), 0);
    // Simplified: beta ≈ vol(portfolio) / vol(market) since correlation is not computable from single series
    const marketVol = 0.012; // ~1.2% daily vol reference for Ibovespa
    const beta = stdDev / marketVol;

    // Max Drawdown
    let peak = performanceData[0].value;
    let maxDD = 0;
    for (const p of performanceData) {
      if (p.value > peak) peak = p.value;
      const dd = peak > 1e-6 ? (peak - p.value) / peak : 0;
      if (dd > maxDD) maxDD = dd;
    }

    return {
      sharpe: Number(sharpe.toFixed(2)),
      beta: Number(beta.toFixed(2)),
      maxDrawdown: Number((maxDD * 100).toFixed(1)),
    };
  }, [performanceData]);

  // Asset Profit & Loss and contributions for Sharpe Ratio
  const assetProfitLoss = useMemo(() => {
    const list: { ticker: string; name: string; profitLoss: number; returnPercent: number; weight: number }[] = [];
    
    positions.forEach(p => {
      const pl = p.profit_loss !== undefined ? p.profit_loss : ((p.current_value || 0) - p.total_cost);
      const retPct = p.return_percent !== undefined ? p.return_percent : (p.total_cost > 0 ? (pl / p.total_cost) * 100 : 0);
      const weight = totalPortfolioValue > 1e-6 ? ((p.current_value || 0) / totalPortfolioValue) * 100 : 0;
      list.push({
        ticker: p.ticker,
        name: p.name,
        profitLoss: pl,
        returnPercent: retPct,
        weight,
      });
    });

    fiPositions.forEach(p => {
      const invested = p.total_invested !== undefined ? p.total_invested : ((p as any).invested_amount || 0);
      const pl = p.net_value - invested;
      const retPct = p.net_return_percent !== undefined ? p.net_return_percent : (invested > 0 ? (pl / invested) * 100 : 0);
      const weight = totalPortfolioValue > 1e-6 ? (p.net_value / totalPortfolioValue) * 100 : 0;
      
      const type = p.asset?.type || p.type || 'Renda Fixa';
      const indexer = p.asset?.indexer || (p as any).index_type || (p as any).indexer || 'CDI';
      const institution = p.asset?.institution || (p as any).institution || '';
      const indexerLabel = indexer === 'PREFIXADO' ? 'Pré' : indexer;
      
      list.push({
        ticker: `${type} ${indexerLabel}`,
        name: institution,
        profitLoss: pl,
        returnPercent: retPct,
        weight,
      });
    });

    return list;
  }, [positions, fiPositions, totalPortfolioValue]);

  // Volatility profiles and aggressive assets for Beta Card
  const volatilityExposure = useMemo(() => {
    let highVolVal = 0;
    let medVolVal = 0;
    let lowVolVal = 0;

    positions.forEach(p => {
      const val = p.current_value || 0;
      if (['STOCK_BR', 'STOCK_US', 'ETF_BR', 'ETF_US', 'CRYPTO', 'BDR'].includes(p.type)) {
        highVolVal += val;
      } else if (['FII', 'FIAGRO'].includes(p.type)) {
        medVolVal += val;
      } else {
        lowVolVal += val;
      }
    });

    fiPositions.forEach(p => {
      lowVolVal += p.net_value;
    });

    const total = highVolVal + medVolVal + lowVolVal;
    if (total < 1e-6) {
      return { highPct: 0, medPct: 0, lowPct: 0, topAggressive: [] };
    }

    const highVolAssets = positions
      .filter(p => ['STOCK_BR', 'STOCK_US', 'ETF_BR', 'ETF_US', 'CRYPTO', 'BDR'].includes(p.type))
      .map(p => ({
        ticker: p.ticker,
        weight: totalPortfolioValue > 1e-6 ? ((p.current_value || 0) / totalPortfolioValue) * 100 : 0,
      }))
      .sort((a, b) => b.weight - a.weight)
      .slice(0, 3);

    return {
      highPct: (highVolVal / total) * 100,
      medPct: (medVolVal / total) * 100,
      lowPct: (lowVolVal / total) * 100,
      topAggressive: highVolAssets,
    };
  }, [positions, fiPositions, totalPortfolioValue]);

  // Concentration metrics for Max Drawdown Card
  const concentrationMetrics = useMemo(() => {
    const list: { ticker: string; weight: number }[] = [];

    positions.forEach(p => {
      const w = totalPortfolioValue > 1e-6 ? ((p.current_value || 0) / totalPortfolioValue) * 100 : 0;
      list.push({ ticker: p.ticker, weight: w });
    });

    fiPositions.forEach(p => {
      const w = totalPortfolioValue > 1e-6 ? (p.net_value / totalPortfolioValue) * 100 : 0;
      const type = p.asset?.type || p.type || 'Renda Fixa';
      const indexer = p.asset?.indexer || (p as any).index_type || (p as any).indexer || 'CDI';
      const institution = p.asset?.institution || (p as any).institution || '';
      const indexerLabel = indexer === 'PREFIXADO' ? 'Pré' : indexer;
      
      list.push({ ticker: `${type} ${indexerLabel} (${institution || 'Renda Fixa'})`, weight: w });
    });

    const sorted = [...list].sort((a, b) => b.weight - a.weight);
    const top3 = sorted.slice(0, 3);
    const top3Sum = top3.reduce((s, x) => s + x.weight, 0);

    return {
      top3,
      top3Sum,
    };
  }, [positions, fiPositions, totalPortfolioValue]);

  const topGainers = useMemo(() => {
    return [...assetProfitLoss]
      .filter(item => item.profitLoss > 0)
      .sort((a, b) => b.profitLoss - a.profitLoss)
      .slice(0, 3);
  }, [assetProfitLoss]);

  const topLosers = useMemo(() => {
    return [...assetProfitLoss]
      .filter(item => item.profitLoss < 0)
      .sort((a, b) => a.profitLoss - b.profitLoss)
      .slice(0, 3);
  }, [assetProfitLoss]);

  // ═══════════════════════════════════════════════════════════════════════════
  // SECTION 4: Geração de Renda (Proventos)
  // ═══════════════════════════════════════════════════════════════════════════

  const dividendsMonthly = useMemo(() => {
    const now = new Date();
    const twelveMonthsAgo = new Date(now);
    twelveMonthsAgo.setMonth(now.getMonth() - 12);

    const grouped: Record<string, { label: string; rawDate: Date; nacionais: number; internacionais: number; rendaFixa: number }> = {};

    dividends.forEach(div => {
      const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.ex_date;
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
      const dateStr = (d.payment_date && !d.payment_date.startsWith('0001')) ? d.payment_date : d.ex_date;
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
    };
  }, [dividends, positions]);

  // ═══════════════════════════════════════════════════════════════════════════
  // SECTION 5: Fundamentos da Carteira
  // ═══════════════════════════════════════════════════════════════════════════

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

  // ═══════════════════════════════════════════════════════════════════════════
  // RENDERING — CUSTOM TOOLTIPS
  // ═══════════════════════════════════════════════════════════════════════════

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

  const BenchmarkTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload || payload.length === 0) return null;
    return (
      <div style={{
        background: 'rgba(15, 23, 42, 0.95)',
        border: '1px solid rgba(255,255,255,0.1)',
        padding: '0.85rem 1rem',
        borderRadius: '10px',
        boxShadow: '0 8px 32px rgba(0,0,0,0.45)',
        backdropFilter: 'blur(12px)',
      }}>
        <p style={{ margin: '0 0 0.4rem 0', fontWeight: 700, color: '#fff', fontSize: '0.85rem' }}>{label}</p>
        {payload.map((entry: any, i: number) => (
          <p key={i} style={{ margin: '0.2rem 0', fontSize: '0.78rem', color: entry.color, display: 'flex', justifyContent: 'space-between', gap: '1.5rem' }}>
            <span>{entry.name}:</span>
            <span style={{ fontWeight: 700, fontVariantNumeric: 'tabular-nums' }}>{entry.value.toFixed(2)}%</span>
          </p>
        ))}
      </div>
    );
  };

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

  // Custom Pie label renderer
  const renderPieLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, pct, name }: any) => {
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

  // ═══════════════════════════════════════════════════════════════════════════
  // RENDER
  // ═══════════════════════════════════════════════════════════════════════════

  if (positions.length === 0 && fiPositions.length === 0) {
    return (
      <div className="text-center text-secondary" style={{ padding: '3rem' }}>
        <span className="text-2xl" style={{ display: 'block', marginBottom: '0.5rem' }}>📊</span>
        <p>Adicione ativos à carteira para visualizar a análise completa.</p>
      </div>
    );
  }


  const avgMonthly = incomeKPIs.totalDiv12m / MONTHS_FOR_YIELD;

  // 4. Top performers & worst
  const rankedPositions = useMemo(() =>
    [...positions]
      .filter(p => p.return_percent !== undefined)
      .sort((a, b) => (b.return_percent || 0) - (a.return_percent || 0)),
    [positions]
  );
  const topPerformers = rankedPositions.slice(0, TOP_N);
  const worstPerformers = [...rankedPositions].reverse().slice(0, TOP_N);

  const maxAbsReturn = Math.max(...[...topPerformers, ...worstPerformers].map(p => Math.abs(p.return_percent || 0)), 1);

  // NEW: Valuation Data (Graham / Bazin)
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

  // NEW: Renda Fixa Liquidez
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
    return [
      { label: 'Liquidez Diária / Vencido', value: daily, color: '#4ade80' },
      { label: 'Até 1 ano', value: upTo1Year, color: '#60a5fa' },
      { label: '1 a 3 anos', value: upTo3Years, color: '#fbbf24' },
      { label: 'Longo Prazo (> 3 anos)', value: longTerm, color: '#f87171' },
    ].filter(i => i.value > 0);
  }, [fiPositions]);

  return (
    <div className="flex-col gap-xl">
      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* SEÇÃO 1: Alocação Estratégica                                      */}
      {/* ════════════════════════════════════════════════════════════════════ */}
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
                    <Pie data={[]} dataKey="value" cx="50%" cy="50%" innerRadius={0} outerRadius={0}>
                      {/* @ts-ignore - Recharts center label pattern */}
                    </Pie>
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

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* SEÇÃO 2: Comparação com Benchmarks                                 */}
      {/* ════════════════════════════════════════════════════════════════════ */}
      <AnalysisCard id="section-benchmarks">
        <SectionTitle
          emoji="📈"
          title="Comparação com Benchmarks"
          subtitle="Rentabilidade acumulada da carteira vs principais indicadores do mercado"
        />

        {benchmarkData.length > 0 ? (
          <>
            <ResponsiveContainer width="100%" height={340}>
              <LineChart data={benchmarkData} margin={{ top: 10, right: 15, left: 5, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
                <XAxis
                  dataKey="label"
                  stroke="rgba(255,255,255,0.4)"
                  fontSize={11}
                  tickMargin={10}
                  axisLine={false}
                  tickLine={false}
                  interval="preserveStartEnd"
                />
                <YAxis
                  stroke="rgba(255,255,255,0.4)"
                  fontSize={11}
                  tickFormatter={(v) => `${v}%`}
                  axisLine={false}
                  tickLine={false}
                />
                <Tooltip content={<BenchmarkTooltip />} />
                <Legend
                  wrapperStyle={{ paddingTop: '16px', fontSize: '0.75rem' }}
                  iconType="circle"
                  iconSize={8}
                />
                <Line type="monotone" dataKey="portfolio" name="Carteira" stroke={BENCHMARK_COLORS.portfolio} strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 0 }} />
                <Line type="monotone" dataKey="cdi" name="CDI" stroke={BENCHMARK_COLORS.cdi} strokeWidth={1.5} dot={false} strokeDasharray="6 3" opacity={0.7} />
                <Line type="monotone" dataKey="ipca" name="IPCA+" stroke={BENCHMARK_COLORS.ipca} strokeWidth={1.5} dot={false} strokeDasharray="4 4" opacity={0.6} />
                <Line type="monotone" dataKey="ifix" name="IFIX" stroke={BENCHMARK_COLORS.ifix} strokeWidth={1.5} dot={false} strokeDasharray="8 4" opacity={0.6} />
                <Line type="monotone" dataKey="sp500" name="S&P 500" stroke={BENCHMARK_COLORS.sp500} strokeWidth={1.5} dot={false} strokeDasharray="3 6" opacity={0.6} />
              </LineChart>
            </ResponsiveContainer>

            <div style={{
              marginTop: '0.75rem',
              padding: '0.6rem 0.85rem',
              background: 'rgba(0,242,254,0.04)',
              borderRadius: '10px',
              border: '1px solid rgba(0,242,254,0.1)',
              fontSize: '0.72rem',
              color: 'var(--text-secondary)',
              lineHeight: 1.5,
            }}>
              💡 <strong style={{ color: 'var(--text-primary)' }}>Nota:</strong> Os benchmarks utilizam taxas estimadas para fins de comparação visual.
              Para resultados precisos, conecte uma fonte de dados de mercado em tempo real.
            </div>
          </>
        ) : (
          <div style={{ height: '280px', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', border: '1px dashed var(--panel-border)', borderRadius: '12px', color: 'var(--text-secondary)' }}>
            <span style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>📉</span>
            <p style={{ fontSize: '0.85rem', margin: 0 }}>Dados de performance insuficientes para gerar a comparação.</p>
          </div>
        )}
      </AnalysisCard>

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* SEÇÃO 3: Termômetro de Risco (Scorecards)                         */}
      {/* ════════════════════════════════════════════════════════════════════ */}
      <AnalysisCard id="section-risk">
        <SectionTitle
          emoji="🌡️"
          title="Termômetro de Risco"
          subtitle="Indicadores-chave de risco e eficiência da carteira"
        />

        {riskMetrics.sharpe !== null ? (
          <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
            {/* Sharpe Ratio */}
            <KPIScorecard
              icon="⚡"
              label="Índice de Sharpe"
              value={riskMetrics.sharpe!.toFixed(2)}
              subtitle={
                riskMetrics.sharpe! >= 1
                  ? 'Excelente eficiência ajustada ao risco'
                  : riskMetrics.sharpe! >= 0.5
                    ? 'Eficiência moderada — espaço para otimizar'
                    : 'Eficiência abaixo do ideal — risco não compensado'
              }
              description={
                riskMetrics.sharpe! >= 1
                  ? `O retorno gerado supera a volatilidade dos ativos. Cada 1% de risco assumido entregou mais de 1% de retorno excedente, demonstrando excelente equilíbrio.`
                  : riskMetrics.sharpe! >= 0.5
                    ? `O retorno compensa a volatilidade de forma moderada. É possível otimizar a carteira reduzindo ativos de alta oscilação ou melhorando a diversificação.`
                    : `A volatilidade da carteira é alta demais para o retorno que ela entrega. Indica que a carteira corre risco pouco eficiente para o ganho obtido.`
              }
              color={riskMetrics.sharpe! >= 1 ? '#4ade80' : riskMetrics.sharpe! >= 0.5 ? '#fbbf24' : '#f87171'}
              alertLevel={riskMetrics.sharpe! >= 1 ? 'safe' : riskMetrics.sharpe! >= 0.5 ? 'moderate' : 'danger'}
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                {topGainers.length > 0 && (
                  <div>
                    <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#4ade80', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                      <span>📈</span> Ativos mais lucrativos
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                      {topGainers.map(item => (
                        <AssetRiskDetailRow
                          key={item.ticker}
                          ticker={item.ticker}
                          subText={`Peso: ${item.weight.toFixed(1)}%`}
                          valueText={`+${formatMoney(item.profitLoss, kpiCurrency || 'BRL')} (${item.returnPercent.toFixed(1)}%)`}
                          valueColor="#4ade80"
                        />
                      ))}
                    </div>
                  </div>
                )}

                {topLosers.length > 0 && (
                  <div style={{ marginTop: '0.25rem' }}>
                    <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#f87171', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                      <span>📉</span> Ativos detratores
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                      {topLosers.map(item => (
                        <AssetRiskDetailRow
                          key={item.ticker}
                          ticker={item.ticker}
                          subText={`Peso: ${item.weight.toFixed(1)}%`}
                          valueText={`${formatMoney(item.profitLoss, kpiCurrency || 'BRL')} (${item.returnPercent.toFixed(1)}%)`}
                          valueColor="#f87171"
                        />
                      ))}
                    </div>
                  </div>
                )}

                {topGainers.length === 0 && topLosers.length === 0 && (
                  <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', textAlign: 'center' }}>
                    Nenhum ativo com ganhos ou perdas significativos.
                  </div>
                )}
              </div>
            </KPIScorecard>

            {/* Beta */}
            <KPIScorecard
              icon="📊"
              label="Beta"
              value={riskMetrics.beta!.toFixed(2)}
              subtitle={
                riskMetrics.beta! <= 0.8
                  ? 'Carteira defensiva — menos volátil que o mercado'
                  : riskMetrics.beta! <= 1.2
                    ? 'Volatilidade próxima ao mercado'
                    : 'Carteira agressiva — mais volátil que o mercado'
              }
              description={
                riskMetrics.beta! <= 0.8
                  ? `A carteira tende a oscilar cerca de ${riskMetrics.beta!.toFixed(2)}x a variação do Ibovespa, oferecendo um perfil defensivo que amortece as quedas do mercado.`
                  : riskMetrics.beta! <= 1.2
                    ? `A carteira varia de forma muito semelhante ao Ibovespa (cerca de ${riskMetrics.beta!.toFixed(2)}x), replicando o comportamento médio do mercado.`
                    : `A carteira tende a variar ${riskMetrics.beta!.toFixed(2)}x mais que o Ibovespa, o que amplifica os ganhos em altas, mas aumenta o risco de perdas em quedas.`
              }
              color={riskMetrics.beta! <= 0.8 ? '#4ade80' : riskMetrics.beta! <= 1.2 ? '#fbbf24' : '#f87171'}
              alertLevel={riskMetrics.beta! <= 0.8 ? 'safe' : riskMetrics.beta! <= 1.2 ? 'moderate' : 'danger'}
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                <div>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '0.4rem' }}>
                    Perfil de Oscilação da Carteira
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                    <AssetRiskDetailRow
                      ticker="Alta Oscilação"
                      subText="Ações, ETFs, BDRs, Cripto"
                      valueText={`${volatilityExposure.highPct.toFixed(1)}%`}
                      barPct={volatilityExposure.highPct}
                      barColor="linear-gradient(90deg, #f87171, #ef4444)"
                    />
                    <AssetRiskDetailRow
                      ticker="Média Oscilação"
                      subText="FIIs, FIAGROs"
                      valueText={`${volatilityExposure.medPct.toFixed(1)}%`}
                      barPct={volatilityExposure.medPct}
                      barColor="linear-gradient(90deg, #fbbf24, #f59e0b)"
                    />
                    <AssetRiskDetailRow
                      ticker="Baixa Oscilação"
                      subText="Renda Fixa & Caixa"
                      valueText={`${volatilityExposure.lowPct.toFixed(1)}%`}
                      barPct={volatilityExposure.lowPct}
                      barColor="linear-gradient(90deg, #4ade80, #10b981)"
                    />
                  </div>
                </div>

                {volatilityExposure.topAggressive.length > 0 && (
                  <div style={{ marginTop: '0.25rem' }}>
                    <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#fbbf24', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                      <span>⚡</span> Ativos mais Voláteis (por peso)
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                      {volatilityExposure.topAggressive.map(item => (
                        <div key={item.ticker} style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
                          <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</span>
                          <span>Peso na Carteira: <strong style={{ color: 'var(--text-primary)' }}>{item.weight.toFixed(1)}%</strong></span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </KPIScorecard>

            {/* Max Drawdown */}
            <KPIScorecard
              icon="📉"
              label="Drawdown Máximo"
              value={`-${riskMetrics.maxDrawdown!.toFixed(1)}%`}
              subtitle={
                riskMetrics.maxDrawdown! <= 10
                  ? 'Drawdown contido — volatilidade controlada'
                  : riskMetrics.maxDrawdown! <= 25
                    ? 'Drawdown moderado — considere proteger posições'
                    : 'Drawdown severo — revise a alocação de risco'
              }
              description={
                riskMetrics.maxDrawdown! <= 10
                  ? `A maior queda da carteira a partir do seu pico recente foi de -${riskMetrics.maxDrawdown!.toFixed(1)}%. Este comportamento indica excelente controle de perdas temporárias.`
                  : riskMetrics.maxDrawdown! <= 25
                    ? `A carteira sofreu uma queda máxima de -${riskMetrics.maxDrawdown!.toFixed(1)}% em relação ao seu pico recente. Sugere volatilidade intermediária a ser monitorada.`
                    : `A carteira sofreu uma queda severa de -${riskMetrics.maxDrawdown!.toFixed(1)}% a partir do seu pico recente, indicando alta sensibilidade a cenários de forte estresse.`
              }
              color={riskMetrics.maxDrawdown! <= 10 ? '#4ade80' : riskMetrics.maxDrawdown! <= 25 ? '#fbbf24' : '#f87171'}
              alertLevel={riskMetrics.maxDrawdown! <= 10 ? 'safe' : riskMetrics.maxDrawdown! <= 25 ? 'moderate' : 'danger'}
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                <div>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '0.4rem' }}>
                    Concentração nos 3 Maiores Ativos
                  </div>
                  <AssetRiskDetailRow
                    ticker="Concentração Top 3"
                    subText={
                      concentrationMetrics.top3Sum > 50
                        ? 'Concentração elevada (risco alto)'
                        : concentrationMetrics.top3Sum > 30
                          ? 'Concentração moderada'
                          : 'Bem diversificada'
                    }
                    valueText={`${concentrationMetrics.top3Sum.toFixed(1)}%`}
                    barPct={concentrationMetrics.top3Sum}
                    barColor={concentrationMetrics.top3Sum > 50 ? '#f87171' : concentrationMetrics.top3Sum > 30 ? '#fbbf24' : '#4ade80'}
                  />
                </div>

                {concentrationMetrics.top3.length > 0 && (
                  <div style={{ marginTop: '0.25rem' }}>
                    <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '0.4rem' }}>
                      Maiores Alocações Individuais
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                      {concentrationMetrics.top3.map((item, idx) => (
                        <AssetRiskDetailRow
                          key={item.ticker}
                          ticker={`${idx + 1}. ${item.ticker}`}
                          subText="Alocação"
                          valueText={`${item.weight.toFixed(1)}%`}
                          barPct={item.weight}
                          barColor="#00f2fe"
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </KPIScorecard>
          </div>
        ) : (
          <div style={{
            padding: '2rem',
            textAlign: 'center',
            border: '1px dashed var(--panel-border)',
            borderRadius: '12px',
            color: 'var(--text-secondary)',
          }}>
            <span style={{ fontSize: '1.75rem', display: 'block', marginBottom: '0.5rem' }}>📊</span>
            <p style={{ fontSize: '0.85rem', margin: 0 }}>Dados de performance insuficientes para calcular métricas de risco.</p>
            <p style={{ fontSize: '0.75rem', margin: '0.4rem 0 0 0', opacity: 0.7 }}>É necessário ao menos 10 dias de histórico.</p>
          </div>
        )}
      </AnalysisCard>

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* SEÇÃO 4: Geração de Renda (Proventos)                             */}
      {/* ════════════════════════════════════════════════════════════════════ */}
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

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* SEÇÃO 5: Fundamentos da Carteira                                   */}
      {/* ════════════════════════════════════════════════════════════════════ */}
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
    
      {/* ── 4. Top & Worst Performers ── */}
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
      {/* ── 5. Monthly Income Goal ── */}
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
      {/* ── 7. Valuation & Margem de Segurança ── */}
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
      {/* ── 8. Sazonalidade de Proventos ── */}
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
                          background: 'rgba(251, 191, 36, 0.8)'
                      }} />
                   )}
                   {item.pctPast > 0 && (
                      <div style={{ 
                          width: '100%', 
                          height: `${(item.pctPast / (item.pctPast + item.pctFuture)) * 100}%`, 
                          background: item.isCurrent ? '#4ade80' : 'rgba(96,165,250,0.6)' 
                      }} />
                   )}
                </div>
             </div>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem', padding: '0 2px' }}>
           {dividendSeasonality.map((item, i) => (
              <div key={i} style={{ flex: 1, textAlign: 'center' }}>
                 <span style={{ fontSize: '0.65rem', color: item.isCurrent ? '#4ade80' : 'var(--text-secondary)', fontWeight: item.isCurrent ? 700 : 400 }}>
                    {item.monthLabel}
                 </span>
              </div>
           ))}
        </div>
        
        {upcomingDividends.length > 0 && (
           <div style={{ marginTop: '1.25rem', padding: '0.75rem', background: 'rgba(74,222,128,0.05)', borderRadius: '8px', border: '1px solid rgba(74,222,128,0.2)' }}>
              <p style={{ fontSize: '0.75rem', color: '#4ade80', fontWeight: 600, margin: 0, marginBottom: '0.2rem' }}>💰 Proventos a Receber</p>
              <p style={{ fontSize: '1.1rem', color: 'var(--text-primary)', fontWeight: 700, margin: 0, fontVariantNumeric: 'tabular-nums' }}>
                 {formatMoney(upcomingDividends.reduce((s, d) => s + d.net_amount, 0), 'BRL')}
              </p>
           </div>
        )}
      </AnalysisCard>
      {/* ── 9. Liquidez e Renda Fixa ── */}
      {fiPositions.length > 0 && (
        <AnalysisCard>
          <SectionTitle emoji="💧" title="Liquidez da Renda Fixa" subtitle="Perfil de vencimento dos seus ativos" />
          
          <div style={{ display: 'flex', height: '12px', borderRadius: '6px', overflow: 'hidden', marginBottom: '1rem' }}>
            {fiLiquidity.map((item, i) => (
              <div key={i} style={{ width: `${(item.value / fiLiquidity.reduce((s, x) => s + x.value, 0)) * 100}%`, background: item.color }} title={`${item.label}: ${formatMoney(item.value, kpiCurrency)}
`} />
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
                   <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{pct.toFixed(1)}
%</span>
                 </div>
               );
            })}

          </div>
        </AnalysisCard>
      )}

    </div>
  );
}
