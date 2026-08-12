import { render, screen, fireEvent } from '@testing-library/react';
import CreateAlertModal from './CreateAlertModal';
import React from 'react';
import { Quote } from './types';

describe('CreateAlertModal Component', () => {
  const mockQuote: Quote = {
    symbol: 'PETR4',
    name: 'Petrobras PN',
    price: 35.5,
    change: 1.2,
    changePercent: 3.5,
  };

  it('renders modal and triggers onClose on ESC key press', () => {
    const onCloseMock = vi.fn();
    const onSubmitMock = vi.fn();

    render(
      <CreateAlertModal
        activeQuote={mockQuote}
        alertTargetPrice="40.00"
        alertCondition="ABOVE"
        isCreatingAlert={false}
        alertErrorMsg={null}
        alertSuccessMsg={null}
        onTargetPriceChange={vi.fn()}
        onConditionChange={vi.fn()}
        onSubmit={onSubmitMock}
        onClose={onCloseMock}
      />
    );

    expect(screen.getByText(/Criar Alerta de Preço/i)).toBeInTheDocument();
    expect(screen.getByText(/PETR4/i)).toBeInTheDocument();

    // Trigger ESC keydown
    fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' });
    expect(onCloseMock).toHaveBeenCalledTimes(1);
  });
});
