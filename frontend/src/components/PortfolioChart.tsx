'use client';

import React, { useEffect, useRef } from 'react';
import { createChart, ColorType, IChartApi, ISeriesApi, AreaSeries, LineSeries } from 'lightweight-charts';

interface ChartPoint {
  date: string;
  value: number;
  total_invested: number;
}

interface PortfolioChartProps {
  data: ChartPoint[];
}

export default function PortfolioChart({ data }: PortfolioChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const valueSeriesRef = useRef<ISeriesApi<'Area'> | null>(null);
  const costSeriesRef = useRef<ISeriesApi<'Line'> | null>(null);

  useEffect(() => {
    if (!containerRef.current || data.length === 0) return;

    // Remove chart antigo se houver re-render
    if (chartRef.current) {
      chartRef.current.remove();
    }

    // Configuração do container do gráfico
    const chart = createChart(containerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor: 'rgba(255, 255, 255, 0.45)',
        fontSize: 10,
      },
      grid: {
        vertLines: { color: 'rgba(255, 255, 255, 0.02)' },
        horzLines: { color: 'rgba(255, 255, 255, 0.02)' },
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

    // Série 1: Valor de Mercado (Area Series Neon Cyan)
    const valueSeries = chart.addSeries(AreaSeries, {
      lineColor: '#00f2fe',
      topColor: 'rgba(0, 242, 254, 0.15)',
      bottomColor: 'rgba(0, 242, 254, 0.0)',
      lineWidth: 2,
      priceFormat: {
        type: 'price',
        precision: 2,
        minMove: 0.01,
      },
    });

    // Série 2: Custo de Aquisição (Line Series Dashed White)
    const costSeries = chart.addSeries(LineSeries, {
      color: 'rgba(255, 255, 255, 0.35)',
      lineWidth: 1.5,
      lineStyle: 2, // Dashed
      priceFormat: {
        type: 'price',
        precision: 2,
        minMove: 0.01,
      },
    });

    // Formata pontos para a biblioteca
    const valueData = data.map((pt) => ({
      time: pt.date,
      value: pt.value,
    }));

    const costData = data.map((pt) => ({
      time: pt.date,
      value: pt.total_invested,
    }));

    valueSeries.setData(valueData);
    costSeries.setData(costData);

    chart.timeScale().fitContent();

    chartRef.current = chart;
    valueSeriesRef.current = valueSeries;
    costSeriesRef.current = costSeries;

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
  }, [data]);

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      {/* Legenda Dinâmica no Topo */}
      <div style={{ display: 'flex', gap: '1.5rem', marginBottom: '0.8rem', fontSize: '0.75rem', fontWeight: 600 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
          <span style={{ display: 'inline-block', width: '8px', height: '8px', borderRadius: '50%', background: '#00f2fe' }}></span>
          <span style={{ color: 'var(--text-primary)' }}>Evolução Patrimonial</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
          <span style={{ display: 'inline-block', width: '10px', height: '2px', borderTop: '2px dashed rgba(255,255,255,0.6)' }}></span>
          <span style={{ color: 'var(--text-secondary)' }}>Custo Médio Investido</span>
        </div>
      </div>
      <div ref={containerRef} style={{ width: '100%', minHeight: '300px' }} />
    </div>
  );
}
