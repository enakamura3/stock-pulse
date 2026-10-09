'use client';

import React, { useEffect, useRef, useState } from 'react';
import { createChart, ColorType, IChartApi, ISeriesApi, AreaSeries, LineSeries, LineStyle } from 'lightweight-charts';
import { useTheme } from '@/components/ThemeProvider';
import { useThemeColors } from '@/hooks/useThemeColors';
import { formatMoney, formatPercentage } from '@/components/portfolio/helpers';

interface ChartPoint {
  date: string;
  value: number;
  total_invested: number;
}

interface PortfolioChartProps {
  data: ChartPoint[];
}

export function toRgba(colorStr: string, opacity: number): string {
  if (!colorStr) return `rgba(255, 107, 0, ${opacity})`;
  if (colorStr.startsWith('rgb(')) {
    return colorStr.replace('rgb(', 'rgba(').replace(')', `, ${opacity})`);
  }
  if (colorStr.startsWith('rgba(')) {
    return colorStr.replace(/,\s*[\d.]+\)$/, `, ${opacity})`);
  }
  if (colorStr.startsWith('#')) {
    let hex = colorStr.slice(1);
    if (hex.length === 3) {
      hex = hex.split('').map(c => c + c).join('');
    }
    const r = parseInt(hex.substring(0, 2), 16) || 0;
    const g = parseInt(hex.substring(2, 4), 16) || 0;
    const b = parseInt(hex.substring(4, 6), 16) || 0;
    return `rgba(${r}, ${g}, ${b}, ${opacity})`;
  }
  return colorStr;
}

