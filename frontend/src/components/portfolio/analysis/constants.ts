export const ALLOCATION_COLORS: Record<string, string> = {
  'Renda Variável': 'var(--accent-color)',
  'Renda Fixa': 'var(--color-warning)',
};

export const CATEGORY_COLORS: Record<string, string> = {
  'Ações (B3)': 'var(--accent-color)',
  'FIIs': 'var(--color-info)',
  'FIAGROs': 'var(--color-success)',
  'ETFs Nacionais': 'var(--color-info)',
  'BDRs': 'var(--color-warning)',
  'Ações EUA': 'var(--color-danger)',
  'ETF Internacional': 'var(--accent-color)',
  'Cripto': 'var(--color-warning)',
  'Renda Fixa': 'var(--color-warning)',
  'Tesouro Direto': 'var(--color-success)',
  'Desconhecido': 'var(--text-muted)',
};

export const EXPOSURE_COLORS = {
  local: 'var(--color-success)',
  global: 'var(--accent-color)',
};

export const BENCHMARK_COLORS = {
  portfolio: 'var(--accent-color)',
  cdi: 'var(--color-warning)',
  ipca: 'var(--color-danger)',
  ifix: 'var(--color-info)',
  ibov: '#ec4899',
  sp500: 'var(--color-success)',
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
