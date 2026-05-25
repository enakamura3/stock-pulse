import { render } from '@testing-library/react';
import PortfolioChart from './PortfolioChart';
import React from 'react';
import { vi } from 'vitest';

vi.mock('lightweight-charts', () => {
  const addSeriesMock = vi.fn().mockReturnValue({
    setData: vi.fn(),
  });
  const fitContentMock = vi.fn();
  const removeMock = vi.fn();
  const applyOptionsMock = vi.fn();

  return {
    AreaSeries: 'AreaSeries',
    LineSeries: 'LineSeries',
    ColorType: { Solid: 'Solid' },
    createChart: vi.fn().mockReturnValue({
      addSeries: addSeriesMock,
      timeScale: vi.fn().mockReturnValue({ fitContent: fitContentMock }),
      remove: removeMock,
      applyOptions: applyOptionsMock,
    }),
  };
});

describe('PortfolioChart', () => {
  it('renders without crashing with empty data', () => {
    const { container } = render(<PortfolioChart data={[]} />);
    expect(container).toBeInTheDocument();
  });

  it('renders and calls chart creation with data', () => {
    const data = [
      { date: '2023-01-01', value: 100, total_invested: 90 },
      { date: '2023-01-02', value: 105, total_invested: 90 },
    ];
    const { container } = render(<PortfolioChart data={data} />);
    expect(container).toBeInTheDocument();
  });
});
