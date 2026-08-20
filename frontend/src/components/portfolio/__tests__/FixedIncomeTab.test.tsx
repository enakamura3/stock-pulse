import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FixedIncomeTab from '../FixedIncomeTab';
import * as api from '@/lib/api';

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
          json: () => Promise.resolve({ success: 2, errors: [] }),
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
      expect(window.alert).toHaveBeenCalledWith('Importação concluída com sucesso! 2 registros importados.');
    });
  });
});
