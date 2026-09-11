import React, { useState } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FixedIncomeModal, { FixedIncomeModalProps } from './FixedIncomeModal';

function FixedIncomeModalWrapper(props: Partial<FixedIncomeModalProps> = {}) {
  const [fiAmount, setFiAmount] = useState<string | number>(props.fiAmount ?? '');
  const [fiInstitution, setFiInstitution] = useState<string>(props.fiInstitution ?? '');
  const [fiType, setFiType] = useState<string>(props.fiType ?? 'CDB');
  const [fiDebtType, setFiDebtType] = useState<string>(props.fiDebtType ?? 'POS');
  const [fiIndexer, setFiIndexer] = useState<string>(props.fiIndexer ?? 'CDI');
  const [fiRate, setFiRate] = useState<string | number>(props.fiRate ?? '');
  const [fiApplicationDate, setFiApplicationDate] = useState<string>(props.fiApplicationDate ?? '2026-09-10');
  const [fiMaturityDate, setFiMaturityDate] = useState<string>(props.fiMaturityDate ?? '2028-09-10');

  const defaultProps: FixedIncomeModalProps = {
    showFIModal: true,
    setShowFIModal: vi.fn(),
    fiInstitution,
    setFiInstitution: props.setFiInstitution ?? setFiInstitution,
    fiType: props.fiType ?? fiType,
    setFiType: props.setFiType ?? setFiType,
    fiDebtType: props.fiDebtType ?? fiDebtType,
    setFiDebtType: props.setFiDebtType ?? setFiDebtType,
    fiIndexer: props.fiIndexer ?? fiIndexer,
    setFiIndexer: props.setFiIndexer ?? setFiIndexer,
    fiRate,
    setFiRate: props.setFiRate ?? setFiRate,
    fiAmount,
    setFiAmount: props.setFiAmount ?? setFiAmount,
    fiApplicationDate,
    setFiApplicationDate: props.setFiApplicationDate ?? setFiApplicationDate,
    fiMaturityDate,
    setFiMaturityDate: props.setFiMaturityDate ?? setFiMaturityDate,
    isAddingFI: false,
    handleAddFixedIncome: vi.fn((e) => e.preventDefault()),
    ...props,
  };

  return <FixedIncomeModal {...defaultProps} fiAmount={fiAmount} setFiAmount={setFiAmount} />;
}

