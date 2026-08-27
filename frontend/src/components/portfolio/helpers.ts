import type { Position, FixedIncomePosition, TreasuryPosition } from './types';

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

/**
 * Exporta os dados do Resumo Diário para arquivo CSV com download automático.
 */
export function exportDailyReportCSV(
  positions: Position[],
  fiPositions: FixedIncomePosition[] = [],
  treasuryPositions: TreasuryPosition[] = [],
  kpiCurrency: string = 'BRL',
  filenamePrefix = 'resumo_diario'
): string {
  const lines: string[] = [];

  // Seção 1: Renda Variável
  lines.push('--- RENDA VARIÁVEL ---');
  lines.push('Ticker,Nome,Categoria,Quantidade,Preço Médio,Fech. Anterior,Preço Atual,Var. Dia (R$),Var. Dia (%),Impacto Carteira (R$)');

  positions.forEach(pos => {
    const isUSD = pos.currency?.toUpperCase() === 'USD' || pos.type === 'STOCK_US' || pos.type === 'ETF_US';
    const rate = (kpiCurrency === 'BRL' && isUSD) ? (pos.fx_rate_to_brl ?? 1.0) : 1.0;
    const absChange = pos.daily_change ?? 0;
    const currentPrice = pos.current_price ?? 0;
    const prevClose = (pos.previous_close != null && pos.previous_close > 1e-6)
      ? pos.previous_close
      : currentPrice - absChange;
    const impact = absChange * (pos.quantity ?? 0) * rate;

    const safeTicker = `"${(pos.ticker || '').replace(/"/g, '""')}"`;
    const safeName = `"${(pos.name || '').replace(/"/g, '""')}"`;
    const safeCat = `"${getAssetCategory(pos.type)}"`;

    lines.push([
      safeTicker,
      safeName,
      safeCat,
      (pos.quantity ?? 0).toString(),
      (pos.average_price ?? 0).toFixed(2),
      prevClose.toFixed(2),
      currentPrice.toFixed(2),
      absChange.toFixed(2),
      (pos.daily_change_percent ?? 0).toFixed(2),
      impact.toFixed(2),
    ].join(','));
  });

  // Seção 2: Renda Fixa Privada
  if (fiPositions.length > 0) {
    lines.push('');
    lines.push('--- RENDA FIXA PRIVADA ---');
    lines.push('Instituição,Tipo,Taxa,Vencimento,Valor Líquido (R$),Taxa Diária Est. (%),Ganho Diário Est. (R$),Rent. Acumulada (%)');

    fiPositions.forEach(p => {
      const taxa = p.asset.debt_type === 'POS'
        ? `${p.asset.rate.toFixed(2)}% ${p.asset.indexer}`
        : `${p.asset.rate.toFixed(2)}% a.a.`;
      const dailyRatePct = calculateDailyFixedIncomeRate(p.asset.indexer || p.asset.debt_type, p.asset.rate);
      const dailyGain = calculateEstimatedDailyGain(p.net_value ?? 0, dailyRatePct);

      lines.push([
        `"${(p.asset.institution || '').replace(/"/g, '""')}"`,
        `"${p.asset.type}"`,
        `"${taxa}"`,
        p.asset.maturity_date ? p.asset.maturity_date.split('T')[0] : '',
        (p.net_value ?? 0).toFixed(2),
        dailyRatePct.toFixed(4),
        dailyGain.toFixed(2),
        (p.net_return_percent ?? 0).toFixed(2),
      ].join(','));
    });
  }

  // Seção 3: Tesouro Direto
  if (treasuryPositions.length > 0) {
    lines.push('');
    lines.push('--- TESOURO DIRETO ---');
    lines.push('Título,Tipo,Vencimento,Valor Líquido (R$),Taxa Diária Est. (%),Ganho Diário Est. (R$),Rent. Acumulada (%)');

    treasuryPositions.forEach(p => {
      const returnPct = p.total_invested > 1e-6
        ? ((p.net_value - p.total_invested) / p.total_invested) * 100
        : 0;
      const dailyRatePct = calculateDailyFixedIncomeRate(p.treasury_type, p.contracted_rate ?? 0);
      const dailyGain = calculateEstimatedDailyGain(p.net_value ?? 0, dailyRatePct);

      lines.push([
        `"${(p.ticker || '').replace(/"/g, '""')}"`,
        `"${p.treasury_type}"`,
        p.maturity_date ? p.maturity_date.split('T')[0] : '',
        (p.net_value ?? 0).toFixed(2),
        dailyRatePct.toFixed(4),
        dailyGain.toFixed(2),
        returnPct.toFixed(2),
      ].join(','));
    });
  }

  const csvContent = lines.join('\n');

  if (typeof window !== 'undefined' && typeof document !== 'undefined' && typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function') {
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    const today = new Date().toISOString().split('T')[0];
    link.setAttribute('href', url);
    link.setAttribute('download', `${filenamePrefix}_${today}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    if (typeof URL.revokeObjectURL === 'function') {
      URL.revokeObjectURL(url);
    }
  }

  return csvContent;
}


