import React, { useState, useEffect } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import TransactionModal, { TransactionModalProps } from './TransactionModal';

function TransactionModalWrapper(props: Partial<TransactionModalProps> = {}) {
  const [txUnitPrice, setTxUnitPrice] = useState<string | number>(props.txUnitPrice ?? '');
  const [txFee, setTxFee] = useState<string | number>(props.txFee ?? '');
  const [txQuantity, setTxQuantity] = useState<string | number>(props.txQuantity ?? '10');
  const [txType, setTxType] = useState<'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS'>(props.txType ?? 'BUY');
  const [txAssetType, setTxAssetType] = useState<string>(props.txAssetType ?? 'STOCK_BR');
  const [searchQuery, setSearchQuery] = useState<string>(props.searchQuery ?? 'PETR4');
  const [txExchangeRate, setTxExchangeRate] = useState<string | number>(props.txExchangeRate ?? 1.0);
  const [txExecutedAt, setTxExecutedAt] = useState<string>(props.txExecutedAt ?? '2026-09-10');

  const prevTxTypeRef = React.useRef(props.txType);
  if (props.txType !== undefined && props.txType !== prevTxTypeRef.current) {
    prevTxTypeRef.current = props.txType;
    setTxType(props.txType);
  }

  const defaultProps: TransactionModalProps = {
    showTxModal: true,
    setShowTxModal: vi.fn(),
    editingTxId: null,
    setEditingTxId: vi.fn(),
    txTicker: 'PETR4',
    txAssetType,
    setTxAssetType: props.setTxAssetType ?? setTxAssetType,
    searchQuery,
    setSearchQuery: props.setSearchQuery ?? setSearchQuery,
    isSearching: false,
    showDropdown: false,
    searchResults: [],
    handleSelectAsset: vi.fn(),
    isAddingTx: false,
    txType,
    setTxType: props.setTxType ?? setTxType,
    txQuantity,
    setTxQuantity: props.setTxQuantity ?? setTxQuantity,
    txUnitPrice,
    setTxUnitPrice: props.setTxUnitPrice ?? setTxUnitPrice,
    txFee,
    setTxFee: props.setTxFee ?? setTxFee,
    txExchangeRate,
    setTxExchangeRate: props.setTxExchangeRate ?? setTxExchangeRate,
    txExecutedAt,
    setTxExecutedAt: props.setTxExecutedAt ?? setTxExecutedAt,
    selectedAssetCurrency: 'BRL',
    kpiCurrency: 'BRL',
    handleAddTransaction: vi.fn((e) => e.preventDefault()),
    ...props,
  };

  return <TransactionModal {...defaultProps} txType={txType} txUnitPrice={txUnitPrice} txFee={txFee} txQuantity={txQuantity} />;
}

