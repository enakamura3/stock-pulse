export interface Portfolio {
  id: string;
  user_id: string;
  name: string;
  base_currency: string;
  created_at: string;
}

export interface Position {
  asset_id: string;
  ticker: string;
  name: string;
  type: string;
  currency: string;
  quantity: number;
  average_price: number;
  total_cost: number;
  current_price?: number;
  current_value?: number;
  profit_loss?: number;
  return_percent?: number;
  daily_change?: number;
  daily_change_percent?: number;
  graham_value?: number;
  bazin_value?: number;
  pvp?: number;
  pe?: number;
  dividend_yield?: number;
}

export interface Transaction {
  id: string;
  portfolio_id: string;
  asset_id: string;
  ticker?: string;
  asset_name?: string;
  asset_type?: string;
  currency?: string;
  type: string; // "BUY" ou "SELL"
  quantity: number;
  unit_price: number;
  total_cost: number;
  exchange_rate: number;
  executed_at: string;
  created_at: string;
}

export interface PerformancePoint {
  date: string;
  value: number;
  total_invested: number;
  return_pct?: number;
  cdi_return_pct?: number;
  ipca_return_pct?: number;
  ifix_return_pct?: number;
  ibov_return_pct?: number;
  sp500_return_pct?: number;
}

export interface CalculatedDividend {
  asset_id: string;
  ticker: string;
  cum_date: string;
  payment_date: string;
  gross_amount: number;
  net_amount: number;
  currency: string;
  original_gross_amount?: number;
  original_net_amount?: number;
  type: string;
  quantity: number;
  per_share_amount: number;
  asset_type: string;
  asset_name: string;
  is_accrued?: boolean;
}

export interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
  type: string;
}

export interface FixedIncomeAsset {
  id: string;
  portfolio_id: string;
  institution: string;
  type: string; // CDB, LCI, LCA, TESOURO
  debt_type: string; // PRE, POS, HIBRIDO
  indexer: string; // CDI, SELIC, IPCA, PREFIXADO
  rate: number;
  maturity_date: string;
  created_at?: string;
  updated_at?: string;
}

export interface FixedIncomeTransaction {
  id: string;
  asset_id: string;
  type: string; // BUY, SELL, MATURITY
  amount: number;
  date: string;
  created_at?: string;
}

export interface FixedIncomePosition {
  asset: FixedIncomeAsset;
  start_date: string;
  total_invested: number;
  gross_value: number;
  net_value: number;
  net_return_percent: number;
  gross_return_percent: number;
  iof_amount: number;
  ir_amount: number;
  ir_rate: number;
  iof_rate: number;
  days_in_portfolio: number;
  days_to_maturity: number;
  is_matured: boolean;
}

export interface UnifiedTransaction {
  id: string;
  portfolio_id: string;
  module: string;
  date: string;
  asset_name: string;
  asset_type: string;
  type: string;
  quantity: number | null;
  unit_price: number | null;
  exchange_rate: number | null;
  total_value: number;
  currency: string;
  maturity_date?: string;
  resulting_quantity?: number;
  resulting_invested?: number;
}

export interface TreasuryPosition {
  asset_id: string;
  ticker: string;
  treasury_type: string;
  maturity_date: string;
  has_coupons: boolean;
  start_date: string;
  total_invested: number;
  gross_value: number;
  net_value: number;
  is_matured: boolean;
  days_to_maturity: number;
  taxes_calculated: number;
  b3_fee: number;
  ir_tax: number;
  iof_tax: number;
}

export interface TreasuryPerfPoint {
  date: string;
  value: number;
  total_invested: number;
}
