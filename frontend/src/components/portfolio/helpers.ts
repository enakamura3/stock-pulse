export const getAssetCategory = (dbType: string) => {
  switch (dbType) {
    case 'STOCK_BR': return 'Ações (B3)';
    case 'FII': return 'FIIs';
    case 'FIAGRO': return 'FIAGROs';
    case 'ETF_BR': return 'ETFs Nacionais';
    case 'BDR': return 'BDRs';
    case 'STOCK_US': return 'Ações EUA';
    case 'ETF_US': return 'ETF Internacional';
    case 'CRYPTO': return 'Cripto';
    case 'CDB':
    case 'LCI':
    case 'LCA':
    case 'TESOURO':
    case 'DEBENTURE':
    case 'CRI':
    case 'CRA': return 'Renda Fixa';
    default: return 'Desconhecido';
  }
};

export const formatMoney = (val: number, currency: string) => {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: currency || 'BRL',
  }).format(val);
};

export const formatPercentage = (val: number) => {
  const formatted = Math.abs(val) < 1e-6 ? 0 : val;
  const isPos = formatted > 1e-6;
  const fixed = formatted.toFixed(2);
  if (fixed === '0.00' || fixed === '-0.00') {
    return '0.00%';
  }
  return `${isPos ? '+' : ''}${fixed}%`;
};

export const formatQuantity = (val: number) => {
  return new Intl.NumberFormat('pt-BR', {
    maximumFractionDigits: 3,
  }).format(val);
};
