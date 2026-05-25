'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/context/AuthContext';
import dynamic from 'next/dynamic';

// Importa o gráfico dinamicamente desativando SSR para evitar erros de renderização no servidor (Lightweight Charts)
const PortfolioChart = dynamic(() => import('@/components/PortfolioChart'), { ssr: false });

interface Portfolio {
  id: string;
  user_id: string;
  name: string;
  base_currency: string;
  created_at: string;
}

interface Position {
  asset_id: string;
  ticker: string;
  name: string;
  type: string;
  currency: string;
  quantity: number;
  average_price: number;
  total_cost: number;
  current_price?: number;
  current_value?: number;
  profit_loss?: number;
  return_percent?: number;
  graham_value?: number;
  bazin_value?: number;
}

interface Transaction {
  id: string;
  portfolio_id: string;
  asset_id: string;
  ticker?: string;
  asset_name?: string;
  asset_type?: string;
  currency?: string;
  type: string; // "BUY" ou "SELL"
  quantity: number;
  unit_price: number;
  total_cost: number;
  exchange_rate: number;
  executed_at: string;
  created_at: string;
}

interface PerformancePoint {
  date: string;
  value: number;
  total_invested: number;
}

interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
  type: string;
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export default function PortfolioPage() {
  const { user, logout, isLoading: authLoading } = useAuth();

  // Estados de Portfólios
  const [portfolios, setPortfolios] = useState<Portfolio[]>([]);
  const [activePortfolioId, setActivePortfolioId] = useState<string>('');
  const [positions, setPositions] = useState<Position[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [performanceData, setPerformanceData] = useState<PerformancePoint[]>([]);
  
  // Período do gráfico
  const [period, setPeriod] = useState<string>('ALL');

  // Modais
  const [showPortfolioModal, setShowPortfolioModal] = useState(false);
  const [showTxModal, setShowTxModal] = useState(false);
  
  // Loading states
  const [isLoadingPortfolios, setIsLoadingPortfolios] = useState(true);
  const [isLoadingDetails, setIsLoadingDetails] = useState(false);
  const [isLoadingPerformance, setIsLoadingPerformance] = useState(false);

  // Forms - Portfólio
  const [newPortfolioName, setNewPortfolioName] = useState('');
  const [newPortfolioCurrency, setNewPortfolioCurrency] = useState('BRL');
  const [isCreatingPortfolio, setIsCreatingPortfolio] = useState(false);

  // Forms - Transação
  const [txTicker, setTxTicker] = useState('');
  const [txType, setTxType] = useState<'BUY' | 'SELL'>('BUY');
  const [txQuantity, setTxQuantity] = useState<string | number>('');
  const [txUnitPrice, setTxUnitPrice] = useState<string | number>('');
  const [txExchangeRate, setTxExchangeRate] = useState<string | number>(1.0);
  const [txExecutedAt, setTxExecutedAt] = useState<string>(new Date().toISOString().split('T')[0]);
  const [isAddingTx, setIsAddingTx] = useState(false);
  
  // Autocomplete na Transação
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [selectedAssetCurrency, setSelectedAssetCurrency] = useState('BRL');

  // 1. CARREGA TODOS OS PORTFÓLIOS
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
    } catch (e) {
      console.error('Erro ao buscar portfólios:', e);
    } finally {
      setIsLoadingPortfolios(false);
    }
  }, []);

  // 2. CARREGA DETALHES CONSOLIDADOS (POSIÇÕES & DADOS ATUAIS)
  const loadPortfolioDetails = useCallback(async (id: string) => {
    if (!id) return;
    setIsLoadingDetails(true);
    try {
      // Detalhes / Posições
      const resDetails = await fetch(`${API_URL}/portfolios/${id}`, { credentials: 'include', cache: 'no-store' });
      if (resDetails.ok) {
        const data = await resDetails.json();
        setPositions(data.positions || []);
      }

      // Histórico de Transações
      const resTxs = await fetch(`${API_URL}/portfolios/${id}/transactions`, { credentials: 'include', cache: 'no-store' });
      if (resTxs.ok) {
        const txsData = await resTxs.json();
        setTransactions(txsData || []);
      }
    } catch (e) {
      console.error('Erro ao buscar detalhes do portfólio:', e);
    } finally {
      setIsLoadingDetails(false);
    }
  }, []);

  // 3. CARREGA DADOS DE EVOLUÇÃO PATRIMONIAL (GRÁFICO)
  const loadPerformance = useCallback(async (id: string, selectPeriod: string) => {
    if (!id) return;
    setIsLoadingPerformance(true);
    try {
      const res = await fetch(`${API_URL}/portfolios/${id}/performance?period=${selectPeriod}`, { credentials: 'include', cache: 'no-store' });
      if (res.ok) {
        const data = await res.json();
        setPerformanceData(data || []);
      }
    } catch (e) {
      console.error('Erro ao buscar série histórica do gráfico:', e);
    } finally {
      setIsLoadingPerformance(false);
    }
  }, []);

  // Onmount ou troca de usuário
  useEffect(() => {
    if (user) {
      loadPortfolios();
    }
  }, [user, loadPortfolios]);

  // Carrega dados da carteira quando a ID muda ou o período muda
  useEffect(() => {
    if (activePortfolioId) {
      loadPortfolioDetails(activePortfolioId);
      loadPerformance(activePortfolioId, period);
    }
  }, [activePortfolioId, period, loadPortfolioDetails, loadPerformance]);

  // Efeito Debounce para busca autocomplete dentro do Modal de Cadastro
  useEffect(() => {
    if (!searchQuery.trim()) {
      setSearchResults([]);
      setShowDropdown(false);
      return;
    }

    const delayDebounce = setTimeout(async () => {
      setIsSearching(true);
      try {
        const res = await fetch(`${API_URL}/assets/search?q=${encodeURIComponent(searchQuery)}`, { credentials: 'include', cache: 'no-store' });
        if (res.ok) {
          const data = await res.json();
          setSearchResults(data || []);
          setShowDropdown(true);
        }
      } catch (e) {
        console.error('Erro na busca de ativos:', e);
      } finally {
        setIsSearching(false);
      }
    }, 350);

    return () => clearTimeout(delayDebounce);
  }, [searchQuery]);

  // Seleciona ativo na lista de busca
  const handleSelectAsset = async (symbol: string) => {
    setTxTicker(symbol);
    setSearchQuery(symbol);
    setShowDropdown(false);

    // Consulta cotação em tempo real para obter a moeda padrão do ativo (ex: USD para AAPL)
    try {
      const res = await fetch(`${API_URL}/quotes/${encodeURIComponent(symbol)}`, { credentials: 'include', cache: 'no-store' });
      if (res.ok) {
        const quote = await res.json();
        setSelectedAssetCurrency(quote.currency || 'BRL');
        if (quote.currency === 'USD') {
          // Busca cotação atual do USD para ajudar o usuário com exchange_rate sugerido
          const rateRes = await fetch(`${API_URL}/quotes/USDBRL=X`, { credentials: 'include', cache: 'no-store' });
          if (rateRes.ok) {
            const rateQuote = await rateRes.json();
            setTxExchangeRate(rateQuote.price || 5.25);
          } else {
            setTxExchangeRate(5.25);
          }
        } else {
          setTxExchangeRate(1.0);
        }
      }
    } catch (e) {
      console.error(e);
      setSelectedAssetCurrency('BRL');
      setTxExchangeRate(1.0);
    }
  };

  // 4. SUBMETE CRIAÇÃO DE CARTEIRA
  const handleCreatePortfolio = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPortfolioName.trim()) return;
    setIsCreatingPortfolio(true);
    try {
      const res = await fetch(`${API_URL}/portfolios`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newPortfolioName, base_currency: newPortfolioCurrency }),
        credentials: 'include', cache: 'no-store',
      });
      if (res.ok) {
        const data = await res.json();
        setNewPortfolioName('');
        setShowPortfolioModal(false);
        await loadPortfolios(data.id);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsCreatingPortfolio(false);
    }
  };

  // 5. SUBMETE EXCLUSÃO DE CARTEIRA ATIVA
  const handleDeletePortfolio = async () => {
    if (portfolios.length <= 1) {
      alert('Você precisa manter pelo menos uma carteira ativa no sistema.');
      return;
    }
    if (!confirm('Deseja realmente apagar esta carteira? Todas as transações históricas serão excluídas definitivamente.')) return;
    try {
      const res = await fetch(`${API_URL}/portfolios/${activePortfolioId}`, {
        method: 'DELETE',
        credentials: 'include', cache: 'no-store',
      });
      if (res.ok) {
        await loadPortfolios();
      }
    } catch (e) {
      console.error(e);
    }
  };

  // 6. SUBMETE INSERÇÃO DE TRANSAÇÃO
  const handleAddTransaction = async (e: React.FormEvent) => {
    e.preventDefault();
    const parsedQty = parseFloat(txQuantity.toString());
    const parsedPrice = parseFloat(txUnitPrice.toString());
    const parsedRate = parseFloat(txExchangeRate.toString());

    if (!txTicker || isNaN(parsedQty) || parsedQty <= 0 || (txType !== 'SPLIT' && (isNaN(parsedPrice) || parsedPrice <= 0))) {
      alert('Preencha todos os campos obrigatórios corretamente.');
      return;
    }

    // Validação local de saldo para transações de venda (SELL)
    if (txType === 'SELL') {
      const activePosition = positions.find((p) => p.ticker.toUpperCase() === txTicker.toUpperCase());
      const currentQty = activePosition ? activePosition.quantity : 0;
      if (parsedQty > currentQty) {
        alert(`Saldo insuficiente de ativos. Você possui apenas ${currentQty} cotas de ${txTicker}.`);
        return;
      }
    }

    setIsAddingTx(true);
    try {
      const res = await fetch(`${API_URL}/portfolios/${activePortfolioId}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ticker: txTicker,
          type: txType,
          quantity: parsedQty,
          unit_price: txType === 'SPLIT' ? 0 : parsedPrice,
          exchange_rate: isNaN(parsedRate) || parsedRate <= 0 ? 1.0 : parsedRate,
          executed_at: txExecutedAt,
        }),
        credentials: 'include', cache: 'no-store',
      });

      if (res.ok) {
        // Limpa inputs
        setTxTicker('');
        setSearchQuery('');
        setTxQuantity('');
        setTxUnitPrice('');
        setTxExchangeRate(1.0);
        setSelectedAssetCurrency('BRL');
        setShowTxModal(false);

        // Recarrega todos os dados de forma consolidada
        await loadPortfolioDetails(activePortfolioId);
        await loadPerformance(activePortfolioId, period);
      } else {
        const err = await res.json();
        alert(err.error || 'Erro ao cadastrar transação.');
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsAddingTx(false);
    }
  };

  // 7. REMOVE UMA TRANSAÇÃO CADASTRADA
  const handleDeleteTransaction = async (txId: string) => {
    if (!confirm('Deseja realmente excluir esta transação? A rentabilidade histórica e a média da carteira serão recalculadas automaticamente.')) return;
    try {
      const res = await fetch(`${API_URL}/portfolios/${activePortfolioId}/transactions/${txId}`, {
        method: 'DELETE',
        credentials: 'include', cache: 'no-store',
      });
      if (res.ok) {
        await loadPortfolioDetails(activePortfolioId);
        await loadPerformance(activePortfolioId, period);
      }
    } catch (e) {
      console.error(e);
    }
  };

  // Formatadores Auxiliares
  const getActivePortfolio = () => portfolios.find((p) => p.id === activePortfolioId);

  const formatMoney = (val: number, currency: string) => {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: currency || 'BRL',
    }).format(val);
  };

  const formatPercentage = (val: number) => {
    const isPos = val >= 0;
    return `${isPos ? '+' : ''}${val.toFixed(2)}%`;
  };

  if (authLoading || isLoadingPortfolios) {
    return (
      <main className="container">
        <div className="glass-panel">
          <span className="loading-spinner" style={{ borderTopColor: '#00f2fe', width: 40, height: 40 }}></span>
          <p style={{ marginTop: '1.5rem', color: 'var(--text-secondary)' }}>Carregando dados financeiros seguros...</p>
        </div>
      </main>
    );
  }

  if (!user) return null;

  const activeP = getActivePortfolio();
  
  // Cálculos Consolidados de Resumo Financeiro (KPI Cards)
  const totalCost = positions.reduce((acc, pos) => acc + pos.total_cost, 0);
  const currentValue = positions.reduce((acc, pos) => acc + (pos.current_value || 0), 0);
  const profitLoss = currentValue - totalCost;
  const returnPercent = totalCost > 0 ? (profitLoss / totalCost) * 100 : 0.0;
  const kpiCurrency = activeP ? activeP.base_currency : 'BRL';

  return (
    <main className="container" style={{ maxWidth: 1100 }}>
      {/* Header Centralizado com Navbar Tabs */}
      <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', borderBottom: '1px solid var(--panel-border)', paddingBottom: '1.25rem', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '2.3rem', background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', margin: 0, fontWeight: 800 }}>StockPulse</h1>
          
          {/* Navegação entre telas do Dashboard */}
          <div style={{ display: 'flex', gap: '1.5rem', marginTop: '0.8rem' }}>
            <a href="/dashboard" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              📊 Monitoramento
            </a>
            <a href="/dashboard/portfolio" style={{ color: 'var(--accent-color)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 700, borderBottom: '2px solid var(--accent-color)', paddingBottom: '3px', display: 'flex', alignItems: 'center', gap: '5px' }}>
              💼 Minha Carteira
            </a>
          </div>
        </div>
        
        <div style={{ display: 'flex', alignItems: 'center', gap: '1.25rem' }}>
          <div style={{ textAlign: 'right', fontSize: '0.8rem' }}>
            <span style={{ display: 'block', fontWeight: 600 }}>{user.name}</span>
            <span style={{ color: 'var(--text-secondary)', fontSize: '0.7rem' }}>Sessão Segura</span>
          </div>
          <button className="primary-button" onClick={logout} style={{ padding: '0.5rem 1.25rem', fontSize: '0.85rem' }}>
            Sair
          </button>
        </div>
      </div>

      {/* Seletor Lateral e Superior de Múltiplas Carteiras */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem', flexWrap: 'wrap', gap: '1rem' }}>
        {/* Abas de Portfólio */}
        <div style={{ display: 'flex', gap: '0.5rem', overflowX: 'auto', paddingBottom: '0.2rem' }}>
          {portfolios.map((p) => (
            <button
              key={p.id}
              onClick={() => setActivePortfolioId(p.id)}
              style={{
                padding: '0.5rem 1.1rem',
                fontSize: '0.85rem',
                borderRadius: '8px',
                border: '1px solid',
                borderColor: activePortfolioId === p.id ? 'var(--accent-color)' : 'var(--panel-border)',
                background: activePortfolioId === p.id ? 'rgba(0, 242, 254, 0.08)' : 'transparent',
                color: activePortfolioId === p.id ? 'var(--accent-color)' : 'var(--text-secondary)',
                cursor: 'pointer',
                fontWeight: activePortfolioId === p.id ? 700 : 600,
                whiteSpace: 'nowrap',
                transition: 'all 0.15s ease',
              }}
            >
              💼 {p.name} <span style={{ fontSize: '0.65rem', opacity: 0.65, marginLeft: '3px' }}>({p.base_currency})</span>
            </button>
          ))}
          <button
            onClick={() => setShowPortfolioModal(true)}
            style={{
              padding: '0.5rem 1rem',
              fontSize: '0.85rem',
              borderRadius: '8px',
              border: '1px dashed var(--accent-color)',
              background: 'transparent',
              color: 'var(--accent-color)',
              cursor: 'pointer',
              fontWeight: 600,
              whiteSpace: 'nowrap',
            }}
          >
            + Criar Carteira
          </button>
        </div>

        {/* Botão de Excluir Portfólio */}
        {activeP && portfolios.length > 1 && (
          <button
            onClick={handleDeletePortfolio}
            style={{
              background: 'none',
              border: 'none',
              color: '#ff4a5a',
              cursor: 'pointer',
              fontSize: '0.85rem',
              fontWeight: 600,
              padding: '0.5rem',
            }}
            title="Excluir carteira atual"
          >
            🗑️ Excluir Carteira
          </button>
        )}
      </div>

      {isLoadingDetails ? (
        <div className="glass-panel" style={{ minHeight: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 35, height: 35 }}></span>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
          
          {/* KPI Dashboard Grid */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(230px, 1fr))', gap: '1.25rem' }}>
            {/* Card 1: Patrimônio */}
            <div className="glass-panel" style={{ padding: '1.5rem', textAlign: 'left', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase', fontWeight: 600, letterSpacing: '0.05em' }}>
                Patrimônio Atual
              </span>
              <span style={{ fontSize: '1.8rem', fontWeight: 800, color: '#fff', marginTop: '0.4rem', letterSpacing: '-0.02em' }}>
                {formatMoney(currentValue, kpiCurrency)}
              </span>
            </div>
            
            {/* Card 2: Aplicado */}
            <div className="glass-panel" style={{ padding: '1.5rem', textAlign: 'left', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase', fontWeight: 600, letterSpacing: '0.05em' }}>
                Total Investido
              </span>
              <span style={{ fontSize: '1.8rem', fontWeight: 800, color: '#fff', marginTop: '0.4rem', letterSpacing: '-0.02em' }}>
                {formatMoney(totalCost, kpiCurrency)}
              </span>
            </div>

            {/* Card 3: Lucro nominal */}
            <div className="glass-panel" style={{ padding: '1.5rem', textAlign: 'left', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase', fontWeight: 600, letterSpacing: '0.05em' }}>
                Lucro / Prejuízo
              </span>
              <span style={{
                fontSize: '1.8rem',
                fontWeight: 800,
                color: profitLoss >= 0 ? '#00e676' : '#ff3d00',
                marginTop: '0.4rem',
                letterSpacing: '-0.02em'
              }}>
                {profitLoss >= 0 ? '▲' : '▼'} {formatMoney(profitLoss, kpiCurrency)}
              </span>
            </div>

            {/* Card 4: Rentabilidade percentual */}
            <div className="glass-panel" style={{ padding: '1.5rem', textAlign: 'left', display: 'flex', flexDirection: 'column', justifyContent: 'center', position: 'relative' }}>
              <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase', fontWeight: 600, letterSpacing: '0.05em' }}>
                Rentabilidade
              </span>
              <span style={{
                fontSize: '1.8rem',
                fontWeight: 800,
                color: profitLoss >= 0 ? '#00e676' : '#ff3d00',
                marginTop: '0.4rem',
                letterSpacing: '-0.02em'
              }}>
                {formatPercentage(returnPercent)}
              </span>
              {totalCost > 0 && profitLoss > 0 && (
                <span className="pulse-dot" style={{ position: 'absolute', top: '15px', right: '15px', width: '8px', height: '8px', background: '#00e676', borderRadius: '50%' }}></span>
              )}
            </div>
          </div>

          {/* Gráfico de Evolução Patrimonial */}
          <div className="glass-panel" style={{ padding: '1.75rem 2rem', textAlign: 'left', display: 'flex', flexDirection: 'column', minHeight: '380px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.25rem', flexWrap: 'wrap', gap: '0.8rem' }}>
              <div>
                <h3 style={{ margin: 0, fontSize: '1.05rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  📈 Evolução da Carteira
                </h3>
                <p style={{ margin: '0.1rem 0 0 0', fontSize: '0.75rem', color: 'var(--text-secondary)' }}>Valores ponderados na moeda base ({kpiCurrency})</p>
              </div>
              
              {/* Seletores de Período */}
              <div style={{ display: 'flex', gap: '0.35rem', background: 'rgba(255,255,255,0.02)', padding: '0.2rem', borderRadius: '6px', border: '1px solid var(--panel-border)' }}>
                {['1M', '3M', '6M', '1Y', 'ALL'].map((p) => (
                  <button
                    key={p}
                    onClick={() => setPeriod(p)}
                    style={{
                      padding: '0.25rem 0.65rem',
                      fontSize: '0.7rem',
                      borderRadius: '4px',
                      border: 'none',
                      background: period === p ? 'var(--accent-gradient)' : 'transparent',
                      color: period === p ? '#000' : 'var(--text-secondary)',
                      cursor: 'pointer',
                      fontWeight: 700,
                      transition: 'all 0.15s ease',
                    }}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>

            {isLoadingPerformance ? (
              <div style={{ height: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center', width: '100%' }}>
                <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 30, height: 30 }}></span>
              </div>
            ) : performanceData.length > 0 ? (
              <PortfolioChart data={performanceData} />
            ) : (
              <div style={{ height: '300px', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', width: '100%', color: 'var(--text-secondary)', border: '1px dashed var(--panel-border)', borderRadius: '12px' }}>
                <span style={{ fontSize: '1.7rem', marginBottom: '0.5rem' }}>💼</span>
                <p style={{ margin: 0, fontSize: '0.85rem' }}>Cadastre a sua primeira transação abaixo para começar a visualizar o histórico de rentabilidade.</p>
              </div>
            )}
          </div>

          {/* Seção Dupla de Posições e Histórico */}
          <div style={{ display: 'flex', gap: '2rem', flexFlow: 'row wrap', alignItems: 'stretch' }}>
            
            {/* Posições Ativas (Esquerda) */}
            <div style={{ flex: '2 1 600px', display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              <div className="glass-panel" style={{ padding: '1.75rem 2rem', textAlign: 'left', minHeight: '380px', display: 'flex', flexDirection: 'column' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.25rem' }}>
                  <h3 style={{ margin: 0, fontSize: '1.05rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                    📦 Posições Ativas
                  </h3>
                  <button
                    className="primary-button"
                    onClick={() => setShowTxModal(true)}
                    style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}
                  >
                    + Lançar Operação
                  </button>
                </div>

                <div style={{ overflowX: 'auto', flex: 1 }}>
                  {positions.length > 0 ? (
                    <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '0.85rem' }}>
                      <thead>
                        <tr style={{ borderBottom: '1px solid var(--panel-border)', color: 'var(--text-secondary)' }}>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600 }}>Ativo</th>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600, textAlign: 'right' }}>Qtd</th>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600, textAlign: 'right' }}>Preço Médio</th>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600, textAlign: 'right' }}>Cotação Atual</th>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600, textAlign: 'right' }}>Custo Total</th>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600, textAlign: 'right' }}>Valor Atual</th>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600, textAlign: 'right' }}>Retorno</th>
                          <th style={{ padding: '0.75rem 0.5rem', fontWeight: 600, textAlign: 'center' }}>Valuation</th>
                        </tr>
                      </thead>
                      <tbody>
                        {positions.map((pos) => {
                          const isPos = (pos.profit_loss || 0) >= 0;
                          return (
                            <tr key={pos.asset_id} style={{ borderBottom: '1px solid rgba(255,255,255,0.02)', verticalAlign: 'middle' }}>
                              <td style={{ padding: '0.9rem 0.5rem' }}>
                                <span style={{ display: 'block', fontWeight: 700, color: 'var(--accent-color)' }}>{pos.ticker}</span>
                                <span style={{ display: 'block', fontSize: '0.7rem', color: 'var(--text-secondary)', maxWidth: '140px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                  {pos.name}
                                </span>
                              </td>
                              <td style={{ padding: '0.9rem 0.5rem', textAlign: 'right', fontWeight: 600, fontFamily: 'monospace' }}>
                                {pos.quantity}
                              </td>
                              <td style={{ padding: '0.9rem 0.5rem', textAlign: 'right', fontFamily: 'monospace' }}>
                                {formatMoney(pos.average_price, pos.currency)}
                              </td>
                              <td style={{ padding: '0.9rem 0.5rem', textAlign: 'right', fontFamily: 'monospace', fontWeight: 600 }}>
                                {pos.current_price ? formatMoney(pos.current_price, pos.currency) : '--'}
                              </td>
                              <td style={{ padding: '0.9rem 0.5rem', textAlign: 'right', fontFamily: 'monospace' }}>
                                {formatMoney(pos.total_cost, kpiCurrency)}
                              </td>
                              <td style={{ padding: '0.9rem 0.5rem', textAlign: 'right', fontFamily: 'monospace', fontWeight: 700, color: '#fff' }}>
                                {pos.current_value ? formatMoney(pos.current_value, kpiCurrency) : '--'}
                              </td>
                              <td style={{ padding: '0.9rem 0.5rem', textAlign: 'right', fontWeight: 700, color: isPos ? '#00e676' : '#ff3d00' }}>
                                {pos.return_percent !== undefined ? formatPercentage(pos.return_percent) : '--'}
                              </td>
                              <td style={{ padding: '0.9rem 0.5rem', textAlign: 'center' }}>
                                {pos.graham_value && pos.current_price ? (
                                  <span style={{
                                    display: 'inline-block',
                                    padding: '0.2rem 0.5rem',
                                    borderRadius: '4px',
                                    fontSize: '0.7rem',
                                    fontWeight: 700,
                                    backgroundColor: pos.current_price < pos.graham_value ? 'rgba(0, 230, 118, 0.1)' : 'rgba(255, 61, 0, 0.1)',
                                    color: pos.current_price < pos.graham_value ? '#00e676' : '#ff3d00',
                                    border: `1px solid ${pos.current_price < pos.graham_value ? 'rgba(0, 230, 118, 0.3)' : 'rgba(255, 61, 0, 0.3)'}`
                                  }} title={`Preço Justo (Graham): ${formatMoney(pos.graham_value, pos.currency)}`}>
                                    {pos.current_price < pos.graham_value ? 'DESCONTADA' : 'CARA'}
                                  </span>
                                ) : (
                                  <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>--</span>
                                )}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '240px', color: 'var(--text-secondary)' }}>
                      <span style={{ fontSize: '1.5rem', marginBottom: '0.5rem' }}>📁</span>
                      <p style={{ margin: 0, fontSize: '0.85rem' }}>Esta carteira ainda não possui ativos ativos.</p>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Histórico de Operações (Direita) */}
            <div style={{ flex: '1 1 350px', display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              <div className="glass-panel" style={{ padding: '1.75rem 1.5rem', textAlign: 'left', minHeight: '380px', display: 'flex', flexDirection: 'column' }}>
                <h3 style={{ margin: '0 0 1.25rem 0', fontSize: '1.05rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  📜 Últimas Operações
                </h3>
                
                <div style={{ overflowY: 'auto', flex: 1, display: 'flex', flexDirection: 'column', gap: '0.75rem', maxHeight: '310px' }}>
                  {transactions.length > 0 ? (
                    transactions.map((tx) => {
                      const isBuy = tx.type === 'BUY';
                      const isSplit = tx.type === 'SPLIT';
                      return (
                        <div
                          key={tx.id}
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            padding: '0.75rem',
                            background: 'rgba(255,255,255,0.015)',
                            border: '1px solid var(--panel-border)',
                            borderRadius: '8px',
                            fontSize: '0.8rem',
                          }}
                        >
                          <div>
                            <span style={{
                              display: 'inline-block',
                              padding: '0.15rem 0.4rem',
                              borderRadius: '4px',
                              fontSize: '0.65rem',
                              fontWeight: 700,
                              background: isBuy ? 'rgba(0, 230, 118, 0.08)' : isSplit ? 'rgba(0, 242, 254, 0.08)' : 'rgba(255, 61, 0, 0.08)',
                              color: isBuy ? '#00e676' : isSplit ? '#00f2fe' : '#ff3d00',
                              marginRight: '0.5rem',
                            }}>
                              {isBuy ? 'COMPRA' : isSplit ? 'SPLIT' : 'VENDA'}
                            </span>
                            <span style={{ fontWeight: 700, color: '#fff' }}>{tx.ticker}</span>
                            <span style={{ display: 'block', fontSize: '0.65rem', color: 'var(--text-secondary)', marginTop: '0.2rem' }}>
                              {new Date(tx.executed_at).toISOString().split('T')[0].replace(/-/g, '/')}
                            </span>
                          </div>
                          
                          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                            <div style={{ textAlign: 'right' }}>
                              <span style={{ display: 'block', fontWeight: 700 }}>
                                {isSplit ? `Fator: ${tx.quantity}x` : `${tx.quantity} un.`}
                              </span>
                              {!isSplit && (
                                <span style={{ display: 'block', fontSize: '0.7rem', color: 'var(--text-secondary)' }}>
                                  {formatMoney(tx.unit_price, tx.currency || 'BRL')}
                                </span>
                              )}
                            </div>
                            
                            <button
                              onClick={() => handleDeleteTransaction(tx.id)}
                              style={{
                                background: 'none',
                                border: 'none',
                                color: 'rgba(255,255,255,0.2)',
                                cursor: 'pointer',
                                fontSize: '0.95rem',
                                padding: '0.25rem',
                              }}
                              onMouseEnter={(e) => e.currentTarget.style.color = '#ff4a5a'}
                              onMouseLeave={(e) => e.currentTarget.style.color = 'rgba(255,255,255,0.2)'}
                              title="Remover operação"
                            >
                              ✕
                            </button>
                          </div>
                        </div>
                      );
                    })
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '200px', color: 'var(--text-secondary)', border: '1px dashed var(--panel-border)', borderRadius: '8px' }}>
                      <span style={{ fontSize: '1.25rem', marginBottom: '0.5rem' }}>📝</span>
                      <p style={{ margin: 0, fontSize: '0.75rem', textAlign: 'center' }}>Sem transações registradas nesta carteira.</p>
                    </div>
                  )}
                </div>
              </div>
            </div>

          </div>

        </div>
      )}

      {/* MODAL 1: CRIAÇÃO DE CARTEIRA */}
      {showPortfolioModal && (
        <div style={{
          position: 'fixed', top: 0, left: 0, width: '100%', height: '100%',
          background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100
        }}>
          <div className="glass-panel" style={{ maxWidth: '400px', width: '90%', padding: '2rem', textAlign: 'left' }}>
            <h3 style={{ fontSize: '1.4rem', fontWeight: 800, background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', marginBottom: '1.5rem' }}>
              💼 Nova Carteira
            </h3>
            
            <form onSubmit={handleCreatePortfolio} style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
              <div className="form-group">
                <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                  Nome da Carteira
                </label>
                <input
                  className="form-input"
                  type="text"
                  value={newPortfolioName}
                  onChange={(e) => setNewPortfolioName(e.target.value)}
                  placeholder="Ex: Minha Aposentadoria, Ações B3..."
                  required
                  disabled={isCreatingPortfolio}
                />
              </div>

              <div className="form-group">
                <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                  Moeda Base
                </label>
                <select
                  value={newPortfolioCurrency}
                  onChange={(e) => setNewPortfolioCurrency(e.target.value)}
                  disabled={isCreatingPortfolio}
                  style={{
                    width: '100%',
                    padding: '0.75rem 1rem',
                    background: 'rgba(255,255,255,0.03)',
                    border: '1px solid var(--panel-border)',
                    borderRadius: '8px',
                    color: '#fff',
                    fontSize: '0.95rem',
                    outline: 'none',
                  }}
                >
                  <option value="BRL" style={{ background: '#1c1f24' }}>BRL (R$) - Real Brasileiro</option>
                  <option value="USD" style={{ background: '#1c1f24' }}>USD ($) - Dólar Americano</option>
                </select>
              </div>

              <div style={{ display: 'flex', gap: '0.75rem', marginTop: '0.5rem' }}>
                <button
                  type="button"
                  onClick={() => setShowPortfolioModal(false)}
                  style={{ flex: 1, padding: '0.75rem', background: 'transparent', border: '1px solid var(--panel-border)', color: 'var(--text-secondary)', borderRadius: '8px', cursor: 'pointer', fontWeight: 600 }}
                >
                  Cancelar
                </button>
                <button
                  className="primary-button"
                  type="submit"
                  disabled={isCreatingPortfolio}
                  style={{ flex: 1, padding: '0.75rem' }}
                >
                  {isCreatingPortfolio ? 'Criando...' : 'Salvar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 2: CADASTRO DE TRANSAÇÃO (Lançar Operação) */}
      {showTxModal && (
        <div style={{
          position: 'fixed', top: 0, left: 0, width: '100%', height: '100%',
          background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100
        }}>
          <div className="glass-panel" style={{ maxWidth: '460px', width: '90%', padding: '2rem', textAlign: 'left', maxHeight: '90vh', overflowY: 'auto' }}>
            <h3 style={{ fontSize: '1.4rem', fontWeight: 800, background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', marginBottom: '1.5rem' }}>
              📝 Registrar Operação
            </h3>
            
            <form onSubmit={handleAddTransaction} style={{ display: 'flex', flexDirection: 'column', gap: '1.1rem' }}>
              
              {/* Autocomplete Asset Search */}
              <div className="form-group" style={{ position: 'relative' }}>
                <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                  Ativo / Ticker
                </label>
                <input
                  className="form-input"
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Pesquise o ticker (Ex: PETR4, AAPL, IVV)..."
                  required
                  disabled={isAddingTx}
                  autoComplete="off"
                />
                {isSearching && (
                  <span className="loading-spinner" style={{ position: 'absolute', right: '15px', top: '55%', borderTopColor: 'var(--accent-color)' }}></span>
                )}

                {/* Suggestions List */}
                {showDropdown && searchResults.length > 0 && (
                  <div className="glass-panel" style={{
                    position: 'absolute', top: '100%', left: 0, width: '100%',
                    marginTop: '0.4rem', zIndex: 101, padding: '0.4rem',
                    textAlign: 'left', maxHeight: '200px', overflowY: 'auto',
                    boxShadow: '0 16px 40px rgba(0,0,0,0.7)', border: '1px solid var(--panel-border)'
                  }}>
                    {searchResults.map((item) => (
                      <div
                        key={item.symbol}
                        onClick={() => handleSelectAsset(item.symbol)}
                        style={{ padding: '0.55rem 0.8rem', borderRadius: '6px', cursor: 'pointer', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                        onMouseEnter={(e) => e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.04)'}
                        onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
                      >
                        <div>
                          <span style={{ fontWeight: 700, color: 'var(--accent-color)', marginRight: '0.6rem' }}>{item.symbol}</span>
                          <span style={{ fontSize: '0.8rem', opacity: 0.85 }}>{item.name}</span>
                        </div>
                        <span style={{ fontSize: '0.6rem', padding: '0.15rem 0.35rem', background: 'rgba(0, 242, 254, 0.08)', color: 'var(--accent-color)', borderRadius: '3px', textTransform: 'uppercase' }}>
                          {item.exchange}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Tipo de Operação (BUY/SELL) */}
              <div className="form-group">
                <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                  Operação
                </label>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <button
                    type="button"
                    onClick={() => setTxType('BUY')}
                    disabled={isAddingTx}
                    style={{
                      flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', fontWeight: 700, fontSize: '0.8rem',
                      border: txType === 'BUY' ? '1px solid #00e676' : '1px solid var(--panel-border)',
                      background: txType === 'BUY' ? 'rgba(0, 230, 118, 0.08)' : 'transparent',
                      color: txType === 'BUY' ? '#00e676' : 'var(--text-secondary)',
                      transition: 'all 0.15s ease'
                    }}
                  >
                    🟢 COMPRA
                  </button>
                  <button
                    type="button"
                    onClick={() => setTxType('SELL')}
                    disabled={isAddingTx}
                    style={{
                      flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', fontWeight: 700, fontSize: '0.8rem',
                      border: txType === 'SELL' ? '1px solid #ff3d00' : '1px solid var(--panel-border)',
                      background: txType === 'SELL' ? 'rgba(255, 61, 0, 0.08)' : 'transparent',
                      color: txType === 'SELL' ? '#ff3d00' : 'var(--text-secondary)',
                      transition: 'all 0.15s ease'
                    }}
                  >
                    🔴 VENDA
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setTxType('SPLIT');
                      setTxUnitPrice(0);
                    }}
                    disabled={isAddingTx}
                    style={{
                      flex: 1, padding: '0.6rem', borderRadius: '6px', cursor: 'pointer', fontWeight: 700, fontSize: '0.8rem',
                      border: txType === 'SPLIT' ? '1px solid #00f2fe' : '1px solid var(--panel-border)',
                      background: txType === 'SPLIT' ? 'rgba(0, 242, 254, 0.08)' : 'transparent',
                      color: txType === 'SPLIT' ? '#00f2fe' : 'var(--text-secondary)',
                      transition: 'all 0.15s ease'
                    }}
                  >
                    ✂️ SPLIT
                  </button>
                </div>
              </div>

              {/* Quantidade & Preço Unitário */}
              <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                    {txType === 'SPLIT' ? 'Fator / Multiplicador' : 'Quantidade'}
                  </label>
                  <input
                    className="form-input"
                    type="number"
                    step="any"
                    value={txQuantity}
                    onChange={(e) => setTxQuantity(e.target.value)}
                    placeholder={txType === 'SPLIT' ? "Ex: 10 (desdobra) ou 0.1 (agrupa)" : "0"}
                    required
                    disabled={isAddingTx}
                  />
                  {txType === 'SPLIT' && (
                    <span style={{ fontSize: '0.65rem', color: 'var(--text-secondary)', marginTop: '0.4rem', display: 'block' }}>
                      Ex: Desdobramento 1 para 10 = Fator 10. Agrupamento 10 para 1 = Fator 0.1.
                    </span>
                  )}
                </div>

                {txType !== 'SPLIT' && (
                  <div className="form-group" style={{ flex: 1 }}>
                    <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                      Preço Unitário ({selectedAssetCurrency})
                    </label>
                    <input
                      className="form-input"
                      type="number"
                      step="any"
                      value={txUnitPrice}
                      onChange={(e) => setTxUnitPrice(e.target.value)}
                      placeholder="0.00"
                      required
                      disabled={isAddingTx}
                    />
                  </div>
                )}
              </div>

              {/* Taxa de Câmbio (Mostrada dinamicamente apenas se o ativo for USD e a carteira BRL) */}
              {selectedAssetCurrency === 'USD' && kpiCurrency === 'BRL' && (
                <div className="form-group">
                  <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: '#ffc107', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                    Taxa Cambial USDBRL de Aquisição
                  </label>
                  <input
                    className="form-input"
                    type="number"
                    step="any"
                    value={txExchangeRate}
                    onChange={(e) => setTxExchangeRate(e.target.value)}
                    placeholder="Ex: 5.2500"
                    required
                    disabled={isAddingTx}
                    style={{ borderColor: 'rgba(255, 193, 7, 0.4)' }}
                  />
                  <span style={{ fontSize: '0.65rem', color: 'var(--text-secondary)', marginTop: '0.2rem', display: 'block' }}>
                    Sugerido com base no fechamento cambial recente. Ajuste se comprou com outra taxa cambial.
                  </span>
                </div>
              )}

              {/* Data da Operação */}
              <div className="form-group">
                <label className="form-label" style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase' }}>
                  Data de Execução
                </label>
                <input
                  className="form-input"
                  type="date"
                  value={txExecutedAt}
                  onChange={(e) => setTxExecutedAt(e.target.value)}
                  required
                  disabled={isAddingTx}
                />
              </div>

              {/* Ações */}
              <div style={{ display: 'flex', gap: '0.75rem', marginTop: '0.5rem' }}>
                <button
                  type="button"
                  onClick={() => setShowTxModal(false)}
                  style={{ flex: 1, padding: '0.75rem', background: 'transparent', border: '1px solid var(--panel-border)', color: 'var(--text-secondary)', borderRadius: '8px', cursor: 'pointer', fontWeight: 600 }}
                >
                  Cancelar
                </button>
                <button
                  className="primary-button"
                  type="submit"
                  disabled={isAddingTx}
                  style={{ flex: 1, padding: '0.75rem' }}
                >
                  {isAddingTx ? 'Registrando...' : 'Lançar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

    </main>
  );
}
