import type { Position, FixedIncomePosition, TreasuryPosition } from './types';

export const ASSET_TYPE_OPTIONS = [
  { value: 'STOCK_BR', label: 'Ação (B3) — ex: PETR4, TAEE11, SANB11' },
  { value: 'FII', label: 'FII – Fundo Imobiliário — ex: HGLG11, MXRF11' },
  { value: 'FIAGRO', label: 'Fiagro — ex: VGIA11, KNCA11' },
  { value: 'ETF_BR', label: 'ETF Nacional (B3) — ex: BOVA11, SMAL11, IVVB11' },
  { value: 'BDR', label: 'BDR — ex: AAPL34, MSFT34' },
  { value: 'STOCK_US', label: 'Ação Internacional (EUA) — ex: AAPL, MSFT' },
  { value: 'ETF_US', label: 'ETF Internacional (EUA) — ex: SPY, QQQ, VOO' },
  { value: 'CRYPTO', label: 'Criptoativo — ex: BTC-USD, ETH-BRL' },
] as const;

export function determineAssetTypeLocal(ticker: string, name: string, currency: string): string {
  const upperTicker = (ticker || '').toUpperCase().trim();
  const upperCurrency = (currency || '').toUpperCase().trim();
  const lowerName = (name || '').toLowerCase().trim();

  if (upperTicker.includes('-') || upperCurrency === 'CRYPTO') {
    return 'CRYPTO';
  }

  if (!upperTicker.endsWith('.SA')) {
    if (lowerName.includes('etf') || lowerName.includes('trust') || lowerName.includes('fund')) {
      return 'ETF_US';
    }
    return 'STOCK_US';
  }

  // É do Brasil (.SA)
  if (upperTicker.endsWith('34.SA') || upperTicker.endsWith('35.SA') || upperTicker.endsWith('39.SA')) {
    return 'BDR';
  }

  if (upperTicker.endsWith('11.SA')) {
    if (lowerName.includes('fiagro') || lowerName.includes('agro')) {
      return 'FIAGRO';
    }

    const isEtf =
      lowerName.includes('etf') ||
      lowerName.includes('ishares') ||
      lowerName.includes('índice') ||
      lowerName.includes('indice') ||
      lowerName.includes('sp500') ||
      lowerName.includes('nasdaq') ||
      lowerName.includes('bovespa') ||
      lowerName.includes('hashdex') ||
      lowerName.includes('trend') ||
      lowerName.includes('investo');
    if (isEtf) {
      return 'ETF_BR';
    }

    const isFii =
      lowerName.includes('fii') ||
      lowerName.includes('fundo') ||
      lowerName.includes('fdo') ||
      lowerName.includes('imob') ||
      lowerName.includes('lajes') ||
      lowerName.includes('shopping') ||
      lowerName.includes('logística') ||
      lowerName.includes('logistica') ||
      lowerName.includes('tijolo') ||
      lowerName.includes('recebíveis') ||
      lowerName.includes('recebiveis');
    if (isFii) {
      return 'FII';
    }

    return 'STOCK_BR';
  }

  return 'STOCK_BR';
}

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

/**
 * Converte um valor numérico ou string monetária (ex: "2,35", "1.234,56", 2.35) em float numérico.
 * Trata vírgulas e pontos decimais de forma resiliente.
 */
export function parseCurrency(val: string | number | undefined | null): number {
  if (val === undefined || val === null || val === '') return 0;
  if (typeof val === 'number') return isNaN(val) ? 0 : val;
  const str = val.toString().trim();
  if (!str) return 0;

  // Remove caracteres não numéricos exceto vírgula, ponto e sinal de menos
  const sanitized = str.replace(/[^\d,.-]/g, '');
  if (!sanitized) return 0;

  if (sanitized.includes(',') && sanitized.includes('.')) {
    if (sanitized.lastIndexOf(',') > sanitized.lastIndexOf('.')) {
      // Formato brasileiro: 1.234,56
      const cleaned = sanitized.replace(/\./g, '').replace(',', '.');
      const parsed = parseFloat(cleaned);
      return isNaN(parsed) ? 0 : parsed;
    } else {
      // Formato internacional: 1,234.56
      const cleaned = sanitized.replace(/,/g, '');
      const parsed = parseFloat(cleaned);
      return isNaN(parsed) ? 0 : parsed;
    }
  }

  if (sanitized.includes(',')) {
    const parsed = parseFloat(sanitized.replace(',', '.'));
    return isNaN(parsed) ? 0 : parsed;
  }

  const parsed = parseFloat(sanitized);
  return isNaN(parsed) ? 0 : parsed;
}

