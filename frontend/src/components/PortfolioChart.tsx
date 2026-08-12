'use client';

import React, { useEffect, useRef, useState } from 'react';
import { createChart, ColorType, IChartApi, ISeriesApi, AreaSeries } from 'lightweight-charts';
import { useTheme } from '@/components/ThemeProvider';
import { useThemeColors } from '@/hooks/useThemeColors';

interface ChartPoint {
  date: string;
  value: number;
  total_invested: number;
}

interface PortfolioChartProps {
  data: ChartPoint[];
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

  useEffect(() => {
    if (!containerRef.current || data.length === 0) return;

    // Remove chart antigo se houver re-render
    if (chartRef.current) {
      chartRef.current.remove();
    }

    const isLight = theme === 'light';
    const textColor = isLight ? 'rgba(24, 24, 27, 0.65)' : 'rgba(250, 250, 250, 0.45)';
    const gridColor = isLight ? 'rgba(0, 0, 0, 0.05)' : 'rgba(255, 255, 255, 0.02)';

    const toRgba = (colorStr: string, opacity: number) => {
      if (colorStr.startsWith('rgb(')) {
        return colorStr.replace('rgb(', 'rgba(').replace(')', `, ${opacity})`);
      }
      return colorStr;
    };

    // Configuração do container do gráfico
    const chart = createChart(containerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor,
        fontSize: 10,
      },
      grid: {
        vertLines: { color: gridColor },
        horzLines: { color: gridColor },
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

    let valueSeries: ISeriesApi<"Area"> | null = null;
    let investedSeries: ISeriesApi<"Area"> | null = null;

    // Formatador de preço
    const priceFormat = viewMode === 'percent' 
      ? { type: 'custom' as const, formatter: (price: number) => `${price.toFixed(2)}%`, minMove: 0.01 }
      : { type: 'price' as const, precision: 2, minMove: 0.01 };

    // Série 1: Valor de Mercado
    if (showValue) {
      valueSeries = chart.addSeries(AreaSeries, {
        lineColor: colors.accent,
        topColor: toRgba(colors.accent, 0.15),
        bottomColor: toRgba(colors.accent, 0.0),
        lineWidth: 2,
        priceFormat,
      });

      const valueData = data.map((pt) => {
        let val = pt.value;
        if (viewMode === 'percent') {
          val = pt.total_invested > 0 ? ((pt.value - pt.total_invested) / pt.total_invested) * 100 : 0;
        }
        return { time: pt.date, value: val };
      });
      valueSeries.setData(valueData);
    }

    // Série 2: Valor Investido
    if (showInvested) {
      investedSeries = chart.addSeries(AreaSeries, {
        lineColor: colors.success,
        topColor: toRgba(colors.success, 0.15),
        bottomColor: toRgba(colors.success, 0.0),
        lineWidth: 2,
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
      chart.remove();
      chartRef.current = null;
    };
  }, [data, showValue, showInvested, viewMode, theme, colors]);

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      {/* Controles Dinâmicos */}
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.8rem', fontSize: '0.75rem', fontWeight: 600 }}>
        {/* Toggle Linhas */}
        <div style={{ display: 'flex', gap: '1.5rem' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }}>
            <input 
              type="checkbox" 
              checked={showValue} 
              onChange={e => setShowValue(e.target.checked)} 
              style={{ accentColor: 'var(--accent-color)' }} 
            />
            <span style={{ color: 'var(--text-primary)' }}>Evolução Patrimonial</span>
          </label>
          <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }}>
            <input 
              type="checkbox" 
              checked={showInvested} 
              onChange={e => setShowInvested(e.target.checked)} 
              style={{ accentColor: 'var(--color-success)' }} 
            />
            <span style={{ color: 'var(--text-secondary)' }}>Valor Investido</span>
          </label>
        </div>
        
        {/* Toggle Moeda / Percentual */}
        <div style={{ display: 'flex', gap: '8px', background: 'var(--input-bg)', padding: '2px', borderRadius: '6px' }}>
          <button 
            onClick={() => setViewMode('currency')}
            style={{ 
              padding: '2px 8px', 
              borderRadius: '4px', 
              border: 'none',
              background: viewMode === 'currency' ? 'var(--accent-gradient)' : 'transparent', 
              color: viewMode === 'currency' ? 'var(--accent-foreground)' : 'var(--text-secondary)',
              cursor: 'pointer',
              fontWeight: 600
            }}
          >
            R$
          </button>
          <button 
            onClick={() => setViewMode('percent')}
            style={{ 
              padding: '2px 8px', 
              borderRadius: '4px', 
              border: 'none',
              background: viewMode === 'percent' ? 'var(--accent-gradient)' : 'transparent', 
              color: viewMode === 'percent' ? 'var(--accent-foreground)' : 'var(--text-secondary)',
              cursor: 'pointer',
              fontWeight: 600
            }}
          >
            %
          </button>
        </div>
      </div>
      <div ref={containerRef} style={{ width: '100%', minHeight: '300px' }} />
    </div>
  );
}
