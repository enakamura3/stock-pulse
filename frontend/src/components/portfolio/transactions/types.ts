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
  background: '#1E293B',
  color: '#FFFFFF',
  fontSize: '0.82rem',
  outline: 'none',
  cursor: 'pointer',
};

export const OPTION_STYLE: React.CSSProperties = { background: '#1c1f24' };

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
      ? { text: 'APLICAÇÃO', color: '#2196F3', bg: 'rgba(33,150,243,0.08)' }
      : { text: 'RESGATE', color: '#FF9800', bg: 'rgba(255,152,0,0.08)' };
  }
  switch (tx.type) {
    case 'BUY':          return { text: 'COMPRA',      color: '#00e676', bg: 'rgba(0,230,118,0.08)' };
    case 'SELL':         return { text: 'VENDA',       color: '#ff3d00', bg: 'rgba(255,61,0,0.08)' };
    case 'BONUS':        return { text: 'BÔNUS',       color: '#00e676', bg: 'rgba(0,230,118,0.08)' };
    case 'SPLIT':        return { text: 'SPLIT',       color: '#00f2fe', bg: 'rgba(0,242,254,0.08)' };
    case 'REVERSE_SPLIT':return { text: 'AGRUPAMENTO', color: '#e040fb', bg: 'rgba(156,39,176,0.08)' };
    default:             return { text: tx.type,       color: '#aaa',    bg: 'rgba(255,255,255,0.05)' };
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
          gradient: 'linear-gradient(135deg, rgba(33,150,243,0.15) 0%, rgba(33,150,243,0.03) 100%)',
          borderColor: 'rgba(33,150,243,0.25)',
        }
      : {
          emoji: categoryEmoji,
          gradient: 'linear-gradient(135deg, rgba(255,152,0,0.15) 0%, rgba(255,152,0,0.03) 100%)',
          borderColor: 'rgba(255,152,0,0.25)',
        };
  }

  switch (tx.type) {
    case 'BUY':
      return {
        emoji: '🛒',
        gradient: 'linear-gradient(135deg, rgba(0,230,118,0.15) 0%, rgba(0,230,118,0.03) 100%)',
        borderColor: 'rgba(0,230,118,0.25)',
      };
    case 'SELL':
      return {
        emoji: '💰',
        gradient: 'linear-gradient(135deg, rgba(255,61,0,0.15) 0%, rgba(255,61,0,0.03) 100%)',
        borderColor: 'rgba(255,61,0,0.25)',
      };
    case 'BONUS':
      return {
        emoji: '🎁',
        gradient: 'linear-gradient(135deg, rgba(0,230,118,0.15) 0%, rgba(0,230,118,0.03) 100%)',
        borderColor: 'rgba(0,230,118,0.25)',
      };
    case 'SPLIT':
      return {
        emoji: '⚡',
        gradient: 'linear-gradient(135deg, rgba(0,242,254,0.15) 0%, rgba(0,242,254,0.03) 100%)',
        borderColor: 'rgba(0,242,254,0.25)',
      };
    case 'REVERSE_SPLIT':
      return {
        emoji: '🔄',
        gradient: 'linear-gradient(135deg, rgba(156,39,176,0.15) 0%, rgba(156,39,176,0.03) 100%)',
        borderColor: 'rgba(156,39,176,0.25)',
      };
    default:
      return {
        emoji: '📄',
        gradient: 'linear-gradient(135deg, rgba(255,255,255,0.08) 0%, rgba(255,255,255,0.02) 100%)',
        borderColor: 'rgba(255,255,255,0.15)',
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
    return { id: 'RF', name: 'Renda Fixa & Tesouro', emoji: '💵', color: '#2196F3' };
  }
  const type = (tx.asset_type || '').toUpperCase();
  if (type.includes('STOCK') || type === 'EQUITY' || type === 'EQUITY_BR' || type === 'EQUITY_US') {
    return { id: 'STOCK', name: 'Ações', emoji: '📈', color: '#00e676' };
  }
  if (type.includes('FII') || type.includes('REAL_ESTATE')) {
    return { id: 'FII', name: 'FIIs', emoji: '🏢', color: '#00f2fe' };
  }
  if (type.includes('ETF')) {
    return { id: 'ETF', name: 'ETFs', emoji: '🌐', color: '#e040fb' };
  }
  if (type.includes('CRYPTO')) {
    return { id: 'CRYPTO', name: 'Cripto', emoji: '₿', color: '#ff9800' };
  }
  if (type.includes('BDR')) {
    return { id: 'BDR', name: 'BDRs', emoji: '📦', color: '#ffc107' };
  }
  return { id: 'OTHER', name: 'Outros', emoji: '🎯', color: '#9e9e9e' };
}