describe('FixedIncomeModal', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [],
    });
  });

  it('does not render when showFIModal is false', () => {
    const { container } = render(<FixedIncomeModalWrapper showFIModal={false} />);
    expect(container.firstChild).toBeNull();
  });

  it('fetches banks list on mount when open and populates datalist', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [
        { name: 'Banco Itaú', ispb: '' },
        { name: 'Banco Inter', ispb: '00416968' },
        { name: '', ispb: '999' },
      ],
    });

    render(<FixedIncomeModalWrapper />);

    await waitFor(() => {
      const option = document.querySelector('option[value="Banco Inter"]');
      expect(option).not.toBeNull();
    });
    expect(screen.getByText(/🏛️ Nova Aplicação \(Renda Fixa\)/i)).toBeInTheDocument();
  });

  it('handles bank fetch returning non-array', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ error: 'bad data' }),
    });

    render(<FixedIncomeModalWrapper />);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled();
    });
  });

  it('handles bank fetch failure gracefully', async () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

    render(<FixedIncomeModalWrapper />);

    await waitFor(() => {
      expect(errorSpy).toHaveBeenCalled();
    });
    errorSpy.mockRestore();
  });

  it('applies ATM mask when typing into Valor Aplicado', () => {
    render(<FixedIncomeModalWrapper />);
    const amountInput = screen.getByPlaceholderText('Ex: 5.000,00') as HTMLInputElement;

    fireEvent.change(amountInput, { target: { value: '500000' } });
    expect(amountInput.value).toBe('5.000,00');

    // Paste formatted value
    fireEvent.paste(amountInput, {
      clipboardData: {
        getData: (format: string) => (format === 'text' ? '12500.50' : ''),
      },
    });
    expect(amountInput.value).toBe('12.500,50');

    // Empty paste
    fireEvent.paste(amountInput, {
      clipboardData: {
        getData: () => '',
      },
    });
    expect(amountInput.value).toBe('12.500,50');
  });

  it('handles form field changes and debt type variations (PRE, HIBRIDO, POS)', () => {
    const setFiInstitutionMock = vi.fn();
    const setFiTypeMock = vi.fn();
    const setFiDebtTypeMock = vi.fn();
    const setFiIndexerMock = vi.fn();
    const setFiRateMock = vi.fn();
    const setFiAppDateMock = vi.fn();
    const setFiMatDateMock = vi.fn();

    const { rerender } = render(
      <FixedIncomeModalWrapper
        fiDebtType="POS"
        setFiInstitution={setFiInstitutionMock}
        setFiType={setFiTypeMock}
        setFiDebtType={setFiDebtTypeMock}
        setFiIndexer={setFiIndexerMock}
        setFiRate={setFiRateMock}
        setFiApplicationDate={setFiAppDateMock}
        setFiMaturityDate={setFiMatDateMock}
      />
    );

    // Institution input
    const instInput = screen.getByLabelText(/Instituição \(Banco\/Corretora\)/i);
    fireEvent.change(instInput, { target: { value: 'XP Investimentos' } });
    expect(setFiInstitutionMock).toHaveBeenCalledWith('XP Investimentos');

    // Type select
    const typeSelect = screen.getByLabelText(/Tipo de Produto/i);
    fireEvent.change(typeSelect, { target: { value: 'LCI' } });
    expect(setFiTypeMock).toHaveBeenCalledWith('LCI');

    // Rentabilidade select
    const debtTypeSelect = screen.getByLabelText(/Rentabilidade/i);
    fireEvent.change(debtTypeSelect, { target: { value: 'PRE' } });
    expect(setFiDebtTypeMock).toHaveBeenCalledWith('PRE');

    // Rate input
    const rateInput = screen.getByLabelText(/% do Indexador/i);
    fireEvent.change(rateInput, { target: { value: '110' } });
    expect(setFiRateMock).toHaveBeenCalledWith('110');

    // Application date
    const appDateInput = screen.getByLabelText(/Data de Aplicação/i);
    fireEvent.change(appDateInput, { target: { value: '2026-09-12' } });
    expect(setFiAppDateMock).toHaveBeenCalledWith('2026-09-12');

    // Maturity date
    const matDateInput = screen.getByLabelText(/Data de Vencimento/i);
    fireEvent.change(matDateInput, { target: { value: '2028-12-31' } });
    expect(setFiMatDateMock).toHaveBeenCalledWith('2028-12-31');

    // With PRE, indexador is hidden and label changes
    rerender(<FixedIncomeModalWrapper fiDebtType="PRE" />);
    expect(screen.queryByLabelText(/^Indexador$/i)).toBeNull();
    expect(screen.getByLabelText(/Taxa ao Ano \(%\)/i)).toBeInTheDocument();

    // With POS, change indexer
    rerender(
      <FixedIncomeModalWrapper
        fiDebtType="POS"
        setFiIndexer={setFiIndexerMock}
      />
    );
    const indexerSelect = screen.getByLabelText(/^Indexador$/i);
    fireEvent.change(indexerSelect, { target: { value: 'SELIC' } });
    expect(setFiIndexerMock).toHaveBeenCalledWith('SELIC');

    // With HIBRIDO, IPCA indexer is available
    rerender(<FixedIncomeModalWrapper fiDebtType="HIBRIDO" fiIndexer="IPCA" />);
    expect(screen.getByText('IPCA')).toBeInTheDocument();
  });

  it('handles form submission and cancel/close buttons', () => {
    const setShowFIModalMock = vi.fn();
    const handleAddFixedIncomeMock = vi.fn((e) => e.preventDefault());

    const { rerender } = render(
      <FixedIncomeModalWrapper
        setShowFIModal={setShowFIModalMock}
        handleAddFixedIncome={handleAddFixedIncomeMock}
        isAddingFI={false}
      />
    );

    // Close button
    fireEvent.click(screen.getByText('✕'));
    expect(setShowFIModalMock).toHaveBeenCalledWith(false);

    // Cancel button
    fireEvent.click(screen.getByText('Cancelar'));
    expect(setShowFIModalMock).toHaveBeenCalledWith(false);

    // Submit button
    const submitBtn = screen.getByText('Aplicar');
    fireEvent.submit(submitBtn.closest('form')!);
    expect(handleAddFixedIncomeMock).toHaveBeenCalled();

    // Loading state
    rerender(<FixedIncomeModalWrapper isAddingFI={true} />);
    expect(screen.getByText('Cadastrando...')).toBeDisabled();
  });
});
