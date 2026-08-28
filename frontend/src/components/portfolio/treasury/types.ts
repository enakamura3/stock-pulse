import { TreasuryPosition, TreasuryPerfPoint } from '../types';

export interface NewTreasuryTx {
  ticker: string;
  treasury_type: string;
  maturity_date: string;
  has_coupons: boolean;
  type: 'SUBSCRIPTION' | 'REDEMPTION';
  quantity: number | '';
  unit_price: number | '';
  contracted_rate: number | '';
  transaction_date: string;
}

export type SortKey =
  | 'ticker'
  | 'treasury_type'
  | 'maturity_date'
  | 'total_invested'
  | 'gross_value'
  | 'net_value'
  | 'net_return'
  | 'iof_tax'
  | 'ir_tax'
  | 'b3_fee'
  | 'status';

export type SortDir = 'asc' | 'desc';

export function fmt(value: number, currency = 'BRL'): string {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

export function fmtPct(value: number): string {
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(2)}%`;
}

export function getTreasuryTypeLabel(t: string): string {
  const map: Record<string, string> = {
    SELIC: '📈 Tesouro Selic',
    PREFIXADO: '🔒 Prefixado',
    'IPCA+': '🏷️ IPCA+',
  };
  return map[t] || t;
}

export function getTreasuryTypeBadgeColor(t: string): string {
  const map: Record<string, string> = {
    SELIC: '#4caf50',
    PREFIXADO: '#2196f3',
    'IPCA+': '#ff9800',
  };
  return map[t] || '#9e9e9e';
}
