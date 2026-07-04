import { render, screen } from '@testing-library/react';
import PortfolioInsights from '../PortfolioInsights';
import { Position, CalculatedDividend, FixedIncomePosition } from '../types';
import { describe, it, expect } from 'vitest';
import React from 'react';

describe('PortfolioInsights', () => {
  const mockPositions: Position[] = [
    { 
      ticker: 'AAPL', 
      type: 'Stock', 
      current_value: 1000, 
      total_cost: 800, 
      return_percent: 25, 
      current_price: 150, 
      graham_value: 200, 
      bazin_value: 180, 
      currency: 'USD' 
    }
  ];
  
  const mockDividends: CalculatedDividend[] = [
    { 
      ticker: 'AAPL', 
      type: 'Dividend', 
      net_amount: 10, 
      is_accrued: false, 
      payment_date: '2023-10-01', 
      ex_date: '2023-09-15',
      gross_amount: 10,
      currency: 'USD'
    }
  ];
  
  const mockFiPositions: FixedIncomePosition[] = [
    { 
      name: 'Tesouro Direto', 
      net_value: 5000, 
      days_to_maturity: 100,
      invested_amount: 4000
    }
  ];

  it('deve renderizar o componente sem erros de variável indefinida', () => {
    render(
      <PortfolioInsights 
        positions={mockPositions} 
        dividends={mockDividends} 
        fiPositions={mockFiPositions} 
        kpiCurrency="BRL" 
      />
    );
    
    // Verifica se os blocos principais estão sendo renderizados corretamente
    expect(screen.getByText(/Concentração da Carteira/i)).toBeInTheDocument();
    expect(screen.getByText(/Alocação por Categoria/i)).toBeInTheDocument();
    expect(screen.getByText(/Yield da Carteira/i)).toBeInTheDocument();
    expect(screen.getByText(/Valuation e Descontos/i)).toBeInTheDocument();
    expect(screen.getByText(/Sazonalidade de Proventos/i)).toBeInTheDocument();
    expect(screen.getByText(/Liquidez da Renda Fixa/i)).toBeInTheDocument();
  });
});
