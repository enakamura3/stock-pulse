'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import { useAuth } from '@/context/AuthContext';
import { PortfolioProvider, usePortfolio } from '@/context/PortfolioContext';
import { getAssetCategory, getDividendAssetCategory, formatMoney } from '@/components/portfolio/helpers';
import { apiFetch } from '@/lib/api';

import PortfolioHeader from '@/components/portfolio/PortfolioHeader';
import PortfolioTabs from '@/components/portfolio/PortfolioTabs';
import PortfolioSummaryCards from '@/components/portfolio/PortfolioSummaryCards';
import AssetList from '@/components/portfolio/AssetList';
import TransactionHistory from '@/components/portfolio/TransactionHistory';
import DividendsHistory from '@/components/portfolio/DividendsHistory';
import DailyReport from '@/components/portfolio/DailyReport';
import FixedIncomeTab from '@/components/portfolio/FixedIncomeTab';
import TreasuryTab from '@/components/portfolio/TreasuryTab';
import PortfolioAnalysis from '@/components/portfolio/PortfolioAnalysis';
import Modals from '@/components/portfolio/Modals';

const PortfolioChart = dynamic(() => import('@/components/PortfolioChart'), { ssr: false });

import AppSidebar from '@/components/AppSidebar';

function PortfolioContent() {
  const { user, logout, isLoading: authLoading } = useAuth();
  const portfolio = usePortfolio();

  const {
    portfolios,
    activePortfolioId,
    setActivePortfolioId,
    kpiCurrency,
    positions,
    fiPositions,
    treasuryPositions,
    transactions,
    performanceData,
    dividends,
    isLoadingPortfolios,
    isLoadingDetails,
    isLoadingPerformance,
    isLoadingDividends,
    isLoadingTreasury,
    activeTab,
    setActiveTab,
    activeCategoryFilter,
    setActiveCategoryFilter,
    filterTxTicker,
    setFilterTxTicker,
    filterChartTicker,
    setFilterChartTicker,
    filterDivYear,
    setFilterDivYear,
    filterDivMonth,
    setFilterDivMonth,
    period,
    setPeriod,
    setShowPortfolioModal,
    setShowTxModal,
    setShowFIModal,
    setEditingTxId,
    handleDeletePortfolio,
    handleSetDefaultPortfolio,
    handleEditTransaction,
    handleDeleteTransaction,
    handleFileUpload,
    handleExportPortfolio,
    loadTreasuryPositions,
    lastFetchedAt,
    loadPortfolioDetails,
    loadDividends,
    loadPerformance,
  } = portfolio;

  // Dynamic categories per active tab without hardcoding
  const dynamicCategories = React.useMemo(() => {
    if (activeTab === 'ativos') {
      const cats = Array.from(new Set(positions.map(p => getAssetCategory(p.type)).filter(c => c && c !== 'Desconhecido'))).sort();
      return cats.length > 1 ? ['Todas', ...cats] : [];
    }
    if (activeTab === 'renda-fixa') {
      const types = Array.from(new Set(fiPositions.map(p => p.asset?.type).filter(Boolean))).sort();
      return types.length > 1 ? ['Todas', ...types] : [];
    }
    if (activeTab === 'tesouro') {
      const types = Array.from(new Set(treasuryPositions.map(p => p.treasury_type).filter(Boolean))).sort();
      return types.length > 1 ? ['Todas', ...types] : [];
    }
    if (activeTab === 'proventos') {
      const cats = Array.from(new Set(dividends.map(d => getDividendAssetCategory(d)).filter(c => c && c !== 'Outros' && c !== 'Desconhecido'))).sort();
      return cats.length > 1 ? ['Todas', ...cats] : [];
    }
    return [];
  }, [activeTab, positions, fiPositions, treasuryPositions, dividends]);

  // Reset activeCategoryFilter to 'Todas' on tab change
  React.useEffect(() => {
    setActiveCategoryFilter('Todas');
  }, [activeTab, setActiveCategoryFilter]);

  // Reset if activeCategoryFilter is no longer present in dynamicCategories
  React.useEffect(() => {
    if (activeCategoryFilter !== 'Todas' && dynamicCategories.length > 0 && !dynamicCategories.includes(activeCategoryFilter)) {
      setActiveCategoryFilter('Todas');
    }
  }, [dynamicCategories, activeCategoryFilter, setActiveCategoryFilter]);

  const filteredPositions = React.useMemo(() => {
    if (activeTab !== 'ativos' || activeCategoryFilter === 'Todas') {
      return positions;
    }
    return positions.filter(pos => getAssetCategory(pos.type) === activeCategoryFilter);
  }, [positions, activeTab, activeCategoryFilter]);

  const filteredTreasuryPositions = React.useMemo(() => {
    if (activeTab !== 'tesouro' || activeCategoryFilter === 'Todas') {
      return treasuryPositions;
    }
    return treasuryPositions.filter(p => p.treasury_type === activeCategoryFilter);
  }, [treasuryPositions, activeTab, activeCategoryFilter]);

  const categoryFilteredDividends = React.useMemo(() => {
    if (activeTab !== 'proventos' || activeCategoryFilter === 'Todas') {
      return dividends;
    }
    return dividends.filter(div => getDividendAssetCategory(div) === activeCategoryFilter);
  }, [dividends, activeTab, activeCategoryFilter]);

  const filteredDividends = React.useMemo(() => {
    return categoryFilteredDividends.filter(div => {
      const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.cum_date;
      if (!dateStr) return true;
      const year = dateStr.substring(0, 4);
      const month = dateStr.substring(5, 7);
      return (filterDivYear === 'Todos' || year === filterDivYear) && (filterDivMonth === 'Todos' || month === filterDivMonth);
    });
  }, [categoryFilteredDividends, filterDivYear, filterDivMonth]);

  const availableYears = React.useMemo(() => {
    return Array.from(new Set(categoryFilteredDividends.map(d => ((d.payment_date && !d.payment_date.startsWith('0001') ? d.payment_date : d.cum_date) || '').substring(0, 4)).filter(Boolean))).sort((a, b) => b.localeCompare(a));
  }, [categoryFilteredDividends]);

  const eqCost = positions.reduce((acc, pos) => acc + pos.total_cost, 0);
  const eqValue = positions.reduce((acc, pos) => acc + (pos.current_value || 0), 0);
  
  const fiCost = fiPositions.reduce((acc, pos) => acc + pos.total_invested, 0);
  const fiValue = fiPositions.reduce((acc, pos) => acc + pos.net_value, 0);

  const tdCost = treasuryPositions.reduce((acc, pos) => acc + pos.total_invested, 0);
  const tdValue = treasuryPositions.reduce((acc, pos) => acc + pos.net_value, 0);

  const totalCost = eqCost + fiCost + tdCost;
  const currentValue = eqValue + fiValue + tdValue;
  const profitLoss = currentValue - totalCost;
  const returnPercent = totalCost > 0 ? (profitLoss / totalCost) * 100 : 0.0;
  
  const twelveMonthsAgo = new Date();
  twelveMonthsAgo.setMonth(twelveMonthsAgo.getMonth() - 12);
  const divs12m = dividends.filter(div => {
    const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.cum_date;
    return dateStr && new Date(dateStr) >= twelveMonthsAgo;
  });
  const sumDivs12m = divs12m.reduce((acc, div) => acc + ((div as any).total_value || div.net_amount || 0), 0);
  const avgDividends12m = sumDivs12m / 12;

  const getCategoryLabel = React.useCallback((cat: string) => {
    if (cat === 'Todas') return 'Todas';
    if (activeTab === 'tesouro') {
      const map: Record<string, string> = {
        SELIC: 'Tesouro Selic',
        PREFIXADO: 'Prefixado',
        'IPCA+': 'IPCA+',
        IPCA: 'IPCA+',
      };
      return map[cat] || cat;
    }
    if (activeTab === 'renda-fixa') {
      if (cat.toUpperCase() === 'DEBENTURE') return 'Debêntures';
    }
    return cat;
  }, [activeTab]);

  const filteredEqCost = React.useMemo(() => filteredPositions.reduce((acc, pos) => acc + (pos.total_cost || 0), 0), [filteredPositions]);
  const filteredEqValue = React.useMemo(() => filteredPositions.reduce((acc, pos) => acc + (pos.current_value || 0), 0), [filteredPositions]);
  const filteredEqProfitLoss = filteredEqValue - filteredEqCost;
  const filteredEqReturnPercent = filteredEqCost > 1e-6 ? (filteredEqProfitLoss / filteredEqCost) * 100 : 0.0;

  const filteredEqDividends = React.useMemo(() => {
    const activeTickers = new Set(filteredPositions.map(p => p.ticker).filter(Boolean));
    const activeAssetIds = new Set(filteredPositions.map(p => p.asset_id).filter(Boolean));
    return dividends.filter(d => (d.ticker && activeTickers.has(d.ticker)) || (d.asset_id && activeAssetIds.has(d.asset_id)));
  }, [filteredPositions, dividends]);

  const filteredEqDividendsTotal = React.useMemo(() => {
    return filteredEqDividends.reduce((acc, d) => acc + ((d as any).total_value || d.net_amount || 0), 0);
  }, [filteredEqDividends]);

  const equityKpis = React.useMemo(() => [
    { label: 'Total Investido', value: formatMoney(filteredEqCost, kpiCurrency), icon: '💰' },
    { label: 'Patrimônio Atual', value: formatMoney(filteredEqValue, kpiCurrency), icon: '📊' },
    {
      label: 'Lucro / Prejuízo',
      value: formatMoney(filteredEqProfitLoss, kpiCurrency),
      icon: '💵',
      sub: `${filteredEqReturnPercent >= 0 ? '+' : ''}${filteredEqReturnPercent.toFixed(2)}% (${filteredEqProfitLoss >= 0 ? '+' : ''}${formatMoney(filteredEqProfitLoss, kpiCurrency)})`,
      subColor: filteredEqProfitLoss >= 0 ? 'var(--color-success)' : 'var(--color-danger)',
    },
    {
      label: 'Proventos Recebidos',
      value: formatMoney(filteredEqDividendsTotal, kpiCurrency),
      icon: '🪙',
      sub: 'Total acumulado',
      subColor: 'var(--accent-color)',
    },
    {
      label: 'Ativos em Carteira',
      value: `${filteredPositions.length}`,
      icon: '🏷️',
      sub: activeCategoryFilter && activeCategoryFilter !== 'Todas' ? getCategoryLabel(activeCategoryFilter) : 'Todas as categorias',
      subColor: 'var(--text-secondary)',
    },
  ], [filteredEqCost, filteredEqValue, filteredEqProfitLoss, filteredEqReturnPercent, filteredEqDividendsTotal, filteredPositions.length, activeCategoryFilter, kpiCurrency, getCategoryLabel]);

  if (authLoading || isLoadingPortfolios) {
    return (
      <main className="container">
        <div className="glass-panel flex-col items-center justify-center" style={{ minHeight: '300px' }}>
          <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 40, height: 40 }}></span>
          <p className="text-secondary mt-lg">Carregando dados financeiros seguros...</p>
        </div>
      </main>
    );
  }

  if (!user) return null;

  return (
    <div className="app-layout">
      <AppSidebar
        userName={user?.name || 'Investidor'}
        onLogout={logout}
        activeTab={activeTab}
        onSelectTab={setActiveTab}
      />

      <main className="app-main-content">
        <PortfolioTabs 
          portfolios={portfolios} 
          activePortfolioId={activePortfolioId} setActivePortfolioId={setActivePortfolioId} 
          setShowPortfolioModal={setShowPortfolioModal} handleDeletePortfolio={handleDeletePortfolio} 
          handleExportPortfolio={handleExportPortfolio}
          handleSetDefaultPortfolio={handleSetDefaultPortfolio}
        />

        {isLoadingDetails ? (
          <div className="glass-panel flex-row items-center justify-center" style={{ minHeight: '300px' }}>
            <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 35, height: 35 }}></span>
          </div>
        ) : (
          <div className="flex-col gap-xl">
            <PortfolioSummaryCards totalCost={totalCost} currentValue={currentValue} profitLoss={profitLoss} returnPercent={returnPercent} avgDividends12m={avgDividends12m} kpiCurrency={kpiCurrency} isLoadingTreasury={isLoadingTreasury} />

            {dynamicCategories.length > 1 && (
              <div className="flex-row gap-sm flex-wrap" data-testid="contextual-filter-pills" aria-label="Filtro de categorias">
                {dynamicCategories.map(cat => (
                  <button
                    key={cat} onClick={() => setActiveCategoryFilter(cat)}
                    className={`badge ${activeCategoryFilter === cat ? 'font-bold' : 'font-semibold'}`}
                    style={{ padding: '0.4rem 1rem', borderRadius: '20px', cursor: 'pointer', border: activeCategoryFilter === cat ? '1px solid var(--accent-color)' : '1px solid var(--panel-border)', background: activeCategoryFilter === cat ? 'var(--accent-bg)' : 'var(--panel-bg)', color: activeCategoryFilter === cat ? 'var(--accent-color)' : 'var(--text-secondary)' }}
                  >
                    {getCategoryLabel(cat)}
                  </button>
                ))}
              </div>
            )}

          {activeTab === 'ativos' && (
            <div className="flex-col gap-xl w-full">
              {/* ── KPI Cards ── */}
              <div className="flex-row gap-md flex-wrap" data-testid="equity-kpi-cards">
                {equityKpis.map((card, idx) => (
                  <div
                    key={idx}
                    className="card"
                    style={{ flex: '1 1 180px', minWidth: 160, padding: '1.25rem 1.5rem' }}
                  >
                    <div style={{ fontSize: '1.4rem', marginBottom: '0.4rem' }}>{card.icon}</div>
                    <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)', marginBottom: '0.35rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>{card.label}</div>
                    <div style={{ fontSize: '1.2rem', fontWeight: 700, color: 'var(--text-primary)' }}>{card.value}</div>
                    {card.sub && (
                      <div style={{ fontSize: '0.75rem', color: card.subColor, marginTop: '0.25rem', fontWeight: 600 }}>
                        {card.sub}
                      </div>
                    )}
                  </div>
                ))}
              </div>

              <div className="card flex-col" style={{ padding: '1.75rem 2rem', minHeight: '380px' }}>
                <div className="flex-row justify-between items-center mb-lg flex-wrap gap-md">
                  <div>
                    <h3 className="card-title">📈 Evolução da Renda Variável</h3>
                    <p className="text-xs text-secondary mt-sm">Valores ponderados na moeda base ({kpiCurrency})</p>
                  </div>
                  <div className="flex-row gap-sm" style={{ background: 'var(--input-bg)', padding: '0.2rem', borderRadius: '6px', border: '1px solid var(--panel-border)' }}>
                    <select 
                      value={filterChartTicker} 
                      onChange={(e) => setFilterChartTicker(e.target.value)}
                      style={{ background: 'transparent', border: 'none', color: 'var(--text-primary)', outline: 'none', cursor: 'pointer', fontSize: '0.75rem', padding: '0 0.5rem', fontWeight: 600 }}
                    >
                      <option value="Todos" style={{ background: 'var(--option-bg)', color: 'var(--option-color)' }}>Todos os Tickers</option>
                      {Array.from(new Set(filteredPositions.map(p => p.ticker))).sort().map(t => (
                        <option key={t} value={t} style={{ background: 'var(--option-bg)', color: 'var(--option-color)' }}>{t}</option>
                      ))}
                    </select>
                  </div>
                  <div className="flex-row gap-sm" style={{ background: 'var(--input-bg)', padding: '0.2rem', borderRadius: '6px', border: '1px solid var(--panel-border)' }}>
                    {['1M', '3M', '6M', '1Y', 'ALL'].map((p) => (
                      <button key={p} onClick={() => setPeriod(p)} style={{ padding: '0.25rem 0.65rem', fontSize: '0.7rem', borderRadius: '4px', border: 'none', background: period === p ? 'var(--accent-gradient)' : 'transparent', color: period === p ? 'var(--accent-foreground)' : 'var(--text-secondary)', cursor: 'pointer', fontWeight: 700 }}>
                        {p}
                      </button>
                    ))}
                  </div>
                </div>

                {isLoadingPerformance ? (
                  <div className="flex-row items-center justify-center w-full" style={{ height: '300px' }}>
                    <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 30, height: 30 }}></span>
                  </div>
                ) : performanceData.length > 0 ? (
                  <PortfolioChart data={performanceData} />
                ) : (
                  <div className="flex-col items-center justify-center w-full text-secondary" style={{ height: '300px', border: '1px dashed var(--panel-border)', borderRadius: '12px' }}>
                    <span className="text-2xl mb-sm">💼</span>
                    <p className="text-sm m-0">Cadastre a sua primeira transação abaixo para começar a visualizar o histórico de rentabilidade.</p>
                  </div>
                )}
              </div>

              <AssetList positions={filteredPositions} kpiCurrency={kpiCurrency} onImportCsv={handleFileUpload} onLaunchOperation={() => { setEditingTxId(null); setShowTxModal(true); }} />
            </div>
          )}

          {activeTab === 'operacoes' && (
            <div className="flex-col gap-xl w-full">
              <TransactionHistory transactions={transactions} filterTxTicker={filterTxTicker} setFilterTxTicker={setFilterTxTicker} handleEditTransaction={handleEditTransaction} handleDeleteTransaction={handleDeleteTransaction} onLaunchOperation={() => { setEditingTxId(null); setShowTxModal(true); }} kpiCurrency={kpiCurrency} />
            </div>
          )}

          {activeTab === 'proventos' && (
            <DividendsHistory dividends={filteredDividends} allDividends={categoryFilteredDividends} filterDivYear={filterDivYear} setFilterDivYear={setFilterDivYear} filterDivMonth={filterDivMonth} setFilterDivMonth={setFilterDivMonth} availableYears={availableYears} isLoadingDividends={isLoadingDividends} />
          )}

          {activeTab === 'analise' && (
            <PortfolioAnalysis
              positions={positions}
              dividends={dividends}
              fiPositions={fiPositions}
              treasuryPositions={treasuryPositions}
              performanceData={performanceData}
              kpiCurrency={kpiCurrency}
            />
          )}

          {activeTab === 'diario' && (
            <DailyReport
              positions={positions}
              fiPositions={fiPositions}
              treasuryPositions={treasuryPositions}
              dividends={dividends}
              kpiCurrency={kpiCurrency}
              lastFetchedAt={lastFetchedAt}
              onRefresh={async (forceRealtime?: boolean) => {
                if (activePortfolioId) {
                  if (forceRealtime) {
                    try {
                      await apiFetch('/market/quotes/invalidate', { method: 'POST' });
                    } catch {}
                  }
                  await loadPortfolioDetails(activePortfolioId);
                  await loadDividends(activePortfolioId);
                  await loadPerformance(activePortfolioId, period);
                }
              }}
              isRefreshing={isLoadingDetails}
              onGoToAssets={() => setActiveTab('ativos')}
            />
          )}

          {activeTab === 'renda-fixa' && (
            <FixedIncomeTab portfolioId={activePortfolioId} onLaunchOperation={() => setShowFIModal(true)} categoryFilter={activeCategoryFilter} />
          )}

          {activeTab === 'tesouro' && (
            <TreasuryTab
              portfolioId={activePortfolioId}
              positions={filteredTreasuryPositions}
              isLoadingPositions={isLoadingTreasury}
              onRefresh={async () => { await loadTreasuryPositions(activePortfolioId); }}
            />
          )}
        </div>
      )}

      {/* Renderizado sem prop drilling! Todos os modais consomem o PortfolioContext */}
      <Modals />
      </main>
    </div>
  );
}

export default function PortfolioPage() {
  return (
    <PortfolioProvider>
      <PortfolioContent />
    </PortfolioProvider>
  );
}
