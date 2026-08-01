export interface Item {
  id: string;
  watchlist_id: string;
  asset_id: string;
  ticker: string;
  name: string;
  type: string;
  currency: string;
  added_at: string;
  price?: number;
  change?: number;
  change_percent?: number;
  graham_value?: number;
  bazin_value?: number;
}

export interface Watchlist {
  id: string;
  user_id: string;
  name: string;
  created_at: string;
  items?: Item[];
}

export interface SearchResult {
  symbol: string;
  name: string;
  exchange: string;
  type: string;
}

export interface Quote {
  symbol: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
  high: number;
  low: number;
  volume: number;
  currency: string;
}
