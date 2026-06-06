import React from 'react';
import { CalculatedDividend } from './types';
import { getAssetCategory, formatMoney } from './helpers';
import dynamic from 'next/dynamic';

const DividendsChart = dynamic(() => import('@/components/DividendsChart'), { ssr: false });

interface DividendsHistoryProps {
  dividends: CalculatedDividend[];
  filterDivYear: string;
  setFilterDivYear: (y: string) => void;
  filterDivMonth: string;
  setFilterDivMonth: (m: string) => void;
  availableYears: string[];
  isLoadingDividends: boolean;
}

export default function DividendsHistory({
  dividends, filterDivYear, setFilterDivYear, filterDivMonth, setFilterDivMonth, availableYears, isLoadingDividends
}: DividendsHistoryProps) {

  const totalRV = dividends.filter(d => !d.is_accrued).reduce((acc, curr) => acc + curr.net_amount, 0);
  const totalRF = dividends.filter(d => d.is_accrued).reduce((acc, curr) => acc + curr.net_amount, 0);

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
            <div className="flex-row gap-md mb-lg flex-wrap">
              <div className="card" style={{ flex: '1', background: 'rgba(255,255,255,0.02)', padding: '1rem', border: '1px solid var(--panel-border)' }}>
                <div className="text-secondary text-xs mb-xs">Proventos RV (Caixa Livre)</div>
                <div className="font-bold text-xl" style={{ color: '#00e676' }}>{formatMoney(totalRV, 'BRL')}</div>
              </div>
              <div className="card" style={{ flex: '1', background: 'rgba(255,255,255,0.02)', padding: '1rem', border: '1px solid var(--panel-border)' }}>
                <div className="text-secondary text-xs mb-xs">Rendimento RF (Juros Retidos)</div>
                <div className="font-bold text-xl" style={{ color: '#FFB300' }}>{formatMoney(totalRF, 'BRL')}</div>
              </div>
              <div className="card" style={{ flex: '1', background: 'rgba(255,255,255,0.02)', padding: '1rem', border: '1px solid var(--panel-border)' }}>
                <div className="text-secondary text-xs mb-xs">Geração de Valor Total</div>
                <div className="font-bold text-xl">{formatMoney(totalRV + totalRF, 'BRL')}</div>
              </div>
            </div>

            <div style={{ height: '350px', marginBottom: '2rem' }}>
              <DividendsChart data={dividends} />
            </div>
            
            <div className="table-container" style={{ border: '1px solid var(--panel-border)', borderRadius: '8px' }}>
              <table className="data-table">
                <thead>
                  <tr style={{ background: 'rgba(255,255,255,0.03)' }}>
                    <th className="text-center">Ativo</th>
                    <th className="text-center">Papel</th>
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
                  {dividends.map((div, i) => (
                    <tr key={i}>
                      <td className="text-center font-semibold">{div.ticker}</td>
                      <td className="text-center text-secondary text-xs">{div.is_accrued ? 'Renda Fixa' : getAssetCategory(div.asset_type)}</td>
                      <td className="text-center">
                        <span className="badge" style={{
                          backgroundColor: div.is_accrued ? 'rgba(255, 179, 0, 0.15)' :
                                          !div.type ? 'rgba(255,255,255,0.1)' : 
                                          div.type.toLowerCase().includes('jcp') ? 'rgba(255, 152, 0, 0.15)' :
                                          div.type.toLowerCase().includes('rendimento') ? 'rgba(156, 39, 176, 0.15)' :
                                          div.type.toLowerCase().includes('amorti') ? 'rgba(244, 67, 54, 0.15)' :
                                          'rgba(33, 150, 243, 0.15)',
                          color: div.is_accrued ? '#FFB300' :
                                 !div.type ? '#aaa' : 
                                 div.type.toLowerCase().includes('jcp') ? '#ff9800' :
                                 div.type.toLowerCase().includes('rendimento') ? '#e040fb' :
                                 div.type.toLowerCase().includes('amorti') ? '#ff5252' :
                                 '#64b5f6'
                        }}>
                          {div.is_accrued ? '🏦 JUROS ACUMULADOS' : (div.type || 'DIVIDENDO')}
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
                  ))}
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
