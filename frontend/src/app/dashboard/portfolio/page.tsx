'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import { useAuth } from '@/context/AuthContext';
import { PortfolioProvider, usePortfolio } from '@/context/PortfolioContext';
import { getAssetCategory } from '@/components/portfolio/helpers';

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
  } = portfolio;

  if (authLoading || isLoadingPortfolios) {
    return (
      <main className="container">
        <div className="glass-panel flex-col items-center justify-center" style={{ minHeight: '300px' }}>
          <span className="loading-spinner" style={{ borderTopColor: '#00f2fe', width: 40, height: 40 }}></span>
          <p className="text-secondary mt-lg">Carregando dados financeiros seguros...</p>
        </div>
      </main>
    );
  }

  if (!user) return null;

  const filteredPositions = positions.filter(pos => activeCategoryFilter === 'Todas' || activeCategoryFilter === 'Renda Variável' || getAssetCategory(pos.type) === activeCategoryFilter);
  const filteredTransactions = transactions.filter(tx => {
    if (activeCategoryFilter === 'Todas') return true;
    if (activeCategoryFilter === 'Renda Variável') return tx.module !== 'RF';
    if (activeCategoryFilter === 'Renda Fixa') return tx.module === 'RF' && tx.asset_type !== 'TESOURO';
    if (activeCategoryFilter === 'Tesouro Direto') return tx.module === 'RF' && tx.asset_type === 'TESOURO';
    return getAssetCategory(tx.asset_type || '') === activeCategoryFilter;
  });
  const categoryFilteredDividends = dividends.filter(div => {
    if (activeCategoryFilter === 'Todas') return true;
    if (activeCategoryFilter === 'Renda Variável') return !div.is_accrued;
    if (activeCategoryFilter === 'Renda Fixa') return div.is_accrued && div.asset_type !== 'TESOURO';
    if (activeCategoryFilter === 'Tesouro Direto') return div.is_accrued && div.asset_type === 'TESOURO';
    if (div.is_accrued || getAssetCategory(div.asset_type) !== activeCategoryFilter) return false;
    return true;
  });

  const filteredDividends = categoryFilteredDividends.filter(div => {
    const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.cum_date;
    if (!dateStr) return true;
    const year = dateStr.substring(0, 4);
    const month = dateStr.substring(5, 7);
    return (filterDivYear === 'Todos' || year === filterDivYear) && (filterDivMonth === 'Todos' || month === filterDivMonth);
  });

  const availableYears = Array.from(new Set(categoryFilteredDividends.map(d => ((d.payment_date && !d.payment_date.startsWith('0001') ? d.payment_date : d.cum_date) || '').substring(0, 4)).filter(Boolean))).sort((a, b) => b.localeCompare(a));

  const includeFI = activeCategoryFilter === 'Todas' || activeCategoryFilter === 'Renda Fixa';
  const filteredFI = includeFI ? fiPositions : [];

  const includeTreasury = activeCategoryFilter === 'Todas' || activeCategoryFilter === 'Tesouro Direto';
  const filteredTreasury = includeTreasury ? treasuryPositions : [];

  const eqCost = filteredPositions.reduce((acc, pos) => acc + pos.total_cost, 0);
  const eqValue = filteredPositions.reduce((acc, pos) => acc + (pos.current_value || 0), 0);
  
  const fiCost = filteredFI.reduce((acc, pos) => acc + pos.total_invested, 0);
  const fiValue = filteredFI.reduce((acc, pos) => acc + pos.net_value, 0);

  const tdCost = filteredTreasury.reduce((acc, pos) => acc + pos.total_invested, 0);
  const tdValue = filteredTreasury.reduce((acc, pos) => acc + pos.net_value, 0);

  const totalCost = eqCost + fiCost + tdCost;
  const currentValue = eqValue + fiValue + tdValue;
  const profitLoss = currentValue - totalCost;
  const returnPercent = totalCost > 0 ? (profitLoss / totalCost) * 100 : 0.0;
  
  const twelveMonthsAgo = new Date();
  twelveMonthsAgo.setMonth(twelveMonthsAgo.getMonth() - 12);
  const divs12m = categoryFilteredDividends.filter(div => {
    const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.cum_date;
    return dateStr && new Date(dateStr) >= twelveMonthsAgo;
  });
  const sumDivs12m = divs12m.reduce((acc, div) => acc + ((div as any).total_value || div.net_amount || 0), 0);
  const avgDividends12m = sumDivs12m / 12;

  const availableCategories = Array.from(new Set(positions.map(pos => getAssetCategory(pos.type)))).sort();
  const filterCategories = ['Todas'];
  if (positions.length > 0) filterCategories.push('Renda Variável', ...availableCategories);
  if (fiPositions.length > 0) {
    filterCategories.push('Renda Fixa');
  }
  if (treasuryPositions.length > 0) {
    filterCategories.push('Tesouro Direto');
  }

  return (
    <main className="container" style={{ maxWidth: 1400 }}>
      <PortfolioHeader userName={user?.name || 'Investidor'} onLogout={logout} />

      <PortfolioTabs 
        portfolios={portfolios} 
        activePortfolioId={activePortfolioId} setActivePortfolioId={setActivePortfolioId} 
        setShowPortfolioModal={setShowPortfolioModal} handleDeletePortfolio={handleDeletePortfolio} 
        handleExportPortfolio={handleExportPortfolio}
        handleSetDefaultPortfolio={handleSetDefaultPortfolio}
      />

      <div className="flex-row gap-sm mb-lg flex-wrap">
        {filterCategories.map(cat => (
          <button
            key={cat} onClick={() => setActiveCategoryFilter(cat)}
            className={`badge ${activeCategoryFilter === cat ? 'font-bold' : 'font-semibold'}`}
            style={{ padding: '0.4rem 1rem', borderRadius: '20px', cursor: 'pointer', border: activeCategoryFilter === cat ? '1px solid var(--accent-color)' : '1px solid var(--panel-border)', background: activeCategoryFilter === cat ? 'rgba(0, 242, 254, 0.1)' : 'rgba(255, 255, 255, 0.02)', color: activeCategoryFilter === cat ? '#fff' : 'var(--text-secondary)' }}
          >
            {cat}
          </button>
        ))}
      </div>

      {isLoadingDetails ? (
        <div className="glass-panel flex-row items-center justify-center" style={{ minHeight: '300px' }}>
          <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 35, height: 35 }}></span>
        </div>
      ) : (
        <div className="flex-col gap-xl">
          <PortfolioSummaryCards totalCost={totalCost} currentValue={currentValue} profitLoss={profitLoss} returnPercent={returnPercent} avgDividends12m={avgDividends12m} kpiCurrency={kpiCurrency} isLoadingTreasury={isLoadingTreasury} />

          <div className="flex-row gap-md mt-xl mb-lg" style={{ borderBottom: '1px solid rgba(255,255,255,0.1)' }}>
            <button onClick={() => setActiveTab('ativos')} style={{ background: 'none', border: 'none', padding: '0.75rem 1rem', cursor: 'pointer', color: activeTab === 'ativos' ? '#00e676' : 'var(--text-secondary)', borderBottom: activeTab === 'ativos' ? '2px solid #00e676' : '2px solid transparent', fontWeight: activeTab === 'ativos' ? 700 : 500, fontSize: '0.9rem' }}>
              📊 Renda Variável
            </button>
            <button onClick={() => setActiveTab('renda-fixa')} style={{ background: 'none', border: 'none', padding: '0.75rem 1rem', cursor: 'pointer', color: activeTab === 'renda-fixa' ? '#00e676' : 'var(--text-secondary)', borderBottom: activeTab === 'renda-fixa' ? '2px solid #00e676' : '2px solid transparent', fontWeight: activeTab === 'renda-fixa' ? 700 : 500, fontSize: '0.9rem' }}>
              🏛️ Renda Fixa
            </button>
            <button onClick={() => setActiveTab('tesouro')} style={{ background: 'none', border: 'none', padding: '0.75rem 1rem', cursor: 'pointer', color: activeTab === 'tesouro' ? '#00e676' : 'var(--text-secondary)', borderBottom: activeTab === 'tesouro' ? '2px solid #00e676' : '2px solid transparent', fontWeight: activeTab === 'tesouro' ? 700 : 500, fontSize: '0.9rem' }}>
              🏛️ Tesouro Direto
            </button>
            <button onClick={() => setActiveTab('operacoes')} style={{ background: 'none', border: 'none', padding: '0.75rem 1rem', cursor: 'pointer', color: activeTab === 'operacoes' ? '#00e676' : 'var(--text-secondary)', borderBottom: activeTab === 'operacoes' ? '2px solid #00e676' : '2px solid transparent', fontWeight: activeTab === 'operacoes' ? 700 : 500, fontSize: '0.9rem' }}>
              📜 Histórico de Operações
            </button>
            <button onClick={() => setActiveTab('proventos')} style={{ background: 'none', border: 'none', padding: '0.75rem 1rem', cursor: 'pointer', color: activeTab === 'proventos' ? '#00e676' : 'var(--text-secondary)', borderBottom: activeTab === 'proventos' ? '2px solid #00e676' : '2px solid transparent', fontWeight: activeTab === 'proventos' ? 700 : 500, fontSize: '0.9rem' }}>
              💰 Proventos
            </button>
            <button onClick={() => setActiveTab('analise')} style={{ background: 'none', border: 'none', padding: '0.75rem 1rem', cursor: 'pointer', color: activeTab === 'analise' ? '#00e676' : 'var(--text-secondary)', borderBottom: activeTab === 'analise' ? '2px solid #00e676' : '2px solid transparent', fontWeight: activeTab === 'analise' ? 700 : 500, fontSize: '0.9rem' }}>
              🔬 Análise da Carteira
            </button>
            <button onClick={() => setActiveTab('diario')} style={{ background: 'none', border: 'none', padding: '0.75rem 1rem', cursor: 'pointer', color: activeTab === 'diario' ? '#00e676' : 'var(--text-secondary)', borderBottom: activeTab === 'diario' ? '2px solid #00e676' : '2px solid transparent', fontWeight: activeTab === 'diario' ? 700 : 500, fontSize: '0.9rem' }}>
              📈 Resumo Diário
            </button>
          </div>

          {activeTab === 'ativos' && (
            <div className="flex-col gap-xl w-full">
              <div className="card flex-col" style={{ padding: '1.75rem 2rem', minHeight: '380px' }}>
                <div className="flex-row justify-between items-center mb-lg flex-wrap gap-md">
                  <div>
                    <h3 className="card-title">📈 Evolução da Renda Variável</h3>
                    <p className="text-xs text-secondary mt-sm">Valores ponderados na moeda base ({kpiCurrency})</p>
                  </div>
                  {activeCategoryFilter !== 'Renda Fixa' && (
                    <>
                    <div className="flex-row gap-sm" style={{ background: 'rgba(255,255,255,0.02)', padding: '0.2rem', borderRadius: '6px', border: '1px solid var(--panel-border)' }}>
                      <select 
                        value={filterChartTicker} 
                        onChange={(e) => setFilterChartTicker(e.target.value)}
                        style={{ background: 'transparent', border: 'none', color: 'var(--text-primary)', outline: 'none', cursor: 'pointer', fontSize: '0.75rem', padding: '0 0.5rem', fontWeight: 600 }}
                      >
                        <option value="Todos" style={{ background: '#1c1f24', color: '#fff' }}>Todos os Tickers</option>
                        {Array.from(new Set(filteredPositions.map(p => p.ticker))).sort().map(t => (
                          <option key={t} value={t} style={{ background: '#1c1f24', color: '#fff' }}>{t}</option>
                        ))}
                      </select>
                    </div>
                    <div className="flex-row gap-sm" style={{ background: 'rgba(255,255,255,0.02)', padding: '0.2rem', borderRadius: '6px', border: '1px solid var(--panel-border)' }}>
                      {['1M', '3M', '6M', '1Y', 'ALL'].map((p) => (
                        <button key={p} onClick={() => setPeriod(p)} style={{ padding: '0.25rem 0.65rem', fontSize: '0.7rem', borderRadius: '4px', border: 'none', background: period === p ? 'var(--accent-gradient)' : 'transparent', color: period === p ? '#000' : 'var(--text-secondary)', cursor: 'pointer', fontWeight: 700 }}>
                          {p}
                        </button>
                      ))}
                    </div>
                    </>
                  )}
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
              <TransactionHistory transactions={filteredTransactions} filterTxTicker={filterTxTicker} setFilterTxTicker={setFilterTxTicker} handleEditTransaction={handleEditTransaction} handleDeleteTransaction={handleDeleteTransaction} onLaunchOperation={() => { setEditingTxId(null); setShowTxModal(true); }} kpiCurrency={kpiCurrency} />
            </div>
          )}

          {activeTab === 'proventos' && (
            <DividendsHistory dividends={filteredDividends} allDividends={categoryFilteredDividends} filterDivYear={filterDivYear} setFilterDivYear={setFilterDivYear} filterDivMonth={filterDivMonth} setFilterDivMonth={setFilterDivMonth} availableYears={availableYears} isLoadingDividends={isLoadingDividends} />
          )}

          {activeTab === 'analise' && (
            <PortfolioAnalysis
              positions={filteredPositions}
              dividends={categoryFilteredDividends}
              fiPositions={filteredFI}
              treasuryPositions={filteredTreasury}
              performanceData={performanceData}
              kpiCurrency={kpiCurrency}
            />
          )}

          {activeTab === 'diario' && (
            <DailyReport positions={filteredPositions} treasuryPositions={treasuryPositions} kpiCurrency={kpiCurrency} />
          )}

          {activeTab === 'renda-fixa' && (
            <FixedIncomeTab portfolioId={activePortfolioId} onLaunchOperation={() => setShowFIModal(true)} />
          )}

          {activeTab === 'tesouro' && (
            <TreasuryTab
              portfolioId={activePortfolioId}
              positions={treasuryPositions}
              isLoadingPositions={isLoadingTreasury}
              onRefresh={async () => { await loadTreasuryPositions(activePortfolioId); }}
            />
          )}
        </div>
      )}

      {/* Renderizado sem prop drilling! Todos os modais consomem o PortfolioContext */}
      <Modals />
    </main>
  );
}

export default function PortfolioPage() {
  return (
    <PortfolioProvider>
      <PortfolioContent />
    </PortfolioProvider>
  );
}
