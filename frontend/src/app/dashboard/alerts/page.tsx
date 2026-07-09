'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/context/AuthContext';
import Link from 'next/link';

interface Alert {
  id: string;
  user_id: string;
  asset_id: string;
  ticker: string;
  asset_name: string;
  currency: string;
  target_price: number;
  condition: 'ABOVE' | 'BELOW';
  status: 'ACTIVE' | 'TRIGGERED' | 'DISABLED';
  triggered_at?: string;
  created_at: string;
}

interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
  type: string;
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export default function AlertsPage() {
  const { user, logout, isLoading: authLoading } = useAuth();

  // Estados principais
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [isLoadingAlerts, setIsLoadingAlerts] = useState(true);

  // Estados de busca/criação
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [selectedAsset, setSelectedAsset] = useState<SearchResult | null>(null);

  // Formulário
  const [targetPrice, setTargetPrice] = useState('');
  const [condition, setCondition] = useState<'ABOVE' | 'BELOW'>('ABOVE');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [formSuccess, setFormSuccess] = useState<string | null>(null);

  // CARREGAR ALERTAS DO BANCO
  const loadAlerts = useCallback(async () => {
    try {
      const res = await fetch(`${API_URL}/alerts`, {
        credentials: 'include',
      });
      if (res.ok) {
        const data = await res.json();
        setAlerts(data || []);
      }
    } catch (e) {
      console.error('Erro ao buscar alertas:', e);
    } finally {
      setIsLoadingAlerts(false);
    }
  }, []);

  useEffect(() => {
    if (user) {
      loadAlerts();
    }
  }, [user, loadAlerts]);

  // Debounce para autocomplete do input de busca de ativos
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

  // SELEÇÃO DE ATIVO
  const handleSelectAsset = (asset: SearchResult) => {
    setSelectedAsset(asset);
    setSearchQuery(asset.symbol);
    setShowDropdown(false);
    setFormError(null);
  };

  // SUBMIT DO ALERTA
  const handleCreateAlert = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedAsset || !targetPrice.trim()) {
      setFormError('Por favor, selecione um ativo válido e insira o preço alvo.');
      return;
    }

    setIsSubmitting(true);
    setFormError(null);
    setFormSuccess(null);

    try {
      const res = await fetch(`${API_URL}/alerts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ticker: selectedAsset.symbol,
          target_price: parseFloat(targetPrice),
          condition: condition,
        }),
        credentials: 'include',
      });

      const data = await res.json();
      if (res.ok) {
        setFormSuccess('Alerta configurado com sucesso!');
        setTargetPrice('');
        setSearchQuery('');
        setSelectedAsset(null);
        await loadAlerts();
        setTimeout(() => setFormSuccess(null), 3000);
      } else {
        setFormError(data.error || 'Erro ao registrar alerta.');
      }
    } catch (e) {
      setFormError('Falha ao conectar com o servidor.');
    } finally {
      setIsSubmitting(false);
    }
  };

  // TOGGLE STATUS (ATIVE <-> DISABLED)
  const handleToggleStatus = async (id: string) => {
    try {
      const res = await fetch(`${API_URL}/alerts/${id}/toggle`, {
        method: 'PUT',
        credentials: 'include',
      });
      if (res.ok) {
        // Atualiza status localmente de forma otimista
        const data = await res.json();
        setAlerts((prev) =>
          prev.map((a) => (a.id === id ? { ...a, status: data.status as 'ACTIVE' | 'DISABLED' } : a))
        );
      }
    } catch (e) {
      console.error(e);
    }
  };

  // EXCLUSÃO
  const handleDeleteAlert = async (id: string) => {
    if (!confirm('Deseja realmente excluir permanentemente este alerta?')) return;
    try {
      const res = await fetch(`${API_URL}/alerts/${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        setAlerts((prev) => prev.filter((a) => a.id !== id));
      }
    } catch (e) {
      console.error(e);
    }
  };

  // Formatadores
  const formatMoney = (val: number, currency: string) => {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: currency || 'BRL',
    }).format(val);
  };

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr);
    const yyyy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    const dd = String(d.getDate()).padStart(2, '0');
    const hh = String(d.getHours()).padStart(2, '0');
    const min = String(d.getMinutes()).padStart(2, '0');
    return `${yyyy}/${mm}/${dd} às ${hh}:${min}`;
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

  // Filtragem dos alertas
  const activeAlerts = alerts.filter((a) => a.status === 'ACTIVE');
  const triggeredAlerts = alerts.filter((a) => a.status === 'TRIGGERED');
  const disabledAlerts = alerts.filter((a) => a.status === 'DISABLED');

  return (
    <main className="container" style={{ maxWidth: 1100 }}>
      {/* Header */}
      <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', borderBottom: '1px solid var(--panel-border)', paddingBottom: '1.25rem', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '2.3rem', background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', margin: 0, fontWeight: 800 }}>stock-pulse</h1>
          
          <div style={{ display: 'flex', gap: '1.5rem', marginTop: '0.8rem' }}>
            <Link href="/dashboard/portfolio" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              💼 Minha Carteira
            </Link>
            <Link href="/dashboard" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              📊 Monitoramento
            </Link>
            <Link href="/dashboard/alerts" style={{ color: 'var(--accent-color)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 700, borderBottom: '2px solid var(--accent-color)', paddingBottom: '3px', display: 'flex', alignItems: 'center', gap: '5px' }}>
              🔔 Meus Alertas
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

      {/* Grid Central */}
      <div style={{ display: 'flex', flexFlow: 'row wrap', gap: '2rem', alignItems: 'stretch' }}>
        
        {/* Painel Esquerdo: Criar Alerta */}
        <div style={{ flex: '1 1 350px' }}>
          <div className="glass-panel" style={{ padding: '2rem', textAlign: 'left', height: '100%' }}>
            <h2 style={{ fontSize: '1.5rem', fontWeight: 800, color: '#fff', margin: '0 0 1.5rem 0' }}>
              Novo Alerta
            </h2>

            {formError && (
              <div className="alert-error" style={{ marginBottom: '1.5rem', margin: 0 }}>
                ⚠️ {formError}
              </div>
            )}

            {formSuccess && (
              <div style={{
                background: 'rgba(0, 230, 118, 0.08)',
                border: '1px solid #00e676',
                borderRadius: '10px',
                padding: '1rem',
                color: '#e2e8f0',
                fontSize: '0.85rem',
                marginBottom: '1.5rem'
              }}>
                🎉 {formSuccess}
              </div>
            )}

            <form onSubmit={handleCreateAlert} style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
              
              {/* Autocomplete Ticker */}
              <div className="form-group" style={{ position: 'relative', margin: 0 }}>
                <label className="form-label" style={{ display: 'block', marginBottom: '0.4rem', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', color: 'var(--accent-color)' }}>
                  Buscar Ativo
                </label>
                <input
                  className="form-input"
                  type="text"
                  value={searchQuery}
                  onChange={(e) => {
                    setSearchQuery(e.target.value);
                    if (selectedAsset) setSelectedAsset(null);
                  }}
                  onFocus={() => { if (searchResults.length > 0) setShowDropdown(true); }}
                  placeholder="Ticker... Ex: VALE3.SA, AAPL"
                  autoComplete="off"
                  required
                  style={{ width: '100%', padding: '0.6rem 0.9rem' }}
                />
                {isSearching && (
                  <div style={{ position: 'absolute', right: '12px', top: '55%' }}>
                    <span className="loading-spinner" style={{ width: 15, height: 15, borderTopColor: 'var(--accent-color)' }}></span>
                  </div>
                )}

                {/* Dropdown de Busca */}
                {showDropdown && searchResults.length > 0 && (
                  <div className="glass-panel" style={{
                    position: 'absolute',
                    top: '100%',
                    left: 0,
                    width: '100%',
                    marginTop: '0.5rem',
                    zIndex: 10,
                    padding: '0.4rem',
                    textAlign: 'left',
                    maxHeight: '200px',
                    overflowY: 'auto',
                    boxShadow: '0 16px 40px rgba(0, 0, 0, 0.6)'
                  }}>
                    {searchResults.map((item) => (
                      <div
                        key={item.symbol}
                        onClick={() => handleSelectAsset(item)}
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                          padding: '0.5rem 0.75rem',
                          borderRadius: '6px',
                          cursor: 'pointer',
                          transition: 'background-color 0.2s ease',
                          fontSize: '0.85rem'
                        }}
                        onMouseEnter={(e) => e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.04)'}
                        onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
                      >
                        <div>
                          <span style={{ fontWeight: 700, color: 'var(--accent-color)', marginRight: '0.5rem' }}>{item.symbol}</span>
                          <span style={{ opacity: 0.85 }}>{item.name}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {selectedAsset && (
                <div style={{ background: 'rgba(255,255,255,0.015)', border: '1px solid var(--panel-border)', padding: '0.8rem', borderRadius: '8px', fontSize: '0.8rem' }}>
                  <span style={{ color: 'var(--text-secondary)', display: 'block' }}>Empresa Selecionada</span>
                  <span style={{ fontWeight: 700, color: '#fff', fontSize: '0.9rem' }}>{selectedAsset.name}</span>
                  <span style={{ display: 'block', color: 'var(--accent-color)', fontSize: '0.7rem', textTransform: 'uppercase', marginTop: '0.2rem' }}>
                    Bolsa: {selectedAsset.exchange} | Tipo: {selectedAsset.type}
                  </span>
                </div>
              )}

              {/* Condição */}
              <div className="form-group" style={{ margin: 0 }}>
                <label className="form-label" style={{ display: 'block', marginBottom: '0.4rem', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', color: 'var(--accent-color)' }}>
                  Condição
                </label>
                <select
                  className="form-input"
                  value={condition}
                  onChange={(e) => setCondition(e.target.value as 'ABOVE' | 'BELOW')}
                  style={{ background: '#111827', width: '100%', padding: '0.6rem 0.9rem' }}
                >
                  <option value="ABOVE">Sobe acima de (▲)</option>
                  <option value="BELOW">Cai abaixo de (▼)</option>
                </select>
              </div>

              {/* Preço Alvo */}
              <div className="form-group" style={{ margin: 0 }}>
                <label className="form-label" style={{ display: 'block', marginBottom: '0.4rem', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', color: 'var(--accent-color)' }}>
                  Preço Alvo
                </label>
                <input
                  className="form-input"
                  type="number"
                  step="0.01"
                  required
                  value={targetPrice}
                  onChange={(e) => setTargetPrice(e.target.value)}
                  placeholder="Ex: 38.50"
                  style={{ width: '100%', padding: '0.6rem 0.9rem' }}
                />
              </div>

              <button
                className="primary-button"
                type="submit"
                disabled={isSubmitting || !selectedAsset}
                style={{
                  width: '100%',
                  padding: '0.75rem',
                  fontSize: '0.9rem',
                  marginTop: '0.5rem',
                  background: (!selectedAsset) ? 'rgba(255,255,255,0.03)' : 'linear-gradient(135deg, #00f2fe 0%, #4facfe 100%)',
                  color: (!selectedAsset) ? 'var(--text-secondary)' : '#0b0f19',
                  cursor: (!selectedAsset) ? 'not-allowed' : 'pointer',
                  fontWeight: 700,
                  border: 'none'
                }}
              >
                {isSubmitting ? 'Salvando...' : '🔔 Ativar Alerta'}
              </button>

            </form>
          </div>
        </div>

        {/* Painel Direito: Listagens das Categorias de Alertas */}
        <div style={{ flex: '2 1 600px', display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          
          {/* Seção 1: Alertas Ativos */}
          <div className="glass-panel" style={{ padding: '1.5rem 2rem', textAlign: 'left' }}>
            <h3 style={{ margin: '0 0 1rem 0', fontSize: '1.1rem', fontWeight: 800, color: '#00f2fe', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              🟢 Alertas Ativos ({activeAlerts.length})
            </h3>
            
            {isLoadingAlerts ? (
              <div style={{ padding: '2rem', textAlign: 'center' }}>
                <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)' }}></span>
              </div>
            ) : activeAlerts.length === 0 ? (
              <p style={{ margin: 0, color: 'var(--text-secondary)', fontSize: '0.85rem', padding: '1rem 0' }}>
                Nenhum alerta ativo no momento. Use o painel ao lado para criar o seu primeiro alerta de preços!
              </p>
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '1rem' }}>
                {activeAlerts.map((a) => (
                  <div key={a.id} style={{ background: 'rgba(255,255,255,0.015)', border: '1px solid var(--panel-border)', borderRadius: '12px', padding: '1.2rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                      <span style={{ fontWeight: 800, fontSize: '1.1rem', color: '#fff', display: 'block' }}>{a.ticker}</span>
                      <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', display: 'block', marginBottom: '0.5rem', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '160px' }}>
                        {a.asset_name}
                      </span>
                      <span style={{ fontSize: '0.85rem', fontWeight: 700, color: a.condition === 'ABOVE' ? '#00e676' : '#ff3d00' }}>
                        {a.condition === 'ABOVE' ? '▲ Acima de' : '▼ Abaixo de'} {formatMoney(a.target_price, a.currency)}
                      </span>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.8rem' }}>
                      {/* Toggle Switch */}
                      <button
                        onClick={() => handleToggleStatus(a.id)}
                        style={{
                          background: 'rgba(0, 242, 254, 0.1)',
                          border: '1px solid #00f2fe',
                          color: '#00f2fe',
                          borderRadius: '20px',
                          padding: '0.25rem 0.65rem',
                          fontSize: '0.7rem',
                          fontWeight: 700,
                          cursor: 'pointer',
                          transition: 'all 0.2s'
                        }}
                        onMouseEnter={(e) => { e.currentTarget.style.background = '#00f2fe'; e.currentTarget.style.color = '#000'; }}
                        onMouseLeave={(e) => { e.currentTarget.style.background = 'rgba(0, 242, 254, 0.1)'; e.currentTarget.style.color = '#00f2fe'; }}
                        title="Desativar Alerta"
                      >
                        Pausar
                      </button>

                      {/* Excluir */}
                      <button
                        onClick={() => handleDeleteAlert(a.id)}
                        style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.2)', fontSize: '1.1rem', cursor: 'pointer', transition: 'color 0.15s' }}
                        onMouseEnter={(e) => e.currentTarget.style.color = '#ff4a5a'}
                        onMouseLeave={(e) => e.currentTarget.style.color = 'rgba(255,255,255,0.2)'}
                        title="Excluir Alerta"
                      >
                        ✕
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Seção 2: Alertas Disparados */}
          <div className="glass-panel" style={{ padding: '1.5rem 2rem', textAlign: 'left' }}>
            <h3 style={{ margin: '0 0 1rem 0', fontSize: '1.1rem', fontWeight: 800, color: '#ffb300', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              🔔 Disparados recentemente ({triggeredAlerts.length})
            </h3>
            
            {isLoadingAlerts ? (
              <div style={{ padding: '2rem', textAlign: 'center' }}>
                <span className="loading-spinner" style={{ borderTopColor: '#ffb300' }}></span>
              </div>
            ) : triggeredAlerts.length === 0 ? (
              <p style={{ margin: 0, color: 'var(--text-secondary)', fontSize: '0.85rem', padding: '1rem 0' }}>
                Nenhum alerta disparado recentemente.
              </p>
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '1rem' }}>
                {triggeredAlerts.map((a) => (
                  <div key={a.id} style={{ background: 'rgba(255,179,0,0.02)', border: '1px solid rgba(255,179,0,0.15)', borderRadius: '12px', padding: '1.2rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                      <span style={{ fontWeight: 800, fontSize: '1.1rem', color: '#ffb300', display: 'block' }}>{a.ticker}</span>
                      <span style={{ fontSize: '0.7rem', color: '#ffb300', display: 'block', fontWeight: 600, marginBottom: '0.2rem' }}>
                        Disparou {a.triggered_at ? formatDate(a.triggered_at) : ''}
                      </span>
                      <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', display: 'block' }}>
                        Meta era: {a.condition === 'ABOVE' ? '▲ acima de' : '▼ abaixo de'} {formatMoney(a.target_price, a.currency)}
                      </span>
                    </div>

                    <button
                      onClick={() => handleDeleteAlert(a.id)}
                      style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.2)', fontSize: '1.1rem', cursor: 'pointer', transition: 'color 0.15s' }}
                      onMouseEnter={(e) => e.currentTarget.style.color = '#ff4a5a'}
                      onMouseLeave={(e) => e.currentTarget.style.color = 'rgba(255,255,255,0.2)'}
                      title="Excluir Histórico"
                    >
                      ✕
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Seção 3: Alertas Desativados */}
          <div className="glass-panel" style={{ padding: '1.5rem 2rem', textAlign: 'left' }}>
            <h3 style={{ margin: '0 0 1rem 0', fontSize: '1.1rem', fontWeight: 800, color: '#9ca3af', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              ⚪ Pausados / Inativos ({disabledAlerts.length})
            </h3>
            
            {isLoadingAlerts ? (
              <div style={{ padding: '2rem', textAlign: 'center' }}>
                <span className="loading-spinner" style={{ borderTopColor: '#9ca3af' }}></span>
              </div>
            ) : disabledAlerts.length === 0 ? (
              <p style={{ margin: 0, color: 'var(--text-secondary)', fontSize: '0.85rem', padding: '1rem 0' }}>
                Nenhum alerta pausado.
              </p>
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '1rem' }}>
                {disabledAlerts.map((a) => (
                  <div key={a.id} style={{ background: 'rgba(255,255,255,0.005)', border: '1px solid var(--panel-border)', borderRadius: '12px', padding: '1.2rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center', opacity: 0.6 }}>
                    <div>
                      <span style={{ fontWeight: 800, fontSize: '1.1rem', color: '#9ca3af', display: 'block' }}>{a.ticker}</span>
                      <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', display: 'block', marginBottom: '0.5rem' }}>
                        Pausado
                      </span>
                      <span style={{ fontSize: '0.85rem', fontWeight: 700, color: 'var(--text-secondary)' }}>
                        {a.condition === 'ABOVE' ? '▲ Acima de' : '▼ Abaixo de'} {formatMoney(a.target_price, a.currency)}
                      </span>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.8rem' }}>
                      {/* Toggle Switch */}
                      <button
                        onClick={() => handleToggleStatus(a.id)}
                        style={{
                          background: 'rgba(255, 255, 255, 0.05)',
                          border: '1px solid #9ca3af',
                          color: '#9ca3af',
                          borderRadius: '20px',
                          padding: '0.25rem 0.65rem',
                          fontSize: '0.7rem',
                          fontWeight: 700,
                          cursor: 'pointer',
                          transition: 'all 0.2s'
                        }}
                        onMouseEnter={(e) => { e.currentTarget.style.background = '#e2e8f0'; e.currentTarget.style.color = '#000'; }}
                        onMouseLeave={(e) => { e.currentTarget.style.background = 'rgba(255, 255, 255, 0.05)'; e.currentTarget.style.color = '#9ca3af'; }}
                        title="Reativar Alerta"
                      >
                        Ativar
                      </button>

                      {/* Excluir */}
                      <button
                        onClick={() => handleDeleteAlert(a.id)}
                        style={{ background: 'none', border: 'none', color: 'rgba(255,255,255,0.2)', fontSize: '1.1rem', cursor: 'pointer', transition: 'color 0.15s' }}
                        onMouseEnter={(e) => e.currentTarget.style.color = '#ff4a5a'}
                        onMouseLeave={(e) => e.currentTarget.style.color = 'rgba(255,255,255,0.2)'}
                        title="Excluir Alerta"
                      >
                        ✕
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

        </div>

      </div>
    </main>
  );
}
