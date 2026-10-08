import { render, screen, fireEvent, act } from '@testing-library/react';
import PortfolioChart from './PortfolioChart';
import React from 'react';
import { vi } from 'vitest';
import { ThemeProvider } from './ThemeProvider';

let crosshairMoveHandler: ((param: any) => void) | null = null;
const subscribeCrosshairMoveMock = vi.fn((fn) => {
  crosshairMoveHandler = fn;
});
const unsubscribeCrosshairMoveMock = vi.fn();
const addSeriesMock = vi.fn().mockReturnValue({
  setData: vi.fn(),
});
const fitContentMock = vi.fn();
const removeMock = vi.fn();
const applyOptionsMock = vi.fn();
const createChartMock = vi.fn().mockImplementation(() => ({
  addSeries: addSeriesMock,
  timeScale: vi.fn().mockReturnValue({ fitContent: fitContentMock }),
  remove: removeMock,
  applyOptions: applyOptionsMock,
  subscribeCrosshairMove: subscribeCrosshairMoveMock,
  unsubscribeCrosshairMove: unsubscribeCrosshairMoveMock,
}));

vi.mock('lightweight-charts', () => {
  return {
    AreaSeries: 'AreaSeries',
    LineSeries: 'LineSeries',
    LineStyle: {
      Solid: 0,
      Dotted: 1,
      Dashed: 2,
      LargeDashed: 3,
      SparseDotted: 4,
    },
    ColorType: { Solid: 'Solid' },
    createChart: (...args: any[]) => createChartMock(...args),
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
    crosshairMoveHandler = null;
  });

  it('renders without crashing with empty data', () => {
    const { container } = render(
      <ThemeProvider>
        <PortfolioChart data={[]} />
      </ThemeProvider>
    );
    expect(container).toBeInTheDocument();
    expect(screen.queryByTestId('portfolio-chart-hud')).not.toBeInTheDocument();
  });

  it('renders with data, disables attribution logo, handles light theme, resize, and toggles', () => {
    localStorage.setItem('theme', 'light');
    document.documentElement.style.setProperty('--accent-color', 'rgb(50, 100, 150)');

    const { rerender } = render(
      <ThemeProvider>
        <PortfolioChart data={data} />
      </ThemeProvider>
    );

    // Verify attributionLogo: false was passed to createChart
    expect(createChartMock).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        layout: expect.objectContaining({
          attributionLogo: false,
        }),
      })
    );

    // Verify HUD is displayed with the last point (2023-01-02)
    const hud = screen.getByTestId('portfolio-chart-hud');
    expect(hud).toBeInTheDocument();
    expect(hud).toHaveTextContent('2023-01-02');

    // Trigger crosshair move with valid string time
    act(() => {
      crosshairMoveHandler?.({
        time: '2023-01-01',
        point: { x: 10, y: 10 },
      });
    });
    expect(screen.getByTestId('portfolio-chart-hud')).toHaveTextContent('2023-01-01');

    // Trigger crosshair move with object time
    act(() => {
      crosshairMoveHandler?.({
        time: { year: 2023, month: 1, day: 2 },
        point: { x: 20, y: 20 },
      });
    });
    expect(screen.getByTestId('portfolio-chart-hud')).toHaveTextContent('2023-01-02');

    // Trigger crosshair move with numeric timestamp
    act(() => {
      crosshairMoveHandler?.({
        time: 1672531200,
        point: { x: 25, y: 25 },
      });
    });

    // Trigger crosshair move with unknown time
    act(() => {
      crosshairMoveHandler?.({
        time: '2029-99-99',
        point: { x: 30, y: 30 },
      });
    });
    // Resets to default/last point
    expect(screen.getByTestId('portfolio-chart-hud')).toHaveTextContent('2023-01-02');

    // Trigger crosshair move with undefined/leaving
    act(() => {
      crosshairMoveHandler?.(null);
    });
    expect(screen.getByTestId('portfolio-chart-hud')).toHaveTextContent('2023-01-02');

    // Trigger window resize
    window.dispatchEvent(new Event('resize'));

    // Toggle to percent mode
    const percentBtn = screen.getByRole('button', { name: '%' });
    fireEvent.click(percentBtn);

    // Toggle back to currency mode
    const currencyBtn = screen.getByRole('button', { name: 'R$' });
    fireEvent.click(currencyBtn);

    // Toggle showValue checkbox
    const showValueCheckbox = screen.getByLabelText(/Evolução Patrimonial/i);
    fireEvent.click(showValueCheckbox);
    fireEvent.click(showValueCheckbox);

    // Toggle showInvested checkbox
    const showInvestedCheckbox = screen.getByLabelText(/Valor Investido/i);
    fireEvent.click(showInvestedCheckbox);
    fireEvent.click(showInvestedCheckbox);

    // Rerender with negative return data to cover isPos = false branch
    rerender(
      <ThemeProvider>
        <PortfolioChart data={[...data, { date: '2023-01-03', value: 80, total_invested: 100 }]} />
      </ThemeProvider>
    );

    expect(screen.getByTestId('portfolio-chart-hud')).toHaveTextContent('2023-01-03');
  });
});
