import { fmt, fmtPct, getTreasuryTypeLabel, getTreasuryTypeBadgeColor } from '../types';

describe('treasury/types utilities', () => {
  describe('fmt', () => {
    it('formats currency values in BRL and custom currencies', () => {
      const formattedBRL = fmt(1234.56);
      expect(formattedBRL).toMatch(/1\.234,56/);

      const formattedUSD = fmt(1234.56, 'USD');
      expect(formattedUSD).toMatch(/1\.234,56/);
    });
  });

  describe('fmtPct', () => {
    it('formats percentage with + sign for positive values', () => {
      expect(fmtPct(5.25)).toBe('+5.25%');
    });

    it('formats percentage without + sign for zero or negative values', () => {
      expect(fmtPct(0)).toBe('0.00%');
      expect(fmtPct(-3.14)).toBe('-3.14%');
    });
  });

  describe('getTreasuryTypeLabel', () => {
    it('returns formatted label for known types', () => {
      expect(getTreasuryTypeLabel('SELIC')).toBe('📈 Tesouro Selic');
      expect(getTreasuryTypeLabel('PREFIXADO')).toBe('🔒 Prefixado');
      expect(getTreasuryTypeLabel('IPCA+')).toBe('🏷️ IPCA+');
    });

    it('returns raw string for unknown types', () => {
      expect(getTreasuryTypeLabel('IGPM')).toBe('IGPM');
    });
  });

  describe('getTreasuryTypeBadgeColor', () => {
    it('returns badge colors for known types', () => {
      expect(getTreasuryTypeBadgeColor('SELIC')).toBe('#4caf50');
      expect(getTreasuryTypeBadgeColor('PREFIXADO')).toBe('#2196f3');
      expect(getTreasuryTypeBadgeColor('IPCA+')).toBe('#ff9800');
    });

    it('returns fallback color for unknown types', () => {
      expect(getTreasuryTypeBadgeColor('OTHER')).toBe('#9e9e9e');
    });
  });
});
