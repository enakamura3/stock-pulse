import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import AssetSearch from './AssetSearch';
import { SearchResult } from './types';

describe('AssetSearch Component', () => {
  const onSearchChange = vi.fn();
  const onFocus = vi.fn();
  const onSelectAsset = vi.fn();

  const mockResults: SearchResult[] = [
    { symbol: 'PETR4', name: 'Petrobras PN', exchange: 'SAO', type: 'EQUITY' },
    { symbol: 'AAPL', name: 'Apple Inc.', exchange: 'NMS', type: 'EQUITY' },
  ];

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders input field with initial query and handles input and focus', () => {
    const { container } = render(
      <AssetSearch
        searchQuery="PETR"
        searchResults={[]}
        isSearching={false}
        showDropdown={false}
        onSearchChange={onSearchChange}
        onFocus={onFocus}
        onSelectAsset={onSelectAsset}
      />
    );

    const input = screen.getByPlaceholderText(/Pesquise ativos.../);
    expect(input).toHaveValue('PETR');

    fireEvent.change(input, { target: { value: 'VALE' } });
    expect(onSearchChange).toHaveBeenCalledWith('VALE');

    fireEvent.focus(input);
    expect(onFocus).toHaveBeenCalled();

    expect(container.querySelector('.loading-spinner')).not.toBeInTheDocument();
  });

  it('shows loading spinner when isSearching is true', () => {
    const { container } = render(
      <AssetSearch
        searchQuery="PETR"
        searchResults={[]}
        isSearching={true}
        showDropdown={false}
        onSearchChange={onSearchChange}
        onFocus={onFocus}
        onSelectAsset={onSelectAsset}
      />
    );

    expect(container.querySelector('.loading-spinner')).toBeInTheDocument();
  });

  it('does not render dropdown when showDropdown is false or results empty', () => {
    const { rerender } = render(
      <AssetSearch
        searchQuery="PETR"
        searchResults={mockResults}
        isSearching={false}
        showDropdown={false}
        onSearchChange={onSearchChange}
        onFocus={onFocus}
        onSelectAsset={onSelectAsset}
      />
    );

    expect(screen.queryByText('Petrobras PN')).not.toBeInTheDocument();

    rerender(
      <AssetSearch
        searchQuery="PETR"
        searchResults={[]}
        showDropdown={true}
        isSearching={false}
        onSearchChange={onSearchChange}
        onFocus={onFocus}
        onSelectAsset={onSelectAsset}
      />
    );

    expect(screen.queryByText('Petrobras PN')).not.toBeInTheDocument();
  });

  it('renders dropdown and selects asset on click', () => {
    render(
      <AssetSearch
        searchQuery="PETR"
        searchResults={mockResults}
        isSearching={false}
        showDropdown={true}
        onSearchChange={onSearchChange}
        onFocus={onFocus}
        onSelectAsset={onSelectAsset}
      />
    );

    expect(screen.getByText('PETR4')).toBeInTheDocument();
    expect(screen.getByText('Petrobras PN')).toBeInTheDocument();
    expect(screen.getByText('SAO')).toBeInTheDocument();

    const appleOption = screen.getByText('AAPL');
    fireEvent.click(appleOption);
    expect(onSelectAsset).toHaveBeenCalledWith('AAPL');
  });
});
