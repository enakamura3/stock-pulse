import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import DividendsCalendarSection from '../DividendsCalendarSection';

describe('DividendsCalendarSection', () => {
  it('renders calendar with past, future and mixed dividend events, handles day click and navigation', () => {
    const today = new Date();
    const curYear = today.getFullYear();
    const curMonth = today.getMonth(); // 0-indexed
    const curMonthStr = String(curMonth + 1).padStart(2, '0');

    const mockDividends = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        type: 'Dividendo',
        payment_date: `${curYear}-${curMonthStr}-10T00:00:00Z`,
        net_amount: 1500, // >= 1000 => '1.5k'
      },
      {
        asset_id: '2',
        ticker: 'VALE3',
        type: 'JCP',
        payment_date: `${curYear}-${curMonthStr}-10T00:00:00Z`,
        net_amount: 500,
      },
      {
        asset_id: '3',
        ticker: 'MXRF11',
        payment_date: '0001-01-01',
        cum_date: `${curYear}-${curMonthStr}-15T00:00:00Z`, // fallback to cum_date
        net_amount: 250, // < 1000
      },
      {
        asset_id: '4',
        ticker: 'INVALID',
        payment_date: '0001-01-01',
        cum_date: '', // null date
        net_amount: 10,
      },
      {
        asset_id: '5',
        ticker: 'BAD_DATE',
        payment_date: 'invalid-date',
        net_amount: 10,
      },
      {
        asset_id: '6',
        ticker: 'FUT1',
        payment_date: new Date(today.getFullYear(), today.getMonth() + 1, 12).toISOString(),
        net_amount: 600,
      },
    ];

    render(
      <DividendsCalendarSection
        dividends={mockDividends as any}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Calendário de Proventos Diário/)).toBeInTheDocument();
    expect(screen.getByText('Total do Mês')).toBeInTheDocument();

    // Click on day 10 cell (which has events)
    const day10Text = screen.getByText('10');
    const day10Cell = day10Text.closest('div')!;
    fireEvent.click(day10Cell);

    // Selected day panel should appear
    expect(screen.getByText(/Proventos do dia 10 de/)).toBeInTheDocument();
    expect(screen.getByText('PETR4')).toBeInTheDocument();

    // Click Close button in details panel
    const closeBtn = screen.getByText('✖ Fechar');
    fireEvent.click(closeBtn);
    expect(screen.queryByText(/Proventos do dia 10 de/)).not.toBeInTheDocument();

    // Click day 10 again to select, then click it again to toggle off
    fireEvent.click(day10Cell);
    expect(screen.getByText(/Proventos do dia 10 de/)).toBeInTheDocument();
    fireEvent.click(day10Cell);
    expect(screen.queryByText(/Proventos do dia 10 de/)).not.toBeInTheDocument();

    // Test month navigation: Next (future month with upcoming events), Prev, Today
    const prevBtn = screen.getByTitle('Mês anterior');
    const nextBtn = screen.getByTitle('Próximo mês');
    const todayBtn = screen.getByText('Hoje');

    // Go to next month (has upcoming event FUT1)
    fireEvent.click(nextBtn);
    expect(screen.getAllByText(/600,00/)[0]).toBeInTheDocument();

    // Go back two months (previous month)
    fireEvent.click(prevBtn);
    fireEvent.click(prevBtn);

    // Reset with today button
    fireEvent.click(todayBtn);
  });

  it('renders empty month placeholder when no events exist in visible month', () => {
    render(
      <DividendsCalendarSection
        dividends={[]}
        kpiCurrency=""
      />
    );

    expect(screen.getByText(/Nenhum provento registrado ou previsto para o mês/)).toBeInTheDocument();
  });
});
