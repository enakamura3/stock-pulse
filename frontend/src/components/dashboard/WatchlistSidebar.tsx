import React from 'react';
import { Watchlist } from './types';

interface WatchlistSidebarProps {
  watchlists: Watchlist[];
  activeWatchlistId: string;
  activeWL?: Watchlist;
  newWatchlistName: string;
  isCreatingList: boolean;
  priceFlashing: Record<string, 'up' | 'down'>;
  onSelectWatchlist: (id: string) => void;
  onDeleteActiveWatchlist: () => void;
  onCreateWatchlist: (e: React.FormEvent) => void;
  onNewWatchlistNameChange: (name: string) => void;
  onSelectAsset: (symbol: string) => void;
  onRemoveFromSidebar: (e: React.MouseEvent, ticker: string) => void;
  formatMoney: (val: number, currency: string) => string;
  formatPercentage: (val: number) => string;
}

export default function WatchlistSidebar({
  watchlists,
  activeWatchlistId,
  activeWL,
  newWatchlistName,
  isCreatingList,
  priceFlashing,
  onSelectWatchlist,
  onDeleteActiveWatchlist,
  onCreateWatchlist,
  onNewWatchlistNameChange,
  onSelectAsset,
  onRemoveFromSidebar,
  formatMoney,
  formatPercentage,
}: WatchlistSidebarProps) {
  return (
    <div style={{ flex: '1 1 320px', display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
      <div className="card" style={{ padding: '1.25rem', textAlign: 'left', display: 'flex', flexDirection: 'column', height: '100%' }}>
        
        {/* Seletor de Watchlists */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h3 className="font-mono" style={{ margin: 0, fontSize: '0.95rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.04em', color: 'var(--accent-color)' }}>
            [WL] Listas de Ativos
          </h3>
          
          {/* Botão de excluir watchlist ativa */}
          {activeWL && watchlists.length > 1 && (
            <button
              onClick={onDeleteActiveWatchlist}
              className="font-mono"
              style={{
                background: 'none',
                border: 'none',
                color: 'var(--color-danger)',
                cursor: 'pointer',
                fontSize: '0.74rem',
                fontWeight: 600,
              }}
              title="Excluir Lista Atual"
            >
              Excluir Lista
            </button>
          )}
        </div>

        {/* Abas das Watchlists */}
        <div style={{ display: 'flex', gap: '0.35rem', overflowX: 'auto', paddingBottom: '0.4rem', marginBottom: '0.85rem', borderBottom: '1px solid var(--panel-border)' }}>
          {watchlists.map((wl) => (
            <button
              key={wl.id}
              onClick={() => onSelectWatchlist(wl.id)}
              className="font-mono"
              style={{
                padding: '0.3rem 0.65rem',
                fontSize: '0.76rem',
                borderRadius: '4px',
                border: '1px solid',
                borderColor: activeWatchlistId === wl.id ? 'var(--accent-color)' : 'var(--panel-border)',
                background: activeWatchlistId === wl.id ? 'rgba(238, 96, 24, 0.1)' : 'transparent',
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
        <form onSubmit={onCreateWatchlist} style={{ display: 'flex', gap: '0.4rem', marginBottom: '1rem' }}>
          <input
            className="form-input font-mono"
            type="text"
            value={newWatchlistName}
            onChange={(e) => onNewWatchlistNameChange(e.target.value)}
            placeholder="Nova Lista..."
            required
            disabled={isCreatingList}
            style={{ padding: '0.4rem 0.7rem', fontSize: '0.78rem', borderRadius: '4px' }}
          />
          <button
            className="primary-button font-mono"
            type="submit"
            disabled={isCreatingList}
            style={{ padding: '0.4rem 0.8rem', fontSize: '0.78rem', whiteSpace: 'nowrap' }}
          >
            + Criar
          </button>
        </form>

        {/* Listagem de itens da Watchlist Ativa */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', maxHeight: '350px', overflowY: 'auto' }}>
          {activeWL?.items && activeWL.items.length > 0 ? (
            activeWL.items.map((item) => {
              const wlPos = item.change !== undefined && item.change >= 0;
              return (
                <div
                  key={item.ticker}
                  onClick={() => onSelectAsset(item.ticker)}
                  className="sidebar-link"
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '0.65rem 0.75rem',
                    borderRadius: '4px',
                    background: 'var(--input-bg)',
                    border: '1px solid var(--panel-border)',
                    cursor: 'pointer',
                    transition: 'all 0.15s ease',
                  }}
                >
                  <div>
                    <span className="font-mono" style={{ display: 'block', fontWeight: 700, color: 'var(--text-primary)', fontSize: '0.85rem' }}>
                      {item.ticker}
                    </span>
                    <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', display: 'block', maxWidth: '140px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {item.name}
                    </span>
                  </div>

                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.65rem' }}>
                    {item.price !== undefined ? (
                      <div style={{ textAlign: 'right' }}>
                        <span
                          className="font-mono font-bold"
                          style={{ 
                            display: 'block', 
                            fontSize: '0.85rem', 
                            color: priceFlashing[item.ticker] === 'up' ? 'var(--color-success)' : priceFlashing[item.ticker] === 'down' ? 'var(--color-danger)' : 'var(--text-primary)', 
                            textShadow: priceFlashing[item.ticker] === 'up' ? '0 0 8px rgba(var(--success-rgb), 0.5)' : priceFlashing[item.ticker] === 'down' ? '0 0 8px rgba(var(--danger-rgb), 0.5)' : 'none',
                            transition: 'all 0.15s ease',
                            fontVariantNumeric: 'tabular-nums',
                          }}
                        >
                          {formatMoney(item.price, item.currency)}
                        </span>
                        <span
                          className="font-mono"
                          style={{ display: 'block', fontSize: '0.72rem', fontWeight: 600, color: wlPos ? 'var(--color-success)' : 'var(--color-danger)', fontVariantNumeric: 'tabular-nums' }}
                        >
                          {item.change_percent !== undefined ? formatPercentage(item.change_percent) : ''}
                        </span>
                        {item.graham_value && item.price ? (
                          <span
                            className="badge font-mono"
                            style={{
                              display: 'inline-block',
                              marginTop: '0.15rem',
                              padding: '0.1rem 0.3rem',
                              borderRadius: '3px',
                              fontSize: '0.58rem',
                              fontWeight: 700,
                              backgroundColor: item.price < item.graham_value ? 'var(--color-success-bg)' : 'var(--color-danger-bg)',
                              color: item.price < item.graham_value ? 'var(--color-success)' : 'var(--color-danger)',
                              border: `1px solid rgba(var(${item.price < item.graham_value ? '--success-rgb' : '--danger-rgb'}), 0.3)`
                            }}
                            title={`Graham: ${formatMoney(item.graham_value, item.currency)}`}
                          >
                            {item.price < item.graham_value ? 'DESC' : 'CARA'}
                          </span>
                        ) : null}
                      </div>
                    ) : (
                      <span className="font-mono" style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>--</span>
                    )}

                    {/* Botão de Excluir Item */}
                    <button
                      onClick={(e) => onRemoveFromSidebar(e, item.ticker)}
                      className="btn-close"
                      style={{
                        fontSize: '1rem',
                        padding: '0.2rem',
                      }}
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
  );
}
