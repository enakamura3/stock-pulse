import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import PortfolioTabs from '../PortfolioTabs';
import { Portfolio } from '../types';

describe('PortfolioTabs Component', () => {
  const setActivePortfolioId = vi.fn();
  const setShowPortfolioModal = vi.fn();
  const handleDeletePortfolio = vi.fn();
  const handleExportPortfolio = vi.fn();
  const handleSetDefaultPortfolio = vi.fn();

  const mockPortfolios: Portfolio[] = [
    {
      id: 'p1',
      user_id: 'u1',
      name: 'Principal',
      description: 'Minha carteira principal',
      base_currency: 'BRL',
      is_default: true,
      created_at: '',
      updated_at: '',
    },
    {
      id: 'p2',
      user_id: 'u1',
      name: 'Secundária',
      description: '',
      base_currency: 'USD',
      is_default: false,
      created_at: '',
      updated_at: '',
    },
  ];

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders portfolio tabs and handles selection and creation modal', () => {
    render(
      <PortfolioTabs
        portfolios={mockPortfolios}
        activePortfolioId="p1"
        setActivePortfolioId={setActivePortfolioId}
        setShowPortfolioModal={setShowPortfolioModal}
        handleDeletePortfolio={handleDeletePortfolio}
        handleExportPortfolio={handleExportPortfolio}
        handleSetDefaultPortfolio={handleSetDefaultPortfolio}
      />
    );

    expect(screen.getByText(/Principal/)).toBeInTheDocument();
    expect(screen.getByText(/Secundária/)).toBeInTheDocument();

    // Select second portfolio
    fireEvent.click(screen.getByText(/Secundária/));
    expect(setActivePortfolioId).toHaveBeenCalledWith('p2');

    // Click create portfolio button
    fireEvent.click(screen.getByText('+ Criar Carteira'));
    expect(setShowPortfolioModal).toHaveBeenCalledWith(true);

    // Active portfolio is default
    expect(screen.getByText('⭐ Carteira Padrão')).toBeInTheDocument();

    // Export backup
    fireEvent.click(screen.getByTitle('Exportar Carteira (CSV)'));
    expect(handleExportPortfolio).toHaveBeenCalled();

    // Delete portfolio (multiple portfolios exist)
    fireEvent.click(screen.getByTitle('Excluir carteira atual'));
    expect(handleDeletePortfolio).toHaveBeenCalled();
  });

  it('handles non-default active portfolio and single portfolio display', () => {
    const { rerender } = render(
      <PortfolioTabs
        portfolios={mockPortfolios}
        activePortfolioId="p2"
        setActivePortfolioId={setActivePortfolioId}
        setShowPortfolioModal={setShowPortfolioModal}
        handleDeletePortfolio={handleDeletePortfolio}
        handleExportPortfolio={handleExportPortfolio}
        handleSetDefaultPortfolio={handleSetDefaultPortfolio}
      />
    );

    // Should show "Definir como Padrão" button
    const setDefaultBtn = screen.getByTitle('Definir como carteira principal ao fazer login');
    expect(setDefaultBtn).toBeInTheDocument();
    fireEvent.click(setDefaultBtn);
    expect(handleSetDefaultPortfolio).toHaveBeenCalled();

    // Rerender with single portfolio: delete button should NOT be shown
    rerender(
      <PortfolioTabs
        portfolios={[mockPortfolios[0]]}
        activePortfolioId="p1"
        setActivePortfolioId={setActivePortfolioId}
        setShowPortfolioModal={setShowPortfolioModal}
        handleDeletePortfolio={handleDeletePortfolio}
        handleExportPortfolio={handleExportPortfolio}
        handleSetDefaultPortfolio={handleSetDefaultPortfolio}
      />
    );

    expect(screen.queryByTitle('Excluir carteira atual')).not.toBeInTheDocument();
  });

  it('does not render actions when activePortfolioId does not match any portfolio', () => {
    render(
      <PortfolioTabs
        portfolios={mockPortfolios}
        activePortfolioId="non-existent"
        setActivePortfolioId={setActivePortfolioId}
        setShowPortfolioModal={setShowPortfolioModal}
        handleDeletePortfolio={handleDeletePortfolio}
        handleExportPortfolio={handleExportPortfolio}
        handleSetDefaultPortfolio={handleSetDefaultPortfolio}
      />
    );

    expect(screen.queryByText('⭐ Carteira Padrão')).not.toBeInTheDocument();
    expect(screen.queryByTitle('Exportar Carteira (CSV)')).not.toBeInTheDocument();
  });
});
