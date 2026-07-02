import React, { useMemo } from 'react';
import { CalculatedDividend } from './types';
import { formatMoney } from './helpers';

interface DividendsMatrixProps {
  data: CalculatedDividend[];
}

export default function DividendsMatrix({ data }: DividendsMatrixProps) {
  const { matrix, years } = useMemo(() => {
    // Record<Year, Array of 12 months + 1 for Total>
    const grouped: Record<string, number[]> = {};

    data.forEach(div => {
      // Usar payment_date se existir e não for nula/0001, senão fallback para ex_date
      const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.ex_date;
      if (!dateStr) return;
      
      const [year, monthStr] = dateStr.split('T')[0].split('-');
      const monthIdx = parseInt(monthStr, 10) - 1; // 0 a 11

      if (!grouped[year]) {
        grouped[year] = Array(13).fill(0); // 12 meses + 1 posição pro Total
      }

      grouped[year][monthIdx] += div.net_amount;
      grouped[year][12] += div.net_amount; // Adiciona ao Total do Ano (índice 12)
    });

    // Ordenar anos em ordem decrescente (do mais recente para o mais antigo)
    const sortedYears = Object.keys(grouped).sort((a, b) => b.localeCompare(a));

    return { matrix: grouped, years: sortedYears };
  }, [data]);

  if (years.length === 0) return null;

  const months = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];

  return (
    <div className="card mb-xl">
      <h3 className="card-title mb-md">📅 Matriz de Proventos (Ano x Mês)</h3>
      <div className="table-container" style={{ border: '1px solid var(--panel-border)', borderRadius: '8px' }}>
        <table className="data-table">
          <thead>
            <tr style={{ background: 'rgba(255,255,255,0.03)' }}>
              <th className="text-center">Ano</th>
              {months.map(m => (
                <th key={m} className="text-right">{m}</th>
              ))}
              <th className="text-right">Total</th>
            </tr>
          </thead>
          <tbody>
            {years.map(year => (
              <tr key={year}>
                <td className="text-center font-bold">{year}</td>
                {months.map((m, idx) => {
                  const val = matrix[year][idx];
                  return (
                    <td key={idx} className="text-right text-xs">
                      {val > 0 ? (
                        formatMoney(val, 'BRL')
                      ) : (
                        <span style={{ opacity: 0.3 }}>-</span>
                      )}
                    </td>
                  );
                })}
                <td className="text-right text-xs font-bold text-success">
                  {formatMoney(matrix[year][12], 'BRL')}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
