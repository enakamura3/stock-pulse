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
    <div className="card" style={{ minHeight: '260px', display: 'flex', flexDirection: 'column', justifyContent: 'center', textAlign: 'left', padding: '1.5rem', borderRadius: '6px' }}>
      {isLoadingQuote ? (
        <div style={{ textAlign: 'center', width: '100%' }}>
          <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 35, height: 35 }}></span>
          <p className="font-mono" style={{ marginTop: '1rem', color: 'var(--text-secondary)', fontSize: '0.82rem' }}>Carregando dados em tempo real...</p>
        </div>
      ) : quoteError ? (
        <div className="alert-error font-mono" style={{ margin: 0, width: '100%', fontSize: '0.85rem' }}>
          ⚠️ {quoteError}
        </div>
      ) : activeQuote ? (
        <div style={{ width: '100%' }}>
          {/* Nome, Ticker e Ícone Estrela */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.75rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
              <div>
                <h2 className="font-mono" style={{ fontSize: '2rem', margin: 0, fontWeight: 700, color: 'var(--text-primary)', display: 'flex', alignItems: 'center', gap: '10px' }}>
                  {activeQuote.symbol}
                  
                  {/* ÍCONE ESTRELA PARA FAVORITAR DENTRO DA WATCHLIST ATIVA */}
                  <button
                    onClick={onToggleFavorite}
                    disabled={isAddingToWatchlist}
                    style={{
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      fontSize: '1.6rem',
                      color: activeFavorited ? 'var(--color-warning)' : 'var(--text-muted)',
                      transition: 'transform 0.15s ease, color 0.15s ease',
                      padding: 0,
                      lineHeight: 1,
                    }}
                    onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.2)'}
                    onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
                    title={activeFavorited ? 'Remover dos Favoritos' : 'Adicionar aos Favoritos'}
                  >
                    {activeFavorited ? '★' : '☆'}
                  </button>
                </h2>
                <p style={{ margin: '0.1rem 0 0 0', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                  {activeQuote.name}
                </p>
              </div>
            </div>

            {/* Refresh e Badge */}
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '0.35rem' }}>
              <div style={{ display: 'flex', gap: '0.5rem' }}>
                <button
                  className="primary-button"
                  onClick={onOpenAlertModal}
                  style={{ padding: '0.35rem 0.85rem', fontSize: '0.78rem' }}
                >
                  🔔 Criar Alerta
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => onRefreshQuote(activeQuote.symbol, true)}
                  style={{ padding: '0.35rem 0.85rem', fontSize: '0.78rem' }}
                >
                  🔄 Atualizar
                </button>
              </div>
              {cacheStatus === 'hit' && (
                <span className="badge badge-neutral font-mono" style={{ fontSize: '0.68rem', color: 'var(--accent-color)', borderColor: 'rgba(238, 96, 24, 0.3)' }}>
                  ⚡ Redis Cache
                </span>
              )}
              {cacheStatus === 'miss' && (
                <span className="badge badge-neutral font-mono" style={{ fontSize: '0.68rem', color: 'var(--color-warning)', borderColor: 'rgba(var(--warning-rgb), 0.3)' }}>
                  🌐 Yahoo API
                </span>
              )}
            </div>
          </div>

          {/* Preço e Variação */}
          <div style={{ display: 'flex', alignItems: 'baseline', gap: '1rem', marginBottom: '1.25rem' }}>
            <span
              className="font-mono font-bold"
              style={{ 
                fontSize: '2.5rem', 
                color: priceFlashing[activeQuote.symbol] === 'up' ? 'var(--color-success)' : priceFlashing[activeQuote.symbol] === 'down' ? 'var(--color-danger)' : 'var(--text-primary)', 
                textShadow: priceFlashing[activeQuote.symbol] === 'up' ? '0 0 12px rgba(var(--success-rgb), 0.5)' : priceFlashing[activeQuote.symbol] === 'down' ? '0 0 12px rgba(var(--danger-rgb), 0.5)' : 'none',
                transition: 'all 0.2s ease',
                letterSpacing: '-0.02em',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {formatMoney(activeQuote.price, activeQuote.currency)}
            </span>
            <span
              className={`badge ${activeQuote.change >= 0 ? 'badge-success' : 'badge-danger'} font-mono`}
              style={{
                fontSize: '0.95rem',
                fontWeight: 700,
                padding: '0.25rem 0.6rem',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {activeQuote.change >= 0 ? '▲' : '▼'} {formatPercentage(activeQuote.change_percent)}
            </span>
          </div>

          {/* Grid Secundária */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '0.75rem' }}>
            <div style={{ background: 'var(--input-bg)', padding: '0.65rem 0.85rem', borderRadius: '4px', border: '1px solid var(--panel-border)' }}>
              <span className="font-mono text-muted" style={{ display: 'block', fontSize: '0.68rem', textTransform: 'uppercase', marginBottom: '0.15rem', fontWeight: 600 }}>
                Mínima
              </span>
              <span className="font-mono font-bold" style={{ fontSize: '0.95rem', fontVariantNumeric: 'tabular-nums' }}>
                {formatMoney(activeQuote.low, activeQuote.currency)}
              </span>
            </div>
            <div style={{ background: 'var(--input-bg)', padding: '0.65rem 0.85rem', borderRadius: '4px', border: '1px solid var(--panel-border)' }}>
              <span className="font-mono text-muted" style={{ display: 'block', fontSize: '0.68rem', textTransform: 'uppercase', marginBottom: '0.15rem', fontWeight: 600 }}>
                Máxima
              </span>
              <span className="font-mono font-bold" style={{ fontSize: '0.95rem', fontVariantNumeric: 'tabular-nums' }}>
                {formatMoney(activeQuote.high, activeQuote.currency)}
              </span>
            </div>
            <div style={{ background: 'var(--input-bg)', padding: '0.65rem 0.85rem', borderRadius: '4px', border: '1px solid var(--panel-border)' }}>
              <span className="font-mono text-muted" style={{ display: 'block', fontSize: '0.68rem', textTransform: 'uppercase', marginBottom: '0.15rem', fontWeight: 600 }}>
                Volume
              </span>
              <span className="font-mono font-bold" style={{ fontSize: '0.95rem', fontVariantNumeric: 'tabular-nums' }}>
                {new Intl.NumberFormat('pt-BR').format(activeQuote.volume)}
              </span>
            </div>
          </div>
        </div>
      ) : (
        <div style={{ textAlign: 'center', width: '100%', color: 'var(--text-secondary)' }}>
          <p className="font-mono text-muted" style={{ margin: 0, fontSize: '0.95rem' }}>
            🔎 Pesquise um ativo no campo superior ou selecione um dos seus favoritos ao lado para ver a cotação.
          </p>
        </div>
      )}
    </div>
  );
}
