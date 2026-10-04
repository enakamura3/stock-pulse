import { render, screen, act } from '@testing-library/react';
import React from 'react';
import IdeStatusBar from './IdeStatusBar';
import * as marketHours from '@/lib/marketHours';

describe('IdeStatusBar', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders correctly with default props', () => {
    render(<IdeStatusBar />);

    expect(screen.getByTestId('ide-statusbar')).toBeInTheDocument();
    expect(screen.getByText('LIVE WS')).toBeInTheDocument();
    expect(screen.getByText(/stock-pulse/i)).toBeInTheDocument();
    expect(screen.getByText(/B3:/i)).toBeInTheDocument();
  });

  it('renders disconnected state and custom portfolio', () => {
    render(
      <IdeStatusBar
        wsConnected={false}
        activePortfolioName="FIIs & Dividendos"
        currency="USD"
        version="v3.0-custom"
      />
    );

    expect(screen.getByText('DISCONNECTED')).toBeInTheDocument();
    expect(screen.getByText('FIIS & DIVIDENDOS')).toBeInTheDocument();
    expect(screen.getByText('[USD]')).toBeInTheDocument();
    expect(screen.getByText('stock-pulse v3.0-custom')).toBeInTheDocument();
  });

  it('updates market status on interval trigger', () => {
    const spy = vi.spyOn(marketHours, 'getMarketStatus');
    render(<IdeStatusBar />);

    expect(spy).toHaveBeenCalledTimes(1);

    act(() => {
      vi.advanceTimersByTime(60000);
    });

    expect(spy).toHaveBeenCalledTimes(2);
  });
});
