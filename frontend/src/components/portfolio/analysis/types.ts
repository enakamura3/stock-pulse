import { Position, CalculatedDividend, FixedIncomePosition, PerformancePoint, TreasuryPosition } from '../types';

export interface PortfolioAnalysisProps {
  positions: Position[];
  dividends: CalculatedDividend[];
  fiPositions: FixedIncomePosition[];
  treasuryPositions?: TreasuryPosition[];
  performanceData: PerformancePoint[];
  kpiCurrency: string;
}

export interface BenchmarkPoint {
  label: string;
  portfolio: number;
  cdi: number;
  ipca: number;
  ifix: number;
  ibov: number;
  sp500: number;
}
