import { render, screen, fireEvent } from '@testing-library/react';
import Modals from './Modals';
import React from 'react';
import { ThemeProvider } from '../ThemeProvider';

describe('Modals Component', () => {
  it('renders open modals and triggers ESC key close', () => {
    const setShowPortfolioModalMock = vi.fn();
    const setShowTxModalMock = vi.fn();
    const setShowFIModalMock = vi.fn();
    const setShowFIEditModalMock = vi.fn();

    render(
      <ThemeProvider>
        <Modals
          showPortfolioModal={true}
          setShowPortfolioModal={setShowPortfolioModalMock}
          showTxModal={true}
          setShowTxModal={setShowTxModalMock}
          txType="CONVERSION"
          positions={[{ asset_id: 'a1', ticker: 'ORIG11.SA', name: 'Fundo Origem', type: 'FII', currency: 'BRL', quantity: 100, average_price: 10, total_cost: 1000 }]}
          convSourceTicker="ORIG11.SA"
          setConvSourceTicker={vi.fn()}
          convSourceQty={100}
          setConvSourceQty={vi.fn()}
          showFIModal={true}
          setShowFIModal={setShowFIModalMock}
          showFIEditModal={true}
          setShowFIEditModal={setShowFIEditModalMock}
        />
      </ThemeProvider>
    );

    // Trigger ESC keydown
    fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' });

    expect(setShowPortfolioModalMock).toHaveBeenCalledWith(false);
    expect(setShowTxModalMock).toHaveBeenCalledWith(false);
    expect(setShowFIModalMock).toHaveBeenCalledWith(false);
    expect(setShowFIEditModalMock).toHaveBeenCalledWith(false);
  });
});
