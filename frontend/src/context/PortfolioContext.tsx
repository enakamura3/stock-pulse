'use client';

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/context/AuthContext';
import { apiFetch } from '@/lib/api';
import {
  Portfolio,
  Position,
  PerformancePoint,
  CalculatedDividend,
  SearchResult,
  FixedIncomePosition,
  UnifiedTransaction,
  TreasuryPosition,
} from '@/components/portfolio/types';
import { getAssetCategory } from '@/components/portfolio/helpers';

interface PortfolioContextType {
  // Data
  portfolios: Portfolio[];
  activePortfolioId: string;
  activePortfolio?: Portfolio;
  kpiCurrency: string;
  positions: Position[];
  fiPositions: FixedIncomePosition[];
  treasuryPositions: TreasuryPosition[];
  transactions: UnifiedTransaction[];
  performanceData: PerformancePoint[];
  dividends: CalculatedDividend[];

  // Loading
  isLoadingPortfolios: boolean;
  isLoadingDetails: boolean;
  isLoadingPerformance: boolean;
  isLoadingDividends: boolean;
  isLoadingTreasury: boolean;

  // Filters & Tabs
  activeTab: 'ativos' | 'operacoes' | 'proventos' | 'insights' | 'analise' | 'diario' | 'renda-fixa' | 'tesouro';
  setActiveTab: (tab: 'ativos' | 'operacoes' | 'proventos' | 'insights' | 'analise' | 'diario' | 'renda-fixa' | 'tesouro') => void;
  activeCategoryFilter: string;
  setActiveCategoryFilter: (cat: string) => void;
  filterTxTicker: string;
  setFilterTxTicker: (t: string) => void;
  filterChartTicker: string;
  setFilterChartTicker: (t: string) => void;
  filterDivYear: string;
  setFilterDivYear: (y: string) => void;
  filterDivMonth: string;
  setFilterDivMonth: (m: string) => void;
  period: string;
  setPeriod: (p: string) => void;

  // Modals visibility
  showPortfolioModal: boolean;
  setShowPortfolioModal: (s: boolean) => void;
  showTxModal: boolean;
  setShowTxModal: (s: boolean) => void;
  showFIModal: boolean;
  setShowFIModal: (s: boolean) => void;
  showFIEditModal: boolean;
  setShowFIEditModal: (s: boolean) => void;

  // Portfolio Form
  newPortfolioName: string;
  setNewPortfolioName: (s: string) => void;
  newPortfolioCurrency: string;
  setNewPortfolioCurrency: (s: string) => void;
  isCreatingPortfolio: boolean;

  // Transaction Form
  txTicker: string;
  setTxTicker: (s: string) => void;
  txType: 'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS';
  setTxType: (t: 'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS') => void;
  txQuantity: string | number;
  setTxQuantity: (q: string | number) => void;
  txUnitPrice: string | number;
  setTxUnitPrice: (p: string | number) => void;
  txFee: string | number;
  setTxFee: (f: string | number) => void;
  txExchangeRate: string | number;
  setTxExchangeRate: (r: string | number) => void;
  txExecutedAt: string;
  setTxExecutedAt: (d: string) => void;
  isAddingTx: boolean;
  editingTxId: string | null;
  setEditingTxId: (id: string | null) => void;

  // Asset Search Form
  searchQuery: string;
  setSearchQuery: (s: string) => void;
  searchResults: SearchResult[];
  isSearching: boolean;
  showDropdown: boolean;
  setShowDropdown: (s: boolean) => void;
  selectedAssetCurrency: string;

  // Fixed Income Form
  fiInstitution: string;
  setFiInstitution: (s: string) => void;
  fiType: string;
  setFiType: (s: string) => void;
  fiDebtType: string;
  setFiDebtType: (s: string) => void;
  fiIndexer: string;
  setFiIndexer: (s: string) => void;
  fiRate: string | number;
  setFiRate: (s: string | number) => void;
  fiAmount: string | number;
  setFiAmount: (s: string | number) => void;
  fiApplicationDate: string;
  setFiApplicationDate: (s: string) => void;
  fiMaturityDate: string;
  setFiMaturityDate: (s: string) => void;
  fiTxType: string;
  setFiTxType: (s: string) => void;
  fiEditTxAssetName: string;
  setFiEditTxAssetName: (s: string) => void;
  isAddingFI: boolean;

