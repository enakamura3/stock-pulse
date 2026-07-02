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

interface DividendsYearlyChartProps {
  data: CalculatedDividend[];
}

export default function DividendsYearlyChart({ data }: DividendsYearlyChartProps) {
  const chartData = useMemo(() => {
    const grouped = data.reduce((acc, div) => {
      // Use payment date if available, else ex_date
      const year = div.payment_date && !div.payment_date.startsWith('0001') 
        ? div.payment_date.substring(0, 4) 
        : div.ex_date.substring(0, 4);
        
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
          background: 'rgba(15, 23, 42, 0.95)',
          border: '1px solid rgba(255,255,255,0.1)',
          padding: '1rem',
          borderRadius: '8px',
          boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
          backdropFilter: 'blur(10px)'
        }}>
          <p style={{ margin: '0 0 0.5rem 0', fontWeight: 700, color: '#fff' }}>Ano: {label}</p>
          <p style={{ margin: '0', color: '#3b82f6', fontWeight: 700, display: 'flex', justifyContent: 'space-between', gap: '1rem' }}>
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
        <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
        <XAxis dataKey="name" stroke="rgba(255,255,255,0.5)" fontSize={12} tickLine={false} axisLine={false} />
        <YAxis stroke="rgba(255,255,255,0.5)" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => `R$${val}`} />
        <Tooltip content={<CustomTooltip />} cursor={{ fill: 'rgba(255,255,255,0.05)' }} />
        <Bar dataKey="total" fill="#3b82f6" radius={[4, 4, 0, 0]} maxBarSize={40} />
      </BarChart>
    </ResponsiveContainer>
  );
}
