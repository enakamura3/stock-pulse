import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import React from 'react';
import TreasuryTab from '../TreasuryTab';
import { TreasuryPosition } from '../types';
import { apiFetch } from '@/lib/api';

vi.mock('@/lib/api', () => ({
  apiFetch: vi.fn(),
}));

vi.mock('next/dynamic', () => ({
  default: () => {
    return function MockChart({ data }: any) {
      return <div data-testid="portfolio-chart">{data.length} points</div>;
    };
  },
}));

describe('TreasuryTab Component', () => {
  const mockOnRefresh = vi.fn();

  const mockPositions: TreasuryPosition[] = [
    {
      transaction_id: 'tx-1',
      ticker: 'Tesouro Selic 2029',
      treasury_type: 'SELIC',
      maturity_date: '2029-03-01T00:00:00Z',
      days_to_maturity: 1500,
      total_invested: 1000,
      gross_value: 1200,
      net_value: 1150,
      iof_tax: 0,
      ir_tax: 35,
      b3_fee: 15,
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
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    (apiFetch as any).mockResolvedValue({
      ok: true,
      json: async () => [{ date: '2024-01-01', value: 1000 }],
    });
  });

  it('renders KPI cards, chart, and table', async () => {
    render(
      <TreasuryTab
        portfolioId="p1"
        positions={mockPositions}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    expect(screen.getByText('Total Investido')).toBeInTheDocument();
    expect(screen.getByText('Valor Bruto')).toBeInTheDocument();
    expect(screen.getByText('Valor Líquido')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByTestId('portfolio-chart')).toBeInTheDocument();
    });
  });

  it('renders empty chart state when performance data is empty', async () => {
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => [],
    });

    render(
      <TreasuryTab
        portfolioId="p1"
        positions={[]}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('Nenhum dado histórico disponível ainda.')).toBeInTheDocument();
    });
  });

  it('handles error when fetching performance data', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    (apiFetch as any).mockRejectedValueOnce(new Error('Network failure'));

    render(
      <TreasuryTab
        portfolioId="p1"
        positions={[]}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith('Erro ao buscar performance Tesouro:', expect.any(Error));
    });
    consoleSpy.mockRestore();
  });

  it('handles sorting columns in TreasuryTab', () => {
    render(
      <TreasuryTab
        portfolioId="p1"
        positions={mockPositions}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    const tickerHeader = screen.getByText(/Título/);
    fireEvent.click(tickerHeader); // asc -> desc
    fireEvent.click(tickerHeader); // desc -> asc

    const typeHeader = screen.getByText(/Tipo/);
    fireEvent.click(typeHeader);
  });

  it('opens new application modal and creates a transaction successfully', async () => {
    render(
      <TreasuryTab
        portfolioId="p1"
        positions={mockPositions}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    const addBtn = screen.getByText('+ Nova Aplicação');
    fireEvent.click(addBtn);

    expect(screen.getByText(/Nova Aplicação — Tesouro Direto/)).toBeInTheDocument();

    // Fill form
    fireEvent.change(screen.getByPlaceholderText('Ex: TESOURO SELIC 2027'), {
      target: { value: 'TESOURO SELIC 2029' },
    });
    fireEvent.change(document.getElementById('treasury-tx-date')!, {
      target: { value: '2024-03-01' },
    });
    fireEvent.change(document.getElementById('treasury-maturity-date')!, {
      target: { value: '2029-03-01' },
    });
    fireEvent.change(document.getElementById('treasury-contracted-rate')!, {
      target: { value: '0.15' },
    });
    fireEvent.change(document.getElementById('treasury-quantity')!, {
      target: { value: '0.5' },
    });
    fireEvent.change(document.getElementById('treasury-unit-price')!, {
      target: { value: '14000' },
    });

    // Toggle coupon checkbox
    const couponCheckbox = document.getElementById('treasury-has-coupons')!;
    fireEvent.click(couponCheckbox);

    // Mock API response for POST
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ id: 'new-tx' }),
    });

    const form = document.querySelector('form')!;
    fireEvent.submit(form);

    await waitFor(() => {
      expect(mockOnRefresh).toHaveBeenCalled();
    });
  });

  it('handles validation error and api failure on modal submit', async () => {
    render(
      <TreasuryTab
        portfolioId="p1"
        positions={mockPositions}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    fireEvent.click(screen.getByText('+ Nova Aplicação'));

    // Submit with empty fields
    const form = document.querySelector('form')!;
    fireEvent.submit(form);

    expect(screen.getByText(/Preencha todos os campos obrigatórios/)).toBeInTheDocument();

    // Fill required fields
    fireEvent.change(screen.getByPlaceholderText('Ex: TESOURO SELIC 2027'), {
      target: { value: 'TESOURO SELIC 2029' },
    });
    fireEvent.change(document.getElementById('treasury-tx-date')!, {
      target: { value: '2024-03-01' },
    });
    fireEvent.change(document.getElementById('treasury-quantity')!, {
      target: { value: '0.5' },
    });
    fireEvent.change(document.getElementById('treasury-unit-price')!, {
      target: { value: '14000' },
    });

    // Mock API error
    (apiFetch as any).mockResolvedValueOnce({
      ok: false,
      json: async () => ({ error: 'Saldo insuficiente' }),
    });

    fireEvent.submit(form);

    await waitFor(() => {
      expect(screen.getByText(/Saldo insuficiente/)).toBeInTheDocument();
    });

    // Cancel modal
    fireEvent.click(screen.getByText('Cancelar'));
    expect(screen.queryByText(/Nova Aplicação — Tesouro Direto/)).not.toBeInTheDocument();
  });

  it('opens redemption modal via table and handles type toggle', async () => {
    render(
      <TreasuryTab
        portfolioId="p1"
        positions={mockPositions}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    const redeemBtns = screen.getAllByText('Resgatar');
    fireEvent.click(redeemBtns[0]);

    expect(screen.getByText(/Novo Resgate — Tesouro Direto/)).toBeInTheDocument();

    // Toggle between Application and Redemption
    const appBtn = screen.getByText('📥 Aplicação');
    fireEvent.click(appBtn);
    expect(screen.getByText(/Nova Aplicação — Tesouro Direto/)).toBeInTheDocument();

    const redBtn = screen.getByText('📤 Resgate');
    fireEvent.click(redBtn);
    expect(screen.getByText(/Novo Resgate — Tesouro Direto/)).toBeInTheDocument();

    // Close via '✕'
    fireEvent.click(screen.getByText('✕'));
    expect(screen.queryByText(/Novo Resgate/)).not.toBeInTheDocument();
  });

  it('handles CSV import success and failure', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});

    render(
      <TreasuryTab
        portfolioId="p1"
        positions={mockPositions}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    const fileInput = document.querySelector('input[type="file"]')!;

    // Successful import
    (apiFetch as any).mockResolvedValueOnce({ ok: true });
    fireEvent.change(fileInput, { target: { files: [new File(['csv-content'], 'test.csv')] } });

    await waitFor(() => {
      expect(mockOnRefresh).toHaveBeenCalled();
    });

    // Failed import
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    fireEvent.change(fileInput, { target: { files: [new File(['csv-content'], 'test.csv')] } });

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Erro ao importar arquivo CSV.');
    });

    // Exception during import
    (apiFetch as any).mockRejectedValueOnce(new Error('Network error'));
    fireEvent.change(fileInput, { target: { files: [new File(['csv-content'], 'test.csv')] } });

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Erro de conexão ao importar arquivo.');
    });

    alertSpy.mockRestore();
  });

  it('handles CSV export success and failure', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    window.URL.createObjectURL = vi.fn().mockReturnValue('blob:http://test');
    window.URL.revokeObjectURL = vi.fn();

    render(
      <TreasuryTab
        portfolioId="p1"
        positions={mockPositions}
        isLoadingPositions={false}
        onRefresh={mockOnRefresh}
      />
    );

    // Export success
    (apiFetch as any).mockResolvedValueOnce({
      ok: true,
      blob: async () => new Blob(['data']),
    });

    fireEvent.click(screen.getByText('📤 Exportar'));

    await waitFor(() => {
      expect(window.URL.createObjectURL).toHaveBeenCalled();
    });

    // Export failure
    (apiFetch as any).mockResolvedValueOnce({ ok: false });
    fireEvent.click(screen.getByText('📤 Exportar'));

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Erro ao exportar posições.');
    });

    // Export exception
    (apiFetch as any).mockRejectedValueOnce(new Error('Network error'));
    fireEvent.click(screen.getByText('📤 Exportar'));

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Erro ao exportar posições.');
    });

    alertSpy.mockRestore();
  });
});
