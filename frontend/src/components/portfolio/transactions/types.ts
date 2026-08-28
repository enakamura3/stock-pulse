import React from 'react';
import { UnifiedTransaction } from '../types';

export interface TransactionWithBalance extends UnifiedTransaction {
  resulting_quantity: number;
  resulting_invested: number;
}

export const MONTHS = [
  { value: '01', label: 'Janeiro' },
  { value: '02', label: 'Fevereiro' },
  { value: '03', label: 'Março' },
  { value: '04', label: 'Abril' },
  { value: '05', label: 'Maio' },
  { value: '06', label: 'Junho' },
  { value: '07', label: 'Julho' },
  { value: '08', label: 'Agosto' },
  { value: '09', label: 'Setembro' },
  { value: '10', label: 'Outubro' },
  { value: '11', label: 'Novembro' },
  { value: '12', label: 'Dezembro' },
];

export const TX_TYPES = [
  { value: 'Todos', label: 'Tipo: Todos' },
  { value: 'BUY', label: 'Compra' },
  { value: 'SELL', label: 'Venda' },
  { value: 'BONUS', label: 'Bônus' },
  { value: 'SPLIT', label: 'Split' },
  { value: 'REVERSE_SPLIT', label: 'Agrupamento' },
  { value: 'SUBSCRIPTION', label: 'Aplicação/Resgate' },
];

export const PAGE_SIZE = 20;

export const SELECT_STYLE: React.CSSProperties = {
  padding: '0.4rem 0.75rem',
  borderRadius: '6px',
  border: '1px solid var(--panel-border)',
  background: 'var(--option-bg)',
  color: 'var(--option-color)',
  fontSize: '0.82rem',
  outline: 'none',
  cursor: 'pointer',
};

export const OPTION_STYLE: React.CSSProperties = { background: 'var(--option-bg)', color: 'var(--option-color)' };

export function formatDateStr(dateStr: string | null | undefined): string {
  if (!dateStr) return 'N/A';
  return dateStr.substring(0, 10).replace(/-/g, '/');
}

export function formatDateGroupLabel(dateStr: string): string {
  const [year, month, day] = dateStr.split('-');
  const monthName = MONTHS.find((m) => m.value === month)?.label ?? month;
  return `${parseInt(day)} de ${monthName} de ${year}`;
}

export function getBadge(tx: UnifiedTransaction): { text: string; color: string; bg: string } {
  const isRF = tx.module === 'RF';
  if (isRF) {
    return tx.type === 'SUBSCRIPTION'
      ? { text: 'APLICAÇÃO', color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
      : { text: 'RESGATE', color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' };
  }
  switch (tx.type) {
    case 'BUY':          return { text: 'COMPRA',      color: 'var(--color-success)', bg: 'var(--color-success-bg)' };
    case 'SELL':         return { text: 'VENDA',       color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' };
    case 'BONUS':        return { text: 'BÔNUS',       color: 'var(--color-success)', bg: 'var(--color-success-bg)' };
    case 'SPLIT':        return { text: 'SPLIT',       color: 'var(--accent-color)', bg: 'var(--accent-bg)' };
    case 'REVERSE_SPLIT':return { text: 'AGRUPAMENTO', color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' };
    default:             return { text: tx.type,       color: 'var(--text-secondary)', bg: 'var(--panel-bg)' };
  }
}

export function getTransactionCircleDetails(tx: UnifiedTransaction): {
  emoji: string;
  gradient: string;
  borderColor: string;
} {
  const isRF = tx.module === 'RF';
  const categoryEmoji = isRF ? (tx.asset_type === 'TESOURO' ? '🏛️' : '🏦') : '📈';

  if (isRF) {
    return tx.type === 'SUBSCRIPTION'
      ? {
          emoji: categoryEmoji,
          gradient: 'var(--color-info-bg)',
          borderColor: 'rgba(var(--info-rgb), 0.3)',
        }
      : {
          emoji: categoryEmoji,
          gradient: 'var(--color-warning-bg)',
          borderColor: 'rgba(var(--warning-rgb), 0.3)',
        };
  }

  switch (tx.type) {
    case 'BUY':
      return {
        emoji: '🛒',
        gradient: 'var(--color-success-bg)',
        borderColor: 'rgba(var(--success-rgb), 0.3)',
      };
    case 'SELL':
      return {
        emoji: '💰',
        gradient: 'var(--color-danger-bg)',
        borderColor: 'rgba(var(--danger-rgb), 0.3)',
      };
    case 'BONUS':
      return {
        emoji: '🎁',
        gradient: 'var(--color-success-bg)',
        borderColor: 'rgba(var(--success-rgb), 0.3)',
      };
    case 'SPLIT':
      return {
        emoji: '⚡',
        gradient: 'var(--accent-bg)',
        borderColor: 'rgba(var(--accent-rgb), 0.3)',
      };
    case 'REVERSE_SPLIT':
      return {
        emoji: '🔄',
        gradient: 'var(--color-danger-bg)',
        borderColor: 'rgba(var(--danger-rgb), 0.3)',
      };
    default:
      return {
        emoji: '📄',
        gradient: 'var(--input-bg)',
        borderColor: 'var(--panel-border)',
      };
  }
}

export function getMacroAssetCategory(tx: UnifiedTransaction): {
  id: string;
  name: string;
  emoji: string;
  color: string;
} {
  if (tx.module === 'RF') {
    return { id: 'RF', name: 'Renda Fixa & Tesouro', emoji: '💵', color: 'var(--color-warning)' };
  }
  const type = (tx.asset_type || '').toUpperCase();
  if (type.includes('STOCK') || type === 'EQUITY' || type === 'EQUITY_BR' || type === 'EQUITY_US') {
    return { id: 'STOCK', name: 'Ações', emoji: '📈', color: 'var(--accent-color)' };
  }
  if (type.includes('FII') || type.includes('REAL_ESTATE')) {
    return { id: 'FII', name: 'FIIs', emoji: '🏢', color: 'var(--color-info)' };
  }
  if (type.includes('ETF')) {
    return { id: 'ETF', name: 'ETFs', emoji: '🌐', color: 'var(--color-info)' };
  }
  if (type.includes('CRYPTO')) {
    return { id: 'CRYPTO', name: 'Cripto', emoji: '₿', color: 'var(--color-warning)' };
  }
  if (type.includes('BDR')) {
    return { id: 'BDR', name: 'BDRs', emoji: '📦', color: 'var(--color-warning)' };
  }
  return { id: 'OTHER', name: 'Outros', emoji: '🎯', color: 'var(--text-muted)' };
}