  // Actions
  loadPortfolios: (selectId?: string) => Promise<void>;
  loadPortfolioDetails: (id: string) => Promise<void>;
  loadPerformance: (id: string, selectPeriod: string, filterTickers?: string[]) => Promise<void>;
  loadDividends: (id: string) => Promise<void>;
  loadTreasuryPositions: (id: string) => Promise<void>;
  setActivePortfolioId: (id: string) => void;
  handleSelectAsset: (symbol: string) => Promise<void>;
  handleCreatePortfolio: (e: React.FormEvent) => Promise<void>;
  handleDeletePortfolio: () => Promise<void>;
  handleSetDefaultPortfolio: () => Promise<void>;
  handleAddTransaction: (e: React.FormEvent) => Promise<void>;
  handleEditTransaction: (tx: UnifiedTransaction) => void;
  handleDeleteTransaction: (txId: string) => Promise<void>;
  handleAddFixedIncome: (e: React.FormEvent) => Promise<void>;
  handleUpdateFITransaction: (e: React.FormEvent) => Promise<void>;
  handleFileUpload: (e: React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  handleExportPortfolio: () => Promise<void>;
  handleLinkTelegram: () => Promise<void>;
}

const PortfolioContext = createContext<PortfolioContextType | undefined>(undefined);

export function PortfolioProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();

  const [portfolios, setPortfolios] = useState<Portfolio[]>([]);
  const [activePortfolioId, setActivePortfolioId] = useState<string>('');
  const [activeCategoryFilter, setActiveCategoryFilter] = useState<string>('Todas');
  const [positions, setPositions] = useState<Position[]>([]);
  const [fiPositions, setFiPositions] = useState<FixedIncomePosition[]>([]);
  const [treasuryPositions, setTreasuryPositions] = useState<TreasuryPosition[]>([]);
  const [isLoadingTreasury, setIsLoadingTreasury] = useState(false);
  const [transactions, setTransactions] = useState<UnifiedTransaction[]>([]);
  const [performanceData, setPerformanceData] = useState<PerformancePoint[]>([]);
  const [dividends, setDividends] = useState<CalculatedDividend[]>([]);
  
  const [filterTxTicker, setFilterTxTicker] = useState<string>('');
  const [filterChartTicker, setFilterChartTicker] = useState<string>('Todos');
  const currentYear = new Date().getFullYear().toString();
  const currentMonth = String(new Date().getMonth() + 1).padStart(2, '0');
  const [filterDivYear, setFilterDivYear] = useState<string>(currentYear);
  const [filterDivMonth, setFilterDivMonth] = useState<string>(currentMonth);
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
  const [txFee, setTxFee] = useState<string | number>('');
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

  const activePortfolio = portfolios.find((p) => p.id === activePortfolioId);
  const kpiCurrency = activePortfolio?.base_currency || 'BRL';

  const loadPortfolios = useCallback(async (selectId?: string) => {
    setIsLoadingPortfolios(true);
    try {
      const res = await apiFetch(`/portfolios`, { cache: 'no-store' });
      if (res.ok) {
        const data: Portfolio[] = await res.json();
        setPortfolios(data || []);
        if (data && data.length > 0) {
          const defaultP = data.find(p => p.is_default);
          const nextId = selectId || (defaultP ? defaultP.id : data[0].id);
          setActivePortfolioId(nextId);
        }
      }
    } catch (e) { console.error('Erro ao buscar portfólios:', e); } finally { setIsLoadingPortfolios(false); }
  }, []);

  const loadTreasuryPositions = useCallback(async (id: string) => {
    if (!id) return;
    setIsLoadingTreasury(true);
    try {
      const res = await apiFetch(`/portfolios/${id}/treasury/positions`, { cache: 'no-store' });
      if (res.ok) setTreasuryPositions(await res.json() || []);
    } catch (e) {
      console.error('Erro ao carregar Tesouro:', e);
    } finally {
      setIsLoadingTreasury(false);
    }
  }, []);

