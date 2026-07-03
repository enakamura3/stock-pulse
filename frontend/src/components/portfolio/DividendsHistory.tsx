import React, { useMemo } from 'react';
import { CalculatedDividend } from './types';
import { getAssetCategory, formatMoney } from './helpers';
import dynamic from 'next/dynamic';

const DividendsMatrix = dynamic(() => import('./DividendsMatrix'), { ssr: false });

interface DividendsHistoryProps {
  dividends: CalculatedDividend[];
  allDividends?: CalculatedDividend[];
  filterDivYear: string;
  setFilterDivYear: (y: string) => void;
  filterDivMonth: string;
  setFilterDivMonth: (m: string) => void;
  availableYears: string[];
  isLoadingDividends: boolean;
}

export default function DividendsHistory({
  dividends, allDividends = [], filterDivYear, setFilterDivYear, filterDivMonth, setFilterDivMonth, availableYears, isLoadingDividends
}: DividendsHistoryProps) {

  // Função utilitária para checar pagamento
  const isPaid = (div: CalculatedDividend) => {
    if (!div.payment_date || div.payment_date.startsWith('0001')) return false;
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const [year, month, day] = div.payment_date.split('T')[0].split('-');
    const payDate = new Date(parseInt(year), parseInt(month) - 1, parseInt(day));
    return payDate <= today;
  };

  const formatType = (div: CalculatedDividend) => {
    if (div.is_accrued) return 'Juros';
    if (!div.type) return 'Dividendo';
    const lower = div.type.toLowerCase();
    if (lower.includes('jcp')) return 'JCP';
    if (lower.includes('rendimento')) return 'Rendimento';
    if (lower.includes('amorti')) return 'Amortização';
    return div.type.charAt(0).toUpperCase() + div.type.slice(1).toLowerCase();
  };

  // Agrupamentos e Reduções
  const stats = useMemo(() => {
    const s = {
      totalPaid: 0, totalPending: 0,
      rvPaid: 0, rvPending: 0,
      rfPaid: 0, rfPending: 0,
      types: {} as Record<string, number>
    };

    dividends.forEach(d => {
      const paid = isPaid(d);
      const amt = d.net_amount;
      const groupStr = d.is_accrued ? 'Renda Fixa' : getAssetCategory(d.asset_type);

      if (paid) s.totalPaid += amt; else s.totalPending += amt;
      
      if (d.is_accrued) {
        if (paid) s.rfPaid += amt; else s.rfPending += amt;
      } else {
        if (paid) s.rvPaid += amt; else s.rvPending += amt;
      }

      if (s.types[groupStr]) s.types[groupStr] += amt;
      else s.types[groupStr] = amt;
    });

    return s;
  }, [dividends]);

  return (
    <div className="flex-col gap-lg">
      <div className="card">
        <div className="flex-row justify-between items-center mb-lg flex-wrap gap-md">
          <h3 className="card-title">💰 Histórico de Proventos</h3>
          <div className="flex-row gap-sm">
            <select
              value={filterDivYear}
              onChange={(e) => setFilterDivYear(e.target.value)}
              style={{ padding: '0.3rem 0.6rem', borderRadius: '6px', border: '1px solid var(--panel-border)', background: '#1E293B', color: '#FFFFFF', fontSize: '0.8rem', outline: 'none', cursor: 'pointer', width: 'auto' }}
            >
              <option value="Todos">Todos os Anos</option>
              {availableYears.map(year => (
                <option key={year} value={year}>{year}</option>
              ))}
            </select>
            <select
              value={filterDivMonth}
              onChange={(e) => setFilterDivMonth(e.target.value)}
              style={{ padding: '0.3rem 0.6rem', borderRadius: '6px', border: '1px solid var(--panel-border)', background: '#1E293B', color: '#FFFFFF', fontSize: '0.8rem', outline: 'none', cursor: 'pointer', width: 'auto' }}
            >
              <option value="Todos">Todos os Meses</option>
              {['01','02','03','04','05','06','07','08','09','10','11','12'].map(m => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
          </div>
        </div>

        {isLoadingDividends ? (
          <div className="text-center text-secondary p-xl">Carregando proventos...</div>
        ) : dividends.length > 0 ? (
          <>
            {/* Main KPIs */}
            <div className="flex-row gap-md mb-md flex-wrap">
              <div className="card" style={{ flex: '1', background: 'rgba(255,255,255,0.02)', padding: '1rem', border: '1px solid var(--panel-border)' }}>
                <div className="text-secondary text-xs mb-xs font-bold">Proventos RV (Caixa Livre)</div>
                <div className="font-bold text-2xl" style={{ color: '#00e676' }}>{formatMoney(stats.rvPaid + stats.rvPending, 'BRL')}</div>
                <div className="text-xs text-secondary mt-xs" style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span>Pago: {formatMoney(stats.rvPaid, 'BRL')}</span>
                  <span>Pendente: {formatMoney(stats.rvPending, 'BRL')}</span>
                </div>
              </div>
              <div className="card" style={{ flex: '1', background: 'rgba(255,255,255,0.02)', padding: '1rem', border: '1px solid var(--panel-border)' }}>
                <div className="text-secondary text-xs mb-xs font-bold">Rendimento RF (Juros Retidos)</div>
                <div className="font-bold text-2xl" style={{ color: '#FFB300' }}>{formatMoney(stats.rfPaid + stats.rfPending, 'BRL')}</div>
                <div className="text-xs text-secondary mt-xs" style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span>Pago: {formatMoney(stats.rfPaid, 'BRL')}</span>
                  <span>Pendente: {formatMoney(stats.rfPending, 'BRL')}</span>
                </div>
              </div>
              <div className="card" style={{ flex: '1', background: 'rgba(255,255,255,0.02)', padding: '1rem', border: '1px solid var(--panel-border)' }}>
                <div className="text-secondary text-xs mb-xs font-bold">Geração de Valor Total</div>
                <div className="font-bold text-2xl" style={{ color: '#fff' }}>{formatMoney(stats.totalPaid + stats.totalPending, 'BRL')}</div>
                <div className="text-xs text-secondary mt-xs" style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ color: 'rgba(0, 230, 118, 0.8)' }}>Pago: {formatMoney(stats.totalPaid, 'BRL')}</span>
                  <span style={{ color: 'rgba(255, 152, 0, 0.8)' }}>Pendente: {formatMoney(stats.totalPending, 'BRL')}</span>
                </div>
              </div>
            </div>

            {/* Sub-cards by Type */}
            <div className="flex-row gap-sm mb-lg flex-wrap">
              {Object.entries(stats.types).sort((a, b) => b[1] - a[1]).map(([type, total]) => (
                <div key={type} style={{ background: 'rgba(255,255,255,0.03)', padding: '0.5rem 1rem', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.05)', display: 'flex', flexDirection: 'column' }}>
                  <span className="text-xs text-secondary mb-xs">{type}</span>
                  <span className="font-bold text-sm">{formatMoney(total, 'BRL')}</span>
                </div>
              ))}
            </div>

            <DividendsMatrix 
              data={allDividends.length > 0 ? allDividends : dividends} 
              onYearClick={(y) => { setFilterDivYear(y); setFilterDivMonth('Todos'); }}
              onMonthClick={(y, m) => { setFilterDivYear(y); setFilterDivMonth(m); }}
              activeYear={filterDivYear}
              activeMonth={filterDivMonth}
            />

            {/* Table */}
            <div className="table-container" style={{ border: '1px solid var(--panel-border)', borderRadius: '8px' }}>
              <table className="data-table">
                <thead>
                  <tr style={{ background: 'rgba(255,255,255,0.03)' }}>
                    <th className="text-center">Status</th>
                    <th className="text-center">Ativo</th>
                    <th className="text-center">Categoria</th>
                    <th className="text-center">Tipo</th>
                    <th className="text-center">Data Com</th>
                    <th className="text-center">Pagamento</th>
                    <th className="text-center">Qtd</th>
                    <th className="text-right">Vlr / Cota</th>
                    <th className="text-right">Vlr Bruto</th>
                    <th className="text-right">Vlr Líquido</th>
                  </tr>
                </thead>
                <tbody>
                  {dividends.map((div, i) => {
                    const paid = isPaid(div);
                    const typeStr = formatType(div);
                    return (
                      <tr key={i}>
                        <td className="text-center">
                          {paid ? (
                            <span className="badge" style={{ backgroundColor: 'rgba(0, 230, 118, 0.15)', color: '#00e676' }}>Pago</span>
                          ) : (
                            <span className="badge" style={{ backgroundColor: 'rgba(255, 152, 0, 0.15)', color: '#ff9800' }}>Pendente</span>
                          )}
                        </td>
                        <td className="text-center font-semibold">{div.ticker}</td>
                        <td className="text-center text-secondary text-xs">{div.is_accrued ? 'Renda Fixa' : getAssetCategory(div.asset_type)}</td>
                        <td className="text-center">
                          <span className="badge" style={{
                            backgroundColor: div.is_accrued ? 'rgba(0, 188, 212, 0.15)' :
                                            typeStr === 'JCP' ? 'rgba(255, 152, 0, 0.15)' :
                                            typeStr === 'Rendimento' ? 'rgba(156, 39, 176, 0.15)' :
                                            typeStr === 'Amortização' ? 'rgba(244, 67, 54, 0.15)' :
                                            'rgba(33, 150, 243, 0.15)',
                            color: div.is_accrued ? '#00bcd4' :
                                   typeStr === 'JCP' ? '#ff9800' :
                                   typeStr === 'Rendimento' ? '#e040fb' :
                                   typeStr === 'Amortização' ? '#ff5252' :
                                   '#64b5f6'
                          }}>
                            {typeStr}
                          </span>
                        </td>
                        <td className="text-center text-secondary text-xs">{new Date(div.ex_date).toISOString().split('T')[0].replace(/-/g, '/')}</td>
                        <td className="text-center text-secondary text-xs">{(!div.payment_date || div.payment_date.startsWith('0001')) ? '--' : new Date(div.payment_date).toISOString().split('T')[0].replace(/-/g, '/')}</td>
                        <td className="text-center font-semibold">{div.is_accrued ? '--' : div.quantity}</td>
                        <td className="text-right font-semibold">{div.is_accrued ? '--' : formatMoney(div.per_share_amount, div.currency)}</td>
                        <td className="text-right">
                          {formatMoney(div.gross_amount, div.currency)}
                          {div.currency === 'BRL' && div.original_gross_amount && (
                            <div className="text-xs text-secondary">(US$ {div.original_gross_amount.toFixed(2)})</div>
                          )}
                        </td>
                        <td className="text-right font-bold text-success">
                          {formatMoney(div.net_amount, div.currency)}
                          {div.currency === 'BRL' && div.original_net_amount && (
                            <div className="text-xs" style={{ color: 'rgba(0, 230, 118, 0.7)' }}>(US$ {div.original_net_amount.toFixed(2)})</div>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </>
        ) : (
          <div className="text-center text-secondary p-xl">
            <span className="text-2xl block mb-sm">🏜️</span>
            <p>Nenhum provento recebido ainda.</p>
            <p className="text-xs opacity-70">Aguarde a "Data Com" das suas ações para começar a receber!</p>
          </div>
        )}
      </div>
    </div>
  );
}
