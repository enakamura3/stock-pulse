import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import DividendsCalendarSection from '../analysis/DividendsCalendarSection';
import { CalculatedDividend } from '../types';

describe('DividendsCalendarSection', () => {
  const today = new Date();
  const year = today.getFullYear();
  const month = String(today.getMonth() + 1).padStart(2, '0');

  const mockDividends: CalculatedDividend[] = [
    {
      asset_id: '1',
      ticker: 'PETR4',
      cum_date: `${year}-${month}-01`,
      payment_date: `${year}-${month}-15T00:00:00Z`,
      gross_amount: 150,
      net_amount: 150,
      currency: 'BRL',
      type: 'Dividendo',
      quantity: 100,
      per_share_amount: 1.5,
      asset_type: 'STOCK_BR',
      asset_name: 'Petrobras',
    },
  ];

  it('renders daily dividends calendar with month controls and summary pills', () => {
    render(
      <DividendsCalendarSection
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Calendário de Proventos Diário/i)).toBeInTheDocument();
    expect(screen.getByText(/Total do Mês/i)).toBeInTheDocument();
    expect(screen.getByText(/Já Recebido/i)).toBeInTheDocument();
    expect(screen.getAllByText(/A Receber/i).length).toBeGreaterThanOrEqual(1);

    // Verify day 15 cell is present

    expect(screen.getByText('15')).toBeInTheDocument();
  });

  it('allows month navigation and day selection', () => {
    render(
      <DividendsCalendarSection
        dividends={mockDividends}
        kpiCurrency="BRL"
      />
    );

    // Click next month
    const nextBtn = screen.getByTitle(/Próximo mês/i);
    fireEvent.click(nextBtn);

    // Click Today button to return
    const todayBtn = screen.getByRole('button', { name: /Hoje/i });
    fireEvent.click(todayBtn);

    // Click day 15 cell to open day drawer
    const day15 = screen.getByText('15');
    const dayCell = day15.closest('div')!;
    fireEvent.click(dayCell);

    expect(screen.getByText(/Proventos do dia 15/i)).toBeInTheDocument();
    expect(screen.getByText(/PETR4/i)).toBeInTheDocument();
  });
});
