import { describe, it, expect } from 'vitest';
import { formatPercentage, formatMoney, getAssetCategory, formatQuantity } from '../helpers';

describe('Portfolio Helpers', () => {
  describe('formatPercentage', () => {
    it('formats positive percentages with + sign', () => {
      expect(formatPercentage(5.25)).toBe('+5.25%');
      expect(formatPercentage(0.01)).toBe('+0.01%');
      expect(formatPercentage(100)).toBe('+100.00%');
    });

    it('formats negative percentages with - sign and no +', () => {
      expect(formatPercentage(-3.45)).toBe('-3.45%');
      expect(formatPercentage(-0.01)).toBe('-0.01%');
    });

    it('formats exact zero and near-zero values without + sign (0.00%)', () => {
      expect(formatPercentage(0)).toBe('0.00%');
      expect(formatPercentage(0.0000001)).toBe('0.00%');
      expect(formatPercentage(-0.0000001)).toBe('0.00%');
      expect(formatPercentage(0.001)).toBe('0.00%');
      expect(formatPercentage(-0.001)).toBe('0.00%');
    });
  });

  describe('formatMoney', () => {
    it('formats currency correctly in pt-BR', () => {
      const brl = formatMoney(1234.56, 'BRL');
      expect(brl).toContain('1.234,56');
      const usd = formatMoney(100, 'USD');
      expect(usd).toContain('100,00');
    });
  });

  describe('formatQuantity', () => {
    it('formats numbers with max 3 decimal places', () => {
      expect(formatQuantity(10)).toBe('10');
      expect(formatQuantity(10.5)).toBe('10,5');
      expect(formatQuantity(10.5555)).toBe('10,556');
    });
  });

  describe('getAssetCategory', () => {
    it('maps asset dbTypes to user friendly categories', () => {
      expect(getAssetCategory('STOCK_BR')).toBe('Ações (B3)');
      expect(getAssetCategory('FII')).toBe('FIIs');
      expect(getAssetCategory('FIAGRO')).toBe('FIAGROs');
      expect(getAssetCategory('ETF_BR')).toBe('ETFs Nacionais');
      expect(getAssetCategory('BDR')).toBe('BDRs');
      expect(getAssetCategory('STOCK_US')).toBe('Ações EUA');
      expect(getAssetCategory('ETF_US')).toBe('ETF Internacional');
      expect(getAssetCategory('CRYPTO')).toBe('Cripto');
      expect(getAssetCategory('CDB')).toBe('Renda Fixa');
      expect(getAssetCategory('TESOURO')).toBe('Renda Fixa');
      expect(getAssetCategory('UNKNOWN')).toBe('Desconhecido');
    });
  });
});
