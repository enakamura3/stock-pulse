export type MarketStatusType = 'OPEN' | 'CLOSED' | 'PRE_MARKET' | 'AFTER_MARKET';

export interface MarketStatus {
  status: MarketStatusType;
  label: string;
  color: string;
  badgeBg: string;
  description: string;
}

export function getMarketStatus(date: Date = new Date()): MarketStatus {
  // Converte para horário de Brasília (UTC-3)
  const brtString = date.toLocaleString('en-US', { timeZone: 'America/Sao_Paulo' });
  const brtDate = new Date(brtString);
  const day = brtDate.getDay(); // 0 = Domingo, 6 = Sábado
  const hours = brtDate.getHours();
  const minutes = brtDate.getMinutes();
  const totalMinutes = hours * 60 + minutes;

  // Finais de Semana: Mercado Fechado
  if (day === 0 || day === 6) {
    return {
      status: 'CLOSED',
      label: 'Mercado Fechado',
      color: 'var(--color-danger)',
      badgeBg: 'var(--color-danger-bg)',
      description: 'Fim de semana (B3 fechada)',
    };
  }

  // Pré-abertura B3: 09:00 até 09:59 (540 a 599 min)
  if (totalMinutes >= 540 && totalMinutes < 600) {
    return {
      status: 'PRE_MARKET',
      label: 'Pré-Mercado',
      color: 'var(--color-warning)',
      badgeBg: 'var(--color-warning-bg)',
      description: 'Leilão de pré-abertura (09:00 - 10:00)',
    };
  }

  // Pregão Regular B3: 10:00 até 17:55 (600 a 1075 min)
  if (totalMinutes >= 600 && totalMinutes <= 1075) {
    return {
      status: 'OPEN',
      label: 'Mercado Aberto',
      color: 'var(--color-success)',
      badgeBg: 'var(--color-success-bg)',
      description: 'Pregão regular B3 (10:00 - 17:55)',
    };
  }

  // After-market B3: 17:56 até 18:30 (1076 a 1110 min)
  if (totalMinutes > 1075 && totalMinutes <= 1110) {
    return {
      status: 'AFTER_MARKET',
      label: 'After-Market',
      color: 'var(--accent-color)',
      badgeBg: 'var(--accent-bg)',
      description: 'Negociação pós-fechamento (17:56 - 18:30)',
    };
  }

  // Fora de horário: Mercado Fechado
  return {
    status: 'CLOSED',
    label: 'Mercado Fechado',
    color: 'var(--color-danger)',
    badgeBg: 'var(--color-danger-bg)',
    description: 'Fora do horário de negociação',
  };
}