  const loadPortfolioDetails = useCallback(async (id: string) => {
    if (!id) return;
    setIsLoadingDetails(true);
    try {
      const resDetails = await apiFetch(`/portfolios/${id}`, { cache: 'no-store' });
      if (resDetails.ok) setPositions((await resDetails.json()).positions || []);
      const resTxs = await apiFetch(`/portfolios/${id}/history`, { cache: 'no-store' });
      if (resTxs.ok) setTransactions(await resTxs.json() || []);
      
      const resFI = await apiFetch(`/portfolios/${id}/fixed-income/positions`, { cache: 'no-store' });
      if (resFI.ok) setFiPositions(await resFI.json() || []);

      await loadTreasuryPositions(id);
    } catch (e) { console.error('Erro ao buscar detalhes:', e); } finally { setIsLoadingDetails(false); }
  }, [loadTreasuryPositions]);

  const loadPerformance = useCallback(async (id: string, selectPeriod: string, filterTickers: string[] = []) => {
    if (!id) return;
    setIsLoadingPerformance(true);
    try {
      let url = `/portfolios/${id}/performance?period=${selectPeriod}`;
      if (filterTickers.length > 0) url += `&tickers=${filterTickers.join(',')}`;
      const res = await apiFetch(url, { cache: 'no-store' });
      if (res.ok) setPerformanceData(await res.json() || []);
    } catch (e) { console.error('Erro ao buscar série histórica:', e); } finally { setIsLoadingPerformance(false); }
  }, []);