describe('TransactionModal', () => {
  it('does not render when showTxModal is false', () => {
    const { container } = render(<TransactionModalWrapper showTxModal={false} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders modal with price and fee inputs', () => {
    render(<TransactionModalWrapper />);
    expect(screen.getByText(/➕ Nova Transação/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Preço Unitário/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Corretagem \/ Taxas/i)).toBeInTheDocument();
  });

  it('applies ATM mask when typing digits into Preço Unitário', () => {
    render(<TransactionModalWrapper />);
    const priceInput = screen.getByLabelText(/Preço Unitário/i) as HTMLInputElement;

    // Type '2' -> '0,02'
    fireEvent.change(priceInput, { target: { value: '2' } });
    expect(priceInput.value).toBe('0,02');

    // Type '3' -> '0,23'
    fireEvent.change(priceInput, { target: { value: '0,023' } });
    expect(priceInput.value).toBe('0,23');

    // Type '5' -> '2,35'
    fireEvent.change(priceInput, { target: { value: '0,235' } });
    expect(priceInput.value).toBe('2,35');

    // Type '0' -> '23,50'
    fireEvent.change(priceInput, { target: { value: '2,350' } });
    expect(priceInput.value).toBe('23,50');
  });

  it('handles backspacing correctly in Preço Unitário', () => {
    render(<TransactionModalWrapper txUnitPrice="2,35" />);
    const priceInput = screen.getByLabelText(/Preço Unitário/i) as HTMLInputElement;
    expect(priceInput.value).toBe('2,35');

    // Backspace from '2,35' -> '2,3' -> becomes '0,23'
    fireEvent.change(priceInput, { target: { value: '2,3' } });
    expect(priceInput.value).toBe('0,23');

    // Backspace from '0,23' -> '0,2' -> becomes '0,02'
    fireEvent.change(priceInput, { target: { value: '0,2' } });
    expect(priceInput.value).toBe('0,02');

    // Backspace from '0,02' -> '0,0' -> becomes empty ''
    fireEvent.change(priceInput, { target: { value: '0,0' } });
    expect(priceInput.value).toBe('');
  });

  it('handles pasting formatted and unformatted prices via clipboard paste event', () => {
    render(<TransactionModalWrapper />);
    const priceInput = screen.getByLabelText(/Preço Unitário/i) as HTMLInputElement;

    // Paste "38.50"
    fireEvent.paste(priceInput, {
      clipboardData: {
        getData: (format: string) => (format === 'text' ? '38.50' : ''),
      },
    });
    expect(priceInput.value).toBe('38,50');

    // Paste "100" (integer)
    fireEvent.paste(priceInput, {
      clipboardData: {
        getData: (format: string) => (format === 'text' ? '100' : ''),
      },
    });
    expect(priceInput.value).toBe('100,00');

    // Paste invalid string does not change value
    fireEvent.paste(priceInput, {
      clipboardData: {
        getData: () => '',
      },
    });
    expect(priceInput.value).toBe('100,00');
  });

  it('applies ATM mask and paste to Corretagem / Taxas input', () => {
    render(<TransactionModalWrapper />);
    const feeInput = screen.getByLabelText(/Corretagem \/ Taxas/i) as HTMLInputElement;

    fireEvent.change(feeInput, { target: { value: '450' } });
    expect(feeInput.value).toBe('4,50');

    fireEvent.paste(feeInput, {
      clipboardData: {
        getData: (format: string) => (format === 'text' ? '12.50' : ''),
      },
    });
    expect(feeInput.value).toBe('12,50');

    // Paste invalid on fee
    fireEvent.paste(feeInput, {
      clipboardData: {
        getData: () => '',
      },
    });
    expect(feeInput.value).toBe('12,50');
  });

  it('calculates total operation in summary card dynamically for BUY and SELL', () => {
    const { rerender } = render(<TransactionModalWrapper txQuantity="100" txUnitPrice="2,35" txFee="4,50" txType="BUY" />);

    // BUY: Gross: 100 * 2.35 = 235.00, Fee: 4.50, Net: 239.50
    expect(screen.getByText(/Valor dos Ativos:/i)).toBeInTheDocument();
    expect(screen.getByText(/R\$ 235,00/i)).toBeInTheDocument();
    expect(screen.getByText(/R\$ 4,50/i)).toBeInTheDocument();
    expect(screen.getByText(/R\$ 239,50/i)).toBeInTheDocument();

    // Switch to SELL: Gross: 235.00, Fee: 4.50, Net: 230.50
    rerender(<TransactionModalWrapper txQuantity="100" txUnitPrice="2,35" txFee="4,50" txType="SELL" />);
    expect(screen.getByText(/R\$ 230,50/i)).toBeInTheDocument();
  });

  it('supports editing mode with readonly ticker and update button label', () => {
    render(<TransactionModalWrapper editingTxId="tx-999" txTicker="ITUB4" />);
    expect(screen.getByText(/✏️ Editar Transação/i)).toBeInTheDocument();
    const tickerInput = screen.getByDisplayValue('ITUB4') as HTMLInputElement;
    expect(tickerInput).toBeDisabled();
    expect(screen.getByText('Salvar Alterações')).toBeInTheDocument();
  });

  it('handles close and cancel buttons', () => {
    const setShowTxModalMock = vi.fn();
    const setEditingTxIdMock = vi.fn();

    render(
      <TransactionModalWrapper
        setShowTxModal={setShowTxModalMock}
        setEditingTxId={setEditingTxIdMock}
      />
    );

    // Close button (X)
    const closeBtn = screen.getByText('✕');
    fireEvent.click(closeBtn);
    expect(setShowTxModalMock).toHaveBeenCalledWith(false);
    expect(setEditingTxIdMock).toHaveBeenCalledWith(null);

    // Cancel button
    const cancelBtn = screen.getByText('Cancelar');
    fireEvent.click(cancelBtn);
    expect(setShowTxModalMock).toHaveBeenCalledWith(false);
  });

  it('shows loading spinner when isSearching is true', () => {
    const { container } = render(<TransactionModalWrapper isSearching={true} />);
    expect(container.querySelector('.loading-spinner')).toBeInTheDocument();
  });

  it('renders search results dropdown and allows selecting an asset', () => {
    const handleSelectAssetMock = vi.fn();
    render(
      <TransactionModalWrapper
        showDropdown={true}
        searchResults={[
          { symbol: 'PETR4', name: 'Petrobras PN', exchange: 'B3' },
          { symbol: 'VALE3', name: 'Vale ON', exchange: 'B3' },
        ]}
        handleSelectAsset={handleSelectAssetMock}
      />
    );

    expect(screen.getByText('PETR4')).toBeInTheDocument();
    expect(screen.getByText('Vale ON')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Petrobras PN'));
    expect(handleSelectAssetMock).toHaveBeenCalledWith('PETR4');
  });

  it('handles search input typing', () => {
    const setSearchQueryMock = vi.fn();
    render(<TransactionModalWrapper setSearchQuery={setSearchQueryMock} />);
    const searchInput = screen.getByPlaceholderText(/Pesquise o ticker/i);
    fireEvent.change(searchInput, { target: { value: 'BBAS3' } });
    expect(setSearchQueryMock).toHaveBeenCalledWith('BBAS3');
  });

  it('switches transaction types: SELL, SPLIT, REVERSE_SPLIT, and BUY', () => {
    render(<TransactionModalWrapper />);

    // Click VENDA
    fireEvent.click(screen.getByText(/🔴 VENDA/i));

    // Click SPLIT
    fireEvent.click(screen.getByText(/✂️ SPLIT/i));
    expect(screen.getByText(/Fator \/ Multiplicador/i)).toBeInTheDocument();
    expect(screen.getByText(/Ex: Desdobramento 1 para 10 = Fator 10/i)).toBeInTheDocument();

    // Click REVERSE_SPLIT
    fireEvent.click(screen.getByText(/🗜️ AGRUP/i));
    expect(screen.getByText(/Ex: Agrupamento 10 para 1 = Fator 10/i)).toBeInTheDocument();

    // Click COMPRA back
    fireEvent.click(screen.getByText(/🟢 COMPRA/i));
    expect(screen.queryByText(/Fator \/ Multiplicador/i)).toBeNull();
  });

  it('handles empty or non-numeric quantity gracefully', () => {
    render(<TransactionModalWrapper txQuantity="" txUnitPrice="10,00" />);
    expect(screen.queryByText(/Total Operação:/i)).toBeNull();
  });

  it('handles form changes for asset type, quantity, and executed date', () => {
    const setTxAssetTypeMock = vi.fn();
    const setTxExecutedAtMock = vi.fn();

    render(
      <TransactionModalWrapper
        setTxAssetType={setTxAssetTypeMock}
        setTxExecutedAt={setTxExecutedAtMock}
      />
    );

    const assetTypeSelect = screen.getByRole('combobox');
    fireEvent.change(assetTypeSelect, { target: { value: 'FII' } });
    expect(setTxAssetTypeMock).toHaveBeenCalledWith('FII');

    const qtyInput = screen.getByLabelText(/Quantidade/i);
    fireEvent.change(qtyInput, { target: { value: '50' } });

    const dateInput = screen.getByLabelText(/Data de Execução/i);
    fireEvent.change(dateInput, { target: { value: '2026-09-11' } });
    expect(setTxExecutedAtMock).toHaveBeenCalledWith('2026-09-11');
  });

  it('shows exchange rate input when selectedAssetCurrency is USD and kpiCurrency is BRL', () => {
    const setTxExchangeRateMock = vi.fn();
    render(
      <TransactionModalWrapper
        selectedAssetCurrency="USD"
        kpiCurrency="BRL"
        setTxExchangeRate={setTxExchangeRateMock}
      />
    );

    expect(screen.getByText(/Taxa Cambial USDBRL/i)).toBeInTheDocument();
    const rateInput = screen.getByPlaceholderText('Ex: 5,2500');
    fireEvent.change(rateInput, { target: { value: '54500' } });
    expect(setTxExchangeRateMock).toHaveBeenCalledWith('5,4500');
  });

  it('shows submitting state when isAddingTx is true and calls submit handler', () => {
    const handleAddTransactionMock = vi.fn((e) => e.preventDefault());
    const { rerender } = render(
      <TransactionModalWrapper
        isAddingTx={true}
        handleAddTransaction={handleAddTransactionMock}
      />
    );

    expect(screen.getByText('Registrando...')).toBeInTheDocument();
    expect(screen.getByText('Registrando...')).toBeDisabled();

    rerender(
      <TransactionModalWrapper
        isAddingTx={false}
        handleAddTransaction={handleAddTransactionMock}
      />
    );

    const submitBtn = screen.getByText('Lançar');
    expect(submitBtn).not.toBeDisabled();
    fireEvent.submit(submitBtn.closest('form')!);
    expect(handleAddTransactionMock).toHaveBeenCalled();
  });
});
