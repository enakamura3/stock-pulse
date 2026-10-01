import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import DividendsMatrix from '../DividendsMatrix';
import { CalculatedDividend } from '../types';

describe('DividendsMatrix Component', () => {
  it('returns null when there are no valid years', () => {
    const { container } = render(<DividendsMatrix data={[]} />);
    expect(container.firstChild).toBeNull();

    const { container: container2 } = render(
      <DividendsMatrix
        data={[
          {
            id: '1',
            asset_id: 'a1',
            ticker: 'PETR4',
            type: 'DIVIDENDO',
            payment_date: '',
            cum_date: '',
            gross_amount: 10,
            net_amount: 10,
            quantity: 1,
            rate_per_share: 10,
          },
        ]}
      />
    );
    expect(container2.firstChild).toBeNull();
  });

  it('renders matrix and handles year and month clicks', () => {
    const mockData: CalculatedDividend[] = [
      {
        id: '1',
        asset_id: 'a1',
        ticker: 'PETR4',
        type: 'DIVIDENDO',
        payment_date: '2024-03-15T00:00:00Z',
        cum_date: '2024-02-15T00:00:00Z',
        gross_amount: 100,
        net_amount: 100,
        quantity: 10,
        rate_per_share: 10,
      },
      {
        id: '2',
        asset_id: 'a2',
        ticker: 'VALE3',
        type: 'JCP',
        payment_date: '0001-01-01T00:00:00Z', // fallback to cum_date
        cum_date: '2023-05-10T00:00:00Z',
        gross_amount: 50,
        net_amount: 42.5,
        quantity: 5,
        rate_per_share: 10,
      },
      {
        id: '3',
        asset_id: 'a3',
        ticker: 'ITUB4',
        type: 'DIVIDENDO',
        payment_date: '',
        cum_date: '', // invalid, skipped
        gross_amount: 0,
        net_amount: 0,
        quantity: 0,
        rate_per_share: 0,
      },
    ];

    const onYearClick = vi.fn();
    const onMonthClick = vi.fn();

    const { rerender } = render(
      <DividendsMatrix
        data={mockData}
        onYearClick={onYearClick}
        onMonthClick={onMonthClick}
      />
    );

    expect(screen.getByText(/Mapa de Proventos/i)).toBeInTheDocument();
    expect(screen.getByText('2024')).toBeInTheDocument();
    expect(screen.getByText('2023')).toBeInTheDocument();

    // Click on year '2024' (currently inactive)
    const year2024Cell = screen.getByText('2024');
    fireEvent.click(year2024Cell);
    expect(onYearClick).toHaveBeenCalledWith('2024');

    // Click on active month (Mar/2024)
    const marValueCell = screen.getByTitle('Filtrar Mar/2024');
    fireEvent.click(marValueCell);
    expect(onMonthClick).toHaveBeenCalledWith('2024', '03');

    // Click on zero month cell (no-op)
    const zeroCells = screen.getAllByText('-');
    fireEvent.click(zeroCells[0]);
    expect(onMonthClick).toHaveBeenCalledTimes(1);

    // Rerender with activeYear = '2024' and activeMonth = 'Todos'
    rerender(
      <DividendsMatrix
        data={mockData}
        onYearClick={onYearClick}
        onMonthClick={onMonthClick}
        activeYear="2024"
        activeMonth="Todos"
      />
    );
    fireEvent.click(screen.getByTitle('Remover filtro de ano'));
    expect(onYearClick).toHaveBeenCalledWith('Todos');

    // Rerender with activeYear = '2024' and activeMonth = '03'
    rerender(
      <DividendsMatrix
        data={mockData}
        onYearClick={onYearClick}
        onMonthClick={onMonthClick}
        activeYear="2024"
        activeMonth="03"
      />
    );
    const activeMarCell = screen.getByTitle('Remover filtro');
    fireEvent.click(activeMarCell);
    expect(onMonthClick).toHaveBeenCalledWith('Todos', 'Todos');
  });

  it('renders without callbacks without errors when clicking', () => {
    const mockData: CalculatedDividend[] = [
      {
        id: '1',
        asset_id: 'a1',
        ticker: 'PETR4',
        type: 'DIVIDENDO',
        payment_date: '2024-03-15T00:00:00Z',
        cum_date: '2024-02-15T00:00:00Z',
        gross_amount: 100,
        net_amount: 100,
        quantity: 10,
        rate_per_share: 10,
      },
    ];

    render(<DividendsMatrix data={mockData} />);
    fireEvent.click(screen.getByText('2024'));
    fireEvent.click(screen.getByTitle('Filtrar Mar/2024'));
  });
});
