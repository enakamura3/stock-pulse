import React, { useMemo } from 'react';
import { CalculatedDividend } from './types';
import { formatMoney } from './helpers';

interface DividendsMatrixProps {
  data: CalculatedDividend[];
  onYearClick?: (year: string) => void;
  onMonthClick?: (year: string, month: string) => void;
  activeYear?: string;
  activeMonth?: string;
}

export default function DividendsMatrix({ data, onYearClick, onMonthClick, activeYear, activeMonth }: DividendsMatrixProps) {
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
    <div className="mb-xl">
      <style>{`
        .matrix-table {
          font-variant-numeric: tabular-nums;
        }
        .matrix-table th {
          padding: 0.75rem 0.5rem !important;
          font-size: 0.7rem;
          text-transform: uppercase;
          letter-spacing: 0.05em;
          color: var(--text-secondary);
        }
        .matrix-table td {
          padding: 0.6rem 0.5rem !important;
          transition: all 0.2s ease;
        }
        .matrix-cell-clickable:hover {
          background-color: rgba(255,255,255,0.06) !important;
        }
      `}</style>
      <div className="flex-row justify-between items-center mb-md">
        <h4 className="font-bold text-secondary">📅 Mapa de Proventos (Mensal e Anual)</h4>
      </div>
      <div className="table-container" style={{ border: '1px solid var(--panel-border)', borderRadius: '12px', overflowX: 'auto', overflowY: 'hidden', background: 'rgba(0,0,0,0.1)' }}>
        <table className="data-table matrix-table">
          <thead>
            <tr style={{ background: 'rgba(255,255,255,0.02)' }}>
              <th className="text-center">Ano</th>
              {months.map(m => (
                <th key={m} className="text-right">{m}</th>
              ))}
              <th className="text-right">Total</th>
            </tr>
          </thead>
          <tbody>
            {years.map(year => (
              <tr key={year} style={{ borderBottom: '1px solid rgba(255,255,255,0.03)' }}>
                <td 
                  className="text-center font-bold matrix-cell-clickable text-sm" 
                  style={{ 
                    cursor: 'pointer', 
                    color: activeYear === year && activeMonth === 'Todos' ? '#00f2fe' : 'var(--text-primary)',
                    background: activeYear === year && activeMonth === 'Todos' ? 'rgba(0, 242, 254, 0.08)' : 'transparent',
                    borderRight: '1px solid rgba(255,255,255,0.05)'
                  }}
                  onClick={() => onYearClick && onYearClick(activeYear === year && activeMonth === 'Todos' ? 'Todos' : year)}
                  title={activeYear === year && activeMonth === 'Todos' ? 'Remover filtro de ano' : `Filtrar apenas o ano de ${year}`}
                >
                  {year}
                </td>
                {months.map((m, idx) => {
                  const val = matrix[year][idx];
                  const monthStr = String(idx + 1).padStart(2, '0');
                  const isActive = activeYear === year && activeMonth === monthStr;
                  return (
                    <td 
                      key={idx} 
                      className={`text-right text-sm ${val > 0 ? 'matrix-cell-clickable font-medium' : ''}`}
                      style={{ 
                        cursor: val > 0 ? 'pointer' : 'default',
                        background: isActive ? 'rgba(0, 242, 254, 0.12)' : 'transparent',
                        color: isActive ? '#00f2fe' : (val > 0 ? 'var(--text-primary)' : 'var(--text-secondary)')
                      }}
                      onClick={() => val > 0 && onMonthClick && onMonthClick(isActive ? 'Todos' : year, isActive ? 'Todos' : monthStr)}
                      title={val > 0 ? (isActive ? 'Remover filtro' : `Filtrar ${m}/${year}`) : undefined}
                    >
                      {val > 0 ? (
                        val.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
                      ) : (
                        <span style={{ opacity: 0.2 }}>-</span>
                      )}
                    </td>
                  );
                })}
                <td className="text-right text-sm font-bold text-success" style={{ borderLeft: '1px solid rgba(255,255,255,0.05)', background: 'rgba(0, 230, 118, 0.02)' }}>
                  {matrix[year][12].toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
