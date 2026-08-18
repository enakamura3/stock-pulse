import React, { useState, useMemo } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { useThemeOptional } from '@/components/ThemeProvider';
import { PerformancePoint } from '../types';
import { BENCHMARK_COLORS } from './constants';
import { SectionTitle, AnalysisCard } from './sharedComponents';
import { BenchmarkPoint } from './types';

interface PerformanceBenchmarkSectionProps {
  performanceData: PerformancePoint[];
}

export default function PerformanceBenchmarkSection({ performanceData }: PerformanceBenchmarkSectionProps) {
  const [showReal, setShowReal] = useState(false);
  const themeContext = useThemeOptional();
  const isLight = themeContext?.theme === 'light';
  
  const strokeColor = isLight ? 'rgba(0, 0, 0, 0.5)' : 'rgba(255, 255, 255, 0.4)';
  const gridColor = isLight ? 'rgba(0, 0, 0, 0.06)' : 'rgba(255, 255, 255, 0.05)';

  const benchmarkData = useMemo((): BenchmarkPoint[] => {
    if (!performanceData || performanceData.length === 0) return [];

    const step = Math.max(1, Math.floor(performanceData.length / 60));
    const sampled = performanceData.filter((_, i) => i === 0 || i === performanceData.length - 1 || i % step === 0);

    return sampled.map((point) => {
      const d = new Date(point.date);
      const utcDate = new Date(d.getTime() + d.getTimezoneOffset() * 60 * 1000);
      const label = utcDate.toLocaleDateString('pt-BR', { month: 'short', year: '2-digit' });

      return {
        label,
        portfolio: Number((point.return_pct ?? 0).toFixed(2)),
        cdi: Number((point.cdi_return_pct ?? 0).toFixed(2)),
        ipca: Number((point.ipca_return_pct ?? 0).toFixed(2)),
        ifix: Number((point.ifix_return_pct ?? 0).toFixed(2)),
        ibov: Number((point.ibov_return_pct ?? 0).toFixed(2)),
        sp500: Number((point.sp500_return_pct ?? 0).toFixed(2)),
      };
    });
  }, [performanceData]);

  const chartData = useMemo(() => {
    if (!showReal) return benchmarkData;
    return benchmarkData.map((p) => {
      const ipcaFactor = 1 + p.ipca / 100;
      const deflate = (v: number): number => {
        if (Math.abs(ipcaFactor) < 1e-6) return v;
        const real = ((1 + v / 100) / ipcaFactor - 1) * 100;
        return Number(real.toFixed(2));
      };

      return {
        ...p,
        portfolio: deflate(p.portfolio),
        cdi: deflate(p.cdi),
        ifix: deflate(p.ifix),
        ibov: deflate(p.ibov),
        sp500: deflate(p.sp500),
      };
    });
  }, [benchmarkData, showReal]);

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

  return (
    <AnalysisCard id="section-benchmarks">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '0.5rem' }}>
        <SectionTitle
          emoji="📈"
          title="Comparação com Benchmarks"
          subtitle={
            showReal
              ? 'Rentabilidade real acumulada descontando a inflação (IPCA)'
              : 'Rentabilidade nominal acumulada da carteira vs principais indicadores do mercado'
          }
        />
        {benchmarkData.length > 0 && (
          <button
            onClick={() => setShowReal(!showReal)}
            style={{
              background: showReal ? 'var(--accent-bg)' : 'var(--input-bg)',
              border: `1px solid ${showReal ? 'rgba(var(--accent-rgb), 0.4)' : 'var(--panel-border)'}`,
              borderRadius: '8px',
              padding: '0.4rem 0.85rem',
              fontSize: '0.75rem',
              fontWeight: 600,
              color: showReal ? 'var(--accent-color)' : 'var(--text-secondary)',
              cursor: 'pointer',
              transition: 'all 0.2s ease',
            }}
          >
            {showReal ? '📊 Retorno Real (IPCA)' : '📊 Retorno Nominal'}
          </button>
        )}
      </div>

      {benchmarkData.length > 0 ? (
        <>
          <ResponsiveContainer width="100%" height={340}>
            <LineChart data={chartData} margin={{ top: 10, right: 15, left: 5, bottom: 20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={gridColor} vertical={false} />
              <XAxis
                dataKey="label"
                stroke={strokeColor}
                fontSize={11}
                tickMargin={10}
                axisLine={false}
                tickLine={false}
                interval="preserveStartEnd"
              />
              <YAxis
                stroke={strokeColor}
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
              {!showReal && (
                <Line type="monotone" dataKey="ipca" name="IPCA+" stroke={BENCHMARK_COLORS.ipca} strokeWidth={1.5} dot={false} strokeDasharray="4 4" opacity={0.6} />
              )}
              <Line type="monotone" dataKey="ifix" name="IFIX" stroke={BENCHMARK_COLORS.ifix} strokeWidth={1.5} dot={false} strokeDasharray="8 4" opacity={0.6} />
              <Line type="monotone" dataKey="ibov" name="Ibovespa" stroke={BENCHMARK_COLORS.ibov} strokeWidth={1.5} dot={false} strokeDasharray="5 5" opacity={0.6} />
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
            💡 <strong style={{ color: 'var(--text-primary)' }}>Nota:</strong> Os benchmarks utilizam dados históricos reais obtidos da B3 (IFIX e Ibovespa), Banco Central (CDI e IPCA) e S&P 500 (com câmbio ajustado para BRL se a moeda base da carteira for Real). {showReal && 'No modo Retorno Real, os valores são deflacionados pelo IPCA acumulado do período.'}
          </div>
        </>
      ) : (
        <div style={{ height: '280px', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', border: '1px dashed var(--panel-border)', borderRadius: '12px', color: 'var(--text-secondary)' }}>
          <span style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>📉</span>
          <p style={{ fontSize: '0.85rem', margin: 0 }}>Dados de performance insuficientes para gerar a comparação.</p>
        </div>
      )}
    </AnalysisCard>
  );
}

