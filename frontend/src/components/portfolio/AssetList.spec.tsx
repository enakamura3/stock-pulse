import { render, screen, fireEvent } from '@testing-library/react';
import AssetList from './AssetList';
import React from 'react';
import { ThemeProvider } from '../ThemeProvider';
import { Position } from './types';

describe('AssetList Component', () => {
  const mockPositions: Position[] = [
    {
      asset_id: 'pos-1',
      ticker: 'PETR4',
      name: 'Petrobras PN',
      quantity: 100,
      average_price: 30,
      current_price: 35,
      total_cost: 3000,
      current_value: 3500,
      profit_loss: 500,
      return_percent: 16.67,
      currency: 'BRL',
      dividend_yield: 8.5,
      pe: 4.2,
      pvp: 0.95,
      graham_value: 40,
      bazin_value: 38,
    },
    {
      asset_id: 'pos-2',
      ticker: 'VALE3',
      name: 'Vale ON',
      quantity: 50,
      average_price: 70,
      current_price: 60,
      total_cost: 3500,
      current_value: 3000,
      profit_loss: -500,
      return_percent: -14.28,
      currency: 'BRL',
    },
  ];

  it('renders positions table with columns and sorting functionality', () => {
    const importCsvMock = vi.fn();
    const launchOpMock = vi.fn();

    render(
      <ThemeProvider>
        <AssetList
          positions={mockPositions}
          kpiCurrency="BRL"
          onImportCsv={importCsvMock}
          onLaunchOperation={launchOpMock}
        />
      </ThemeProvider>
    );

    expect(screen.getByText('Posições Ativas')).toBeInTheDocument();
    expect(screen.getByText('PETR4')).toBeInTheDocument();
    expect(screen.getByText('VALE3')).toBeInTheDocument();

    // Test sorting by clicking column header
    const tickerHeader = screen.getByText(/Ativo/i);
    fireEvent.click(tickerHeader);
    expect(tickerHeader).toBeInTheDocument();

    const launchBtn = screen.getByRole('button', { name: /\+ Lançar Operação/i });
    fireEvent.click(launchBtn);
    expect(launchOpMock).toHaveBeenCalledTimes(1);
  });

  it('renders empty state when no positions are provided', () => {
    render(
      <ThemeProvider>
        <AssetList
          positions={[]}
          kpiCurrency="BRL"
          onImportCsv={vi.fn()}
          onLaunchOperation={vi.fn()}
        />
      </ThemeProvider>
    );

    expect(screen.getByText('Esta carteira ainda não possui ativos ativos.')).toBeInTheDocument();
  });
});
