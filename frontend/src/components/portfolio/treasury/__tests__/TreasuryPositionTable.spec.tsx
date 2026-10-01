import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import TreasuryPositionTable from '../TreasuryPositionTable';
import { TreasuryPosition } from '../../types';
import { SortKey } from '../types';

describe('TreasuryPositionTable Component', () => {
  const onSort = vi.fn();
  const onOpenModal = vi.fn();
  const onImport = vi.fn();
  const onExport = vi.fn();
  const onRedeem = vi.fn();

  const mockPositions: TreasuryPosition[] = [
    {
      transaction_id: 'tx-1',
      ticker: 'Tesouro Selic 2029',
      treasury_type: 'SELIC',
      maturity_date: '2029-03-01T00:00:00Z',
      days_to_maturity: 1500,
      total_invested: 1000,
      gross_value: 1100,
      net_value: 1080,
      iof_tax: 0,
      ir_tax: 15,
      b3_fee: 5,
      is_matured: false,
      has_coupons: false,
    },
    {
      transaction_id: 'tx-2',
      ticker: 'Tesouro IPCA+ 2026',
      treasury_type: 'IPCA+',
      maturity_date: '2026-08-15T00:00:00Z',
      days_to_maturity: 200,
      total_invested: 2000,
      gross_value: 1900,
      net_value: 1850,
      iof_tax: 10,
      ir_tax: 20,
      b3_fee: 10,
      is_matured: true,
      has_coupons: true,
    },
    {
      transaction_id: 'tx-3',
      ticker: 'Zero Invested',
      treasury_type: 'SELIC',
      maturity_date: '2025-01-01T00:00:00Z',
      days_to_maturity: 50,
      total_invested: 0,
      gross_value: 0,
      net_value: 0,
      iof_tax: 0,
      ir_tax: 0,
      b3_fee: 0,
      is_matured: false,
      has_coupons: false,
    },
  ];

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading spinner when isLoadingPositions or isImporting is true', () => {
    const { container, rerender } = render(
      <TreasuryPositionTable
        positions={[]}
        isLoadingPositions={true}
        isImporting={false}
        sortKey="ticker"
        sortDir="asc"
        onSort={onSort}
        onOpenModal={onOpenModal}
        onImport={onImport}
        onExport={onExport}
        onRedeem={onRedeem}
      />
    );

    expect(container.querySelector('.loading-spinner')).toBeInTheDocument();

    rerender(
      <TreasuryPositionTable
        positions={[]}
        isLoadingPositions={false}
        isImporting={true}
        sortKey="ticker"
        sortDir="asc"
        onSort={onSort}
        onOpenModal={onOpenModal}
        onImport={onImport}
        onExport={onExport}
        onRedeem={onRedeem}
      />
    );

    expect(container.querySelector('.loading-spinner')).toBeInTheDocument();
  });

  it('renders empty positions placeholder and button to open modal', () => {
    render(
      <TreasuryPositionTable
        positions={[]}
        isLoadingPositions={false}
        isImporting={false}
        sortKey="ticker"
        sortDir="asc"
        onSort={onSort}
        onOpenModal={onOpenModal}
        onImport={onImport}
        onExport={onExport}
        onRedeem={onRedeem}
      />
    );

    expect(screen.getByText('Nenhuma posição ativa de Tesouro Direto.')).toBeInTheDocument();
    const addBtn = screen.getByText('+ Adicionar primeira aplicação');
    fireEvent.click(addBtn);
    expect(onOpenModal).toHaveBeenCalledTimes(1);
  });

  it('renders table with positions, handles all sorting keys and actions', () => {
    const { rerender } = render(
      <TreasuryPositionTable
        positions={mockPositions}
        isLoadingPositions={false}
        isImporting={false}
        sortKey="ticker"
        sortDir="asc"
        onSort={onSort}
        onOpenModal={onOpenModal}
        onImport={onImport}
        onExport={onExport}
        onRedeem={onRedeem}
      />
    );

    expect(screen.getByText('Tesouro Selic 2029')).toBeInTheDocument();
    expect(screen.getByText('Tesouro IPCA+ 2026')).toBeInTheDocument();
    expect(screen.getByText('cupons')).toBeInTheDocument();
    expect(screen.getByText('Vencido')).toBeInTheDocument();
    expect(screen.getByText('VENCIDO')).toBeInTheDocument();
    expect(screen.getAllByText('ATIVO')[0]).toBeInTheDocument();

    // Import file
    const fileInput = document.querySelector('input[type="file"]')!;
    fireEvent.change(fileInput, { target: { files: [new File([''], 'test.csv')] } });
    expect(onImport).toHaveBeenCalled();

    // Export button
    fireEvent.click(screen.getByText('📤 Exportar'));
    expect(onExport).toHaveBeenCalled();

    // New application button
    fireEvent.click(screen.getByText('+ Nova Aplicação'));
    expect(onOpenModal).toHaveBeenCalled();

    // Redeem button
    const redeemBtns = screen.getAllByText('Resgatar');
    fireEvent.click(redeemBtns[0]);
    expect(onRedeem).toHaveBeenCalledWith(expect.objectContaining({ transaction_id: 'tx-2' }));

    // Test sorting on all keys
    const sortKeys: SortKey[] = [
      'ticker',
      'treasury_type',
      'maturity_date',
      'total_invested',
      'gross_value',
      'net_value',
      'net_return',
      'iof_tax',
      'ir_tax',
      'b3_fee',
      'status',
    ];

    sortKeys.forEach((key) => {
      rerender(
        <TreasuryPositionTable
          positions={mockPositions}
          isLoadingPositions={false}
          isImporting={false}
          sortKey={key}
          sortDir="desc"
          onSort={onSort}
          onOpenModal={onOpenModal}
          onImport={onImport}
          onExport={onExport}
          onRedeem={onRedeem}
        />
      );

      rerender(
        <TreasuryPositionTable
          positions={mockPositions}
          isLoadingPositions={false}
          isImporting={false}
          sortKey={key}
          sortDir="asc"
          onSort={onSort}
          onOpenModal={onOpenModal}
          onImport={onImport}
          onExport={onExport}
          onRedeem={onRedeem}
        />
      );
    });
  });
});
