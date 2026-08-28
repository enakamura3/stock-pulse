import { render, screen } from '@testing-library/react';
import PortfolioSummaryCards from './PortfolioSummaryCards';
import React from 'react';
import { ThemeProvider } from '../ThemeProvider';

describe('PortfolioSummaryCards Component', () => {
  it('renders all KPI summary cards correctly with positive profit', () => {
    render(
      <ThemeProvider>
        <PortfolioSummaryCards
          totalCost={10000}
          currentValue={12500}
          profitLoss={2500}
          returnPercent={25}
          avgDividends12m={150}
          kpiCurrency="BRL"
          isLoadingTreasury={true}
        />
      </ThemeProvider>
    );

    expect(screen.getByText(/Patrimônio Atual/i)).toBeInTheDocument();
    expect(screen.getByText(/Total Investido/i)).toBeInTheDocument();
    expect(screen.getByText(/Lucro \/ Prejuízo/i)).toBeInTheDocument();
    expect(screen.getByText(/Média de Proventos/i)).toBeInTheDocument();
    expect(screen.getByText(/Tesouro.../i)).toBeInTheDocument();
  });

  it('renders negative profitLoss correctly', () => {
    render(
      <ThemeProvider>
        <PortfolioSummaryCards
          totalCost={10000}
          currentValue={8000}
          profitLoss={-2000}
          returnPercent={-20}
          avgDividends12m={0}
          kpiCurrency="BRL"
        />
      </ThemeProvider>
    );

    expect(screen.getByText(/Lucro \/ Prejuízo/i)).toBeInTheDocument();
  });
});
