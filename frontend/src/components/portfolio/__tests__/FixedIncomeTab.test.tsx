import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FixedIncomeTab from '../FixedIncomeTab';
import * as api from '@/lib/api';

const mockLoadPortfolioDetails = vi.fn();
const mockLoadDividends = vi.fn();

vi.mock('@/context/PortfolioContext', () => ({
  usePortfolioOptional: () => ({
    loadPortfolioDetails: mockLoadPortfolioDetails,
    loadDividends: mockLoadDividends,
  }),
}));

vi.mock('@/lib/api', () => ({
  apiFetch: vi.fn(),
}));

describe('FixedIncomeTab Component', () => {
  const mockPositions = [
    {
      asset: {
        id: 'fi1',
        portfolio_id: 'p1',
        institution: 'Banco Inter',
        type: 'CDB',
        debt_type: 'POS',
        indexer: 'CDI',
        rate: 100,
        maturity_date: '2026-12-31T00:00:00Z',
      },
      start_date: '2024-01-01',
      total_invested: 1000,
      gross_value: 1100,
      net_value: 1080,
      net_return_percent: 8.0,
      is_matured: false,
      days_to_maturity: 500,
    },
    {
      asset: {
        id: 'fi2',
        portfolio_id: 'p1',
        institution: 'XP Investimentos',
        type: 'LCI',
        debt_type: 'PRE',
        indexer: 'PRE',
        rate: 11.5,
        maturity_date: '2027-06-30T00:00:00Z',
      },
      start_date: '2024-06-01',
      total_invested: 2000,
      gross_value: 2150,
      net_value: 2150,
      net_return_percent: 7.5,
      is_matured: false,
      days_to_maturity: 650,
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/positions')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockPositions),
        });
      }
      if (url.includes('/fixed-income/performance')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([
            { date: '2024-01-01', value: 1000, total_invested: 1000 },
            { date: '2024-06-01', value: 3230, total_invested: 3000 },
          ]),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      });
    });
  });

  it('renders fixed income positions and KPI summary cards correctly', async () => {
    render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
      expect(screen.getByText('XP Investimentos')).toBeInTheDocument();
    });

    expect(screen.getByText('100.00% CDI')).toBeInTheDocument();
    expect(screen.getByText('11.50% a.a.')).toBeInTheDocument();
  });

  it('allows sorting positions by institution, rate, and dates', async () => {
    render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
    });

    const institutionHeader = screen.getByText(/Instituição \/ Produto/);
    fireEvent.click(institutionHeader); // sort desc
    fireEvent.click(institutionHeader); // sort asc

    const rateHeader = screen.getByText(/Taxa/);
    fireEvent.click(rateHeader);

    expect(screen.getByText('XP Investimentos')).toBeInTheDocument();
  });

  it('handles redemption modal submission without full page reload', async () => {
    window.alert = vi.fn();
    render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
    });

    const redeemButtons = screen.getAllByRole('button', { name: /RESGATAR/i });
    fireEvent.click(redeemButtons[0]);

    expect(screen.getByText(/Resgatar Aplicação/i)).toBeInTheDocument();

    const amountInput = screen.getByDisplayValue('1080');
    fireEvent.change(amountInput, { target: { value: '500' } });

    const submitButton = screen.getByRole('button', { name: /Confirmar Resgate/i });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(api.apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/fixed-income/assets/fi1/transactions'),
        expect.objectContaining({
          method: 'POST',
        })
      );
      expect(window.alert).toHaveBeenCalledWith('Resgate realizado com sucesso!');
    });
  });

  it('handles CSV bulk import without window.location.reload', async () => {
    window.alert = vi.fn();
    (api.apiFetch as any).mockImplementation((url: string, opts?: any) => {
      if (url.includes('/fixed-income/bulk')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: 2, errors: ['Linha 3 inválida'] }),
        });
      }
      if (url.includes('/fixed-income/positions')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockPositions),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
      });
    });

    const { container } = render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
    });

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    expect(fileInput).toBeInTheDocument();
    const file = new File(['instituicao,tipo\nBanco Inter,CDB'], 'import.csv', { type: 'text/csv' });

    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(api.apiFetch).toHaveBeenCalledWith(
        '/portfolios/p1/fixed-income/bulk',
        expect.objectContaining({ method: 'POST' })
      );
      expect(window.alert).toHaveBeenCalledWith(expect.stringContaining('Importados com sucesso: 2'));
    });
  });

  it('handles CSV bulk import with 0 errors successfully', async () => {
    window.alert = vi.fn();
    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/bulk')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: 5, errors: [] }),
        });
      }
      if (url.includes('/fixed-income/positions')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockPositions),
        });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
    });

    const { container } = render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
    });

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['valid'], 'import2.csv', { type: 'text/csv' });
    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith('Importação concluída com sucesso! 5 registros importados.');
    });
  });

  it('renders matured, near-maturity, and hybrid assets with proper badges', async () => {
    const variedAssets = [
      {
        asset: {
          id: 'fi_matured',
          portfolio_id: 'p1',
          institution: 'Banco C6',
          type: 'CDB',
          debt_type: 'POS',
          indexer: 'CDI',
          rate: 105,
          maturity_date: '2023-01-01T00:00:00Z',
        },
        start_date: '2022-01-01',
        total_invested: 1000,
        gross_value: 1200,
        net_value: 1180,
        net_return_percent: 18.0,
        is_matured: true,
        days_to_maturity: -100,
      },
      {
        asset: {
          id: 'fi_near',
          portfolio_id: 'p1',
          institution: 'BTG Pactual',
          type: 'CRI',
          debt_type: 'HIBRIDO',
          indexer: 'IPCA',
          rate: 6.5,
          maturity_date: '2026-09-01T00:00:00Z',
        },
        start_date: '2024-01-01',
        total_invested: 5000,
        gross_value: 5200,
        net_value: 5200,
        net_return_percent: 4.0,
        is_matured: false,
        days_to_maturity: 15,
      },
    ];

    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/positions')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(variedAssets),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
      });
    });

    render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco C6')).toBeInTheDocument();
      expect(screen.getByText('(Vencido)')).toBeInTheDocument();
      expect(screen.getByText('BTG Pactual')).toBeInTheDocument();
      expect(screen.getByText('(Vence em 15d)')).toBeInTheDocument();
      expect(screen.getByText('IPCA + 6.50%')).toBeInTheDocument();
    });
  });

  it('handles empty state, period changes, operation launch and modal cancel', async () => {
    (api.apiFetch as any).mockImplementation(() => Promise.resolve({
      ok: true,
      json: () => Promise.resolve([]),
    }));

    const onLaunchOperation = vi.fn();
    render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={onLaunchOperation} />);

    await waitFor(() => {
      expect(screen.getByText('Nenhuma aplicação de Renda Fixa encontrada.')).toBeInTheDocument();
      expect(screen.getByText('Nenhum dado histórico de Renda Fixa no período selecionado.')).toBeInTheDocument();
    });

    // Test period clicks
    const periodButtons = ['1M', '3M', '6M', '1Y', 'ALL'];
    for (const p of periodButtons) {
      const btn = screen.getByRole('button', { name: p });
      fireEvent.click(btn);
    }

    // Test launch operation
    const launchBtn = screen.getByRole('button', { name: /\+ Nova Aplicação/i });
    fireEvent.click(launchBtn);
    expect(onLaunchOperation).toHaveBeenCalled();
  });

  it('allows canceling the redemption modal and sorting by all columns', async () => {
    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/positions')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockPositions),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
      });
    });

    const { container } = render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
    });

    // Test sorting on each th column header
    const thElements = container.querySelectorAll('th[style*="cursor: pointer"]');
    thElements.forEach((th) => {
      fireEvent.click(th);
      fireEvent.click(th);
    });

    // Open and cancel redeem modal
    const redeemButtons = screen.getAllByRole('button', { name: /RESGATAR/i });
    fireEvent.click(redeemButtons[0]);
    expect(screen.getByText(/Resgatar Aplicação/i)).toBeInTheDocument();

    const cancelBtn = screen.getByRole('button', { name: /Cancelar/i });
    fireEvent.click(cancelBtn);
    expect(screen.queryByText(/Resgatar Aplicação/i)).not.toBeInTheDocument();
  });

  it('handles API errors when fetching positions and performance', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    
    // Network reject error
    (api.apiFetch as any).mockImplementation(() => Promise.reject(new Error('Network error')));
    const { unmount } = render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalled();
    });

    unmount();

    // HTTP non-ok response
    (api.apiFetch as any).mockImplementation(() => Promise.resolve({ ok: false, status: 500 }));
    render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith('Failed to fetch fixed income positions');
      expect(consoleErrorSpy).toHaveBeenCalledWith('Error fetching fixed income performance:', expect.anything());
    });

    consoleErrorSpy.mockRestore();
  });

  it('handles bulk import error responses and exceptions', async () => {
    window.alert = vi.fn();
    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/positions')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockPositions) });
      }
      if (url.includes('/fixed-income/bulk')) {
        return Promise.resolve({ ok: false, status: 500 });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
    });

    const { container } = render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
    });

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['invalid'], 'test.csv', { type: 'text/csv' });
    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith('Erro ao enviar arquivo.');
    });

    // Test network catch
    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/bulk')) {
        return Promise.reject(new Error('Failed network'));
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
    });

    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith('Erro de conexão.');
    });
  });

  it('handles redemption validations, API errors, and network exceptions', async () => {
    window.alert = vi.fn();
    window.confirm = vi.fn(() => false); // cancel confirmation

    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/positions')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockPositions) });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
    });

    render(<FixedIncomeTab portfolioId="p1" onLaunchOperation={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Banco Inter')).toBeInTheDocument();
    });

    const redeemButtons = screen.getAllByRole('button', { name: /RESGATAR/i });
    fireEvent.click(redeemButtons[0]);

    // Validation: amount > net_value with confirm rejected
    const amountInput = screen.getByDisplayValue('1080');
    fireEvent.change(amountInput, { target: { value: '5000' } });

    const submitButton = screen.getByRole('button', { name: /Confirmar Resgate/i });
    fireEvent.click(submitButton);
    expect(window.confirm).toHaveBeenCalled();

    // Confirm accepted but API returns error
    (window.confirm as any).mockReturnValue(true);
    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/assets/fi1/transactions')) {
        return Promise.resolve({ ok: false, json: () => Promise.resolve({ error: 'Saldo insuficiente' }) });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
    });

    fireEvent.click(submitButton);
    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith('Erro ao resgatar: Saldo insuficiente');
    });

    // Network error during redeem
    (api.apiFetch as any).mockImplementation((url: string) => {
      if (url.includes('/fixed-income/assets/fi1/transactions')) {
        return Promise.reject(new Error('Network failure'));
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
    });

    fireEvent.click(redeemButtons[0]);
    const amountInput2 = screen.getByDisplayValue('1080');
    fireEvent.change(amountInput2, { target: { value: '100' } });
    fireEvent.click(screen.getByRole('button', { name: /Confirmar Resgate/i }));

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith('Erro de conexão');
    });

    // Test zero / empty amount validation
    fireEvent.click(redeemButtons[0]);
    const amountInputZero = screen.getByDisplayValue('1080');
    fireEvent.change(amountInputZero, { target: { value: '0' } });
    const form = screen.getByRole('button', { name: /Confirmar Resgate/i }).closest('form')!;
    fireEvent.submit(form);
    expect(window.alert).toHaveBeenCalledWith('Informe um valor válido para o resgate.');
  });
});
