import React, { useState } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import TransactionModal, { TransactionModalProps } from './TransactionModal';

function TransactionModalWrapper(props: Partial<TransactionModalProps> = {}) {
  const [txUnitPrice, setTxUnitPrice] = useState<string | number>(props.txUnitPrice ?? '');
  const [txFee, setTxFee] = useState<string | number>(props.txFee ?? '');
  const [txQuantity, setTxQuantity] = useState<string | number>(props.txQuantity ?? '10');
  const [txType, setTxType] = useState<'BUY' | 'SELL' | 'SPLIT' | 'REVERSE_SPLIT' | 'BONUS'>('BUY');

  const defaultProps: TransactionModalProps = {
    showTxModal: true,
    setShowTxModal: vi.fn(),
    editingTxId: null,
    setEditingTxId: vi.fn(),
    txTicker: 'PETR4',
    txAssetType: 'STOCK_BR',
    setTxAssetType: vi.fn(),
    searchQuery: 'PETR4',
    setSearchQuery: vi.fn(),
    isSearching: false,
    showDropdown: false,
    searchResults: [],
    handleSelectAsset: vi.fn(),
    isAddingTx: false,
    txType,
    setTxType,
    txQuantity,
    setTxQuantity,
    txUnitPrice,
    setTxUnitPrice,
    txFee,
    setTxFee,
    txExchangeRate: 1.0,
    setTxExchangeRate: vi.fn(),
    txExecutedAt: '2026-09-10',
    setTxExecutedAt: vi.fn(),
    selectedAssetCurrency: 'BRL',
    kpiCurrency: 'BRL',
    handleAddTransaction: vi.fn((e) => e.preventDefault()),
    ...props,
  };

  return (
    <TransactionModal
      {...defaultProps}
      txUnitPrice={txUnitPrice}
      setTxUnitPrice={setTxUnitPrice}
      txFee={txFee}
      setTxFee={setTxFee}
      txQuantity={txQuantity}
      setTxQuantity={setTxQuantity}
      txType={txType}
      setTxType={setTxType}
    />
  );
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
  });

  it('applies ATM mask to Corretagem / Taxas input', () => {
    render(<TransactionModalWrapper />);
    const feeInput = screen.getByLabelText(/Corretagem \/ Taxas/i) as HTMLInputElement;

    fireEvent.change(feeInput, { target: { value: '450' } });
    expect(feeInput.value).toBe('4,50');
  });

  it('calculates total operation in summary card dynamically', () => {
    render(<TransactionModalWrapper txQuantity="100" txUnitPrice="2,35" txFee="4,50" />);

    // Gross: 100 * 2.35 = 235.00
    // Fee: 4.50
    // Total for BUY: 235.00 + 4.50 = 239.50
    expect(screen.getByText(/Valor dos Ativos:/i)).toBeInTheDocument();
    expect(screen.getByText(/R\$ 235,00/i)).toBeInTheDocument();
    expect(screen.getByText(/R\$ 4,50/i)).toBeInTheDocument();
    expect(screen.getByText(/R\$ 239,50/i)).toBeInTheDocument();
  });
});
