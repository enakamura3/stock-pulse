import { render, screen } from '@testing-library/react';
import Home from './page';

describe('Home Page', () => {
  it('renders correctly', () => {
    render(<Home />);
    expect(screen.getByText('stock-pulse')).toBeInTheDocument();
    expect(screen.getByText(/A arquitetura foi estabelecida/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Entrar no Dashboard/i })).toBeInTheDocument();
  });
});
