import React, { useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

interface CalculatedDividend {
  asset_id: string;
  ticker: string;
  cum_date: string;
  payment_date: string;
  gross_amount: number;
  net_amount: number;
  currency: string;
  original_gross_amount?: number;
  original_net_amount?: number;
  is_accrued?: boolean;
}

interface DividendsChartProps {
  data: CalculatedDividend[];
}

export default function DividendsChart({ data }: DividendsChartProps) {
  const chartData = useMemo(() => {
    // Agrupa por mês (YYYY-MM) usando a data de pagamento
    const grouped = data.reduce((acc, div) => {
      const month = div.payment_date ? div.payment_date.substring(0, 7) : div.cum_date.substring(0, 7); // Pega YYYY-MM
      if (!acc[month]) {
        const [yearStr, monthStr] = month.split('-');
        acc[month] = { name: month, rawDate: new Date(parseInt(yearStr), parseInt(monthStr) - 1, 1), BRL: 0, USD: 0, RF: 0 };
      }
      
      if (div.is_accrued) {
        acc[month].RF += div.net_amount;
      } else if (div.original_net_amount !== undefined && div.original_net_amount > 0) {
        // Mostra o valor convertido em BRL, mas agrupa logicamente
        acc[month].USD += div.net_amount;
      } else {
        acc[month].BRL += div.net_amount;
      }
      return acc;
    }, {} as Record<string, { name: string; rawDate: Date; BRL: number; USD: number; RF: number }>);

    // Converte para array e ordena cronologicamente
    return Object.values(grouped)
      .sort((a, b) => a.rawDate.getTime() - b.rawDate.getTime())
      .map(item => ({
        name: item.rawDate.toLocaleDateString('pt-BR', { month: 'short', year: 'numeric' }).toUpperCase(),
        'Nacionais (R$)': Number(item.BRL.toFixed(2)),
        'Internacionais (R$)': Number(item.USD.toFixed(2)),
        'Renda Fixa (R$)': Number(item.RF.toFixed(2)),
      }));
  }, [data]);

  if (!chartData || chartData.length === 0) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', color: 'var(--text-secondary)' }}>
        Gráfico indisponível
      </div>
    );
  }

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      const total = payload.reduce((sum: number, entry: any) => sum + entry.value, 0);
      return (
        <div style={{
          background: 'rgba(15, 23, 42, 0.95)',
          border: '1px solid rgba(255,255,255,0.1)',
          padding: '1rem',
          borderRadius: '8px',
          boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
          backdropFilter: 'blur(10px)'
        }}>
          <p style={{ margin: '0 0 0.5rem 0', fontWeight: 700, color: '#fff' }}>{label}</p>
          {payload.map((entry: any, index: number) => (
            <p key={index} style={{ margin: '0.25rem 0', fontSize: '0.85rem', color: entry.color, display: 'flex', justifyContent: 'space-between', gap: '1rem' }}>
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
    }
    return null;
  };

  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart
        data={chartData}
        margin={{ top: 10, right: 10, left: 0, bottom: 20 }}
      >
        <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
        <XAxis 
          dataKey="name" 
          stroke="rgba(255,255,255,0.4)" 
          fontSize={12} 
          tickMargin={10}
          axisLine={false}
          tickLine={false}
        />
        <YAxis 
          stroke="rgba(255,255,255,0.4)" 
          fontSize={12}
          tickFormatter={(value) => `R$ ${value}`}
          axisLine={false}
          tickLine={false}
        />
        <Tooltip content={<CustomTooltip />} cursor={{ fill: 'rgba(255,255,255,0.02)' }} />
        <Legend wrapperStyle={{ paddingTop: '20px' }} />
        <Bar dataKey="Nacionais (R$)" stackId="a" fill="#00e676" radius={[0, 0, 4, 4]} barSize={40} />
        <Bar dataKey="Internacionais (R$)" stackId="a" fill="#00f2fe" radius={[0, 0, 0, 0]} barSize={40} />
        <Bar dataKey="Renda Fixa (R$)" stackId="a" fill="#FFB300" radius={[4, 4, 0, 0]} barSize={40} />
      </BarChart>
    </ResponsiveContainer>
  );
}
