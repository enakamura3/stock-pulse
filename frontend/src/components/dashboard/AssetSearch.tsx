import React from 'react';
import { SearchResult } from './types';

interface AssetSearchProps {
  searchQuery: string;
  searchResults: SearchResult[];
  isSearching: boolean;
  showDropdown: boolean;
  onSearchChange: (query: string) => void;
  onFocus: () => void;
  onSelectAsset: (symbol: string) => void;
}

export default function AssetSearch({
  searchQuery,
  searchResults,
  isSearching,
  showDropdown,
  onSearchChange,
  onFocus,
  onSelectAsset,
}: AssetSearchProps) {
  return (
    <div style={{ position: 'relative' }}>
      <div className="form-group" style={{ margin: 0 }}>
        <input
          className="form-input"
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          onFocus={onFocus}
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
        <div
          className="card"
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            width: '100%',
            marginTop: '0.4rem',
            zIndex: 10,
            padding: '0.35rem',
            textAlign: 'left',
            maxHeight: '280px',
            overflowY: 'auto',
            boxShadow: '0 12px 32px rgba(0, 0, 0, 0.7)',
            borderRadius: '4px',
            borderColor: 'var(--panel-border-strong)',
          }}
        >
          {searchResults.map((item) => (
            <div
              key={item.symbol}
              onClick={() => onSelectAsset(item.symbol)}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: '0.55rem 0.75rem',
                borderRadius: '4px',
                cursor: 'pointer',
                transition: 'background-color 0.15s ease',
              }}
              className="sidebar-link"
            >
              <div>
                <span className="font-mono" style={{ fontWeight: 700, color: 'var(--accent-color)', marginRight: '0.75rem', fontSize: '0.9rem' }}>
                  {item.symbol}
                </span>
                <span style={{ fontSize: '0.82rem', opacity: 0.85 }}>{item.name}</span>
              </div>
              <span className="badge badge-neutral font-mono" style={{ fontSize: '0.65rem', textTransform: 'uppercase' }}>
                {item.exchange}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