  const loadDividends = useCallback(async (id: string) => {
    if (!id) return;
    setIsLoadingDividends(true);
    try {
      const [resDivs, resFI, resTD] = await Promise.all([
        apiFetch(`/portfolios/${id}/dividends`, { cache: 'no-store' }),
        apiFetch(`/portfolios/${id}/fixed-income/monthly-yields`, { cache: 'no-store' }),
        apiFetch(`/portfolios/${id}/treasury/monthly-yields`, { cache: 'no-store' })
      ]);
      
      let allDividends: CalculatedDividend[] = [];
      if (resDivs.ok) {
        allDividends = await resDivs.json() || [];
      }
      
      if (resFI.ok) {
        const fiYields = await resFI.json() || [];
        const mappedFI = fiYields.map((fy: any) => {
          const [yearStr, monthStr] = fy.month.split('-');
          const date = new Date(parseInt(yearStr), parseInt(monthStr), 0).toISOString().split('T')[0];
          return {
            asset_id: fy.asset_id,
            ticker: fy.asset_name,
            asset_name: fy.asset_name,
            asset_type: fy.asset_type,
            cum_date: date,
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

      if (resTD.ok) {
        const tdYields = await resTD.json() || [];
        const mappedTD = tdYields.map((ty: any) => {
          const [yearStr, monthStr] = ty.month.split('-');
          const date = new Date(parseInt(yearStr), parseInt(monthStr), 0).toISOString().split('T')[0];
          return {
            asset_id: ty.asset_id,
            ticker: ty.asset_name,
            asset_name: ty.asset_name,
            asset_type: 'TESOURO',
            cum_date: date,
            payment_date: date,
            gross_amount: ty.gross_amount,
            net_amount: ty.net_amount,
            currency: 'BRL',
            type: 'YIELD',
            quantity: 1,
            per_share_amount: ty.net_amount,
            is_accrued: true
          } as CalculatedDividend;
        });
        allDividends = [...allDividends, ...mappedTD];
      }
      
      allDividends.sort((a, b) => {
        const dateA = new Date(a.payment_date || a.cum_date).getTime();
        const dateB = new Date(b.payment_date || b.cum_date).getTime();
        return dateB - dateA;
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
        const res = await apiFetch(`/assets/search?q=${encodeURIComponent(searchQuery)}`, { cache: 'no-store' });
        if (res.ok) { setSearchResults(await res.json() || []); setShowDropdown(true); }
      } catch (e) { console.error('Erro na busca:', e); } finally { setIsSearching(false); }
    }, 350);
    return () => clearTimeout(delayDebounce);
  }, [searchQuery, txTicker]);

  const handleSelectAsset = async (symbol: string) => {
    setTxTicker(symbol); setSearchQuery(symbol); setShowDropdown(false);
    try {
      const res = await apiFetch(`/quotes/${encodeURIComponent(symbol)}`, { cache: 'no-store' });
      if (res.ok) {
        const quote = await res.json();
        setSelectedAssetCurrency(quote.currency || 'BRL');
        if (quote.currency === 'USD') {
          const rateRes = await apiFetch(`/quotes/USDBRL=X`, { cache: 'no-store' });
          if (rateRes.ok) setTxExchangeRate((await rateRes.json()).price || 5.25);
          else setTxExchangeRate(5.25);
        } else { setTxExchangeRate(1.0); }
      }
    } catch (e) { setSelectedAssetCurrency('BRL'); setTxExchangeRate(1.0); }
  };

  const handleLinkTelegram = async () => {
    try {
      const res = await apiFetch(`/telegram/link`, { method: 'POST' });
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
      const res = await apiFetch(`/portfolios`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newPortfolioName, base_currency: newPortfolioCurrency }), cache: 'no-store',
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
      const res = await apiFetch(`/portfolios/${activePortfolioId}`, { method: 'DELETE', cache: 'no-store' });
      if (res.ok) await loadPortfolios();
    } catch (e) { console.error(e); }
  };

  const handleSetDefaultPortfolio = async () => {
    if (!activePortfolioId) return;
    try {
      const res = await apiFetch(`/portfolios/${activePortfolioId}/default`, {
        method: 'PUT', cache: 'no-store'
      });
      if (res.ok) {
        await loadPortfolios(activePortfolioId);
      } else {
        alert('Erro ao definir carteira padrão.');
      }
    } catch (e) { console.error(e); }
  };

  const handleAddTransaction = async (e: React.FormEvent) => {
    e.preventDefault();
    const parsedQty = parseFloat(txQuantity.toString());
    const parsedPrice = parseFloat(txUnitPrice.toString());
    const parsedFee = parseFloat(txFee.toString());
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
      const url = editingTxId ? `/portfolios/${activePortfolioId}/transactions/${editingTxId}` : `/portfolios/${activePortfolioId}/transactions`;
      const res = await apiFetch(url, {
        method: editingTxId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ticker: txTicker, type: txType, quantity: parsedQty,
          unit_price: (txType === 'SPLIT' || txType === 'REVERSE_SPLIT') ? 0 : parsedPrice,
          fee: (txType === 'SPLIT' || txType === 'REVERSE_SPLIT' || txType === 'BONUS' || isNaN(parsedFee) || parsedFee < 0) ? 0.0 : parsedFee,
          exchange_rate: isNaN(parsedRate) || parsedRate <= 0 ? 0.0 : parsedRate,
          executed_at: txExecutedAt,
        }), cache: 'no-store',
      });

      if (res.ok) {
        setTxTicker(''); setSearchQuery(''); setTxQuantity(''); setTxUnitPrice(''); setTxFee(''); setTxExchangeRate(1.0);
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

      const assetRes = await apiFetch(`/portfolios/${activePortfolioId}/fixed-income/assets`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(assetPayload), cache: 'no-store'
      });

      if (!assetRes.ok) throw new Error("Erro ao criar ativo");
      const asset = await assetRes.json();

      const txRes = await apiFetch(`/portfolios/${activePortfolioId}/fixed-income/assets/${asset.id}/transactions`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: 'SUBSCRIPTION', amount: parseFloat(fiAmount.toString().replace(/\./g, '').replace(',', '.')), date: fiApplicationDate ? new Date(fiApplicationDate).toISOString() : new Date().toISOString()
        }), cache: 'no-store'
      });

      if (!txRes.ok) throw new Error("Erro ao criar transação");
      
      setShowFIModal(false);
      setFiInstitution(''); setFiRate(''); setFiAmount(''); setFiMaturityDate(''); setFiApplicationDate(new Date().toISOString().split('T')[0]);
      
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
      if (tx.asset_type === 'TESOURO') {
        alert('Para alterar uma operação do Tesouro Direto, utilize a aba "Tesouro Direto" ou exclua esta transação e registre-a novamente.');
        setActiveTab('tesouro');
        return;
      }
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
    setTxQuantity(tx.quantity || 0); setTxUnitPrice(tx.unit_price || 0); setTxFee(tx.fee || 0); setTxExchangeRate(tx.exchange_rate || 0);
    setSelectedAssetCurrency(tx.currency || 'BRL');
    setTxExecutedAt(tx.date ? tx.date.split('T')[0] : ''); setShowTxModal(true);
  };

  const handleUpdateFITransaction = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingTxId || !fiAmount || !fiApplicationDate) return alert('Preencha os campos obrigatórios');
    setIsAddingFI(true);
    try {
      const res = await apiFetch(`/portfolios/${activePortfolioId}/fixed-income/transactions/${editingTxId}`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: fiTxType,
          amount: parseFloat(fiAmount.toString().replace(/\./g, '').replace(',', '.')),
          date: new Date(fiApplicationDate).toISOString(),
          maturity_date: fiMaturityDate ? new Date(fiMaturityDate).toISOString() : undefined
        }), cache: 'no-store'
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
      const res = await apiFetch(`/portfolios/${activePortfolioId}/transactions/bulk`, { method: "POST", body: formData });
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
      let endpoint = `/portfolios/${activePortfolioId}/transactions/${txId}`;
      if (tx?.module === 'RF') {
        if (tx.asset_type === 'TESOURO') {
          endpoint = `/portfolios/${activePortfolioId}/treasury/transactions/${txId}`;
        } else {
          endpoint = `/portfolios/${activePortfolioId}/fixed-income/transactions/${txId}`;
        }
      }

      const res = await apiFetch(endpoint, { method: 'DELETE', cache: 'no-store' });
      if (res.ok) { await loadPortfolioDetails(activePortfolioId); await loadPerformance(activePortfolioId, period); }
    } catch (e) { console.error(e); }
  };

  const handleExportPortfolio = async () => {
    try {
      const res = await apiFetch(`/portfolios/${activePortfolioId}/export`, {
        method: 'GET',
        cache: 'no-store'
      });
      if (res.ok) {
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        
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

  return (
    <PortfolioContext.Provider
      value={{
        portfolios,
        activePortfolioId,
        activePortfolio,
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
        showPortfolioModal,
        setShowPortfolioModal,
        showTxModal,
        setShowTxModal,
        showFIModal,
        setShowFIModal,
        showFIEditModal,
        setShowFIEditModal,
        newPortfolioName,
        setNewPortfolioName,
        newPortfolioCurrency,
        setNewPortfolioCurrency,
        isCreatingPortfolio,
        txTicker, setTxTicker, txType, setTxType, txQuantity, setTxQuantity, txUnitPrice, setTxUnitPrice, txFee, setTxFee, txExchangeRate, setTxExchangeRate, txExecutedAt, setTxExecutedAt, isAddingTx, editingTxId, setEditingTxId,
        searchQuery,
        setSearchQuery,
        searchResults,
        isSearching,
        showDropdown,
        setShowDropdown,
        selectedAssetCurrency,
        fiInstitution,
        setFiInstitution,
        fiType,
        setFiType,
        fiDebtType,
        setFiDebtType,
        fiIndexer,
        setFiIndexer,
        fiRate,
        setFiRate,
        fiAmount,
        setFiAmount,
        fiApplicationDate,
        setFiApplicationDate,
        fiMaturityDate,
        setFiMaturityDate,
        fiTxType,
        setFiTxType,
        fiEditTxAssetName,
        setFiEditTxAssetName,
        isAddingFI,
        loadPortfolios,
        loadPortfolioDetails,
        loadPerformance,
        loadDividends,
        loadTreasuryPositions,
        setActivePortfolioId,
        handleSelectAsset,
        handleCreatePortfolio,
        handleDeletePortfolio,
        handleSetDefaultPortfolio,
        handleAddTransaction,
        handleEditTransaction,
        handleDeleteTransaction,
        handleAddFixedIncome,
        handleUpdateFITransaction,
        handleFileUpload,
        handleExportPortfolio,
        handleLinkTelegram,
      }}
    >
      {children}
    </PortfolioContext.Provider>
  );
}

export function usePortfolio() {
  const context = useContext(PortfolioContext);
  if (context === undefined) {
    throw new Error('usePortfolio deve ser usado dentro de um PortfolioProvider');
  }
  return context;
}
