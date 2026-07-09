'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/context/AuthContext';
import dynamic from 'next/dynamic';

import { Portfolio, Position, Transaction, PerformancePoint, CalculatedDividend, SearchResult, FixedIncomePosition, UnifiedTransaction } from '@/components/portfolio/types';
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

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export default function PortfolioPage() {
  const { user, logout, isLoading: authLoading } = useAuth();

  const [portfolios, setPortfolios] = useState<Portfolio[]>([]);
  const [activePortfolioId, setActivePortfolioId] = useState<string>('');
  const [activeCategoryFilter, setActiveCategoryFilter] = useState<string>('Todas');
  const [positions, setPositions] = useState<Position[]>([]);
  const [fiPositions, setFiPositions] = useState<FixedIncomePosition[]>([]);
  const [transactions, setTransactions] = useState<UnifiedTransaction[]>([]);
  const [performanceData, setPerformanceData] = useState<PerformancePoint[]>([]);
  const [dividends, setDividends] = useState<CalculatedDividend[]>([]);
  
  const [filterTxTicker, setFilterTxTicker] = useState<string>('');
  const [filterChartTicker, setFilterChartTicker] = useState<string>('Todos');
  const [filterDivYear, setFilterDivYear] = useState<string>('Todos');
  const [filterDivMonth, setFilterDivMonth] = useState<string>('Todos');
  const [activeTab, setActiveTab] = useState<'ativos' | 'operacoes' | 'proventos' | 'insights' | 'analise' | 'diario' | 'renda-fixa' | 'tesouro'>('ativos');
  const [period, setPeriod] = useState<string>('ALL');

  const [showPortfolioModal, setShowPortfolioModal] = useState(false);
  const [showTxModal, setShowTxModal] = useState(false);
  const [showFIModal, setShowFIModal] = useState(false);
  const [showFIEditModal, setShowFIEditModal] = useState(false);
  const [fiEditTxAssetName, setFiEditTxAssetName] = useState('');
  
  const [isLoadingPortfolios, setIsLoadingPortfolios] = useState(true);
  const [isLoadingDetails, setIsLoadingDetails] = useState(false);
  const [isLoadingPerformance, setIsLoadingPerformance] = useState(false);
  const [isLoadingDividends, setIsLoadingDividends] = useState(false);

  const [newPortfolioName, setNewPortfolioName] = useState('');
  const [newPortfolioCurrency, setNewPortfolioCurrency] = useState('BRL');
  const [isCreatingPortfolio, setIsCreatingPortfolio] = useState(false);

  const [txTicker, setTxTicker] = useState('');
  const [txType, setTxType] = useState<'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS'>('BUY');
  const [txQuantity, setTxQuantity] = useState<string | number>('');
  const [txUnitPrice, setTxUnitPrice] = useState<string | number>('');
  const [txExchangeRate, setTxExchangeRate] = useState<string | number>(1.0);
  const [txExecutedAt, setTxExecutedAt] = useState<string>(new Date().toISOString().split('T')[0]);
  const [isAddingTx, setIsAddingTx] = useState(false);
  const [editingTxId, setEditingTxId] = useState<string | null>(null);
  
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [selectedAssetCurrency, setSelectedAssetCurrency] = useState('BRL');

  // Fixed Income State
  const [fiInstitution, setFiInstitution] = useState('');
  const [fiType, setFiType] = useState('CDB');
  const [fiDebtType, setFiDebtType] = useState('POS');
  const [fiIndexer, setFiIndexer] = useState('CDI');
  const [fiRate, setFiRate] = useState<string | number>('');
  const [fiAmount, setFiAmount] = useState<string | number>('');
  const [fiApplicationDate, setFiApplicationDate] = useState<string>(new Date().toISOString().split('T')[0]);
  const [fiMaturityDate, setFiMaturityDate] = useState<string>('');
  const [fiTxType, setFiTxType] = useState<string>('SUBSCRIPTION');
  const [isAddingFI, setIsAddingFI] = useState(false);

  const loadPortfolios = useCallback(async (selectId?: string) => {
    setIsLoadingPortfolios(true);
    try {
      const res = await fetch(`${API_URL}/portfolios`, { credentials: 'include', cache: 'no-store' });
      if (res.ok) {
        const data = await res.json();
        setPortfolios(data || []);
        if (data && data.length > 0) {
          const nextId = selectId || data[0].id;
          setActivePortfolioId(nextId);
        }
      }
    } catch (e) { console.error('Erro ao buscar portfólios:', e); } finally { setIsLoadingPortfolios(false); }
  }, []);

  const loadPortfolioDetails = useCallback(async (id: string) => {
    if (!id) return;
    setIsLoadingDetails(true);
    try {
      const resDetails = await fetch(`${API_URL}/portfolios/${id}`, { credentials: 'include', cache: 'no-store' });
      if (resDetails.ok) setPositions((await resDetails.json()).positions || []);
      const resTxs = await fetch(`${API_URL}/portfolios/${id}/history`, { credentials: 'include', cache: 'no-store' });
      if (resTxs.ok) setTransactions(await resTxs.json() || []);
      
      const resFI = await fetch(`${API_URL}/portfolios/${id}/fixed-income/positions`, { credentials: 'include', cache: 'no-store' });
      if (resFI.ok) setFiPositions(await resFI.json() || []);
    } catch (e) { console.error('Erro ao buscar detalhes:', e); } finally { setIsLoadingDetails(false); }
  }, []);

  const loadPerformance = useCallback(async (id: string, selectPeriod: string, filterTickers: string[] = []) => {
    if (!id) return;
    setIsLoadingPerformance(true);
    try {
      let url = `${API_URL}/portfolios/${id}/performance?period=${selectPeriod}`;
      if (filterTickers.length > 0) url += `&tickers=${filterTickers.join(',')}`;
      const res = await fetch(url, { credentials: 'include', cache: 'no-store' });
      if (res.ok) setPerformanceData(await res.json() || []);
    } catch (e) { console.error('Erro ao buscar série histórica:', e); } finally { setIsLoadingPerformance(false); }
  }, []);

  const loadDividends = useCallback(async (id: string) => {
    if (!id) return;
    setIsLoadingDividends(true);
    try {
      const [resDivs, resFI] = await Promise.all([
        fetch(`${API_URL}/portfolios/${id}/dividends`, { credentials: 'include', cache: 'no-store' }),
        fetch(`${API_URL}/portfolios/${id}/fixed-income/monthly-yields`, { credentials: 'include', cache: 'no-store' })
      ]);
      
      let allDividends: CalculatedDividend[] = [];
      if (resDivs.ok) {
        allDividends = await resDivs.json() || [];
      }
      
      if (resFI.ok) {
        const fiYields = await resFI.json() || [];
        const mappedFI = fiYields.map((fy: any) => {
          // Fake dates for FI yields to be at the end of the month
          const [yearStr, monthStr] = fy.month.split('-');
          // Last day of the month
          const date = new Date(parseInt(yearStr), parseInt(monthStr), 0).toISOString().split('T')[0];
          return {
            asset_id: fy.asset_id,
            ticker: fy.asset_name,
            asset_name: fy.asset_name,
            asset_type: fy.asset_type,
            ex_date: date,
            payment_date: date,
            gross_amount: fy.gross_amount,
            net_amount: fy.net_amount,
            currency: 'BRL',
            type: 'YIELD',
            quantity: 1,
            per_share_amount: fy.net_amount,
            is_accrued: true
          } as CalculatedDividend;
        });
        allDividends = [...allDividends, ...mappedFI];
      }
      
      allDividends.sort((a, b) => {
        const dateA = new Date(a.payment_date || a.ex_date).getTime();
        const dateB = new Date(b.payment_date || b.ex_date).getTime();
        return dateB - dateA; // Descending
      });
      
      setDividends(allDividends);
    } catch (e) { console.error('Erro ao buscar proventos:', e); } finally { setIsLoadingDividends(false); }
  }, []);

  useEffect(() => { if (user) loadPortfolios(); }, [user, loadPortfolios]);

  useEffect(() => {
    if (activePortfolioId) {
      loadPortfolioDetails(activePortfolioId);
      loadDividends(activePortfolioId);
    }
  }, [activePortfolioId, loadPortfolioDetails, loadDividends]);

  // Reset chart filter if category changes
  useEffect(() => { setFilterChartTicker('Todos'); }, [activeCategoryFilter]);

  useEffect(() => {
    if (!activePortfolioId) return;
    let targetTickers: string[] = [];
    if (activeCategoryFilter === 'Renda Variável') {
      targetTickers = positions.map(p => p.ticker);
      if (targetTickers.length === 0) targetTickers = ['NONE_FOUND'];
    } else if (activeCategoryFilter !== 'Todas' && activeCategoryFilter !== 'Renda Fixa') {
      const filtered = positions.filter(pos => getAssetCategory(pos.type) === activeCategoryFilter);
      targetTickers = filtered.map(p => p.ticker);
      if (targetTickers.length === 0) targetTickers = ['NONE_FOUND'];
    } else if (activeCategoryFilter === 'Renda Fixa') {
      targetTickers = ['NONE_FOUND'];
    }

    if (filterChartTicker !== 'Todos') {
      targetTickers = [filterChartTicker];
    }

    loadPerformance(activePortfolioId, period, targetTickers);
  }, [activePortfolioId, period, activeCategoryFilter, filterChartTicker, positions, loadPerformance]);

  useEffect(() => {
    if (!searchQuery.trim() || searchQuery === txTicker) {
      setSearchResults([]); setShowDropdown(false); return;
    }
    const delayDebounce = setTimeout(async () => {
      setIsSearching(true);
      try {
        const res = await fetch(`${API_URL}/assets/search?q=${encodeURIComponent(searchQuery)}`, { credentials: 'include', cache: 'no-store' });
        if (res.ok) { setSearchResults(await res.json() || []); setShowDropdown(true); }
      } catch (e) { console.error('Erro na busca:', e); } finally { setIsSearching(false); }
    }, 350);
    return () => clearTimeout(delayDebounce);
  }, [searchQuery, txTicker]);

  const handleSelectAsset = async (symbol: string) => {
    setTxTicker(symbol); setSearchQuery(symbol); setShowDropdown(false);
    try {
      const res = await fetch(`${API_URL}/quotes/${encodeURIComponent(symbol)}`, { credentials: 'include', cache: 'no-store' });
      if (res.ok) {
        const quote = await res.json();
        setSelectedAssetCurrency(quote.currency || 'BRL');
        if (quote.currency === 'USD') {
          const rateRes = await fetch(`${API_URL}/quotes/USDBRL=X`, { credentials: 'include', cache: 'no-store' });
          if (rateRes.ok) setTxExchangeRate((await rateRes.json()).price || 5.25);
          else setTxExchangeRate(5.25);
        } else { setTxExchangeRate(1.0); }
      }
    } catch (e) { setSelectedAssetCurrency('BRL'); setTxExchangeRate(1.0); }
  };

  const handleLinkTelegram = async () => {
    try {
      const res = await fetch(`${API_URL}/telegram/link`, { method: 'POST', credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        const botUsername = data.bot_username || 'StockPulseBot';
        window.open(`https://t.me/${botUsername}?start=${data.token}`, '_blank');
      } else {
        alert('Erro ao gerar link do Telegram.');
      }
    } catch (e) {
      console.error(e);
      alert('Erro ao comunicar com o servidor.');
    }
  };

  const handleCreatePortfolio = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPortfolioName.trim()) return;
    setIsCreatingPortfolio(true);
    try {
      const res = await fetch(`${API_URL}/portfolios`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newPortfolioName, base_currency: newPortfolioCurrency }),
        credentials: 'include', cache: 'no-store',
      });
      if (res.ok) {
        setNewPortfolioName(''); setShowPortfolioModal(false);
        await loadPortfolios((await res.json()).id);
      }
    } catch (e) { console.error(e); } finally { setIsCreatingPortfolio(false); }
  };

  const handleDeletePortfolio = async () => {
    if (portfolios.length <= 1) return alert('Você precisa manter pelo menos uma carteira ativa no sistema.');
    if (!confirm('Deseja realmente apagar esta carteira? Todas as transações serão excluídas.')) return;
    try {
      const res = await fetch(`${API_URL}/portfolios/${activePortfolioId}`, { method: 'DELETE', credentials: 'include', cache: 'no-store' });
      if (res.ok) await loadPortfolios();
    } catch (e) { console.error(e); }
  };

  const handleAddTransaction = async (e: React.FormEvent) => {
    e.preventDefault();
    const parsedQty = parseFloat(txQuantity.toString());
    const parsedPrice = parseFloat(txUnitPrice.toString());
    const parsedRate = parseFloat(txExchangeRate.toString());

    if (!txTicker || isNaN(parsedQty) || parsedQty <= 0 || (txType !== 'SPLIT' && txType !== 'REVERSE_SPLIT' && (isNaN(parsedPrice) || parsedPrice <= 0))) {
      return alert('Preencha todos os campos obrigatórios corretamente.');
    }

    if (txType === 'SELL') {
      const currentQty = positions.find((p) => p.ticker.toUpperCase() === txTicker.toUpperCase())?.quantity || 0;
      if (parsedQty > currentQty) return alert(`Saldo insuficiente. Você possui apenas ${currentQty} cotas.`);
    }

    setIsAddingTx(true);
    try {
      const url = editingTxId ? `${API_URL}/portfolios/${activePortfolioId}/transactions/${editingTxId}` : `${API_URL}/portfolios/${activePortfolioId}/transactions`;
      const res = await fetch(url, {
        method: editingTxId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ticker: txTicker, type: txType, quantity: parsedQty,
          unit_price: (txType === 'SPLIT' || txType === 'REVERSE_SPLIT') ? 0 : parsedPrice,
          exchange_rate: isNaN(parsedRate) || parsedRate <= 0 ? 0.0 : parsedRate,
          executed_at: txExecutedAt,
        }),
        credentials: 'include', cache: 'no-store',
      });

      if (res.ok) {
        setTxTicker(''); setSearchQuery(''); setTxQuantity(''); setTxUnitPrice(''); setTxExchangeRate(1.0);
        setEditingTxId(null); setSelectedAssetCurrency('BRL'); setShowTxModal(false);
        await loadPortfolioDetails(activePortfolioId); await loadPerformance(activePortfolioId, period);
      } else { alert((await res.json()).error || 'Erro ao cadastrar transação.'); }
    } catch (e) { console.error(e); } finally { setIsAddingTx(false); }
  };


  const handleAddFixedIncome = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fiInstitution || !fiRate || !fiAmount) return alert('Preencha os campos obrigatórios');
    setIsAddingFI(true);

    try {
      const assetPayload: any = {
        institution: fiInstitution, type: fiType, debt_type: fiDebtType, indexer: fiIndexer,
        rate: parseFloat(fiRate.toString())
      };
      if (fiMaturityDate) {
        assetPayload.maturity_date = new Date(fiMaturityDate).toISOString();
      }

      const assetRes = await fetch(`${API_URL}/portfolios/${activePortfolioId}/fixed-income/assets`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(assetPayload), credentials: 'include', cache: 'no-store'
      });

      if (!assetRes.ok) throw new Error("Erro ao criar ativo");
      const asset = await assetRes.json();

      const txRes = await fetch(`${API_URL}/portfolios/${activePortfolioId}/fixed-income/assets/${asset.id}/transactions`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: 'SUBSCRIPTION', amount: parseFloat(fiAmount.toString().replace(/\./g, '').replace(',', '.')), date: fiApplicationDate ? new Date(fiApplicationDate).toISOString() : new Date().toISOString()
        }), credentials: 'include', cache: 'no-store'
      });

      if (!txRes.ok) throw new Error("Erro ao criar transação");
      
      setShowFIModal(false);
      setFiInstitution(''); setFiRate(''); setFiAmount(''); setFiMaturityDate(''); setFiApplicationDate(new Date().toISOString().split('T')[0]);
      
      // Reload page state to see changes
      window.location.reload();
    } catch (e) {
      alert("Erro ao salvar aplicação de Renda Fixa.");
      console.error(e);
    } finally {
      setIsAddingFI(false);
    }
  };

  const handleEditTransaction = (tx: UnifiedTransaction) => {
    if (tx.module === 'RF') {
      setEditingTxId(tx.id);
      setFiEditTxAssetName(tx.asset_name);
      setFiTxType(tx.type);
      setFiAmount(Number(tx.total_value).toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }));
      setFiApplicationDate(tx.date ? tx.date.split('T')[0] : '');
      setFiMaturityDate(tx.maturity_date ? tx.maturity_date.split('T')[0] : '');
      setShowFIEditModal(true);
      return;
    }
    
    setEditingTxId(tx.id); setTxTicker(tx.asset_name); setTxType(tx.type as any);
    setTxQuantity(tx.quantity || 0); setTxUnitPrice(tx.unit_price || 0); setTxExchangeRate(tx.exchange_rate || 0);
    setSelectedAssetCurrency(tx.currency || 'BRL');
    setTxExecutedAt(tx.date ? tx.date.split('T')[0] : ''); setShowTxModal(true);
  };

  const handleUpdateFITransaction = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingTxId || !fiAmount || !fiApplicationDate) return alert('Preencha os campos obrigatórios');
    setIsAddingFI(true);
    try {
      const res = await fetch(`${API_URL}/portfolios/${activePortfolioId}/fixed-income/transactions/${editingTxId}`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: fiTxType,
          amount: parseFloat(fiAmount.toString().replace(/\./g, '').replace(',', '.')),
          date: new Date(fiApplicationDate).toISOString(),
          maturity_date: fiMaturityDate ? new Date(fiMaturityDate).toISOString() : undefined
        }),
        credentials: 'include', cache: 'no-store'
      });
      if (!res.ok) throw new Error("Erro ao atualizar transação");
      
      setShowFIEditModal(false);
      setEditingTxId(null);
      setFiAmount('');
      await loadPortfolioDetails(activePortfolioId);
      await loadPerformance(activePortfolioId, period);
    } catch (e) {
      alert("Erro ao salvar transação de Renda Fixa.");
      console.error(e);
    } finally {
      setIsAddingFI(false);
    }
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const formData = new FormData(); formData.append("file", file);
    try {
      const res = await fetch(`${API_URL}/portfolios/${activePortfolioId}/transactions/bulk`, { method: "POST", credentials: "include", body: formData });
      if (res.ok) {
        const data = await res.json();
        if (data.errors?.length > 0) alert(`Importados com sucesso: ${data.success}\nFalhas:\n- ${data.errors.join("\n- ")}`);
        else alert(`Importação concluída com sucesso! ${data.success} registros importados.`);
        await loadPortfolioDetails(activePortfolioId); await loadPerformance(activePortfolioId, period);
      } else alert("Erro ao enviar arquivo.");
    } catch (err) { alert("Erro de conexão."); }
    e.target.value = '';
  };

  const handleDeleteTransaction = async (txId: string) => {
    if (!confirm('Deseja realmente excluir esta transação?')) return;
    try {
      const tx = transactions.find(t => t.id === txId);
      let endpoint = `${API_URL}/portfolios/${activePortfolioId}/transactions/${txId}`;
      if (tx?.module === 'RF') {
        endpoint = `${API_URL}/portfolios/${activePortfolioId}/fixed-income/transactions/${txId}`;
      }

      const res = await fetch(endpoint, { method: 'DELETE', credentials: 'include', cache: 'no-store' });
      if (res.ok) { await loadPortfolioDetails(activePortfolioId); await loadPerformance(activePortfolioId, period); }
    } catch (e) { console.error(e); }
  };

  const handleExportPortfolio = async () => {
    try {
      const res = await fetch(`${API_URL}/portfolios/${activePortfolioId}/export`, {
        method: 'GET',
        credentials: 'include',
        cache: 'no-store'
      });
      if (res.ok) {
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        
        // Obter nome do arquivo do header se disponível, ou fallback
        const disposition = res.headers.get('Content-Disposition');
        let filename = `backup-carteira.zip`;
        if (disposition && disposition.indexOf('filename=') !== -1) {
          const matches = /filename="([^"]*)"/.exec(disposition);
          if (matches != null && matches[1]) { 
            filename = matches[1];
          }
        }
        
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);
      } else {
        alert("Erro ao exportar backup.");
      }
    } catch (e) {
      console.error(e);
      alert("Erro de conexão ao exportar backup.");
    }
  };

  if (authLoading || isLoadingPortfolios) {
    return (
      <main className="container">
        <div className="glass-panel">
          <span className="loading-spinner" style={{ borderTopColor: '#00f2fe', width: 40, height: 40 }}></span>
          <p className="text-secondary mt-lg">Carregando dados financeiros seguros...</p>
        </div>
      </main>
    );
  }

  if (!user) return null;

  const activeP = portfolios.find((p) => p.id === activePortfolioId);
  const kpiCurrency = activeP ? activeP.base_currency : 'BRL';

  const filteredPositions = positions.filter(pos => activeCategoryFilter === 'Todas' || activeCategoryFilter === 'Renda Variável' || getAssetCategory(pos.type) === activeCategoryFilter);
  const filteredTransactions = transactions.filter(tx => activeCategoryFilter === 'Todas' || (activeCategoryFilter === 'Renda Variável' && tx.module !== 'RF') || getAssetCategory(tx.asset_type || '') === activeCategoryFilter || (activeCategoryFilter === 'Renda Fixa' && tx.module === 'RF'));
  const categoryFilteredDividends = dividends.filter(div => {
    if (activeCategoryFilter === 'Todas') return true;
    if (activeCategoryFilter === 'Renda Variável') return !div.is_accrued;
    if (activeCategoryFilter === 'Renda Fixa') return div.is_accrued;
    if (div.is_accrued || getAssetCategory(div.asset_type) !== activeCategoryFilter) return false;
    return true;
  });

  const filteredDividends = categoryFilteredDividends.filter(div => {
    const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.ex_date;
    if (!dateStr) return true;
    const year = dateStr.substring(0, 4);
    const month = dateStr.substring(5, 7);
    return (filterDivYear === 'Todos' || year === filterDivYear) && (filterDivMonth === 'Todos' || month === filterDivMonth);
  });

  const availableYears = Array.from(new Set(categoryFilteredDividends.map(d => ((d.payment_date && !d.payment_date.startsWith('0001') ? d.payment_date : d.ex_date) || '').substring(0, 4)).filter(Boolean))).sort((a, b) => b.localeCompare(a));

  // Se a categoria for Renda Fixa ou Todas, precisamos somar a renda fixa
  const includeFI = activeCategoryFilter === 'Todas' || activeCategoryFilter === 'Renda Fixa';
  const filteredFI = includeFI ? fiPositions : [];

  const eqCost = filteredPositions.reduce((acc, pos) => acc + pos.total_cost, 0);
  const eqValue = filteredPositions.reduce((acc, pos) => acc + (pos.current_value || 0), 0);
  
  const fiCost = filteredFI.reduce((acc, pos) => acc + pos.total_invested, 0);
  const fiValue = filteredFI.reduce((acc, pos) => acc + pos.net_value, 0);

  // Se o filtro for APENAS Renda Fixa, eqCost e eqValue serão 0 (porque filteredPositions estará vazio).
  const totalCost = eqCost + fiCost;
  const currentValue = eqValue + fiValue;
  const profitLoss = currentValue - totalCost;
  const returnPercent = totalCost > 0 ? (profitLoss / totalCost) * 100 : 0.0;
  
  const twelveMonthsAgo = new Date();
  twelveMonthsAgo.setMonth(twelveMonthsAgo.getMonth() - 12);
  const divs12m = categoryFilteredDividends.filter(div => {
    const dateStr = (div.payment_date && !div.payment_date.startsWith('0001')) ? div.payment_date : div.ex_date;
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

  return (
    <main className="container" style={{ maxWidth: 1400 }}>
      <PortfolioHeader userName={user?.name || 'Investidor'} onLogout={logout} />

      <PortfolioTabs 
        portfolios={portfolios} 
        activePortfolioId={activePortfolioId} setActivePortfolioId={setActivePortfolioId} 
        setShowPortfolioModal={setShowPortfolioModal} handleDeletePortfolio={handleDeletePortfolio} 
        handleExportPortfolio={handleExportPortfolio}
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
          <PortfolioSummaryCards totalCost={totalCost} currentValue={currentValue} profitLoss={profitLoss} returnPercent={returnPercent} avgDividends12m={avgDividends12m} kpiCurrency={kpiCurrency} />



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
              performanceData={performanceData}
              kpiCurrency={kpiCurrency}
            />
          )}

          {activeTab === 'diario' && (
            <DailyReport positions={filteredPositions} kpiCurrency={kpiCurrency} />
          )}

          {activeTab === 'renda-fixa' && (
            <FixedIncomeTab portfolioId={activePortfolioId} onLaunchOperation={() => setShowFIModal(true)} />
          )}

          {activeTab === 'tesouro' && (
            <TreasuryTab portfolioId={activePortfolioId} />
          )}
        </div>
      )}

      <Modals 
        showPortfolioModal={showPortfolioModal} setShowPortfolioModal={setShowPortfolioModal}
        newPortfolioName={newPortfolioName} setNewPortfolioName={setNewPortfolioName}
        newPortfolioCurrency={newPortfolioCurrency} setNewPortfolioCurrency={setNewPortfolioCurrency}
        isCreatingPortfolio={isCreatingPortfolio} handleCreatePortfolio={handleCreatePortfolio}
        showTxModal={showTxModal} setShowTxModal={setShowTxModal} editingTxId={editingTxId} setEditingTxId={setEditingTxId}
        txTicker={txTicker} searchQuery={searchQuery} setSearchQuery={setSearchQuery} isSearching={isSearching}
        showDropdown={showDropdown} searchResults={searchResults} handleSelectAsset={handleSelectAsset}
        isAddingTx={isAddingTx} txType={txType} setTxType={setTxType} txQuantity={txQuantity} setTxQuantity={setTxQuantity}
        txUnitPrice={txUnitPrice} setTxUnitPrice={setTxUnitPrice} txExchangeRate={txExchangeRate} setTxExchangeRate={setTxExchangeRate}
        txExecutedAt={txExecutedAt} setTxExecutedAt={setTxExecutedAt} selectedAssetCurrency={selectedAssetCurrency} kpiCurrency={kpiCurrency}
        handleAddTransaction={handleAddTransaction}
        
        showFIModal={showFIModal} setShowFIModal={setShowFIModal}
        showFIEditModal={showFIEditModal} setShowFIEditModal={setShowFIEditModal}
        fiEditTxAssetName={fiEditTxAssetName} setFiEditTxAssetName={setFiEditTxAssetName}
        handleUpdateFITransaction={handleUpdateFITransaction}
        fiInstitution={fiInstitution} setFiInstitution={setFiInstitution}
        fiType={fiType} setFiType={setFiType}
        fiDebtType={fiDebtType} setFiDebtType={setFiDebtType}
        fiIndexer={fiIndexer} setFiIndexer={setFiIndexer}
        fiRate={fiRate} setFiRate={setFiRate}
        fiTxType={fiTxType} setFiTxType={setFiTxType}
        fiAmount={fiAmount} setFiAmount={setFiAmount}
        fiApplicationDate={fiApplicationDate} setFiApplicationDate={setFiApplicationDate}
        fiMaturityDate={fiMaturityDate} setFiMaturityDate={setFiMaturityDate}
        isAddingFI={isAddingFI} handleAddFixedIncome={handleAddFixedIncome}
      />
    </main>
  );
}
