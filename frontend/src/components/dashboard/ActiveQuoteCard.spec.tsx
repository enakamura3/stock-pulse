import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import ActiveQuoteCard from './ActiveQuoteCard';
import { Quote } from './types';

describe('ActiveQuoteCard Component', () => {
  const mockFormatMoney = vi.fn((val: number, cur: string) => `R$ ${val.toFixed(2)}`);
  const formatPercentage = vi.fn((val: number) => `${val.toFixed(2)}%`);
  const onToggleFavorite = vi.fn();
  const onOpenAlertModal = vi.fn();
  const onRefreshQuote = vi.fn();

  const baseQuote: Quote = {
    symbol: 'PETR4',
    name: 'Petrobras PN',
    price: 38.5,
    change: 0.75,
    change_percent: 1.98,
    open: 37.8,
    high: 39.0,
    low: 37.5,
    previous_close: 37.75,
    volume: 1234567,
    market_cap: 500000000,
    currency: 'BRL',
    updated_at: '2024-03-15T12:00:00Z',
  };

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state when isLoadingQuote is true', () => {
    render(
      <ActiveQuoteCard
        activeQuote={null}
        isLoadingQuote={true}
        quoteError={null}
        activeFavorited={false}
        isAddingToWatchlist={false}
        cacheStatus={null}
        priceFlashing={{}}
        onToggleFavorite={onToggleFavorite}
        onOpenAlertModal={onOpenAlertModal}
        onRefreshQuote={onRefreshQuote}
        formatMoney={mockFormatMoney}
        formatPercentage={formatPercentage}
      />
    );

    expect(screen.getByText('Carregando dados em tempo real...')).toBeInTheDocument();
  });

  it('renders error state when quoteError is present and not loading', () => {
    render(
      <ActiveQuoteCard
        activeQuote={null}
        isLoadingQuote={false}
        quoteError="Ativo não encontrado"
        activeFavorited={false}
        isAddingToWatchlist={false}
        cacheStatus={null}
        priceFlashing={{}}
        onToggleFavorite={onToggleFavorite}
        onOpenAlertModal={onOpenAlertModal}
        onRefreshQuote={onRefreshQuote}
        formatMoney={mockFormatMoney}
        formatPercentage={formatPercentage}
      />
    );

    expect(screen.getByText(/Ativo não encontrado/)).toBeInTheDocument();
  });

  it('renders empty prompt when quote is null and no error/loading', () => {
    render(
      <ActiveQuoteCard
        activeQuote={null}
        isLoadingQuote={false}
        quoteError={null}
        activeFavorited={false}
        isAddingToWatchlist={false}
        cacheStatus={null}
        priceFlashing={{}}
        onToggleFavorite={onToggleFavorite}
        onOpenAlertModal={onOpenAlertModal}
        onRefreshQuote={onRefreshQuote}
        formatMoney={mockFormatMoney}
        formatPercentage={formatPercentage}
      />
    );

    expect(screen.getByText(/Pesquise um ativo no campo superior/)).toBeInTheDocument();
  });

  it('renders positive quote with cache hit and flashing up', () => {
    render(
      <ActiveQuoteCard
        activeQuote={baseQuote}
        isLoadingQuote={false}
        quoteError={null}
        activeFavorited={true}
        isAddingToWatchlist={false}
        cacheStatus="hit"
        priceFlashing={{ PETR4: 'up' }}
        onToggleFavorite={onToggleFavorite}
        onOpenAlertModal={onOpenAlertModal}
        onRefreshQuote={onRefreshQuote}
        formatMoney={mockFormatMoney}
        formatPercentage={formatPercentage}
      />
    );

    expect(screen.getByText('PETR4')).toBeInTheDocument();
    expect(screen.getByText('Petrobras PN')).toBeInTheDocument();
    expect(screen.getByText('⚡ Redis Cache')).toBeInTheDocument();
    expect(screen.getByText(/▲/)).toBeInTheDocument();

    const starBtn = screen.getByTitle('Remover dos Favoritos');
    expect(starBtn).toHaveTextContent('★');
    fireEvent.mouseEnter(starBtn);
    fireEvent.mouseLeave(starBtn);
    fireEvent.click(starBtn);
    expect(onToggleFavorite).toHaveBeenCalled();

    const alertBtn = screen.getByText('🔔 Criar Alerta');
    fireEvent.click(alertBtn);
    expect(onOpenAlertModal).toHaveBeenCalled();

    const refreshBtn = screen.getByText('🔄 Atualizar');
    fireEvent.click(refreshBtn);
    expect(onRefreshQuote).toHaveBeenCalledWith('PETR4', true);
  });

  it('renders negative quote with cache miss and flashing down', () => {
    const negativeQuote: Quote = {
      ...baseQuote,
      symbol: 'VALE3',
      name: 'Vale ON',
      change: -1.25,
      change_percent: -2.1,
    };

    render(
      <ActiveQuoteCard
        activeQuote={negativeQuote}
        isLoadingQuote={false}
        quoteError={null}
        activeFavorited={false}
        isAddingToWatchlist={true}
        cacheStatus="miss"
        priceFlashing={{ VALE3: 'down' }}
        onToggleFavorite={onToggleFavorite}
        onOpenAlertModal={onOpenAlertModal}
        onRefreshQuote={onRefreshQuote}
        formatMoney={mockFormatMoney}
        formatPercentage={formatPercentage}
      />
    );

    expect(screen.getByText('VALE3')).toBeInTheDocument();
    expect(screen.getByText('🌐 Yahoo API')).toBeInTheDocument();
    expect(screen.getByText(/▼/)).toBeInTheDocument();

    const starBtn = screen.getByTitle('Adicionar aos Favoritos');
    expect(starBtn).toHaveTextContent('☆');
    expect(starBtn).toBeDisabled();
  });

  it('renders neutral price flashing styling when priceFlashing has no entry for symbol', () => {
    render(
      <ActiveQuoteCard
        activeQuote={baseQuote}
        isLoadingQuote={false}
        quoteError={null}
        activeFavorited={false}
        isAddingToWatchlist={false}
        cacheStatus={null}
        priceFlashing={{}}
        onToggleFavorite={onToggleFavorite}
        onOpenAlertModal={onOpenAlertModal}
        onRefreshQuote={onRefreshQuote}
        formatMoney={mockFormatMoney}
        formatPercentage={formatPercentage}
      />
    );

    expect(screen.getByText('PETR4')).toBeInTheDocument();
  });
});
