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
}

export interface CalculatedDividend {
  asset_id: string;
  ticker: string;
  ex_date: string;
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
}

export interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
  type: string;
}