export default function PortfolioChart({ data }: PortfolioChartProps) {
  const { theme } = useTheme();
  const colors = useThemeColors();
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const valueSeriesRef = useRef<ISeriesApi<'Area'> | null>(null);
  const investedSeriesRef = useRef<ISeriesApi<'Area'> | null>(null);

  const [showValue, setShowValue] = useState(true);
  const [showInvested, setShowInvested] = useState(true);
  const [viewMode, setViewMode] = useState<'currency' | 'percent'>('currency');
  const [hoveredPoint, setHoveredPoint] = useState<ChartPoint | null>(null);

  // Ponto ativo para exibição de métricas na barra de telemetria
  const activePoint = hoveredPoint || (data.length > 0 ? data[data.length - 1] : null);

  useEffect(() => {
    if (!containerRef.current || data.length === 0) return;

    const isLight = theme === 'light';
    const textColor = isLight ? 'rgba(24, 24, 27, 0.65)' : 'rgba(250, 250, 250, 0.45)';
    const gridColor = isLight ? 'rgba(0, 0, 0, 0.05)' : 'rgba(255, 255, 255, 0.02)';

    // Configuração do container do gráfico
    const chart = createChart(containerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor,
        fontSize: 10,
        attributionLogo: false,
      },
      grid: {
        vertLines: { color: gridColor },
        horzLines: { color: gridColor },
      },
      crosshair: {
        vertLine: {
          color: isLight ? 'rgba(0, 0, 0, 0.25)' : 'rgba(255, 255, 255, 0.2)',
          width: 1,
          style: LineStyle.LargeDashed,
        },
        horzLine: {
          color: isLight ? 'rgba(0, 0, 0, 0.25)' : 'rgba(255, 255, 255, 0.2)',
          width: 1,
          style: LineStyle.LargeDashed,
        },
      },
      width: containerRef.current.clientWidth,
      height: 300,
      timeScale: {
        borderVisible: false,
        timeVisible: false,
      },
      rightPriceScale: {
        borderVisible: false,
      },
    });

    let valueSeries: ISeriesApi<'Area'> | null = null;
    let investedSeries: ISeriesApi<'Area'> | null = null;

    // Formatador de preço
    const priceFormat = viewMode === 'percent' 
      ? { type: 'custom' as const, formatter: (price: number) => `${price.toFixed(2)}%`, minMove: 0.01 }
      : { type: 'price' as const, precision: 2, minMove: 0.01 };

    // Série 1: Valor de Mercado (Patrimônio) - Linha fina de 1px com degradê névoa (10%)
    if (showValue) {
      valueSeries = chart.addSeries(AreaSeries, {
        lineColor: colors.accent,
        topColor: toRgba(colors.accent, 0.10),
        bottomColor: toRgba(colors.accent, 0.0),
        lineWidth: 1,
        lineStyle: LineStyle.Solid,
        lastValueVisible: true,
        priceLineVisible: true,
        priceFormat,
      });

      const valueData = data.map((pt) => {
        let val = pt.value;
        if (viewMode === 'percent') {
          val = pt.total_invested > 1e-6 ? ((pt.value - pt.total_invested) / pt.total_invested) * 100 : 0;
        }
        return { time: pt.date, value: val };
      });
      valueSeries.setData(valueData);
    }

    // Série 2: Valor Investido (Referência) - Linha tracejada fina de 1px com degradê névoa (10%)
    if (showInvested) {
      const isPercent = viewMode === 'percent';
      investedSeries = chart.addSeries(AreaSeries, {
        lineColor: colors.success,
        topColor: toRgba(colors.success, isPercent ? 0.0 : 0.10),
        bottomColor: toRgba(colors.success, 0.0),
        lineWidth: 1,
        lineStyle: LineStyle.Dashed,
        lastValueVisible: false, // Suprime badge duplicado/colidente no eixo Y
        priceLineVisible: false,
        priceFormat,
      });

      const investedData = data.map((pt) => {
        let val = pt.total_invested;
        if (viewMode === 'percent') {
          val = 0; // Baseline é zero no modo %
        }
        return { time: pt.date, value: val };
      });
      investedSeries.setData(investedData);
    }

    // Mapa rápido de pontos por data para sincronização com o crosshair
    const dataMap = new Map<string, ChartPoint>();
    data.forEach((pt) => dataMap.set(pt.date, pt));

    const handleCrosshairMove = (param: any) => {
      if (!param || !param.time || param.point === undefined) {
        setHoveredPoint(null);
        return;
      }
      const timeStr = typeof param.time === 'string'
        ? param.time
        : (param.time?.year
            ? `${param.time.year}-${String(param.time.month).padStart(2, '0')}-${String(param.time.day).padStart(2, '0')}`
            : String(param.time));
      const pt = dataMap.get(timeStr);
      setHoveredPoint(pt || null);
    };

    chart.subscribeCrosshairMove(handleCrosshairMove);

    chart.timeScale().fitContent();

    chartRef.current = chart;
    valueSeriesRef.current = valueSeries;
    investedSeriesRef.current = investedSeries;

    // Redimensionamento automático responsivo
    const handleResize = () => {
      if (containerRef.current && chartRef.current) {
        chartRef.current.applyOptions({ width: containerRef.current.clientWidth });
      }
    };
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.unsubscribeCrosshairMove(handleCrosshairMove);
      chart.remove();
      chartRef.current = null;
    };
  }, [data, showValue, showInvested, viewMode, theme, colors]);

  // Cálculos de métricas para o Telemetry HUD
  const pnlMoney = activePoint ? activePoint.value - activePoint.total_invested : 0;
  const pnlPercent = activePoint && activePoint.total_invested > 1e-6
    ? (pnlMoney / activePoint.total_invested) * 100
    : 0;
  const isPos = pnlMoney >= -1e-6;

  return (
    <div className="portfolio-chart-container" style={{ position: 'relative', width: '100%' }}>
      {/* Regra defensiva para suprimir qualquer símbolo de atribuição remanescente */}
      <style>{`
        .portfolio-chart-container a#tv-attr-logo,
        a#tv-attr-logo {
          display: none !important;
          visibility: hidden !important;
          pointer-events: none !important;
        }
      `}</style>

      {/* Controles Dinâmicos */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.6rem', fontSize: '0.72rem', fontWeight: 600, fontFamily: 'var(--font-mono)' }}>
        {/* Toggle Linhas com identificadores visuais */}
        <div style={{ display: 'flex', gap: '1.25rem', alignItems: 'center' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer', userSelect: 'none' }}>
            <input 
              type="checkbox" 
              checked={showValue} 
              onChange={e => setShowValue(e.target.checked)} 
              style={{ accentColor: 'var(--accent-color)' }} 
            />
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', color: 'var(--text-primary)' }}>
              <span style={{ color: colors.accent, fontWeight: 'bold' }}>━━</span> Evolução Patrimonial
            </span>
          </label>
          <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer', userSelect: 'none' }}>
            <input 
              type="checkbox" 
              checked={showInvested} 
              onChange={e => setShowInvested(e.target.checked)} 
              style={{ accentColor: 'var(--color-success)' }} 
            />
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', color: 'var(--text-secondary)' }}>
              <span style={{ color: colors.success, fontWeight: 'bold' }}>╌╌</span> Valor Investido
            </span>
          </label>
        </div>
        
        {/* Toggle Moeda / Percentual */}
        <div style={{ display: 'flex', gap: '2px', background: 'var(--panel-bg)', padding: '2px', border: '1px solid var(--panel-border)' }}>
          <button 
            onClick={() => setViewMode('currency')}
            style={{ 
              padding: '2px 8px', 
              borderRadius: 0, 
              border: viewMode === 'currency' ? '1px solid var(--accent-color)' : '1px solid transparent',
              background: viewMode === 'currency' ? 'var(--accent-bg, rgba(255, 107, 0, 0.15))' : 'transparent', 
              color: viewMode === 'currency' ? 'var(--accent-color)' : 'var(--text-secondary)',
              cursor: 'pointer',
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
              fontSize: '0.7rem',
            }}
          >
            R$
          </button>
          <button 
            onClick={() => setViewMode('percent')}
            style={{ 
              padding: '2px 8px', 
              borderRadius: 0, 
              border: viewMode === 'percent' ? '1px solid var(--accent-color)' : '1px solid transparent',
              background: viewMode === 'percent' ? 'var(--accent-bg, rgba(255, 107, 0, 0.15))' : 'transparent', 
              color: viewMode === 'percent' ? 'var(--accent-color)' : 'var(--text-secondary)',
              cursor: 'pointer',
              fontWeight: 600,
              fontFamily: 'var(--font-mono)',
              fontSize: '0.7rem',
            }}
          >
            %
          </button>
        </div>
      </div>

      {/* Terminal Telemetry HUD Bar */}
      {activePoint && (
        <div
          data-testid="portfolio-chart-hud"
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '12px',
            padding: '0.4rem 0.75rem',
            marginBottom: '0.65rem',
            background: 'var(--panel-bg)',
            border: '1px solid var(--panel-border)',
            fontFamily: 'var(--font-mono)',
            fontSize: '0.72rem',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ color: 'var(--text-secondary)', opacity: 0.7 }}>[DATA]</span>
            <strong style={{ color: 'var(--text-primary)' }}>{activePoint.date}</strong>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '16px', flexWrap: 'wrap' }}>
            {showValue && (
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <span style={{ color: colors.accent, fontWeight: 700 }}>[PAT]</span>
                <span style={{ color: 'var(--text-secondary)' }}>Patrimônio:</span>
                <strong style={{ color: 'var(--text-primary)' }}>
                  {viewMode === 'percent'
                    ? formatPercentage(pnlPercent)
                    : formatMoney(activePoint.value, 'BRL')}
                </strong>
              </div>
            )}

            {showInvested && (
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <span style={{ color: colors.success, fontWeight: 700 }}>[INV]</span>
                <span style={{ color: 'var(--text-secondary)' }}>Investido:</span>
                <strong style={{ color: 'var(--text-primary)' }}>
                  {viewMode === 'percent'
                    ? '0.00% (Ref)'
                    : formatMoney(activePoint.total_invested, 'BRL')}
                </strong>
              </div>
            )}

            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <span style={{ color: 'var(--text-secondary)', opacity: 0.7 }}>[RET]</span>
              <span style={{ color: 'var(--text-secondary)' }}>Retorno:</span>
              <strong style={{ color: isPos ? 'var(--color-success)' : 'var(--color-danger)' }}>
                {isPos ? '+' : ''}{formatMoney(pnlMoney, 'BRL')} ({formatPercentage(pnlPercent)})
              </strong>
            </div>
          </div>
        </div>
      )}

      <div ref={containerRef} style={{ width: '100%', minHeight: '300px' }} />
    </div>
  );
}
