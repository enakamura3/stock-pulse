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

export const DEFAULT_ANNUAL_CDI = 10.40;
export const DEFAULT_ANNUAL_SELIC = 10.50;

/**
 * Calcula a taxa diária equivalente (base 252 dias úteis) para um ativo de renda fixa.
 * Retorna a taxa diária em percentual (ex: 0.0415 para 0.0415% ao dia).
 */
export function calculateDailyFixedIncomeRate(
  indexer: string,
  rate: number,
  benchmarkAnnualRate: number = DEFAULT_ANNUAL_CDI
): number {
  const indexerUpper = (indexer || '').toUpperCase();
  let effectiveAnnualRate = 0;

  if (indexerUpper === 'PREFIXADO' || indexerUpper === 'PRE') {
    effectiveAnnualRate = rate / 100;
  } else if (indexerUpper === 'CDI' || indexerUpper === 'POS') {
    effectiveAnnualRate = (benchmarkAnnualRate / 100) * (rate / 100);
  } else if (indexerUpper === 'SELIC') {
    effectiveAnnualRate = (benchmarkAnnualRate || DEFAULT_ANNUAL_SELIC) / 100;
  } else if (indexerUpper === 'IPCA' || indexerUpper === 'HIBRIDO') {
    // Parcela pré/spread anual
    effectiveAnnualRate = rate / 100;
  } else {
    effectiveAnnualRate = rate / 100;
  }

  if (effectiveAnnualRate <= -1) {
    return 0;
  }

  // r_dia = (1 + r_anual)^(1/252) - 1
  const dailyRateFraction = Math.pow(1 + effectiveAnnualRate, 1 / 252) - 1;
  return dailyRateFraction * 100; // em %
}

/**
 * Calcula o ganho financeiro estimado em 1 dia útil para a posição de renda fixa.
 */
export function calculateEstimatedDailyGain(netValue: number, dailyRatePercent: number): number {
  if (netValue < 1e-6 || dailyRatePercent < 1e-6) {
    return 0;
  }
  return netValue * (dailyRatePercent / 100);
}

