import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import AnnualSummary from '../AnnualSummary';
import { CalculatedDividend } from '../types';

describe('AnnualSummary Component', () => {
  const mockSetSelectedYear = vi.fn();
  const availableYears = ['2026', '2025'];

  it('renders null when there are no dividends', () => {
    const { container } = render(
      <AnnualSummary
        dividends={[]}
        selectedYear="2026"
        setSelectedYear={mockSetSelectedYear}
        availableYears={availableYears}
      />
    );
    expect(container.firstChild).toBeNull();
  });

  it('calculates metrics and renders layout correctly for selected year', () => {
    const mockDividends: CalculatedDividend[] = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        cum_date: '2026-05-10T00:00:00Z',
        payment_date: '2026-06-15T00:00:00Z',
        gross_amount: 120,
        net_amount: 100,
        currency: 'BRL',
        type: 'DIVIDENDO',
        quantity: 100,
        per_share_amount: 1.2,
        asset_type: 'AÇÃO',
        asset_name: 'Petroleo Brasileiro'
      },
      {
        asset_id: '2',
        ticker: 'VALE3',
        cum_date: '2025-05-10T00:00:00Z',
        payment_date: '2025-06-15T00:00:00Z',
        gross_amount: 60,
        net_amount: 51,
        currency: 'BRL',
        type: 'JCP',
        quantity: 100,
        per_share_amount: 0.6,
        asset_type: 'AÇÃO',
        asset_name: 'Vale SA'
      }
    ];

    render(
      <AnnualSummary
        dividends={mockDividends}
        selectedYear="2026"
        setSelectedYear={mockSetSelectedYear}
        availableYears={availableYears}
      />
    );

    // Cabeçalho e Abas
    expect(screen.getByText(/Resumo Anual Consolidado/i)).toBeInTheDocument();
    expect(screen.getByText('2026')).toBeInTheDocument();
    expect(screen.getByText('2025')).toBeInTheDocument();
    expect(screen.getByText('Ver Todos')).toBeInTheDocument();

    // Total Líquido de 2026 (100,00)
    expect(screen.getByText(/Total Líquido/i)).toBeInTheDocument();
    expect(screen.getAllByText(/100,00/)[0]).toBeInTheDocument();

    // YoY Growth contra 2025 (51,00): (100 - 51) / 51 = 96.1%
    expect(screen.getByText(/96\.1% YoY/)).toBeInTheDocument();

    // Tipo de provento (Dividendo)
    expect(screen.getByText('Dividendo')).toBeInTheDocument();

    // Top pagador: PETR4
    expect(screen.getByText('PETR4')).toBeInTheDocument();
  });

  it('displays the monthly average with correct month divisor', () => {
    // Definimos um dividendo para o ano atual
    const currentYear = new Date().getFullYear();
    const mockDividends: CalculatedDividend[] = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        cum_date: `${currentYear}-01-10T00:00:00Z`,
        payment_date: `${currentYear}-01-15T00:00:00Z`,
        gross_amount: 120,
        net_amount: 120,
        currency: 'BRL',
        type: 'Rendimento',
        quantity: 100,
        per_share_amount: 1.2,
        asset_type: 'AÇÃO',
        asset_name: 'Petroleo Brasileiro'
      }
    ];

    render(
      <AnnualSummary
        dividends={mockDividends}
        selectedYear={String(currentYear)}
        setSelectedYear={mockSetSelectedYear}
        availableYears={[String(currentYear)]}
      />
    );

    const currentMonth = new Date().getMonth() + 1;
    const expectedAverage = 120 / currentMonth;

    // Deve exibir o valor médio mensal correto
    const formattedAverage = expectedAverage.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    expect(screen.getByText(new RegExp(formattedAverage.replace(',', '\\,')))).toBeInTheDocument();
    expect(screen.getByText(new RegExp(`decorridos`))).toBeInTheDocument();
  });

  it('handles other types of dividends like JCP, Rendimento, Amortização, Renda Fixa', () => {
    const mockDividends: CalculatedDividend[] = [
      {
        asset_id: '1',
        ticker: 'MXRF11',
        cum_date: '2026-01-10T00:00:00Z',
        payment_date: '2026-01-15T00:00:00Z',
        gross_amount: 10,
        net_amount: 10,
        currency: 'BRL',
        type: 'Rendimento',
        quantity: 100,
        per_share_amount: 0.1,
        asset_type: 'FII',
        asset_name: 'Maxi Renda'
      },
      {
        asset_id: '2',
        ticker: 'ALUP11',
        cum_date: '2026-02-10T00:00:00Z',
        payment_date: '2026-02-15T00:00:00Z',
        gross_amount: 15,
        net_amount: 15,
        currency: 'BRL',
        type: 'Juros Sobre Capital Próprio (JCP)',
        quantity: 100,
        per_share_amount: 0.15,
        asset_type: 'AÇÃO',
        asset_name: 'Alupar'
      },
      {
        asset_id: '3',
        ticker: 'RF-1',
        cum_date: '2026-03-10T00:00:00Z',
        payment_date: '2026-03-15T00:00:00Z',
        gross_amount: 20,
        net_amount: 20,
        currency: 'BRL',
        type: 'Amortização',
        quantity: 1,
        per_share_amount: 20,
        asset_type: 'RENDA_FIXA',
        asset_name: 'CDB',
        is_accrued: true
      }
    ];

    render(
      <AnnualSummary
        dividends={mockDividends}
        selectedYear="2026"
        setSelectedYear={mockSetSelectedYear}
        availableYears={['2026']}
      />
    );

    expect(screen.getByText('Rendimento')).toBeInTheDocument();
    expect(screen.getByText('JCP')).toBeInTheDocument();
    expect(screen.getByText('Renda Fixa')).toBeInTheDocument();
  });

  it('calls setSelectedYear when tab is clicked', () => {
    const mockDividends: CalculatedDividend[] = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        cum_date: '2026-05-10T00:00:00Z',
        payment_date: '2026-06-15T00:00:00Z',
        gross_amount: 100,
        net_amount: 100,
        currency: 'BRL',
        type: 'DIVIDENDO',
        quantity: 100,
        per_share_amount: 1,
        asset_type: 'AÇÃO',
        asset_name: 'Petroleo Brasileiro'
      },
      {
        asset_id: '2',
        ticker: 'VALE3',
        cum_date: '2025-05-10T00:00:00Z',
        payment_date: '2025-06-15T00:00:00Z',
        gross_amount: 50,
        net_amount: 50,
        currency: 'BRL',
        type: 'DIVIDENDO',
        quantity: 100,
        per_share_amount: 0.5,
        asset_type: 'AÇÃO',
        asset_name: 'Vale SA'
      }
    ];

    render(
      <AnnualSummary
        dividends={mockDividends}
        selectedYear="2026"
        setSelectedYear={mockSetSelectedYear}
        availableYears={availableYears}
      />
    );

    const btn2025 = screen.getByText('2025');
    fireEvent.click(btn2025);
    expect(mockSetSelectedYear).toHaveBeenCalledWith('2025');

    const btnTodos = screen.getByText('Ver Todos');
    fireEvent.click(btnTodos);
    expect(mockSetSelectedYear).toHaveBeenCalledWith('Todos');
  });

  it('uses the most recent year as fallback when selectedYear is "Todos"', () => {
    const mockDividends: CalculatedDividend[] = [
      {
        asset_id: '1',
        ticker: 'PETR4',
        cum_date: '2026-05-10T00:00:00Z',
        payment_date: '2026-06-15T00:00:00Z',
        gross_amount: 100,
        net_amount: 100,
        currency: 'BRL',
        type: 'DIVIDENDO',
        quantity: 100,
        per_share_amount: 1,
        asset_type: 'AÇÃO',
        asset_name: 'Petroleo Brasileiro'
      },
      {
        asset_id: '2',
        ticker: 'VALE3',
        cum_date: '2025-05-10T00:00:00Z',
        payment_date: '2025-06-15T00:00:00Z',
        gross_amount: 50,
        net_amount: 50,
        currency: 'BRL',
        type: 'DIVIDENDO',
        quantity: 100,
        per_share_amount: 0.5,
        asset_type: 'AÇÃO',
        asset_name: 'Vale SA'
      }
    ];

    render(
      <AnnualSummary
        dividends={mockDividends}
        selectedYear="Todos"
        setSelectedYear={mockSetSelectedYear}
        availableYears={availableYears}
      />
    );

    // Se selectedYear é 'Todos', deve selecionar por padrão o ano mais recente (2026, total 100,00)
    expect(screen.getAllByText(/100,00/)[0]).toBeInTheDocument();
  });
});
