'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/context/AuthContext';
import { createChart, ColorType } from 'lightweight-charts';
import Link from 'next/link';

interface Item {
  id: string;
  watchlist_id: string;
  asset_id: string;
  ticker: string;
  name: string;
  type: string;
  currency: string;
  added_at: string;
  price?: number;
  change?: number;
  change_percent?: number;
  graham_value?: number;
  bazin_value?: number;
}

interface Watchlist {
  id: string;
  user_id: string;
  name: string;
  created_at: string;
  items?: Item[];
}

interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
  type: string;
}

interface Quote {
  symbol: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
  high: number;
  low: number;
  volume: number;
  currency: string;
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export default function DashboardPage() {
  const { user, logout, isLoading: authLoading } = useAuth();
  
  // Busca e Resultados
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);

  // Cotação Ativa
  const [activeQuote, setActiveQuote] = useState<Quote | null>(null);
  const [isLoadingQuote, setIsLoadingQuote] = useState(false);
  const [quoteError, setQuoteError] = useState<string | null>(null);
  
  // Controle de Watchlists (Múltiplas Listas)
  const [watchlists, setWatchlists] = useState<Watchlist[]>([]);
  const [activeWatchlistId, setActiveWatchlistId] = useState<string>('');
  const [newWatchlistName, setNewWatchlistName] = useState('');
  const [isCreatingList, setIsCreatingList] = useState(false);
  const [isAddingToWatchlist, setIsAddingToWatchlist] = useState(false);

  // Badges de Status (Redis / Yahoo)
  const [cacheStatus, setCacheStatus] = useState<'hit' | 'miss' | 'updating' | null>(null);

  // WebSocket em tempo real e flashes (Fase 3)
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [wsConnected, setWsConnected] = useState(false);
  const [priceFlashing, setPriceFlashing] = useState<Record<string, 'up' | 'down'>>({});

  // Modal de Alertas (Fase 3)
  const [showAlertModal, setShowAlertModal] = useState(false);
  const [alertTargetPrice, setAlertTargetPrice] = useState('');
  const [alertCondition, setAlertCondition] = useState<'ABOVE' | 'BELOW'>('ABOVE');
  const [isCreatingAlert, setIsCreatingAlert] = useState(false);
  const [alertErrorMsg, setAlertErrorMsg] = useState<string | null>(null);
  const [alertSuccessMsg, setAlertSuccessMsg] = useState<string | null>(null);

  const getActiveWatchlist = useCallback(() => watchlists.find((w) => w.id === activeWatchlistId), [watchlists, activeWatchlistId]);
  const activeWL = getActiveWatchlist();

  const openAlertModal = () => {
    if (!activeQuote) return;
    setAlertTargetPrice(activeQuote.price.toString());
    setAlertCondition('ABOVE');
    setAlertErrorMsg(null);
    setAlertSuccessMsg(null);
    setShowAlertModal(true);
  };

