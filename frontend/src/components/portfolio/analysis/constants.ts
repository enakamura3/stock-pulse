export const ALLOCATION_COLORS: Record<string, string> = {
  'Renda Variável': '#60a5fa',
  'Renda Fixa': '#fbbf24',
};

export const CATEGORY_COLORS: Record<string, string> = {
  'Ações (B3)': '#60a5fa',
  'FIIs': '#c084fc',
  'FIAGROs': '#a78bfa',
  'ETFs Nacionais': '#34d399',
  'BDRs': '#f87171',
  'Ações EUA': '#38bdf8',
  'ETF Internacional': '#2dd4bf',
  'Cripto': '#fb923c',
  'Renda Fixa': '#fbbf24',
  'Tesouro Direto': '#10b981',
  'Desconhecido': '#94a3b8',
};

export const EXPOSURE_COLORS = {
  local: '#4ade80',
  global: '#60a5fa',
};

export const BENCHMARK_COLORS = {
  portfolio: 'var(--accent-color)',
  cdi: '#fbbf24',
  ipca: '#f87171',
  ifix: '#c084fc',
  ibov: '#f472b6',
  sp500: '#34d399',
};

export const DIVIDENDS_COLORS = {
  nacionais: 'var(--color-success)',
  internacionais: 'var(--accent-color)',
  rendaFixa: 'var(--color-warning)',
};

export const MONTHS_LABEL = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];

export const isRendaVariavel = (type: string) =>
  ['STOCK_BR', 'FII', 'FIAGRO', 'ETF_BR', 'BDR', 'STOCK_US', 'ETF_US', 'CRYPTO'].includes(type);

export const isExposicaoGlobal = (type: string) =>
  ['BDR', 'STOCK_US', 'ETF_US'].includes(type);

export const isFII = (type: string) =>
  ['FII', 'FIAGRO'].includes(type);

export const isAcaoOuETF = (type: string) =>
  ['STOCK_BR', 'STOCK_US', 'ETF_BR', 'ETF_US', 'BDR'].includes(type);
