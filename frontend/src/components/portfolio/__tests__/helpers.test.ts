import { describe, it, expect } from 'vitest';
import {
  formatPercentage,
  formatMoney,
  getAssetCategory,
  formatQuantity,
  calculateDailyFixedIncomeRate,
  calculateEstimatedDailyGain,
  exportDailyReportCSV,
  DEFAULT_ANNUAL_CDI,
  DEFAULT_ANNUAL_SELIC,
  determineAssetTypeLocal,
  ASSET_TYPE_OPTIONS,
  formatCurrencyInput,
  parseCurrency,
  parsePastedCurrency,
  formatExchangeRateInput,
  parsePastedExchangeRate,
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

  describe('exportDailyReportCSV', () => {
    it('generates CSV content correctly for variable income, fixed income and treasury', () => {
      const mockPositions = [
        {
          asset_id: 'pos1',
          ticker: 'PETR4',
          name: 'Petrobras',
          type: 'STOCK_BR',
          currency: 'BRL',
          quantity: 100,
          average_price: 30,
          total_cost: 3000,
          current_price: 35,
          current_value: 3500,
          daily_change: 1.5,
          daily_change_percent: 4.48,
          previous_close: 33.5,
        },
        {
          asset_id: 'pos2',
          ticker: 'AAPL',
          name: 'Apple Inc',
          type: 'STOCK_US',
          currency: 'USD',
          quantity: 10,
          average_price: 150,
          total_cost: 7500,
          current_price: 180,
          current_value: 9900,
          daily_change: 2.0,
          daily_change_percent: 1.12,
          fx_rate_to_brl: 5.5,
        },
        {
          asset_id: 'pos3',
          ticker: '',
          name: '',
          type: 'STOCK_US',
          currency: 'USD',
          quantity: undefined as any,
          average_price: undefined as any,
          total_cost: 0,
          current_price: undefined as any,
          current_value: 0,
          daily_change: undefined as any,
          daily_change_percent: undefined as any,
          previous_close: null as any,
          fx_rate_to_brl: undefined as any,
        },
      ];

      const mockFI = [
        {
          asset: {
            id: 'fi1',
            portfolio_id: 'p1',
            institution: 'Banco Inter',
            type: 'CDB',
            debt_type: 'POS',
            indexer: 'CDI',
            rate: 110,
            maturity_date: '2027-12-31T00:00:00Z',
          },
          total_invested: 10000,
          gross_value: 11000,
          net_value: 10850,
          net_return_percent: 8.5,
        },
      ];

      const mockTreasury = [
        {
          transaction_id: 'tx1',
          asset_id: 't1',
          ticker: 'Tesouro Selic 2029',
          treasury_type: 'SELIC',
          maturity_date: '2029-03-01T00:00:00Z',
          has_coupons: false,
          start_date: '2024-01-01',
          quantity: 1,
          unit_price: 14000,
          contracted_rate: 10.5,
          total_invested: 14000,
          gross_value: 15000,
          net_value: 14850,
          is_matured: false,
          days_to_maturity: 1000,
          taxes_calculated: 150,
          b3_fee: 10,
          ir_tax: 140,
          iof_tax: 0,
        },
      ];

      // Mock browser download
      const originalCreateObjectURL = window.URL.createObjectURL;
      const originalRevokeObjectURL = window.URL.revokeObjectURL;
      window.URL.createObjectURL = vi.fn().mockReturnValue('blob:mock-url');
      window.URL.revokeObjectURL = vi.fn();

      const csv = exportDailyReportCSV(mockPositions, mockFI, mockTreasury, 'BRL');

      expect(csv).toContain('--- RENDA VARIÁVEL ---');
      expect(csv).toContain('PETR4');
      expect(csv).toContain('AAPL');
      expect(csv).toContain('--- RENDA FIXA PRIVADA ---');
      expect(csv).toContain('Banco Inter');
      expect(csv).toContain('--- TESOURO DIRETO ---');
      expect(csv).toContain('Tesouro Selic 2029');

      window.URL.createObjectURL = originalCreateObjectURL;
      window.URL.revokeObjectURL = originalRevokeObjectURL;
    });

    it('handles empty maturity dates and default empty collections in exportDailyReportCSV', () => {
      const fiWithoutMaturity = [
        {
          asset: {
            id: 'fi_pre',
            portfolio_id: 'p1',
            institution: '',
            type: 'CDB',
            debt_type: 'PRE',
            indexer: '',
            rate: 12,
            maturity_date: '',
          },
          total_invested: 5000,
          gross_value: 5500,
          net_value: undefined as any,
          net_return_percent: undefined as any,
        },
      ];

      const treasuryWithoutMaturity = [
        {
          transaction_id: 'tx_nomat',
          asset_id: 't_nomat',
          ticker: '',
          treasury_type: 'PREFIXADO',
          maturity_date: '',
          has_coupons: false,
          start_date: '2024-01-01',
          quantity: 1,
          unit_price: 1000,
          contracted_rate: undefined as any,
          total_invested: 0,
          gross_value: 1100,
          net_value: undefined as any,
          is_matured: false,
          days_to_maturity: 500,
          taxes_calculated: 20,
          b3_fee: 0,
          ir_tax: 20,
          iof_tax: 0,
        },
      ];

      const csv = exportDailyReportCSV([], fiWithoutMaturity, treasuryWithoutMaturity);
      expect(csv).toContain('CDB');
      expect(csv).toContain('PREFIXADO');

      const csvDefault = exportDailyReportCSV([]);
      expect(csvDefault).toContain('--- RENDA VARIÁVEL ---');
      expect(csvDefault).not.toContain('--- RENDA FIXA PRIVADA ---');
    });
  });

  describe('determineAssetTypeLocal', () => {
    it('infers CRYPTO correctly', () => {
      expect(determineAssetTypeLocal('BTC-USD', 'Bitcoin', 'USD')).toBe('CRYPTO');
      expect(determineAssetTypeLocal('SOL', 'Solana', 'CRYPTO')).toBe('CRYPTO');
    });

    it('infers US stocks and ETFs correctly', () => {
      expect(determineAssetTypeLocal('AAPL', 'Apple Inc.', 'USD')).toBe('STOCK_US');
      expect(determineAssetTypeLocal('SPY', 'SPDR S&P 500 ETF Trust', 'USD')).toBe('ETF_US');
      expect(determineAssetTypeLocal('QQQ', 'Invesco QQQ Trust', 'USD')).toBe('ETF_US');
    });

    it('infers BDRs correctly', () => {
      expect(determineAssetTypeLocal('AAPL34.SA', 'Apple Inc BDR', 'BRL')).toBe('BDR');
      expect(determineAssetTypeLocal('MSFT35.SA', 'Microsoft BDR', 'BRL')).toBe('BDR');
      expect(determineAssetTypeLocal('GOGL39.SA', 'Alphabet BDR', 'BRL')).toBe('BDR');
    });

    it('infers Brazilian 11.SA assets properly (Fiagro, ETF, FII, Stock)', () => {
      expect(determineAssetTypeLocal('VGIA11.SA', 'Valora FIAGRO', 'BRL')).toBe('FIAGRO');
      expect(determineAssetTypeLocal('BOVA11.SA', 'iShares Ibovespa ETF', 'BRL')).toBe('ETF_BR');
      expect(determineAssetTypeLocal('IVVB11.SA', 'iShares S&P 500 Fundo de Índice', 'BRL')).toBe('ETF_BR');
      expect(determineAssetTypeLocal('HGLG11.SA', 'CSHG Logística Fundo Imobiliário', 'BRL')).toBe('FII');
      expect(determineAssetTypeLocal('MXRF11.SA', 'Maxi Renda FII', 'BRL')).toBe('FII');
      expect(determineAssetTypeLocal('TAEE11.SA', 'Transmissora Aliança de Energia Elétrica S.A.', 'BRL')).toBe('STOCK_BR');
      expect(determineAssetTypeLocal('SANB11.SA', 'Banco Santander Brasil S.A.', 'BRL')).toBe('STOCK_BR');
      expect(determineAssetTypeLocal('UNKNOWN11.SA', '', 'BRL')).toBe('STOCK_BR');
    });

    it('infers standard Brazilian equities correctly', () => {
      expect(determineAssetTypeLocal('PETR4.SA', 'Petrobras PN', 'BRL')).toBe('STOCK_BR');
      expect(determineAssetTypeLocal('VALE3.SA', 'Vale ON', 'BRL')).toBe('STOCK_BR');
    });

    it('contains all 8 market asset type options', () => {
      expect(ASSET_TYPE_OPTIONS).toHaveLength(8);
      const values = ASSET_TYPE_OPTIONS.map((o) => o.value);
      expect(values).toEqual([
        'STOCK_BR',
        'FII',
        'FIAGRO',
        'ETF_BR',
        'BDR',
        'STOCK_US',
        'ETF_US',
        'CRYPTO',
      ]);
    });
  });

  describe('formatCurrencyInput (ATM style mask)', () => {
    it('shifts decimal from right to left as digits are typed', () => {
      expect(formatCurrencyInput('2')).toBe('0,02');
      expect(formatCurrencyInput('0,023')).toBe('0,23');
      expect(formatCurrencyInput('0,235')).toBe('2,35');
      expect(formatCurrencyInput('2,350')).toBe('23,50');
      expect(formatCurrencyInput('2350')).toBe('23,50');
      expect(formatCurrencyInput('235000')).toBe('2.350,00');
    });

    it('handles backspacing correctly', () => {
      // User has 2,35 and presses backspace -> input becomes 2,3 -> digits 23 -> 0,23
      expect(formatCurrencyInput('2,3')).toBe('0,23');
      // User has 0,23 and presses backspace -> 0,2 -> 0,02
      expect(formatCurrencyInput('0,2')).toBe('0,02');
      // User has 0,02 and presses backspace -> 0,0 -> digits 00 -> empty
      expect(formatCurrencyInput('0,0')).toBe('');
    });

    it('returns empty string for zero, empty or non-digit input', () => {
      expect(formatCurrencyInput('')).toBe('');
      expect(formatCurrencyInput('0')).toBe('');
      expect(formatCurrencyInput('0,00')).toBe('');
      expect(formatCurrencyInput('abc')).toBe('');
      expect(formatCurrencyInput(undefined)).toBe('');
      expect(formatCurrencyInput(null)).toBe('');
    });

    it('formats existing numbers when loaded into edit mode', () => {
      expect(formatCurrencyInput(2.35)).toBe('2,35');
      expect(formatCurrencyInput(23.5)).toBe('23,50');
      expect(formatCurrencyInput(1250.75)).toBe('1.250,75');
      expect(formatCurrencyInput(0)).toBe('');
    });
  });

  describe('parseCurrency', () => {
    it('parses formatted Brazilian currency strings correctly', () => {
      expect(parseCurrency('2,35')).toBe(2.35);
      expect(parseCurrency('23,50')).toBe(23.5);
      expect(parseCurrency('1.234,56')).toBe(1234.56);
      expect(parseCurrency('R$ 2.350,00')).toBe(2350);
    });

    it('parses international format and raw numbers', () => {
      expect(parseCurrency('2.35')).toBe(2.35);
      expect(parseCurrency(2.35)).toBe(2.35);
      expect(parseCurrency('1,234.56')).toBe(1234.56);
    });

    it('returns 0 for empty, null, undefined or zero inputs', () => {
      expect(parseCurrency('')).toBe(0);
      expect(parseCurrency(null)).toBe(0);
      expect(parseCurrency(undefined)).toBe(0);
      expect(parseCurrency(0)).toBe(0);
      expect(parseCurrency('invalid')).toBe(0);
    });
  });

  describe('parsePastedCurrency', () => {
    it('handles pasted numbers with comma or dot decimals', () => {
      expect(parsePastedCurrency('2,35')).toBe('2,35');
      expect(parsePastedCurrency('2.35')).toBe('2,35');
      expect(parsePastedCurrency('23.5')).toBe('23,50');
      expect(parsePastedCurrency('1234.56')).toBe('1.234,56');
      expect(parsePastedCurrency('1.234,56')).toBe('1.234,56');
      expect(parsePastedCurrency('1,234.56')).toBe('1.234,56');
    });

    it('handles pasted integers by adding ,00', () => {
      expect(parsePastedCurrency('25')).toBe('25,00');
      expect(parsePastedCurrency('100')).toBe('100,00');
    });

    it('cleans currency symbols from pasted content', () => {
      expect(parsePastedCurrency('R$ 23,50')).toBe('23,50');
      expect(parsePastedCurrency('$ 50.00')).toBe('50,00');
      expect(parsePastedCurrency('R$ 1.250,50')).toBe('1.250,50');
    });

    it('returns empty string for invalid pastes', () => {
      expect(parsePastedCurrency('')).toBe('');
      expect(parsePastedCurrency('abc')).toBe('');
      expect(parsePastedCurrency('0')).toBe('');
    });
  });

  describe('formatExchangeRateInput (ATM style mask, 4 decimal places)', () => {
    it('shifts decimal from right to left as digits are typed (4 places)', () => {
      expect(formatExchangeRateInput('5')).toBe('0,0005');
      expect(formatExchangeRateInput('52')).toBe('0,0052');
      expect(formatExchangeRateInput('525')).toBe('0,0525');
      expect(formatExchangeRateInput('5250')).toBe('0,5250');
      expect(formatExchangeRateInput('52500')).toBe('5,2500');
      expect(formatExchangeRateInput('525000')).toBe('52,5000');
    });

    it('handles backspacing correctly', () => {
      // 5,2500 -> backspace -> 0,5250 -> backspace -> 0,0525
      expect(formatExchangeRateInput('5,250')).toBe('0,5250');
      expect(formatExchangeRateInput('0,525')).toBe('0,0525');
      expect(formatExchangeRateInput('0,000')).toBe('');
    });

    it('returns empty string for zero, empty or non-digit input', () => {
      expect(formatExchangeRateInput('')).toBe('');
      expect(formatExchangeRateInput('0')).toBe('');
      expect(formatExchangeRateInput('0,0000')).toBe('');
      expect(formatExchangeRateInput('abc')).toBe('');
      expect(formatExchangeRateInput(undefined)).toBe('');
      expect(formatExchangeRateInput(null)).toBe('');
    });

    it('formats existing numbers when loaded into edit mode', () => {
      expect(formatExchangeRateInput(5.25)).toBe('5,2500');
      expect(formatExchangeRateInput(5.2534)).toBe('5,2534');
      expect(formatExchangeRateInput(1.0)).toBe('1,0000');
      expect(formatExchangeRateInput(0)).toBe('');
    });
  });

  describe('parsePastedExchangeRate', () => {
    it('handles pasted numbers with comma or dot decimals (4 places)', () => {
      expect(parsePastedExchangeRate('5.25')).toBe('5,2500');
      expect(parsePastedExchangeRate('5,25')).toBe('5,2500');
      expect(parsePastedExchangeRate('5,2500')).toBe('5,2500');
      expect(parsePastedExchangeRate('5.2500')).toBe('5,2500');
    });

    it('handles pasted integers by adding ,0000', () => {
      expect(parsePastedExchangeRate('5')).toBe('5,0000');
      expect(parsePastedExchangeRate('10')).toBe('10,0000');
    });

    it('returns empty string for invalid pastes', () => {
      expect(parsePastedExchangeRate('')).toBe('');
      expect(parsePastedExchangeRate('abc')).toBe('');
      expect(parsePastedExchangeRate('0')).toBe('');
    });
  });
});
