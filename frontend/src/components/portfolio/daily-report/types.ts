import { Position, CalculatedDividend } from '../types';

export type SortKey = 'ticker' | 'average_price' | 'previousClose' | 'current_price' | 'daily_change' | 'daily_change_percent' | 'impact';
export type SortDir = 'asc' | 'desc';

export interface AssetTypeBadge {
  label: string;
  color: string;
}

export interface EnrichedPosition {
  pos: Position;
  percent: number;
  absChange: number;
  currentPrice: number;
  previousClose: number;
  qty: number;
  rate: number;
  impact: number;
  portfolioWeight: number;
}

export interface DailyReportCalculations {
  totalDailyChange: number;
  totalPortfolioValue: number;
  totalDailyPercent: number;
  isDailyPos: boolean;
  totalEstimatedFixedIncomeGain: number;
  todayDividends: CalculatedDividend[];
  totalTodayDividends: number;
  enrichedPositions: EnrichedPosition[];
  sortedRows: EnrichedPosition[];
  topRisers: EnrichedPosition[];
  topFallers: EnrichedPosition[];
  sortKey: SortKey;
  sortDir: SortDir;
  handleSort: (key: SortKey) => void;
  getAssetTypeBadge: (pos: Position) => AssetTypeBadge | null;
}
