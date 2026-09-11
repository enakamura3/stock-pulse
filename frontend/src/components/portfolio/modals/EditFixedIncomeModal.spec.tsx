import React, { useState } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import EditFixedIncomeModal, { EditFixedIncomeModalProps } from './EditFixedIncomeModal';

function EditFixedIncomeModalWrapper(props: Partial<EditFixedIncomeModalProps> = {}) {
  const [fiAmount, setFiAmount] = useState<string | number>(props.fiAmount ?? '1.000,00');
  const [fiTxType, setFiTxType] = useState<string>(props.fiTxType ?? 'SUBSCRIPTION');
  const [fiApplicationDate, setFiApplicationDate] = useState<string>(props.fiApplicationDate ?? '2026-09-10');
  const [fiMaturityDate, setFiMaturityDate] = useState<string>(props.fiMaturityDate ?? '2028-09-10');

  const prevFiTxType = React.useRef(props.fiTxType);
  if (props.fiTxType !== undefined && props.fiTxType !== prevFiTxType.current) {
    prevFiTxType.current = props.fiTxType;
    setFiTxType(props.fiTxType);
  }

  const handleSetFiTxType = (t: string) => {
    setFiTxType(t);
    props.setFiTxType?.(t);
  };

  const defaultProps: EditFixedIncomeModalProps = {
    showFIEditModal: true,
    setShowFIEditModal: vi.fn(),
    fiEditTxAssetName: 'CDB Banco Inter 110%',
    fiTxType,
    setFiTxType: handleSetFiTxType,
    fiAmount,
    setFiAmount: props.setFiAmount ?? setFiAmount,
    fiApplicationDate,
    setFiApplicationDate: props.setFiApplicationDate ?? setFiApplicationDate,
    fiMaturityDate,
    setFiMaturityDate: props.setFiMaturityDate ?? setFiMaturityDate,
    isAddingFI: false,
    handleUpdateFITransaction: vi.fn((e) => e.preventDefault()),
    ...props,
  };

  return <EditFixedIncomeModal {...defaultProps} fiTxType={fiTxType} fiAmount={fiAmount} setFiAmount={setFiAmount} />;
}

describe('EditFixedIncomeModal', () => {
  it('does not render when showFIEditModal is false', () => {
    const { container } = render(<EditFixedIncomeModalWrapper showFIEditModal={false} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders modal with asset name disabled and operation buttons', () => {
    render(<EditFixedIncomeModalWrapper />);
    expect(screen.getByText('Editar Operação RF')).toBeInTheDocument();
    const assetInput = screen.getByDisplayValue('CDB Banco Inter 110%');
    expect(assetInput).toBeDisabled();
    expect(screen.getByText('🟢 APLICAÇÃO')).toBeInTheDocument();
    expect(screen.getByText('🔴 RESGATE')).toBeInTheDocument();
  });

  it('applies ATM mask and paste to Valor (R$)', () => {
    render(<EditFixedIncomeModalWrapper />);
    const amountInput = screen.getByPlaceholderText('Ex: 1.000,00') as HTMLInputElement;

    // Type digits
    fireEvent.change(amountInput, { target: { value: '250000' } });
    expect(amountInput.value).toBe('2.500,00');

    // Paste
    fireEvent.paste(amountInput, {
      clipboardData: {
        getData: (format: string) => (format === 'text' ? '5000.00' : ''),
      },
    });
    expect(amountInput.value).toBe('5.000,00');

    // Empty paste
    fireEvent.paste(amountInput, {
      clipboardData: {
        getData: () => '',
      },
    });
    expect(amountInput.value).toBe('5.000,00');
  });

  it('switches operation type between SUBSCRIPTION and REDEMPTION', () => {
    const setFiTxTypeMock = vi.fn();
    render(<EditFixedIncomeModalWrapper setFiTxType={setFiTxTypeMock} />);

    fireEvent.click(screen.getByText('🔴 RESGATE'));
    expect(setFiTxTypeMock).toHaveBeenCalledWith('REDEMPTION');

    fireEvent.click(screen.getByText('🟢 APLICAÇÃO'));
    expect(setFiTxTypeMock).toHaveBeenCalledWith('SUBSCRIPTION');
  });

  it('renders with REDEMPTION active style correctly', () => {
    render(<EditFixedIncomeModalWrapper fiTxType="REDEMPTION" />);
    expect(screen.getByText('🔴 RESGATE')).toBeInTheDocument();
  });

  it('handles application date and maturity date change', () => {
    const setFiAppDateMock = vi.fn();
    const setFiMatDateMock = vi.fn();

    render(
      <EditFixedIncomeModalWrapper
        setFiApplicationDate={setFiAppDateMock}
        setFiMaturityDate={setFiMatDateMock}
      />
    );

    const inputs = screen.getAllByDisplayValue(/202[68]-09-10/);
    fireEvent.change(inputs[0], { target: { value: '2026-09-15' } });
    expect(setFiAppDateMock).toHaveBeenCalledWith('2026-09-15');

    fireEvent.change(inputs[1], { target: { value: '2028-12-31' } });
    expect(setFiMatDateMock).toHaveBeenCalledWith('2028-12-31');
  });

  it('handles close, cancel and submit with loading state', () => {
    const setShowFIEditModalMock = vi.fn();
    const handleUpdateFITransactionMock = vi.fn((e) => e.preventDefault());

    const { rerender } = render(
      <EditFixedIncomeModalWrapper
        setShowFIEditModal={setShowFIEditModalMock}
        handleUpdateFITransaction={handleUpdateFITransactionMock}
        isAddingFI={false}
      />
    );

    // Cancel button
    fireEvent.click(screen.getByText('Cancelar'));
    expect(setShowFIEditModalMock).toHaveBeenCalledWith(false);

    // Submit button
    const submitBtn = screen.getByText('Salvar Operação');
    fireEvent.submit(submitBtn.closest('form')!);
    expect(handleUpdateFITransactionMock).toHaveBeenCalled();

    // Loading state
    rerender(<EditFixedIncomeModalWrapper isAddingFI={true} />);
    expect(screen.getByText('Salvando...')).toBeDisabled();
  });
});
