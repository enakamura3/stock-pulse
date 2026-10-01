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

    // Other key doesn't trigger onClose
    fireEvent.keyDown(window, { key: 'Enter', code: 'Enter' });
    expect(onCloseMock).toHaveBeenCalledTimes(1);
  });

  it('renders form inputs, handles condition change, target price change, submit and cancel button', () => {
    const onCloseMock = vi.fn();
    const onSubmitMock = vi.fn((e) => e.preventDefault());
    const onTargetPriceChangeMock = vi.fn();
    const onConditionChangeMock = vi.fn();

    render(
      <CreateAlertModal
        activeQuote={mockQuote}
        alertTargetPrice="40.00"
        alertCondition="ABOVE"
        isCreatingAlert={false}
        alertErrorMsg={null}
        alertSuccessMsg={null}
        onTargetPriceChange={onTargetPriceChangeMock}
        onConditionChange={onConditionChangeMock}
        onSubmit={onSubmitMock}
        onClose={onCloseMock}
      />
    );

    const select = screen.getByRole('combobox');
    fireEvent.change(select, { target: { value: 'BELOW' } });
    expect(onConditionChangeMock).toHaveBeenCalledWith('BELOW');

    const input = screen.getByPlaceholderText('Ex: 38.50');
    fireEvent.change(input, { target: { value: '42.00' } });
    expect(onTargetPriceChangeMock).toHaveBeenCalledWith('42.00');

    const submitBtn = screen.getByRole('button', { name: /Salvar Alerta/i });
    fireEvent.click(submitBtn);
    expect(onSubmitMock).toHaveBeenCalled();

    const cancelBtn = screen.getByRole('button', { name: /Cancelar/i });
    fireEvent.click(cancelBtn);
    expect(onCloseMock).toHaveBeenCalled();
  });

  it('renders error message and success message when provided', () => {
    const { rerender } = render(
      <CreateAlertModal
        activeQuote={mockQuote}
        alertTargetPrice="40.00"
        alertCondition="ABOVE"
        isCreatingAlert={true}
        alertErrorMsg="Preço inválido"
        alertSuccessMsg={null}
        onTargetPriceChange={vi.fn()}
        onConditionChange={vi.fn()}
        onSubmit={vi.fn()}
        onClose={vi.fn()}
      />
    );

    expect(screen.getByText(/⚠️ Preço inválido/)).toBeInTheDocument();
    expect(screen.getByText('Criando...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Criando.../i })).toBeDisabled();

    rerender(
      <CreateAlertModal
        activeQuote={mockQuote}
        alertTargetPrice="40.00"
        alertCondition="ABOVE"
        isCreatingAlert={false}
        alertErrorMsg={null}
        alertSuccessMsg="Alerta criado com sucesso!"
        onTargetPriceChange={vi.fn()}
        onConditionChange={vi.fn()}
        onSubmit={vi.fn()}
        onClose={vi.fn()}
      />
    );

    expect(screen.getByText(/🎉 Alerta criado com sucesso!/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Salvar Alerta/i })).not.toBeInTheDocument();
  });
});
