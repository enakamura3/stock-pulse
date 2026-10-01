import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import WatchlistSidebar from './WatchlistSidebar';
import { Watchlist } from './types';

describe('WatchlistSidebar Component', () => {
  const onSelectWatchlist = vi.fn();
  const onDeleteActiveWatchlist = vi.fn();
  const onCreateWatchlist = vi.fn((e) => e.preventDefault());
  const onNewWatchlistNameChange = vi.fn();
  const onSelectAsset = vi.fn();
  const onRemoveFromSidebar = vi.fn();
  const formatMoney = vi.fn((val: number, cur: string) => `R$ ${val.toFixed(2)}`);
  const formatPercentage = vi.fn((val: number) => `${val.toFixed(2)}%`);

  const mockWatchlists: Watchlist[] = [
    {
      id: 'wl-1',
      name: 'Favoritos',
      items: [
        {
          ticker: 'PETR4',
          name: 'Petróleo Brasileiro S.A.',
          price: 35.0,
          currency: 'BRL',
          change: 0.5,
          change_percent: 1.45,
          graham_value: 40.0, // price < graham_value -> DESC
        },
        {
          ticker: 'VALE3',
          name: 'Vale S.A.',
          price: 65.0,
          currency: 'BRL',
          change: -1.2,
          change_percent: -1.8,
          graham_value: 50.0, // price >= graham_value -> CARA
        },
        {
          ticker: 'ITUB4',
          name: 'Itaú Unibanco',
          price: 30.0,
          currency: 'BRL',
          change: 0,
          // change_percent undefined
        },
        {
          ticker: 'BBDC4',
          name: 'Banco Bradesco',
          currency: 'BRL',
          // price undefined -> renders '--'
        },
      ],
    },
    {
      id: 'wl-2',
      name: 'Dividendos',
      items: [],
    },
  ];

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders watchlist tabs, delete button when multiple lists, and handles switching and deleting', () => {
    render(
      <WatchlistSidebar
        watchlists={mockWatchlists}
        activeWatchlistId="wl-1"
        activeWL={mockWatchlists[0]}
        newWatchlistName=""
        isCreatingList={false}
        priceFlashing={{ PETR4: 'up', VALE3: 'down' }}
        onSelectWatchlist={onSelectWatchlist}
        onDeleteActiveWatchlist={onDeleteActiveWatchlist}
        onCreateWatchlist={onCreateWatchlist}
        onNewWatchlistNameChange={onNewWatchlistNameChange}
        onSelectAsset={onSelectAsset}
        onRemoveFromSidebar={onRemoveFromSidebar}
        formatMoney={formatMoney}
        formatPercentage={formatPercentage}
      />
    );

    // Watchlist tabs
    expect(screen.getByText('Favoritos')).toBeInTheDocument();
    expect(screen.getByText('Dividendos')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Dividendos'));
    expect(onSelectWatchlist).toHaveBeenCalledWith('wl-2');

    // Delete active watchlist button
    const deleteBtn = screen.getByTitle('Excluir Lista Atual');
    expect(deleteBtn).toBeInTheDocument();
    fireEvent.click(deleteBtn);
    expect(onDeleteActiveWatchlist).toHaveBeenCalled();

    // Items
    expect(screen.getByText('PETR4')).toBeInTheDocument();
    expect(screen.getByText('DESC')).toBeInTheDocument();
    expect(screen.getByText('CARA')).toBeInTheDocument();
    expect(screen.getByText('--')).toBeInTheDocument();

    // Click item to select
    fireEvent.click(screen.getByText('PETR4'));
    expect(onSelectAsset).toHaveBeenCalledWith('PETR4');

    // Click remove item button
    const removeButtons = screen.getAllByTitle('Remover dos favoritos');
    fireEvent.click(removeButtons[0]);
    expect(onRemoveFromSidebar).toHaveBeenCalledWith(expect.anything(), 'PETR4');
  });

  it('handles create watchlist form interactions', () => {
    render(
      <WatchlistSidebar
        watchlists={mockWatchlists}
        activeWatchlistId="wl-1"
        activeWL={mockWatchlists[0]}
        newWatchlistName="FIIs"
        isCreatingList={false}
        priceFlashing={{}}
        onSelectWatchlist={onSelectWatchlist}
        onDeleteActiveWatchlist={onDeleteActiveWatchlist}
        onCreateWatchlist={onCreateWatchlist}
        onNewWatchlistNameChange={onNewWatchlistNameChange}
        onSelectAsset={onSelectAsset}
        onRemoveFromSidebar={onRemoveFromSidebar}
        formatMoney={formatMoney}
        formatPercentage={formatPercentage}
      />
    );

    const input = screen.getByPlaceholderText('Nova Lista...');
    expect(input).toHaveValue('FIIs');

    fireEvent.change(input, { target: { value: 'Tech' } });
    expect(onNewWatchlistNameChange).toHaveBeenCalledWith('Tech');

    const submitBtn = screen.getByText('+ Criar');
    fireEvent.click(submitBtn);
    expect(onCreateWatchlist).toHaveBeenCalled();
  });

  it('renders empty state when active watchlist has no items, and does not show delete button if single watchlist', () => {
    render(
      <WatchlistSidebar
        watchlists={[mockWatchlists[1]]}
        activeWatchlistId="wl-2"
        activeWL={mockWatchlists[1]}
        newWatchlistName=""
        isCreatingList={true}
        priceFlashing={{}}
        onSelectWatchlist={onSelectWatchlist}
        onDeleteActiveWatchlist={onDeleteActiveWatchlist}
        onCreateWatchlist={onCreateWatchlist}
        onNewWatchlistNameChange={onNewWatchlistNameChange}
        onSelectAsset={onSelectAsset}
        onRemoveFromSidebar={onRemoveFromSidebar}
        formatMoney={formatMoney}
        formatPercentage={formatPercentage}
      />
    );

    expect(screen.queryByTitle('Excluir Lista Atual')).not.toBeInTheDocument();
    expect(screen.getByText(/A lista está vazia/)).toBeInTheDocument();

    const input = screen.getByPlaceholderText('Nova Lista...');
    expect(input).toBeDisabled();
  });
});
