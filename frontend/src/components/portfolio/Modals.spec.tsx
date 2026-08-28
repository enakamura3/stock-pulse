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
