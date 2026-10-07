import React, { useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from 'recharts';
import { CalculatedDividend } from './portfolio/types';
import { useTheme } from '@/components/ThemeProvider';

interface DividendsYearlyChartProps {
  data: CalculatedDividend[];
}

export default function DividendsYearlyChart({ data }: DividendsYearlyChartProps) {
  const { theme } = useTheme();
  const isLight = theme === 'light';
  const strokeColor = isLight ? 'rgba(0, 0, 0, 0.5)' : 'rgba(255, 255, 255, 0.5)';
  const gridColor = isLight ? 'rgba(0, 0, 0, 0.06)' : 'rgba(255, 255, 255, 0.05)';

  const chartData = useMemo(() => {
    const grouped = data.reduce((acc, div) => {
      // Use payment date if available, else cum_date
      const year = div.payment_date && !div.payment_date.startsWith('0001') 
        ? div.payment_date.substring(0, 4) 
        : div.cum_date.substring(0, 4);
        
      if (!acc[year]) {
        acc[year] = { name: year, total: 0 };
      }
      acc[year].total += div.net_amount;
      return acc;
    }, {} as Record<string, { name: string; total: number }>);

    return Object.values(grouped).sort((a, b) => a.name.localeCompare(b.name));
  }, [data]);

  if (!chartData || chartData.length === 0) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', color: 'var(--text-secondary)' }}>
        Nenhum dado anual
      </div>
    );
  }

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div style={{
          background: 'var(--panel-bg)',
          border: '1px solid var(--panel-border-strong)',
          padding: '0.75rem 1rem',
          borderRadius: 0,
          boxShadow: 'none',
          fontFamily: 'var(--font-mono)',
        }}>
          <p style={{ margin: '0 0 0.5rem 0', fontWeight: 700, color: 'var(--text-primary)', fontSize: '0.8rem' }}>Ano: {label}</p>
          <p style={{ margin: '0', color: 'var(--accent-color)', fontWeight: 700, display: 'flex', justifyContent: 'space-between', gap: '1.5rem', fontSize: '0.8rem', fontVariantNumeric: 'tabular-nums' }}>
            <span>Total:</span>
            <span>R$ {payload[0].value.toFixed(2)}</span>
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
        <CartesianGrid strokeDasharray="2 2" stroke={gridColor} vertical={false} />
        <XAxis dataKey="name" stroke={strokeColor} fontSize={11} tickLine={false} axisLine={false} />
        <YAxis stroke={strokeColor} fontSize={11} tickLine={false} axisLine={false} tickFormatter={(val) => `R$${val}`} />
        <Tooltip content={<CustomTooltip />} cursor={{ fill: isLight ? 'rgba(0,0,0,0.04)' : 'rgba(255,255,255,0.05)' }} />
        <Bar dataKey="total" fill="var(--accent-color)" radius={[0, 0, 0, 0]} maxBarSize={36} />
      </BarChart>
    </ResponsiveContainer>
  );
}
