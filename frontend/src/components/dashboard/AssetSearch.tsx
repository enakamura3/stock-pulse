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
              onClick={() => onSelectAsset(item.symbol)}
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
  );
}
