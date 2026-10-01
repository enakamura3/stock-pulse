import {
  formatDateStr,
  formatDateGroupLabel,
  getBadge,
  getTransactionCircleDetails,
  getMacroAssetCategory,
  MONTHS,
  TX_TYPES,
  PAGE_SIZE,
  SELECT_STYLE,
  OPTION_STYLE,
} from '../types';
import { UnifiedTransaction } from '../../types';

describe('transactions/types utilities', () => {
  it('exports expected constants', () => {
    expect(MONTHS).toHaveLength(12);
    expect(TX_TYPES.length).toBeGreaterThan(0);
    expect(PAGE_SIZE).toBe(20);
    expect(SELECT_STYLE).toBeDefined();
    expect(OPTION_STYLE).toBeDefined();
  });

  describe('formatDateStr', () => {
    it('returns N/A for null or undefined or empty', () => {
      expect(formatDateStr(null)).toBe('N/A');
      expect(formatDateStr(undefined)).toBe('N/A');
      expect(formatDateStr('')).toBe('N/A');
    });

    it('formats ISO date to YYYY/MM/DD', () => {
      expect(formatDateStr('2024-03-15T00:00:00Z')).toBe('2024/03/15');
    });
  });

  describe('formatDateGroupLabel', () => {
    it('formats date string to Brazilian full date label', () => {
      expect(formatDateGroupLabel('2024-05-10')).toBe('10 de Maio de 2024');
    });

    it('falls back to month number if month value not found in MONTHS', () => {
      expect(formatDateGroupLabel('2024-99-05')).toBe('5 de 99 de 2024');
    });
  });

  describe('getBadge', () => {
    it('returns RF SUBSCRIPTION and REDEMPTION badges', () => {
      const rfSub: UnifiedTransaction = {
        id: '1',
        module: 'RF',
        type: 'SUBSCRIPTION',
        ticker: 'SELIC2029',
        date: '2024-01-01',
        quantity: 1,
        price: 100,
        total_value: 100,
      };
      expect(getBadge(rfSub).text).toBe('APLICAÇÃO');

      const rfRed: UnifiedTransaction = { ...rfSub, type: 'REDEMPTION' };
      expect(getBadge(rfRed).text).toBe('RESGATE');
    });

    it('returns non-RF transaction badges', () => {
      const baseTx: UnifiedTransaction = {
        id: '1',
        module: 'RV',
        type: 'BUY',
        ticker: 'PETR4',
        date: '2024-01-01',
        quantity: 10,
        price: 30,
        total_value: 300,
      };

      expect(getBadge({ ...baseTx, type: 'BUY' }).text).toBe('COMPRA');
      expect(getBadge({ ...baseTx, type: 'SELL' }).text).toBe('VENDA');
      expect(getBadge({ ...baseTx, type: 'BONUS' }).text).toBe('BÔNUS');
      expect(getBadge({ ...baseTx, type: 'SPLIT' }).text).toBe('SPLIT');
      expect(getBadge({ ...baseTx, type: 'REVERSE_SPLIT' }).text).toBe('AGRUPAMENTO');
      expect(getBadge({ ...baseTx, type: 'UNKNOWN_TYPE' as any }).text).toBe('UNKNOWN_TYPE');
    });
  });

  describe('getTransactionCircleDetails', () => {
    it('returns details for RF transactions', () => {
      const rfTesouro: UnifiedTransaction = {
        id: '1',
        module: 'RF',
        asset_type: 'TESOURO',
        type: 'SUBSCRIPTION',
        ticker: 'SELIC2029',
        date: '2024-01-01',
        quantity: 1,
        price: 100,
        total_value: 100,
      };
      const subDetails = getTransactionCircleDetails(rfTesouro);
      expect(subDetails.emoji).toBe('🏛️');

      const rfCdb: UnifiedTransaction = {
        ...rfTesouro,
        asset_type: 'CDB',
        type: 'REDEMPTION',
      };
      const redDetails = getTransactionCircleDetails(rfCdb);
      expect(redDetails.emoji).toBe('🏦');
    });

    it('returns details for non-RF transactions', () => {
      const baseTx: UnifiedTransaction = {
        id: '1',
        module: 'RV',
        type: 'BUY',
        ticker: 'PETR4',
        date: '2024-01-01',
        quantity: 10,
        price: 30,
        total_value: 300,
      };

      expect(getTransactionCircleDetails({ ...baseTx, type: 'BUY' }).emoji).toBe('🛒');
      expect(getTransactionCircleDetails({ ...baseTx, type: 'SELL' }).emoji).toBe('💰');
      expect(getTransactionCircleDetails({ ...baseTx, type: 'BONUS' }).emoji).toBe('🎁');
      expect(getTransactionCircleDetails({ ...baseTx, type: 'SPLIT' }).emoji).toBe('⚡');
      expect(getTransactionCircleDetails({ ...baseTx, type: 'REVERSE_SPLIT' }).emoji).toBe('🔄');
      expect(getTransactionCircleDetails({ ...baseTx, type: 'UNKNOWN' as any }).emoji).toBe('📄');
    });
  });

  describe('getMacroAssetCategory', () => {
    const baseTx: UnifiedTransaction = {
      id: '1',
      module: 'RV',
      type: 'BUY',
      ticker: 'TEST',
      date: '2024-01-01',
      quantity: 1,
      price: 10,
      total_value: 10,
    };

    it('categorizes RF module', () => {
      expect(getMacroAssetCategory({ ...baseTx, module: 'RF' }).id).toBe('RF');
    });

    it('categorizes stocks / equities', () => {
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'STOCK' }).id).toBe('STOCK');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'EQUITY' }).id).toBe('STOCK');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'EQUITY_BR' }).id).toBe('STOCK');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'EQUITY_US' }).id).toBe('STOCK');
    });

    it('categorizes FIIs and real estate', () => {
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'FII' }).id).toBe('FII');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'REAL_ESTATE' }).id).toBe('FII');
    });

    it('categorizes ETFs, crypto, BDRs, and other', () => {
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'ETF' }).id).toBe('ETF');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'CRYPTO' }).id).toBe('CRYPTO');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'BDR' }).id).toBe('BDR');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: 'COMMODITY' }).id).toBe('OTHER');
      expect(getMacroAssetCategory({ ...baseTx, asset_type: undefined }).id).toBe('OTHER');
    });
  });
});
