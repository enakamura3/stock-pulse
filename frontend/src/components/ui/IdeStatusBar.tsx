import React, { useEffect, useState } from 'react';
import { getMarketStatus, MarketStatus } from '@/lib/marketHours';

export interface IdeStatusBarProps {
  wsConnected?: boolean;
  activePortfolioName?: string;
  currency?: string;
  version?: string;
}

export default function IdeStatusBar({
  wsConnected = true,
  activePortfolioName,
  currency = 'BRL',
  version = 'v2.4-term',
}: IdeStatusBarProps) {
  const [market, setMarket] = useState<MarketStatus>(() => getMarketStatus());

  useEffect(() => {
    const interval = setInterval(() => {
      setMarket(getMarketStatus());
    }, 60000);
    return () => clearInterval(interval);
  }, []);

  return (
    <footer className="ide-statusbar" data-testid="ide-statusbar" role="contentinfo">
      <div className="flex-row items-center gap-md">
        {/* WebSocket Connection status */}
        <div className="flex-row items-center gap-xs font-mono">
          <span
            style={{
              width: '6px',
              height: '6px',
              borderRadius: '50%',
              backgroundColor: wsConnected ? 'var(--color-success)' : 'var(--color-danger)',
              boxShadow: wsConnected ? '0 0 5px var(--color-success)' : 'none',
              display: 'inline-block',
            }}
          />
          <span style={{ color: wsConnected ? 'var(--color-success)' : 'var(--color-danger)' }}>
            {wsConnected ? 'LIVE WS' : 'DISCONNECTED'}
          </span>
        </div>

        <span style={{ opacity: 0.3 }}>|</span>

        {/* Market Status */}
        <div className="flex-row items-center gap-xs font-mono" title={market.description}>
          <span
            style={{
              width: '6px',
              height: '6px',
              borderRadius: '50%',
              backgroundColor: market.color,
              display: 'inline-block',
            }}
          />
          <span style={{ color: 'var(--text-secondary)' }}>
            B3: <strong style={{ color: market.color }}>{market.label.toUpperCase()}</strong>
          </span>
        </div>

        {activePortfolioName && (
          <>
            <span style={{ opacity: 0.3 }}>|</span>
            <div className="flex-row items-center gap-xs font-mono text-muted">
              <span>CARTEIRA:</span>
              <span className="text-primary font-semibold">{activePortfolioName.toUpperCase()}</span>
              <span>[{currency}]</span>
            </div>
          </>
        )}
      </div>

      <div className="flex-row items-center gap-md font-mono text-muted">
        <span className="hide-mobile">TERMINAL_MODE: <span className="text-accent">ACTIVE</span></span>
        <span style={{ opacity: 0.3 }} className="hide-mobile">|</span>
        <span>stock-pulse {version}</span>
      </div>
    </footer>
  );
}