  const handleCreateAlertSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeQuote || !alertTargetPrice.trim()) return;
    
    setIsCreatingAlert(true);
    setAlertErrorMsg(null);
    setAlertSuccessMsg(null);

    try {
      const res = await fetch(`${API_URL}/alerts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ticker: activeQuote.symbol,
          target_price: parseFloat(alertTargetPrice),
          condition: alertCondition,
        }),
        credentials: 'include',
      });
      
      const data = await res.json();
      if (res.ok) {
        setAlertSuccessMsg(`Alerta configurado com sucesso! Enviaremos um e-mail quando o preço ficar ${alertCondition === 'ABOVE' ? 'acima de' : 'abaixo de'} ${formatMoney(parseFloat(alertTargetPrice), activeQuote.currency)}.`);
        setTimeout(() => {
          setShowAlertModal(false);
          setAlertSuccessMsg(null);
        }, 3000);
      } else {
        setAlertErrorMsg(data.error || 'Erro ao criar o alerta.');
      }
    } catch (e) {
      setAlertErrorMsg('Falha ao se conectar com o servidor.');
    } finally {
      setIsCreatingAlert(false);
    }
  };

  // 1. CARREGA TODAS AS WATCHLISTS DO USUÁRIO
  const loadWatchlists = useCallback(async (selectId?: string) => {
    try {
      const res = await fetch(`${API_URL}/watchlists`, {
        credentials: 'include',
      });
      if (res.ok) {
        const data = await res.json();
        setWatchlists(data || []);
        if (data && data.length > 0) {
          const nextId = selectId || data[0].id;
          setActiveWatchlistId(nextId);
          loadWatchlistDetails(nextId);
        }
      }
    } catch (e) {
      console.error('Erro ao buscar favoritos:', e);
    }
  }, []);

  // 2. DETALHA A WATCHLIST ATIVA (E PEGA AS COTAÇÕES ATUALIZADAS DO REDIS/GO)
  const loadWatchlistDetails = async (id: string) => {
    try {
      const res = await fetch(`${API_URL}/watchlists/${id}`, {
        credentials: 'include',
      });
      if (res.ok) {
        const data = await res.json();
        setWatchlists((prev) => prev.map((w) => (w.id === id ? data : w)));
      }
    } catch (e) {
      console.error('Erro ao buscar detalhes da watchlist:', e);
    }
  };

  // Carrega listas ativas ao inicializar o usuário
  useEffect(() => {
    if (user) {
      loadWatchlists();
    }
  }, [user, loadWatchlists]);

  // Conexão WebSocket em Tempo Real (Fase 3)
  useEffect(() => {
    if (!user) return;

    const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // O backend roda na porta 8080 em desenvolvimento local
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || `${wsProto}//localhost:8080/api/v1/ws`;
    
    let socket: WebSocket;
    let reconnectTimeout: NodeJS.Timeout;

    const connect = () => {
      console.log('[WS] Tentando estabelecer conexão...', wsUrl);
      socket = new WebSocket(wsUrl);

      socket.onopen = () => {
        console.log('[WS] Conectado ao servidor de tempo real');
        setWsConnected(true);
        setWs(socket);
      };

      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data);
          if (payload.type === 'quote' && payload.data) {
            const updatedQuote = payload.data as Quote;
            
            // 1. Atualiza e pisca o card central se for o ativo ativo
            setActiveQuote((current) => {
              if (current && current.symbol.toUpperCase() === updatedQuote.symbol.toUpperCase()) {
                const direction = updatedQuote.price > current.price ? 'up' : updatedQuote.price < current.price ? 'down' : null;
                if (direction) {
                  setPriceFlashing((prev) => ({ ...prev, [updatedQuote.symbol]: direction }));
                  setTimeout(() => {
                    setPriceFlashing((prev) => {
                      const copy = { ...prev };
                      delete copy[updatedQuote.symbol];
                      return copy;
                    });
                  }, 1000);
                }
                return updatedQuote;
              }
              return current;
            });

            // 2. Atualiza e pisca o preço correspondente na sidebar de favoritos
            setWatchlists((prevWLs) => {
              return prevWLs.map((wl) => {
                if (wl.items) {
                  const updatedItems = wl.items.map((item) => {
                    if (item.ticker.toUpperCase() === updatedQuote.symbol.toUpperCase()) {
                      const direction = updatedQuote.price > (item.price || 0) ? 'up' : updatedQuote.price < (item.price || 0) ? 'down' : null;
                      if (direction) {
                        setPriceFlashing((prev) => ({ ...prev, [item.ticker]: direction }));
                        setTimeout(() => {
                          setPriceFlashing((prev) => {
                            const copy = { ...prev };
                            delete copy[item.ticker];
                            return copy;
                          });
                        }, 1000);
                      }
                      return {
                        ...item,
                        price: updatedQuote.price,
                        change: updatedQuote.change,
                        change_percent: updatedQuote.change_percent,
                      };
                    }
                    return item;
                  });
                  return { ...wl, items: updatedItems };
                }
                return wl;
              });
            });
          }
        } catch (e) {
          console.error('[WS] Erro ao decodificar cotação do WebSocket:', e);
        }
      };

      socket.onclose = () => {
        console.log('[WS] Conexão perdida. Tentando reconexão em 5 segundos...');
        setWsConnected(false);
        setWs(null);
        reconnectTimeout = setTimeout(connect, 5000);
      };

      socket.onerror = () => {
        socket.close();
      };
    };

    connect();

    return () => {
      if (socket) socket.close();
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
    };
  }, [user]);

  // Sincroniza as assinaturas de ativos com a lista ativa atual
  useEffect(() => {
    if (ws && wsConnected && activeWL && activeWL.items && activeWL.items.length > 0) {
      const symbols = activeWL.items.map((item) => item.ticker);
      console.log('[WS] Atualizando assinaturas para:', symbols);
      ws.send(JSON.stringify({ action: 'subscribe', symbols }));
    }
  }, [ws, wsConnected, activeWL]);


  // Efeito Debounce para busca autocomplete
  useEffect(() => {
    if (!searchQuery.trim()) {
      setSearchResults([]);
      setShowDropdown(false);
      return;
    }

    const delayDebounce = setTimeout(async () => {
      setIsSearching(true);
      try {
        const res = await fetch(`${API_URL}/assets/search?q=${encodeURIComponent(searchQuery)}`, {
          credentials: 'include',
        });
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

  // 3. CARREGA/ATUALIZA COTAÇÃO INDIVIDUAL
  const loadQuote = useCallback(async (symbol: string, isRefresh = false) => {
    setIsLoadingQuote(true);
    setQuoteError(null);
    if (isRefresh) {
      setCacheStatus('updating');
    }
    
    try {
      const res = await fetch(`${API_URL}/quotes/${encodeURIComponent(symbol)}`, {
        credentials: 'include',
      });
      
      const cacheHeader = res.headers.get('X-Cache');
      
      const data = await res.json();
      if (res.ok) {
        setActiveQuote(data);
        if (cacheHeader === 'HIT') {
          setCacheStatus('hit');
        } else {
          setCacheStatus('miss');
        }
      } else {
        setQuoteError(data.error || 'Erro ao carregar cotação.');
      }
    } catch (e) {
      setQuoteError('Falha ao se comunicar com o servidor.');
    } finally {
      setIsLoadingQuote(false);
      setTimeout(() => setCacheStatus(null), 3000);
    }
  }, []);

  // 4. CRIA UMA NOVA WATCHLIST
  const handleCreateWatchlist = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWatchlistName.trim()) return;
    setIsCreatingList(true);
    try {
      const res = await fetch(`${API_URL}/watchlists`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newWatchlistName }),
        credentials: 'include',
      });
      if (res.ok) {
        const data = await res.json();
        setNewWatchlistName('');
        await loadWatchlists(data.id);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsCreatingList(false);
    }
  };

  // 5. EXCLUI A WATCHLIST ATIVA
  const handleDeleteActiveWatchlist = async () => {
    if (watchlists.length <= 1) {
      alert('Você precisa manter pelo menos uma lista de favoritos.');
      return;
    }
    if (!confirm('Deseja realmente apagar esta lista de favoritos e todos os seus itens?')) return;
    try {
      const res = await fetch(`${API_URL}/watchlists/${activeWatchlistId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        await loadWatchlists();
      }
    } catch (e) {
      console.error(e);
    }
  };

  // 6. ADICIONA / REMOVE DOS FAVORITOS (ÍCONE DE ESTRELA)
  const isAssetFavorited = (symbol: string) => {
    return activeWL?.items?.some((item) => item.ticker.toUpperCase() === symbol.toUpperCase()) || false;
  };

  const handleToggleFavorite = async () => {
    if (!activeQuote || !activeWatchlistId) return;
    const symbol = activeQuote.symbol;
    const favorited = isAssetFavorited(symbol);
    setIsAddingToWatchlist(true);

    try {
      if (favorited) {
        // Remove dos favoritos
        const res = await fetch(`${API_URL}/watchlists/${activeWatchlistId}/items/${encodeURIComponent(symbol)}`, {
          method: 'DELETE',
          credentials: 'include',
        });
        if (res.ok) {
          await loadWatchlistDetails(activeWatchlistId);
        }
      } else {
        // Adiciona aos favoritos (faz auto-onboarding do ativo inédito)
        const res = await fetch(`${API_URL}/watchlists/${activeWatchlistId}/items`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ticker: symbol }),
          credentials: 'include',
        });
        if (res.ok) {
          await loadWatchlistDetails(activeWatchlistId);
        }
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsAddingToWatchlist(false);
    }
  };

  // 7. REMOÇÃO RÁPIDA NA BARRA LATERAL (LIXEIRA)
  const handleRemoveFromSidebar = async (e: React.MouseEvent, ticker: string) => {
    e.stopPropagation(); // Evita carregar a cotação ao clicar em excluir
    try {
      const res = await fetch(`${API_URL}/watchlists/${activeWatchlistId}/items/${encodeURIComponent(ticker)}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        await loadWatchlistDetails(activeWatchlistId);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleSelectAsset = (symbol: string) => {
    setSearchQuery('');
    setShowDropdown(false);
    loadQuote(symbol);
  };

  // Formatadores
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

  if (authLoading) {
    return (
      <main className="container">
        <div className="glass-panel">
          <span className="loading-spinner" style={{ borderTopColor: '#00f2fe', width: 40, height: 40 }}></span>
          <p style={{ marginTop: '1.5rem', color: 'var(--text-secondary)' }}>Carregando sua sessão segura...</p>
        </div>
      </main>
    );
  }

  if (!user) return null;

  const activeFavorited = activeQuote ? isAssetFavorited(activeQuote.symbol) : false;

  return (
    <main className="container" style={{ maxWidth: 1400 }}>
      {/* Header */}
      <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', borderBottom: '1px solid var(--panel-border)', paddingBottom: '1.25rem', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '2.3rem', background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', margin: 0, fontWeight: 800, display: 'flex', alignItems: 'center', gap: '10px' }}>
            stock-pulse
            <span style={{ 
              width: 8, 
              height: 8, 
              borderRadius: '50%', 
              backgroundColor: wsConnected ? '#00e676' : '#ff3d00', 
              boxShadow: wsConnected ? '0 0 10px #00e676' : '0 0 10px #ff3d00',
              display: 'inline-block',
              transition: 'all 0.3s ease'
            }} title={wsConnected ? 'Conexão em Tempo Real Ativa' : 'Desconectado da cotação tempo real'} />
          </h1>
          
          {/* Navegação entre telas do Dashboard */}
          <div style={{ display: 'flex', gap: '1.5rem', marginTop: '0.8rem' }}>
            <Link href="/dashboard" style={{ color: 'var(--accent-color)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 700, borderBottom: '2px solid var(--accent-color)', paddingBottom: '3px', display: 'flex', alignItems: 'center', gap: '5px' }}>
              📊 Painel
            </Link>
            <Link href="/dashboard/portfolio" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              💼 Carteiras
            </Link>
            <Link href="/dashboard/alerts" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              🔔 Alertas
            </Link>
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

      {/* Main Grid responsiva em Flexbox */}
      <div style={{ display: 'flex', gap: '2rem', flexFlow: 'row wrap', alignItems: 'stretch' }}>
        
        {/* COLUNA ESQUERDA (BUSCA E DETALHE) */}
        <div style={{ flex: '2 1 500px', display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          
          {/* Barra de Busca */}
          <div style={{ position: 'relative' }}>
            <div className="form-group" style={{ margin: 0 }}>
              <input
                className="form-input"
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={() => { if (searchResults.length > 0) setShowDropdown(true); }}
                placeholder="🔍 Pesquise ativos... (Ex: PETR4, AAPL, VALE3, BTC-USD)"
                autoComplete="off"
                style={{ fontSize: '1rem', padding: '0.9rem 1.2rem' }}
              />
              {isSearching && (
                <div style={{ position: 'absolute', right: '15px', top: '35%' }}>
                  <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)' }}></span>
                </div>
              )}
            </div>

            {/* Dropdown da busca */}
            {showDropdown && searchResults.length > 0 && (
              <div className="glass-panel" style={{
                position: 'absolute',
                top: '100%',
                left: 0,
                width: '100%',
                marginTop: '0.5rem',
                zIndex: 10,
                padding: '0.5rem',
                textAlign: 'left',
                maxHeight: '280px',
                overflowY: 'auto',
                boxShadow: '0 16px 40px rgba(0, 0, 0, 0.6)'
              }}>
                {searchResults.map((item) => (
                  <div
                    key={item.symbol}
                    onClick={() => handleSelectAsset(item.symbol)}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      padding: '0.65rem 0.9rem',
                      borderRadius: '8px',
                      cursor: 'pointer',
                      transition: 'background-color 0.2s ease',
                    }}
                    onMouseEnter={(e) => e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.04)'}
                    onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
                  >
                    <div>
                      <span style={{ fontWeight: 700, color: 'var(--accent-color)', marginRight: '0.8rem' }}>{item.symbol}</span>
                      <span style={{ fontSize: '0.85rem', opacity: 0.85 }}>{item.name}</span>
                    </div>
                    <span style={{ fontSize: '0.65rem', padding: '0.2rem 0.4rem', background: 'rgba(0, 242, 254, 0.08)', color: 'var(--accent-color)', borderRadius: '4px', textTransform: 'uppercase' }}>
                      {item.exchange}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Card da Cotação Principal */}
          <div className="glass-panel" style={{ minHeight: '260px', display: 'flex', flexDirection: 'column', justifyContent: 'center', textAlign: 'left', padding: '2rem' }}>
            {isLoadingQuote ? (
              <div style={{ textAlign: 'center', width: '100%' }}>
                <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 35, height: 35 }}></span>
                <p style={{ marginTop: '1rem', color: 'var(--text-secondary)', fontSize: '0.9rem' }}>Carregando dados em tempo real...</p>
              </div>
            ) : quoteError ? (
              <div className="alert-error" style={{ margin: 0, width: '100%' }}>
                ⚠️ {quoteError}
              </div>
            ) : activeQuote ? (
              <div style={{ width: '100%' }}>
                {/* Nome, Ticker e Ícone Estrela */}
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '1.25rem' }}>
                    <div>
                      <h2 style={{ fontSize: '2.4rem', margin: 0, fontWeight: 800, color: '#fff', display: 'flex', alignItems: 'center', gap: '12px' }}>
                        {activeQuote.symbol}
                        
                        {/* ÍCONE ESTRELA PARA FAVORITAR DENTRO DA WATCHLIST ATIVA */}
                        <button
                          onClick={handleToggleFavorite}
                          disabled={isAddingToWatchlist}
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            fontSize: '2rem',
                            color: activeFavorited ? '#ffd700' : 'rgba(255,255,255,0.15)',
                            transition: 'transform 0.15s ease, color 0.15s ease',
                            padding: 0,
                            lineHeight: 1,
                          }}
                          onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.25)'}
                          onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
                          title={activeFavorited ? 'Remover dos Favoritos' : 'Adicionar aos Favoritos'}
                        >
                          {activeFavorited ? '★' : '☆'}
                        </button>
                      </h2>
                      <p style={{ margin: '0.1rem 0 0 0', fontSize: '0.95rem', color: 'var(--text-secondary)' }}>
                        {activeQuote.name}
                      </p>
                    </div>
                  </div>

                  {/* Refresh e Badge */}
                  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '0.4rem' }}>
                    <div style={{ display: 'flex', gap: '0.5rem' }}>
                      <button
                        className="primary-button"
                        onClick={openAlertModal}
                        style={{ padding: '0.4rem 1rem', fontSize: '0.8rem', background: 'linear-gradient(135deg, #ff9800, #f57c00)', border: 'none', color: '#000', fontWeight: 700 }}
                      >
                        🔔 Criar Alerta
                      </button>
                      <button
                        className="primary-button"
                        onClick={() => loadQuote(activeQuote.symbol, true)}
                        style={{ padding: '0.4rem 1rem', fontSize: '0.8rem' }}
                      >
                        🔄 Atualizar
                      </button>
                    </div>
                    {cacheStatus === 'hit' && (
                      <span style={{ fontSize: '0.7rem', color: '#00f2fe', background: 'rgba(0,242,254,0.08)', padding: '0.15rem 0.5rem', borderRadius: '4px', fontWeight: 600 }}>
                        ⚡ Redis Cache
                      </span>
                    )}
                    {cacheStatus === 'miss' && (
                      <span style={{ fontSize: '0.7rem', color: '#ffc107', background: 'rgba(255,193,7,0.08)', padding: '0.15rem 0.5rem', borderRadius: '4px', fontWeight: 600 }}>
                        🌐 Yahoo API
                      </span>
                    )}
                  </div>
                </div>

                {/* Preço e Variação */}
                <div style={{ display: 'flex', alignItems: 'baseline', gap: '1.25rem', marginBottom: '1.8rem' }}>
                  <span style={{ 
                    fontSize: '3rem', 
                    fontWeight: 800, 
                    color: priceFlashing[activeQuote.symbol] === 'up' ? '#00e676' : priceFlashing[activeQuote.symbol] === 'down' ? '#ff3d00' : '#fff', 
                    textShadow: priceFlashing[activeQuote.symbol] === 'up' ? '0 0 15px rgba(0, 230, 118, 0.6)' : priceFlashing[activeQuote.symbol] === 'down' ? '0 0 15px rgba(255, 61, 0, 0.6)' : 'none',
                    transition: 'all 0.2s ease',
                    letterSpacing: '-0.02em' 
                  }}>
                    {formatMoney(activeQuote.price, activeQuote.currency)}
                  </span>
                  <span style={{
                    fontSize: '1.1rem',
                    fontWeight: 700,
                    color: activeQuote.change >= 0 ? '#00e676' : '#ff3d00',
                    background: activeQuote.change >= 0 ? 'rgba(0,230,118,0.08)' : 'rgba(255,61,0,0.08)',
                    padding: '0.3rem 0.7rem',
                    borderRadius: '6px',
                  }}>
                    {activeQuote.change >= 0 ? '▲' : '▼'} {formatPercentage(activeQuote.change_percent)}
                  </span>
                </div>

                {/* Grid Secundária */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '1rem' }}>
                  <div style={{ background: 'rgba(255,255,255,0.015)', padding: '0.8rem 1rem', borderRadius: '8px', border: '1px solid var(--panel-border)' }}>
                    <span style={{ display: 'block', fontSize: '0.7rem', color: 'var(--text-secondary)', textTransform: 'uppercase', marginBottom: '0.2rem', fontWeight: 600 }}>
                      Mínima
                    </span>
                    <span style={{ fontSize: '1rem', fontWeight: 700 }}>
                      {formatMoney(activeQuote.low, activeQuote.currency)}
                    </span>
                  </div>
                  <div style={{ background: 'rgba(255,255,255,0.015)', padding: '0.8rem 1rem', borderRadius: '8px', border: '1px solid var(--panel-border)' }}>
                    <span style={{ display: 'block', fontSize: '0.7rem', color: 'var(--text-secondary)', textTransform: 'uppercase', marginBottom: '0.2rem', fontWeight: 600 }}>
                      Máxima
                    </span>
                    <span style={{ fontSize: '1rem', fontWeight: 700 }}>
                      {formatMoney(activeQuote.high, activeQuote.currency)}
                    </span>
                  </div>
                  <div style={{ background: 'rgba(255,255,255,0.015)', padding: '0.8rem 1rem', borderRadius: '8px', border: '1px solid var(--panel-border)' }}>
                    <span style={{ display: 'block', fontSize: '0.7rem', color: 'var(--text-secondary)', textTransform: 'uppercase', marginBottom: '0.2rem', fontWeight: 600 }}>
                      Volume
                    </span>
                    <span style={{ fontSize: '1rem', fontWeight: 700, fontFamily: 'monospace' }}>
                      {new Intl.NumberFormat('pt-BR').format(activeQuote.volume)}
                    </span>
                  </div>
                </div>
              </div>
            ) : (
              <div style={{ textAlign: 'center', width: '100%', color: 'var(--text-secondary)' }}>
                <p style={{ margin: 0, fontSize: '1.05rem' }}>
                  🔎 Pesquise um ativo no campo superior ou selecione um dos seus favoritos ao lado para ver a cotação.
                </p>
              </div>
            )}
          </div>
        </div>

        {/* COLUNA DIREITA (WATCHLIST SIDEBAR) */}
        <div style={{ flex: '1 1 320px', display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          <div className="glass-panel" style={{ padding: '1.5rem', textAlign: 'left', display: 'flex', flexDirection: 'column', height: '100%' }}>
            
            {/* Seletor de Watchlists */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.25rem' }}>
              <h3 style={{ margin: 0, fontSize: '1.1rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--accent-color)' }}>
                ⭐ Favoritos
              </h3>
              
              {/* Botão de excluir watchlist ativa */}
              {activeWL && watchlists.length > 1 && (
                <button
                  onClick={handleDeleteActiveWatchlist}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: '#ff4a5a',
                    cursor: 'pointer',
                    fontSize: '0.8rem',
                    fontWeight: 600,
                  }}
                  title="Excluir Lista Atual"
                >
                  Excluir Lista
                </button>
              )}
            </div>

            {/* Abas das Watchlists */}
            <div style={{ display: 'flex', gap: '0.5rem', overflowX: 'auto', paddingBottom: '0.5rem', marginBottom: '1rem', borderBottom: '1px solid var(--panel-border)' }}>
              {watchlists.map((wl) => (
                <button
                  key={wl.id}
                  onClick={() => {
                    setActiveWatchlistId(wl.id);
                    loadWatchlistDetails(wl.id);
                  }}
                  style={{
                    padding: '0.4rem 0.8rem',
                    fontSize: '0.8rem',
                    borderRadius: '6px',
                    border: '1px solid',
                    borderColor: activeWatchlistId === wl.id ? 'var(--accent-color)' : 'var(--panel-border)',
                    background: activeWatchlistId === wl.id ? 'rgba(0, 242, 254, 0.08)' : 'transparent',
                    color: activeWatchlistId === wl.id ? 'var(--accent-color)' : 'var(--text-secondary)',
                    cursor: 'pointer',
                    fontWeight: 600,
                    whiteSpace: 'nowrap',
                    transition: 'all 0.15s ease',
                  }}
                >
                  {wl.name}
                </button>
              ))}
            </div>

            {/* Formulário para Criar Nova Watchlist */}
            <form onSubmit={handleCreateWatchlist} style={{ display: 'flex', gap: '0.5rem', marginBottom: '1.25rem' }}>
              <input
                className="form-input"
                type="text"
                value={newWatchlistName}
                onChange={(e) => setNewWatchlistName(e.target.value)}
                placeholder="Nova Lista..."
                required
                disabled={isCreatingList}
                style={{ padding: '0.5rem 0.8rem', fontSize: '0.85rem' }}
              />
              <button
                className="primary-button"
                type="submit"
                disabled={isCreatingList}
                style={{ padding: '0.5rem 1rem', fontSize: '0.85rem', whiteSpace: 'nowrap' }}
              >
                + Criar
              </button>
            </form>

            {/* Listagem de itens da Watchlist Ativa */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.65rem', maxHeight: '350px', overflowY: 'auto' }}>
              {activeWL?.items && activeWL.items.length > 0 ? (
                activeWL.items.map((item) => {
                  const wlPos = item.change !== undefined && item.change >= 0;
                  return (
                    <div
                      key={item.ticker}
                      onClick={() => loadQuote(item.ticker)}
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        padding: '0.75rem',
                        borderRadius: '8px',
                        background: 'rgba(255,255,255,0.015)',
                        border: '1px solid var(--panel-border)',
                        cursor: 'pointer',
                        transition: 'all 0.2s ease',
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.03)';
                        e.currentTarget.style.borderColor = 'rgba(0, 242, 254, 0.2)';
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.015)';
                        e.currentTarget.style.borderColor = 'var(--panel-border)';
                      }}
                    >
                      <div>
                        <span style={{ display: 'block', fontWeight: 700, color: 'var(--text-primary)', fontSize: '0.9rem' }}>
                          {item.ticker}
                        </span>
                        <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', display: 'block', maxWidth: '140px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {item.name}
                        </span>
                      </div>

                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                        {item.price !== undefined ? (
                          <div style={{ textAlign: 'right' }}>
                            <span style={{ 
                              display: 'block', 
                              fontSize: '0.9rem', 
                              fontWeight: 700,
                              color: priceFlashing[item.ticker] === 'up' ? '#00e676' : priceFlashing[item.ticker] === 'down' ? '#ff3d00' : '#fff',
                              textShadow: priceFlashing[item.ticker] === 'up' ? '0 0 10px rgba(0, 230, 118, 0.5)' : priceFlashing[item.ticker] === 'down' ? '0 0 10px rgba(255, 61, 0, 0.5)' : 'none',
                              transition: 'all 0.2s ease'
                            }}>
                              {formatMoney(item.price, item.currency)}
                            </span>
                            <span style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: wlPos ? '#00e676' : '#ff3d00' }}>
                              {item.change_percent !== undefined ? formatPercentage(item.change_percent) : ''}
                            </span>
                            {item.graham_value && item.price ? (
                              <span style={{
                                display: 'inline-block',
                                marginTop: '0.2rem',
                                padding: '0.15rem 0.35rem',
                                borderRadius: '4px',
                                fontSize: '0.6rem',
                                fontWeight: 700,
                                backgroundColor: item.price < item.graham_value ? 'rgba(0, 230, 118, 0.15)' : 'rgba(255, 61, 0, 0.15)',
                                color: item.price < item.graham_value ? '#00e676' : '#ff3d00',
                                border: `1px solid ${item.price < item.graham_value ? 'rgba(0, 230, 118, 0.3)' : 'rgba(255, 61, 0, 0.3)'}`
                              }} title={`Graham: ${formatMoney(item.graham_value, item.currency)}`}>
                                {item.price < item.graham_value ? 'DESC' : 'CARA'}
                              </span>
                            ) : null}
                          </div>
                        ) : (
                          <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>--</span>
                        )}

                        {/* Botão de Excluir Item */}
                        <button
                          onClick={(e) => handleRemoveFromSidebar(e, item.ticker)}
                          style={{
                            background: 'none',
                            border: 'none',
                            color: 'rgba(255,255,255,0.25)',
                            cursor: 'pointer',
                            fontSize: '1rem',
                            padding: '0.2rem',
                            transition: 'color 0.15s ease',
                          }}
                          onMouseEnter={(e) => e.currentTarget.style.color = '#ff4a5a'}
                          onMouseLeave={(e) => e.currentTarget.style.color = 'rgba(255,255,255,0.25)'}
                          title="Remover dos favoritos"
                        >
                          ✕
                        </button>
                      </div>
                    </div>
                  );
                })
              ) : (
                <div style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '2rem 1rem', fontSize: '0.85rem', border: '1px dashed var(--panel-border)', borderRadius: '8px' }}>
                  A lista está vazia. <br /> Pesquise um ativo e clique na estrela (☆) para favoritá-lo aqui!
                </div>
              )}
            </div>

          </div>
        </div>

      </div>

      {/* MODAL DE CRIAÇÃO DE ALERTA (Fase 3) */}
      {showAlertModal && activeQuote && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0,0,0,0.7)',
          backdropFilter: 'blur(8px)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 100,
          padding: '1rem'
        }}>
          <div className="glass-panel" style={{
            maxWidth: '450px',
            width: '100%',
            padding: '2.5rem',
            textAlign: 'left',
            boxShadow: '0 20px 50px rgba(0,0,0,0.8)',
            border: '1px solid rgba(255,255,255,0.08)'
          }}>
            <h2 style={{ margin: '0 0 0.5rem 0', fontSize: '1.6rem', fontWeight: 800, color: '#fff' }}>
              🔔 Criar Alerta de Preço
            </h2>
            <p style={{ margin: '0 0 2rem 0', color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
              Defina as regras para o ativo <strong>{activeQuote.symbol}</strong> ({activeQuote.name}).
            </p>

            {alertErrorMsg && (
              <div className="alert-error" style={{ marginBottom: '1.5rem', margin: 0 }}>
                ⚠️ {alertErrorMsg}
              </div>
            )}

            {alertSuccessMsg ? (
              <div style={{
                background: 'rgba(0, 230, 118, 0.08)',
                border: '1px solid #00e676',
                borderRadius: '10px',
                padding: '1.2rem',
                color: '#e2e8f0',
                fontSize: '0.9rem',
                lineHeight: 1.5,
                marginBottom: '1rem'
              }}>
                🎉 {alertSuccessMsg}
              </div>
            ) : (
              <form onSubmit={handleCreateAlertSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
                
                {/* Condição */}
                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 600, fontSize: '0.8rem', textTransform: 'uppercase', color: 'var(--accent-color)' }}>
                    Condição de Disparo
                  </label>
                  <select
                    className="form-input"
                    value={alertCondition}
                    onChange={(e) => setAlertCondition(e.target.value as 'ABOVE' | 'BELOW')}
                    style={{ background: '#111827', width: '100%', padding: '0.6rem 0.9rem' }}
                  >
                    <option value="ABOVE">Preço sobe acima de (▲)</option>
                    <option value="BELOW">Preço cai abaixo de (▼)</option>
                  </select>
                </div>

                {/* Preço Alvo */}
                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 600, fontSize: '0.8rem', textTransform: 'uppercase', color: 'var(--accent-color)' }}>
                    Preço Alvo ({activeQuote.currency})
                  </label>
                  <input
                    className="form-input"
                    type="number"
                    step="0.01"
                    required
                    value={alertTargetPrice}
                    onChange={(e) => setAlertTargetPrice(e.target.value)}
                    placeholder="Ex: 38.50"
                    style={{ width: '100%', fontSize: '1.1rem', padding: '0.6rem 0.9rem' }}
                  />
                </div>

                {/* Botões */}
                <div style={{ display: 'flex', gap: '1rem', marginTop: '1rem' }}>
                  <button
                    className="primary-button"
                    type="submit"
                    disabled={isCreatingAlert}
                    style={{ flex: 1, padding: '0.8rem', fontSize: '0.9rem', background: 'linear-gradient(135deg, #00f2fe, #4facfe)', color: '#0b0f19', fontWeight: 700 }}
                  >
                    {isCreatingAlert ? 'Criando...' : 'Salvar Alerta'}
                  </button>
                  <button
                    className="primary-button"
                    type="button"
                    onClick={() => setShowAlertModal(false)}
                    style={{ flex: 1, padding: '0.8rem', fontSize: '0.9rem', background: 'rgba(255,255,255,0.05)', color: '#fff', border: '1px solid var(--panel-border)' }}
                  >
                    Cancelar
                  </button>
                </div>

              </form>
            )}

          </div>
        </div>
      )}
    </main>
  );
}
