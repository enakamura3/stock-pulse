import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import RiskConcentrationSection from '../analysis/RiskConcentrationSection';
import { Position, PerformancePoint } from '../types';


describe('RiskConcentrationSection', () => {
  const mockPositions: Position[] = [
    {
      asset_id: '1',
      ticker: 'PETR4',
      name: 'Petrobras PN',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 100,
      average_price: 20,
      total_cost: 2000,
      current_price: 30,
      current_value: 3000, // weight = 3000/4000 = 75%, return = 50%
      return_percent: 50,
      profit_loss: 1000,
    },
    {
      asset_id: '2',
      ticker: 'VALE3',
      name: 'Vale ON',
      type: 'STOCK_BR',
      currency: 'BRL',
      quantity: 20,
      average_price: 60,
      total_cost: 1200,
      current_price: 50,
      current_value: 1000, // weight = 1000/4000 = 25%, return = -16.67%
      return_percent: -16.67,
      profit_loss: -200,
    },
  ];

  // At least 10 performance points needed for risk metrics
  const mockPerformance: PerformancePoint[] = Array.from({ length: 15 }, (_, i) => ({
    date: `2024-01-${String(i + 1).padStart(2, '0')}`,
    value: 4000 + i * 10,
    total_invested: 3200,
  }));

  it('renders risk metrics and performance attribution scorecard', () => {
    render(
      <RiskConcentrationSection
        positions={mockPositions}
        fiPositions={[]}
        treasuryPositions={[]}
        performanceData={mockPerformance}
        kpiCurrency="BRL"
      />
    );

    expect(screen.getByText(/Termômetro de Risco/i)).toBeInTheDocument();

    const titleMatches = screen.getAllByText(/Atribuição de Performance/i);
    expect(titleMatches.length).toBeGreaterThanOrEqual(1);

    // Contribution calculation verification:
    // PETR4: weight 75% * 50% return / 100 = +37.50pp
    // VALE3: weight 25% * -16.67% return / 100 = -4.17pp
    // Total contribution = +33.33pp
    const valueMatches = screen.getAllByText(/\+33.33pp/i);
    expect(valueMatches.length).toBeGreaterThanOrEqual(1);

    // Click the scorecard to expand children
    const attributionCard = titleMatches[titleMatches.length - 1].closest('div[style*="cursor: pointer"]')!;
    fireEvent.click(attributionCard);


    expect(screen.getByText(/PETR4/i)).toBeInTheDocument();
    expect(screen.getByText(/\+37.50pp/i)).toBeInTheDocument();
    expect(screen.getByText(/VALE3/i)).toBeInTheDocument();
    expect(screen.getByText(/-4.17pp/i)).toBeInTheDocument();
  });

});
