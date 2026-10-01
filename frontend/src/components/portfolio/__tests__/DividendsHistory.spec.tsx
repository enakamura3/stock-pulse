import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import DividendsHistory from '../DividendsHistory';
import { CalculatedDividend } from '../types';

// Mock dynamic DividendsMatrix and AnnualSummary to simplify
vi.mock('next/dynamic', () => ({
  default: () => {
    return function MockDividendsMatrix({ onYearClick, onMonthClick }: any) {
      return (
        <div data-testid="dividends-matrix">
          <button data-testid="matrix-year-btn" onClick={() => onYearClick('2024')}>
            Year 2024
          </button>
          <button data-testid="matrix-month-btn" onClick={() => onMonthClick('2024', '05')}>
            Month 05/2024
          </button>
        </div>
      );
    };
  },
}));

vi.mock('../AnnualSummary', () => ({
  default: ({ selectedYear, setSelectedYear }: any) => (
    <div data-testid="annual-summary">
      <button data-testid="summary-year-btn" onClick={() => setSelectedYear('2023')}>
        Select 2023
      </button>
    </div>
  ),
}));

describe('DividendsHistory Component', () => {
  const setFilterDivYear = vi.fn();
  const setFilterDivMonth = vi.fn();

  const mockDividends: CalculatedDividend[] = [
    {
      id: 'div-1',
      asset_id: 'a1',
      ticker: 'PETR4',
      asset_type: 'STOCK',
      type: 'DIVIDENDO',
      cum_date: '2024-04-10T00:00:00Z',
      payment_date: '2024-05-15T00:00:00Z', // past date -> paid
      quantity: 100,
      per_share_amount: 1.5,
      gross_amount: 150,
      net_amount: 150,
      currency: 'BRL',
    },
    {
      id: 'div-2',
      asset_id: 'a2',
      ticker: 'VALE3',
      asset_type: 'STOCK',
      type: 'JCP',
      cum_date: '2024-03-10T00:00:00Z',
      payment_date: '2099-12-31T00:00:00Z', // future date -> pending
      quantity: 50,
      per_share_amount: 2.0,
      gross_amount: 100,
      net_amount: 85,
      currency: 'BRL',
    },
    {
      id: 'div-3',
      asset_id: 'a3',
      ticker: 'HGLG11',
      asset_type: 'FII',
      type: 'Rendimento',
      cum_date: '2024-05-31T00:00:00Z',
      payment_date: '2024-06-15T00:00:00Z',
      quantity: 10,
      per_share_amount: 1.1,
      gross_amount: 11,
      net_amount: 11,
      currency: 'BRL',
    },
    {
      id: 'div-3-dup', // Duplicate monthly yield should be deduped
      asset_id: 'a3',
      ticker: 'HGLG11',
      asset_type: 'FII',
      type: 'Rendimento',
      cum_date: '2024-05-31T00:00:00Z',
      payment_date: '2024-06-15T00:00:00Z',
      quantity: 10,
      per_share_amount: 1.1,
      gross_amount: 11,
      net_amount: 11,
      currency: 'BRL',
    },
    {
      id: 'div-4',
      asset_id: 'a4',
      ticker: 'KNCR11',
      asset_type: 'FII',
      type: 'Amortização',
      cum_date: '2024-05-31T00:00:00Z',
      payment_date: '2024-06-15T00:00:00Z',
      quantity: 20,
      per_share_amount: 0.5,
      gross_amount: 10,
      net_amount: 10,
      currency: 'BRL',
    },
    {
      id: 'div-5',
      asset_id: 'a5',
      ticker: 'AAPL',
      asset_type: 'EQUITY_US',
      type: 'DIVIDEND',
      cum_date: '2024-02-01T00:00:00Z',
      payment_date: '2024-02-15T00:00:00Z',
      quantity: 5,
      per_share_amount: 0.25,
      gross_amount: 6.25,
      net_amount: 4.38,
      currency: 'BRL',
      original_gross_amount: 1.25,
      original_net_amount: 0.88,
    },
    {
      id: 'div-6',
      asset_id: 'a6',
      ticker: 'TESOURO IPCA',
      asset_type: 'TESOURO',
      type: 'JUROS',
      cum_date: '2024-01-01T00:00:00Z',
      payment_date: '2024-01-15T00:00:00Z',
      quantity: 1,
      per_share_amount: 0,
      gross_amount: 50,
      net_amount: 42.5,
      currency: 'BRL',
      is_accrued: true,
    },
    {
      id: 'div-6-dup', // tesouro in same month should consolidate
      asset_id: 'a6',
      ticker: 'TESOURO IPCA',
      asset_type: 'TESOURO',
      type: 'JUROS',
      cum_date: '2024-01-02T00:00:00Z',
      payment_date: '2024-01-16T00:00:00Z',
      quantity: 1,
      per_share_amount: 0,
      gross_amount: 30,
      net_amount: 25.5,
      currency: 'BRL',
      is_accrued: true,
    },
    {
      id: 'div-7',
      asset_id: 'a7',
      ticker: 'UNKNOWN',
      asset_type: 'OTHER',
      type: '',
      cum_date: '0001-01-01T00:00:00Z',
      payment_date: '',
      quantity: 0,
      per_share_amount: 0,
      gross_amount: 0,
      net_amount: 0,
      currency: 'BRL',
    },
  ];

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state when isLoadingDividends is true', () => {
    render(
      <DividendsHistory
        dividends={[]}
        filterDivYear="2024"
        setFilterDivYear={setFilterDivYear}
        filterDivMonth="Todos"
        setFilterDivMonth={setFilterDivMonth}
        availableYears={['2024', '2023']}
        isLoadingDividends={true}
      />
    );

    expect(screen.getByText('Carregando proventos...')).toBeInTheDocument();
  });

  it('renders empty state when dividends is empty and not loading', () => {
    render(
      <DividendsHistory
        dividends={[]}
        filterDivYear="2024"
        setFilterDivYear={setFilterDivYear}
        filterDivMonth="Todos"
        setFilterDivMonth={setFilterDivMonth}
        availableYears={['2024', '2023']}
        isLoadingDividends={false}
      />
    );

    expect(screen.getByText('Nenhum provento recebido ainda.')).toBeInTheDocument();
  });

  it('renders full dividends history table, KPIs, and handles filter dropdowns and matrix callbacks', () => {
    render(
      <DividendsHistory
        dividends={mockDividends}
        allDividends={mockDividends}
        filterDivYear="2024"
        setFilterDivYear={setFilterDivYear}
        filterDivMonth="Todos"
        setFilterDivMonth={setFilterDivMonth}
        availableYears={['2024', '2023']}
        isLoadingDividends={false}
      />
    );

    // Annual summary and matrix are rendered
    expect(screen.getByTestId('annual-summary')).toBeInTheDocument();
    expect(screen.getByTestId('dividends-matrix')).toBeInTheDocument();

    // Trigger matrix clicks
    fireEvent.click(screen.getByTestId('matrix-year-btn'));
    expect(setFilterDivYear).toHaveBeenCalledWith('2024');
    expect(setFilterDivMonth).toHaveBeenCalledWith('Todos');

    fireEvent.click(screen.getByTestId('matrix-month-btn'));
    expect(setFilterDivYear).toHaveBeenCalledWith('2024');
    expect(setFilterDivMonth).toHaveBeenCalledWith('05');

    // Year select filter
    const yearSelect = screen.getByDisplayValue('2024');
    fireEvent.change(yearSelect, { target: { value: '2023' } });
    expect(setFilterDivYear).toHaveBeenCalledWith('2023');

    // Month select filter
    const monthSelect = screen.getByDisplayValue('Todos os Meses');
    fireEvent.change(monthSelect, { target: { value: '06' } });
    expect(setFilterDivMonth).toHaveBeenCalledWith('06');

    // Verify badges and rows
    expect(screen.getByText('PETR4')).toBeInTheDocument();
    expect(screen.getByText('VALE3')).toBeInTheDocument();
    expect(screen.getByText('HGLG11')).toBeInTheDocument();
    expect(screen.getByText('TESOURO IPCA')).toBeInTheDocument();

    // Sorting by each column
    const headers = [
      'Status',
      'Ativo',
      'Categoria',
      'Tipo',
      'Data Com',
      'Pagamento',
      'Qtd',
      'Vlr / Cota',
      'Vlr Bruto',
      'Vlr Líquido',
    ];

    headers.forEach((headerText) => {
      const th = screen.getByRole('columnheader', { name: new RegExp(headerText, 'i') });
      fireEvent.click(th); // desc
      fireEvent.click(th); // asc
    });
  });
});
