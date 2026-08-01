import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { PortfolioProvider, usePortfolio } from './PortfolioContext';
import { AuthProvider } from './AuthContext';

// Mock do useRouter do Next.js
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    prefetch: vi.fn(),
  })
}));

// Mock global fetch
global.fetch = vi.fn();

// Consumer test component
const TestConsumer = () => {
  const { portfolios, isLoadingPortfolios, activePortfolioId } = usePortfolio();
  return (
    <div>
      <div data-testid="loading">{isLoadingPortfolios ? 'Loading' : 'Ready'}</div>
      <div data-testid="portfolio-count">{portfolios.length}</div>
      <div data-testid="active-id">{activePortfolioId}</div>
    </div>
  );
};

describe('PortfolioContext', () => {
  it('throws an error if usePortfolio is used outside PortfolioProvider', () => {
    // Suppress console.error during expected throw test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    
    expect(() => render(<TestConsumer />)).toThrow(
      'usePortfolio deve ser usado dentro de um PortfolioProvider'
    );

    consoleSpy.mockRestore();
  });

  it('provides default state when wrapped in PortfolioProvider and AuthProvider', () => {
    (global.fetch as any).mockResolvedValue({
      ok: true,
      json: async () => [],
    });

    render(
      <AuthProvider>
        <PortfolioProvider>
          <TestConsumer />
        </PortfolioProvider>
      </AuthProvider>
    );

    expect(screen.getByTestId('loading')).toBeInTheDocument();
  });
});
