'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useSearchParams } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { apiFetch } from '@/lib/api';
import { Item, Watchlist, SearchResult, Quote } from '@/components/dashboard/types';
import AppSidebar from '@/components/AppSidebar';
import AssetSearch from '@/components/dashboard/AssetSearch';
import ActiveQuoteCard from '@/components/dashboard/ActiveQuoteCard';
import WatchlistSidebar from '@/components/dashboard/WatchlistSidebar';
import CreateAlertModal from '@/components/dashboard/CreateAlertModal';

export default function DashboardPage() {
  const { user, logout, isLoading: authLoading } = useAuth();
  const searchParams = useSearchParams();
  const tickerParam = searchParams ? searchParams.get('ticker') : null;
  
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

  // WebSocket em tempo real e flashes
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [wsConnected, setWsConnected] = useState(false);
  const [priceFlashing, setPriceFlashing] = useState<Record<string, 'up' | 'down'>>({});

  // Modal de Alertas
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
      const res = await apiFetch(`/alerts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ticker: activeQuote.symbol,
          target_price: parseFloat(alertTargetPrice),
          condition: alertCondition,
        }),
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
      const res = await apiFetch(`/watchlists`);
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

  // 2. DETALHA A WATCHLIST ATIVA
  const loadWatchlistDetails = async (id: string) => {
    try {
      const res = await apiFetch(`/watchlists/${id}`);
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

  // Conexão WebSocket em Tempo Real
  useEffect(() => {
    if (!user) return;

    const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
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
        const res = await apiFetch(`/assets/search?q=${encodeURIComponent(searchQuery)}`);
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
      const res = await apiFetch(`/quotes/${encodeURIComponent(symbol)}`);
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

  // Carrega ativo diretamente se passado na URL (?ticker=XXXX)
  useEffect(() => {
    if (tickerParam && user) {
      loadQuote(tickerParam);
    }
  }, [tickerParam, user, loadQuote]);

  // 4. CRIA UMA NOVA WATCHLIST
  const handleCreateWatchlist = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWatchlistName.trim()) return;
    setIsCreatingList(true);
    try {
      const res = await apiFetch(`/watchlists`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newWatchlistName }),
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
      const res = await apiFetch(`/watchlists/${activeWatchlistId}`, {
        method: 'DELETE',
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
        const res = await apiFetch(`/watchlists/${activeWatchlistId}/items/${encodeURIComponent(symbol)}`, {
          method: 'DELETE',
        });
        if (res.ok) {
          await loadWatchlistDetails(activeWatchlistId);
        }
      } else {
        const res = await apiFetch(`/watchlists/${activeWatchlistId}/items`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ticker: symbol }),
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
    e.stopPropagation();
    try {
      const res = await apiFetch(`/watchlists/${activeWatchlistId}/items/${encodeURIComponent(ticker)}`, {
        method: 'DELETE',
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
          <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 40, height: 40 }}></span>
          <p style={{ marginTop: '1.5rem', color: 'var(--text-secondary)' }}>Carregando sua sessão segura...</p>
        </div>
      </main>
    );
  }

  if (!user) return null;

  const activeFavorited = activeQuote ? isAssetFavorited(activeQuote.symbol) : false;

  return (
    <div className="app-layout">
      <AppSidebar
        userName={user.name}
        wsConnected={wsConnected}
        onLogout={logout}
      />

      <main className="app-main-content">

      {/* Main Grid responsiva em Flexbox */}
      <div style={{ display: 'flex', gap: '2rem', flexFlow: 'row wrap', alignItems: 'stretch' }}>
        
        {/* COLUNA ESQUERDA (BUSCA E DETALHE) */}
        <div style={{ flex: '2 1 500px', display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          {/* Barra de Busca */}
          <AssetSearch
            searchQuery={searchQuery}
            searchResults={searchResults}
            isSearching={isSearching}
            showDropdown={showDropdown}
            onSearchChange={setSearchQuery}
            onFocus={() => { if (searchResults.length > 0) setShowDropdown(true); }}
            onSelectAsset={handleSelectAsset}
          />

          {/* Card da Cotação Principal */}
          <ActiveQuoteCard
            activeQuote={activeQuote}
            isLoadingQuote={isLoadingQuote}
            quoteError={quoteError}
            activeFavorited={activeFavorited}
            isAddingToWatchlist={isAddingToWatchlist}
            cacheStatus={cacheStatus}
            priceFlashing={priceFlashing}
            onToggleFavorite={handleToggleFavorite}
            onOpenAlertModal={openAlertModal}
            onRefreshQuote={loadQuote}
            formatMoney={formatMoney}
            formatPercentage={formatPercentage}
          />
        </div>

        {/* COLUNA DIREITA (WATCHLIST SIDEBAR) */}
        <WatchlistSidebar
          watchlists={watchlists}
          activeWatchlistId={activeWatchlistId}
          activeWL={activeWL}
          newWatchlistName={newWatchlistName}
          isCreatingList={isCreatingList}
          priceFlashing={priceFlashing}
          onSelectWatchlist={(id) => {
            setActiveWatchlistId(id);
            loadWatchlistDetails(id);
          }}
          onDeleteActiveWatchlist={handleDeleteActiveWatchlist}
          onCreateWatchlist={handleCreateWatchlist}
          onNewWatchlistNameChange={setNewWatchlistName}
          onSelectAsset={loadQuote}
          onRemoveFromSidebar={handleRemoveFromSidebar}
          formatMoney={formatMoney}
          formatPercentage={formatPercentage}
        />
      </div>

      {/* MODAL DE CRIAÇÃO DE ALERTA */}
      {showAlertModal && activeQuote && (
        <CreateAlertModal
          activeQuote={activeQuote}
          alertTargetPrice={alertTargetPrice}
          alertCondition={alertCondition}
          isCreatingAlert={isCreatingAlert}
          alertErrorMsg={alertErrorMsg}
          alertSuccessMsg={alertSuccessMsg}
          onTargetPriceChange={setAlertTargetPrice}
          onConditionChange={setAlertCondition}
          onSubmit={handleCreateAlertSubmit}
          onClose={() => setShowAlertModal(false)}
        />
      )}
      </main>
    </div>
  );
}
