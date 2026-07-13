import React, { useMemo, useState } from 'react';
import { CalculatedDividend } from './types';
import { formatMoney, getAssetCategory } from './helpers';
interface AnnualSummaryProps {
  dividends: CalculatedDividend[];
  selectedYear: string;
  setSelectedYear: (y: string) => void;
  availableYears: string[];
}

export default function AnnualSummary({
  dividends,
  selectedYear,
  setSelectedYear,
  availableYears,
}: AnnualSummaryProps) {
  // Agrupa os proventos por ano e calcula as estatísticas detalhadas
  const annualData = useMemo(() => {
    if (!dividends || dividends.length === 0) return [];

    const grouped: Record<string, {
      year: number;
      totalAmount: number;
      currency: string;
      byType: Record<string, number>;
      byTicker: Record<string, number>;
      byMonth: Record<string, number>;
      highestSinglePayment: number;
      highestSinglePaymentTicker: string;
    }> = {};

    dividends.forEach(div => {
      // Usar payment_date se existir e não for nula/0001, senão cum_date
      const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.cum_date;
      if (!dateStr) return;

      const yearStr = dateStr.split('T')[0].split('-')[0];
      const year = parseInt(yearStr, 10);
      if (isNaN(year)) return;

      if (!grouped[yearStr]) {
        grouped[yearStr] = {
          year,
          totalAmount: 0,
          currency: div.currency || 'BRL',
          byType: {},
          byTicker: {},
          byMonth: {},
          highestSinglePayment: 0,
          highestSinglePaymentTicker: ''
        };
      }

      const amount = div.net_amount || 0;
      grouped[yearStr].totalAmount += amount;

      // Normalização por Categoria (ao invés de Tipo)
      let category = getAssetCategory(div.asset_type || '');
      if (div.asset_type === 'TESOURO') {
        category = 'Tesouro Direto';
      } else if (category === 'Desconhecido') {
        category = div.is_accrued ? 'Renda Fixa' : 'Outros';
      }

      grouped[yearStr].byType[category] = (grouped[yearStr].byType[category] || 0) + amount;

      // Ticker do ativo
      const ticker = div.ticker || 'OUTROS';
      grouped[yearStr].byTicker[ticker] = (grouped[yearStr].byTicker[ticker] || 0) + amount;

      // Controle Mensal
      const monthStr = dateStr.split('T')[0].split('-')[1];
      grouped[yearStr].byMonth[monthStr] = (grouped[yearStr].byMonth[monthStr] || 0) + amount;

      // Maior pagamento único
      if (amount > grouped[yearStr].highestSinglePayment) {
        grouped[yearStr].highestSinglePayment = amount;
        grouped[yearStr].highestSinglePaymentTicker = ticker;
      }
    });

    const sortedYears = Object.keys(grouped).sort((a, b) => b.localeCompare(a));
    const currentYear = new Date().getFullYear();
    const currentMonth = new Date().getMonth() + 1; // 1 a 12

    return sortedYears.map(yearStr => {
      const group = grouped[yearStr];
      const y = group.year;

      // Cálculo do divisor de meses
      let monthsCount = 12;
      if (y === currentYear) {
        monthsCount = currentMonth;
        if (monthsCount < 1) monthsCount = 1;
      } else if (y > currentYear) {
        monthsCount = 12;
      }

      const monthlyAverage = group.totalAmount / monthsCount;

      // Cálculo do crescimento YoY%
      let growthPct: number | null = null;
      const prevYearStr = String(y - 1);
      if (grouped[prevYearStr]) {
        const prevTotal = grouped[prevYearStr].totalAmount;
        if (prevTotal > 0) {
          growthPct = ((group.totalAmount - prevTotal) / prevTotal) * 100;
        }
      }

      // Tipos ordenados
      const byTypeList = Object.entries(group.byType)
        .map(([type, amt]) => ({
          type,
          amount: amt,
          pct: group.totalAmount > 0 ? (amt / group.totalAmount) * 100 : 0,
        }))
        .sort((a, b) => b.amount - a.amount);

      // Tickers ordenados (Top 5)
      const topAssetsList = Object.entries(group.byTicker)
        .map(([ticker, amt]) => ({
          ticker,
          amount: amt,
          pct: group.totalAmount > 0 ? (amt / group.totalAmount) * 100 : 0,
        }))
        .sort((a, b) => b.amount - a.amount)
        .slice(0, 5);

      // Mês Campeão
      let bestMonthStr = '';
      let bestMonthAmount = 0;
      Object.entries(group.byMonth).forEach(([m, amt]) => {
        if (amt > bestMonthAmount) {
          bestMonthAmount = amt;
          bestMonthStr = m;
        }
      });

      return {
        year: y,
        currency: group.currency,
        totalAmount: group.totalAmount,
        growthPct,
        monthlyAverage,
        monthsCount,
        byType: byTypeList, // Mantemos o nome da prop como byType para minimizar refatoração, mas agora armazena Categoria
        topAssets: topAssetsList,
        bestMonthStr,
        bestMonthAmount,
        highestSinglePayment: group.highestSinglePayment,
        highestSinglePaymentTicker: group.highestSinglePaymentTicker
      };
    });
  }, [dividends]);

  // Determina o ano ativo selecionado no sumário.
  // Se o filtro global for 'Todos', usamos o ano mais recente disponível no sumário.
  const activeYearData = useMemo(() => {
    if (annualData.length === 0) return null;
    
    if (selectedYear === 'Todos') {
      return annualData[0]; // Ano mais recente
    }
    
    const yearNum = parseInt(selectedYear, 10);
    return annualData.find(d => d.year === yearNum) || annualData[0];
  }, [annualData, selectedYear]);

  if (annualData.length === 0) return null;

  // Lista dos anos ordenados
  const yearsList = annualData.map(d => String(d.year));

  const formatTypeColor = (category: string) => {
    switch (category) {
      case 'Ações (B3)': return '#60a5fa'; // Azul
      case 'FIIs': return '#c084fc'; // Roxo
      case 'FIAGROs': return '#4ade80'; // Verde claro
      case 'ETFs Nacionais': return '#f472b6'; // Rosa
      case 'BDRs': return '#fbbf24'; // Amarelo
      case 'Ações EUA': return '#f87171'; // Vermelho
      case 'ETF Internacional': return '#818cf8'; // Indigo
      case 'Cripto': return '#fcd34d'; // Dourado
      case 'Renda Fixa': return '#22d3ee'; // Ciano
      case 'Tesouro Direto': return '#34d399'; // Verde esmeralda
      default: return '#9ca3af'; // Cinza (Outros)
    }
  };

  const getMonthName = (m: number) => {
    const months = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];
    return months[m - 1] || '';
  };

  return (
    <div className="mb-xl">
      <style>{`
        .summary-tab-btn {
          padding: 0.5rem 1rem;
          border-radius: 8px;
          border: 1px solid var(--panel-border);
          background: rgba(255, 255, 255, 0.02);
          color: var(--text-secondary);
          cursor: pointer;
          font-weight: 500;
          font-size: 0.85rem;
          transition: all 0.2s ease;
        }
        .summary-tab-btn:hover {
          background: rgba(255, 255, 255, 0.06);
          color: var(--text-primary);
        }
        .summary-tab-btn.active {
          background: rgba(0, 230, 118, 0.1);
          border-color: rgba(0, 230, 118, 0.3);
          color: #00e676;
          font-weight: 600;
        }
        .progress-bar-bg {
          background: rgba(255, 255, 255, 0.05);
          height: 6px;
          border-radius: 4px;
          width: 100%;
          overflow: hidden;
        }
        .progress-bar-fill {
          height: 100%;
          border-radius: 4px;
          transition: width 0.3s ease;
        }
        .growth-badge {
          display: inline-flex;
          align-items: center;
          gap: 2px;
          font-size: 0.75rem;
          font-weight: 600;
          padding: 0.15rem 0.4rem;
          border-radius: 12px;
        }
        .growth-badge.positive {
          background: rgba(74, 222, 128, 0.15);
          color: #4ade80;
        }
        .growth-badge.negative {
          background: rgba(248, 113, 113, 0.15);
          color: #f87171;
        }
      `}</style>

      <div className="flex-row justify-between items-center mb-md flex-wrap gap-sm">
        <h4 className="font-bold text-secondary flex-row items-center gap-xs">
          📊 Resumo Anual Consolidado
        </h4>
        
        {/* Abas dos Anos */}
        <div className="flex-row gap-sm flex-wrap">
          {yearsList.map(yr => (
            <button
              key={yr}
              className={`summary-tab-btn ${((selectedYear === 'Todos' && activeYearData?.year === parseInt(yr, 10)) || selectedYear === yr) ? 'active' : ''}`}
              onClick={() => setSelectedYear(yr)}
            >
              {yr}
            </button>
          ))}
          {selectedYear !== 'Todos' && (
            <button
              className="summary-tab-btn"
              onClick={() => setSelectedYear('Todos')}
              style={{ fontSize: '0.8rem', opacity: 0.8 }}
            >
              Ver Todos
            </button>
          )}
        </div>
      </div>

      {activeYearData && (
        <div 
          style={{ 
            background: 'linear-gradient(145deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0.01) 100%)', 
            padding: '1.5rem', 
            borderRadius: '16px', 
            border: '1px solid rgba(255,255,255,0.05)',
            boxShadow: '0 4px 24px rgba(0,0,0,0.15)'
          }}
          className="flex-col gap-lg"
        >
          {/* Top KPIs */}
          <div className="flex-row gap-md mb-md flex-wrap">
            {/* Total Recebido */}
            <div className="card" style={{ flex: '1', minWidth: '200px', background: 'linear-gradient(145deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0.01) 100%)', padding: '1.25rem', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px', boxShadow: '0 4px 20px rgba(0,0,0,0.2)', display: 'flex', flexDirection: 'column' }}>
              <div className="text-secondary text-sm mb-sm font-bold">Total Líquido ({activeYearData.year})</div>
              <div className="font-bold text-3xl mb-xs" style={{ color: '#00e676', letterSpacing: '-0.5px' }}>
                {formatMoney(activeYearData.totalAmount, activeYearData.currency)}
              </div>
              <div className="text-sm text-secondary" style={{ marginTop: 'auto', paddingTop: '0.5rem' }}>
                {activeYearData.growthPct !== null ? (
                  <span className={`growth-badge ${activeYearData.growthPct >= 0 ? 'positive' : 'negative'}`}>
                    {activeYearData.growthPct >= 0 ? '▲' : '▼'} {Math.abs(activeYearData.growthPct).toFixed(1)}% YoY
                  </span>
                ) : (
                  <span style={{ opacity: 0.5 }}>Sem histórico anterior</span>
                )}
              </div>
            </div>

            {/* Média Mensal */}
            <div className="card" style={{ flex: '1', minWidth: '200px', background: 'linear-gradient(145deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0.01) 100%)', padding: '1.25rem', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px', boxShadow: '0 4px 20px rgba(0,0,0,0.2)', display: 'flex', flexDirection: 'column' }}>
              <div className="text-secondary text-sm mb-sm font-bold">Média Mensal</div>
              <div className="font-bold text-3xl mb-xs" style={{ color: '#fff', letterSpacing: '-0.5px' }}>
                {formatMoney(activeYearData.monthlyAverage, activeYearData.currency)}
                <span className="text-secondary font-normal text-sm" style={{ marginLeft: '4px' }}>/mês</span>
              </div>
              <div className="text-sm text-secondary" style={{ marginTop: 'auto', paddingTop: '0.5rem', opacity: 0.7, lineHeight: '1.4' }}>
                {activeYearData.year === new Date().getFullYear() 
                  ? `Calculado sobre ${activeYearData.monthsCount} meses (Jan-${getMonthName(activeYearData.monthsCount)})`
                  : `Calculado sobre 12 meses`
                }
              </div>
            </div>

            {/* Maior Pagamento Único */}
            <div className="card" style={{ flex: '1', minWidth: '200px', background: 'linear-gradient(145deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0.01) 100%)', padding: '1.25rem', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px', boxShadow: '0 4px 20px rgba(0,0,0,0.2)', display: 'flex', flexDirection: 'column' }}>
              <div className="text-secondary text-sm mb-sm font-bold">Recorde de Pagamento</div>
              <div className="font-bold text-3xl mb-xs" style={{ color: '#fff', letterSpacing: '-0.5px' }}>
                {formatMoney(activeYearData.highestSinglePayment, activeYearData.currency)}
              </div>
              <div className="text-sm text-secondary" style={{ marginTop: 'auto', paddingTop: '0.5rem', opacity: 0.7, lineHeight: '1.4' }}>
                {activeYearData.highestSinglePaymentTicker !== 'OUTROS' ? `Pago por ${activeYearData.highestSinglePaymentTicker}` : 'Sem dados suficientes'}
              </div>
            </div>

            {/* Mês Campeão */}
            <div className="card" style={{ flex: '1', minWidth: '200px', background: 'linear-gradient(145deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0.01) 100%)', padding: '1.25rem', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px', boxShadow: '0 4px 20px rgba(0,0,0,0.2)', display: 'flex', flexDirection: 'column' }}>
              <div className="text-secondary text-sm mb-sm font-bold">Mês Campeão</div>
              <div className="font-bold text-3xl mb-xs" style={{ color: '#FFB300', letterSpacing: '-0.5px', textTransform: 'capitalize' }}>
                {activeYearData.bestMonthStr ? getMonthName(parseInt(activeYearData.bestMonthStr, 10)) : '--'}
              </div>
              <div className="text-sm text-secondary" style={{ marginTop: 'auto', paddingTop: '0.5rem', opacity: 0.7, lineHeight: '1.4' }}>
                {formatMoney(activeYearData.bestMonthAmount, activeYearData.currency)} acumulados no mês
              </div>
            </div>
          </div>

          {/* Details Section: Type Dist and Top Assets */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-xl" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '2rem' }}>
            
            {/* Distribuição por Categoria */}
            <div className="flex-col gap-sm">
              <span className="text-secondary text-xs font-bold uppercase tracking-wider mb-sm block">
                🛠️ Distribuição por Categoria
              </span>
              <div className="flex-col gap-md">
                {activeYearData.byType.length > 0 ? (
                  activeYearData.byType.map(item => (
                    <div key={item.type} className="flex-col gap-xs">
                      <div className="flex-row justify-between items-baseline">
                        <span className="font-semibold text-sm text-primary flex-row items-center gap-xs">
                          <span 
                            style={{ 
                              display: 'inline-block', 
                              width: '8px', 
                              height: '8px', 
                              borderRadius: '50%', 
                              backgroundColor: formatTypeColor(item.type) 
                            }} 
                          />
                          {item.type}
                        </span>
                        <div className="text-right">
                          <span className="font-bold text-sm text-primary">
                            {formatMoney(item.amount, activeYearData.currency)}
                          </span>
                          <span className="text-secondary text-xs font-medium ml-xs">
                            ({item.pct.toFixed(1)}%)
                          </span>
                        </div>
                      </div>
                      <div className="progress-bar-bg">
                        <div 
                          className="progress-bar-fill" 
                          style={{ 
                            width: `${item.pct}%`, 
                            backgroundColor: formatTypeColor(item.type) 
                          }} 
                        />
                      </div>
                    </div>
                  ))
                ) : (
                  <span className="text-secondary text-sm">Sem dados por tipo.</span>
                )}
              </div>
            </div>

            {/* Top 5 Ativos Pagadores */}
            <div className="flex-col gap-sm">
              <span className="text-secondary text-xs font-bold uppercase tracking-wider mb-sm block">
                🏆 Top 5 Ativos Pagadores
              </span>
              <div className="flex-col gap-md">
                {activeYearData.topAssets.length > 0 ? (
                  activeYearData.topAssets.map((item, index) => (
                    <div key={item.ticker} className="flex-col gap-xs">
                      <div className="flex-row justify-between items-baseline">
                        <span className="font-bold text-sm text-primary flex-row items-center gap-xs">
                          <span style={{ color: index === 0 ? '#ffd700' : index === 1 ? '#c0c0c0' : '#cd7f32', marginRight: '4px' }}>
                            {index === 0 ? '🥇' : index === 1 ? '🥈' : '🥉'}
                          </span>
                          {item.ticker}
                        </span>
                        <div className="text-right">
                          <span className="font-bold text-sm text-success">
                            {formatMoney(item.amount, activeYearData.currency)}
                          </span>
                          <span className="text-secondary text-xs font-medium ml-xs">
                            ({item.pct.toFixed(1)}%)
                          </span>
                        </div>
                      </div>
                      <div className="progress-bar-bg">
                        <div 
                          className="progress-bar-fill" 
                          style={{ 
                            width: `${item.pct}%`, 
                            backgroundColor: '#00e676'
                          }} 
                        />
                      </div>
                    </div>
                  ))
                ) : (
                  <span className="text-secondary text-sm">Sem dados de ativos.</span>
                )}
              </div>
            </div>

          </div>
        </div>
      )}
    </div>
  );
}
