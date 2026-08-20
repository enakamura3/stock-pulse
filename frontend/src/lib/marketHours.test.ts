import { describe, it, expect } from 'vitest';
import { getMarketStatus } from './marketHours';

describe('marketHours Utility', () => {
  it('returns CLOSED for weekend dates', () => {
    // Saturday: 2026-08-22
    const saturday = new Date('2026-08-22T15:00:00Z');
    expect(getMarketStatus(saturday).status).toBe('CLOSED');

    // Sunday: 2026-08-23
    const sunday = new Date('2026-08-23T15:00:00Z');
    expect(getMarketStatus(sunday).status).toBe('CLOSED');
  });

  it('returns PRE_MARKET between 09:00 and 09:59 BRT on weekdays', () => {
    // 09:30 BRT = 12:30 UTC
    const preMarket = new Date('2026-08-19T12:30:00Z');
    const result = getMarketStatus(preMarket);
    expect(result.status).toBe('PRE_MARKET');
    expect(result.label).toBe('Pré-Mercado');
  });

  it('returns OPEN between 10:00 and 17:55 BRT on weekdays', () => {
    // 14:00 BRT = 17:00 UTC
    const openMarket = new Date('2026-08-19T17:00:00Z');
    const result = getMarketStatus(openMarket);
    expect(result.status).toBe('OPEN');
    expect(result.label).toBe('Mercado Aberto');
  });

  it('returns AFTER_MARKET between 17:56 and 18:30 BRT on weekdays', () => {
    // 18:00 BRT = 21:00 UTC
    const afterMarket = new Date('2026-08-19T21:00:00Z');
    const result = getMarketStatus(afterMarket);
    expect(result.status).toBe('AFTER_MARKET');
    expect(result.label).toBe('After-Market');
  });

  it('returns CLOSED before 09:00 and after 18:30 BRT on weekdays', () => {
    // 08:00 BRT = 11:00 UTC
    const morningClosed = new Date('2026-08-19T11:00:00Z');
    expect(getMarketStatus(morningClosed).status).toBe('CLOSED');

    // 20:00 BRT = 23:00 UTC
    const eveningClosed = new Date('2026-08-19T23:00:00Z');
    expect(getMarketStatus(eveningClosed).status).toBe('CLOSED');
  });

  it('defaults to current time when no date is provided', () => {
    const current = getMarketStatus();
    expect(current).toBeDefined();
    expect(['OPEN', 'CLOSED', 'PRE_MARKET', 'AFTER_MARKET']).toContain(current.status);
  });
});
