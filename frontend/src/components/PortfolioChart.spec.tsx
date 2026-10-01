import { render, screen, fireEvent } from '@testing-library/react';
import PortfolioChart from './PortfolioChart';
import React from 'react';
import { vi } from 'vitest';
import { ThemeProvider } from './ThemeProvider';

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
    createChart: vi.fn().mockImplementation(() => ({
      addSeries: addSeriesMock,
      timeScale: vi.fn().mockReturnValue({ fitContent: fitContentMock }),
      remove: removeMock,
      applyOptions: applyOptionsMock,
    })),
  };
});

describe('PortfolioChart', () => {
  const data = [
    { date: '2023-01-01', value: 100, total_invested: 90 },
    { date: '2023-01-02', value: 105, total_invested: 0 }, // total_invested = 0 to cover val = 0 in percent mode
  ];

  afterEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('renders without crashing with empty data', () => {
    const { container } = render(
      <ThemeProvider>
        <PortfolioChart data={[]} />
      </ThemeProvider>
    );
    expect(container).toBeInTheDocument();
  });

  it('renders with data, handles light theme, resize, toggles and percent mode', () => {
    localStorage.setItem('theme', 'light');
    document.documentElement.style.setProperty('--accent-color', 'rgb(50, 100, 150)');

    const { rerender } = render(
      <ThemeProvider>
        <PortfolioChart data={data} />
      </ThemeProvider>
    );

    // Trigger window resize
    window.dispatchEvent(new Event('resize'));

    // Toggle to percent mode
    const percentBtn = screen.getByRole('button', { name: '%' });
    fireEvent.click(percentBtn);

    // Toggle back to currency mode
    const currencyBtn = screen.getByRole('button', { name: 'R$' });
    fireEvent.click(currencyBtn);

    // Toggle showValue checkbox
    const showValueCheckbox = screen.getByLabelText('Evolução Patrimonial');
    fireEvent.click(showValueCheckbox);
    fireEvent.click(showValueCheckbox);

    // Toggle showInvested checkbox
    const showInvestedCheckbox = screen.getByLabelText('Valor Investido');
    fireEvent.click(showInvestedCheckbox);
    fireEvent.click(showInvestedCheckbox);

    // Rerender to test cleanup and re-creation
    rerender(
      <ThemeProvider>
        <PortfolioChart data={[...data, { date: '2023-01-03', value: 110, total_invested: 95 }]} />
      </ThemeProvider>
    );
  });
});
