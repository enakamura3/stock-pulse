import React from 'react';
import { Quote } from './types';

interface ActiveQuoteCardProps {
  activeQuote: Quote | null;
  isLoadingQuote: boolean;
  quoteError: string | null;
  activeFavorited: boolean;
  isAddingToWatchlist: boolean;
  cacheStatus: 'hit' | 'miss' | 'updating' | null;
  priceFlashing: Record<string, 'up' | 'down'>;
  onToggleFavorite: () => void;
  onOpenAlertModal: () => void;
  onRefreshQuote: (symbol: string, isRefresh?: boolean) => void;
  formatMoney: (val: number, currency: string) => string;
  formatPercentage: (val: number) => string;
}

export default function ActiveQuoteCard({
  activeQuote,
  isLoadingQuote,
  quoteError,
  activeFavorited,
  isAddingToWatchlist,
  cacheStatus,
  priceFlashing,
  onToggleFavorite,
  onOpenAlertModal,
  onRefreshQuote,
  formatMoney,
  formatPercentage,
}: ActiveQuoteCardProps) {
  return (
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
                    onClick={onToggleFavorite}
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
                  onClick={onOpenAlertModal}
                  style={{ padding: '0.4rem 1rem', fontSize: '0.8rem', background: 'linear-gradient(135deg, #ff9800, #f57c00)', border: 'none', color: '#000', fontWeight: 700 }}
                >
                  🔔 Criar Alerta
                </button>
                <button
                  className="primary-button"
                  onClick={() => onRefreshQuote(activeQuote.symbol, true)}
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
  );
}
