import { describe, it, expect } from 'vitest';
import {
  formatPercentage,
  formatMoney,
  getAssetCategory,
  formatQuantity,
  calculateDailyFixedIncomeRate,
  calculateEstimatedDailyGain,
  DEFAULT_ANNUAL_CDI,
  DEFAULT_ANNUAL_SELIC,
} from '../helpers';

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
      const defaultBrl = formatMoney(50, '');
      expect(defaultBrl).toContain('50,00');
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

  describe('calculateDailyFixedIncomeRate and calculateEstimatedDailyGain', () => {
    it('calculates daily rate for PRE fixed income correctly', () => {
      const dailyRate = calculateDailyFixedIncomeRate('PREFIXADO', 12.0);
      expect(dailyRate).toBeGreaterThan(0);
      expect(dailyRate).toBeCloseTo(0.045, 2);
    });

    it('calculates daily rate for CDI / POS fixed income correctly', () => {
      const dailyRate = calculateDailyFixedIncomeRate('CDI', 110.0, 10.40);
      expect(dailyRate).toBeGreaterThan(0);
      expect(dailyRate).toBeCloseTo(0.043, 2);
    });

    it('calculates daily rate for SELIC treasury correctly', () => {
      const dailyRate = calculateDailyFixedIncomeRate('SELIC', 100.0, 10.50);
      expect(dailyRate).toBeGreaterThan(0);
      expect(dailyRate).toBeCloseTo(0.0397, 2);
    });

    it('calculates daily rate for IPCA / HIBRIDO correctly', () => {
      const dailyRate = calculateDailyFixedIncomeRate('IPCA', 6.0);
      expect(dailyRate).toBeGreaterThan(0);
      expect(dailyRate).toBeCloseTo(0.023, 2);
    });

    it('handles default indexer and negative effective rates', () => {
      const defRate = calculateDailyFixedIncomeRate('OTHER', 10.0);
      expect(defRate).toBeGreaterThan(0);

      const negRate = calculateDailyFixedIncomeRate('PRE', -150.0);
      expect(negRate).toBe(0);
    });

    it('calculates estimated daily gain and handles zero values', () => {
      const gain = calculateEstimatedDailyGain(10000, 0.045);
      expect(gain).toBeCloseTo(4.50, 2);

      expect(calculateEstimatedDailyGain(0, 0.045)).toBe(0);
      expect(calculateEstimatedDailyGain(10000, 0)).toBe(0);
    });
  });
});
