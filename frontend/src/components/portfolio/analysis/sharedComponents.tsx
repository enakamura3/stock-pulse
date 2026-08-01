import React, { useState } from 'react';

export function SectionTitle({ emoji, title, subtitle }: { emoji: string; title: string; subtitle?: string }) {
  return (
    <div className="mb-md">
      <h3 className="font-bold" style={{ fontSize: '1rem', color: 'var(--text-primary)' }}>
        {emoji} {title}
      </h3>
      {subtitle && <p className="text-secondary" style={{ fontSize: '0.78rem', marginTop: '0.2rem' }}>{subtitle}</p>}
    </div>
  );
}

export function ProgressBar({
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

export function AlertBadge({ type, message }: { type: 'warning' | 'info' | 'success'; message: string }) {
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

export function AnalysisCard({ children, style, id }: { children: React.ReactNode; style?: React.CSSProperties; id?: string }) {
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

export function AssetRiskDetailRow({
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

export function KPIScorecard({
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

export function StatPill({ label, value, color }: { label: string; value: string; color: string }) {
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

export function ChartTooltipShell({ active, payload, label, formatter }: any) {
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

export function DonutCenterLabel({ viewBox, title, value }: { viewBox?: any; title: string; value: string }) {
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