/**
 * Aplica máscara de entrada monetária estilo ATM/caixa eletrônico (deslocamento da direita para a esquerda).
 * Ex: Digitar "2" -> "0,02"; "23" -> "0,23"; "235" -> "2,35"; "2350" -> "23,50".
 * Se receber um número (ex: ao carregar edição de transação), formata com 2 casas decimais.
 */
export function formatCurrencyInput(val: string | number | undefined | null): string {
  if (val === undefined || val === null || val === '') return '';

  if (typeof val === 'number') {
    if (isNaN(val) || Math.abs(val) < 1e-6) return '';
    return val.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  }

  const digits = val.replace(/\D/g, '');
  if (!digits || Number(digits) === 0) return '';

  const num = Number(digits) / 100;
  return num.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

/**
 * Trata colagem (paste) de valores monetários, reconhecendo números inteiros ou decimais com ponto/vírgula.
 * Ex: Colar "25" -> "25,00"; "2.35" -> "2,35"; "R$ 1.250,50" -> "1.250,50".
 */
export function parsePastedCurrency(pasted: string): string {
  const trimmed = (pasted || '').trim();
  if (!trimmed) return '';

  const sanitized = trimmed.replace(/[^\d,.-]/g, '');
  if (!sanitized) return '';

  let num: number;
  if (sanitized.includes(',') && sanitized.includes('.')) {
    if (sanitized.lastIndexOf(',') > sanitized.lastIndexOf('.')) {
      num = parseFloat(sanitized.replace(/\./g, '').replace(',', '.'));
    } else {
      num = parseFloat(sanitized.replace(/,/g, ''));
    }
  } else if (sanitized.includes(',')) {
    num = parseFloat(sanitized.replace(',', '.'));
  } else if (sanitized.includes('.')) {
    num = parseFloat(sanitized);
  } else {
    // Número inteiro: "25" -> 25.00
    const digitsOnly = sanitized.replace(/\D/g, '');
    num = digitsOnly ? parseFloat(digitsOnly) : 0;
  }

  if (isNaN(num) || Math.abs(num) < 1e-6) return '';
  return num.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

/**
 * Aplica máscara ATM para taxa de câmbio (4 casas decimais, ex: 5,2500).
 * Ex: Digitar "5" -> "0,0005"; "52" -> "0,0052"; "52500" -> "5,2500".
 * Se receber um número (ao carregar edição), formata com 4 casas decimais.
 */
export function formatExchangeRateInput(val: string | number | undefined | null): string {
  if (val === undefined || val === null || val === '') return '';

  if (typeof val === 'number') {
    if (isNaN(val) || Math.abs(val) < 1e-6) return '';
    return val.toLocaleString('pt-BR', { minimumFractionDigits: 4, maximumFractionDigits: 4 });
  }

  const digits = val.replace(/\D/g, '');
  if (!digits || Number(digits) === 0) return '';

  const num = Number(digits) / 10000;
  return num.toLocaleString('pt-BR', { minimumFractionDigits: 4, maximumFractionDigits: 4 });
}

/**
 * Trata colagem de taxa de câmbio, preservando 4 casas decimais.
 * Ex: Colar "5.25" -> "5,2500"; "5,2500" -> "5,2500"; "5" -> "5,0000".
 */
export function parsePastedExchangeRate(pasted: string): string {
  const trimmed = (pasted || '').trim();
  if (!trimmed) return '';

  const sanitized = trimmed.replace(/[^\d,.-]/g, '');
  if (!sanitized) return '';

  let num: number;
  if (sanitized.includes(',') && sanitized.includes('.')) {
    if (sanitized.lastIndexOf(',') > sanitized.lastIndexOf('.')) {
      num = parseFloat(sanitized.replace(/\./g, '').replace(',', '.'));
    } else {
      num = parseFloat(sanitized.replace(/,/g, ''));
    }
  } else if (sanitized.includes(',')) {
    num = parseFloat(sanitized.replace(',', '.'));
  } else if (sanitized.includes('.')) {
    num = parseFloat(sanitized);
  } else {
    const digitsOnly = sanitized.replace(/\D/g, '');
    num = digitsOnly ? parseFloat(digitsOnly) : 0;
  }

  if (isNaN(num) || Math.abs(num) < 1e-6) return '';
  return num.toLocaleString('pt-BR', { minimumFractionDigits: 4, maximumFractionDigits: 4 });
}

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


