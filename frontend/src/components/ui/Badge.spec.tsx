import { render, screen } from '@testing-library/react';
import React from 'react';
import { Badge } from './Badge';
import { StarIcon } from './icons';

describe('Badge Component', () => {
  it('renders correctly with children and default props', () => {
    render(<Badge>Status Neutro</Badge>);
    expect(screen.getByText('Status Neutro')).toBeInTheDocument();
  });

  it('renders all variants and sizes with icons', () => {
    const { rerender } = render(<Badge variant="success" size="sm" icon={<StarIcon data-testid="badge-icon" />}>Ativo</Badge>);
    expect(screen.getByText('Ativo')).toBeInTheDocument();
    expect(screen.getByTestId('badge-icon')).toBeInTheDocument();

    rerender(<Badge variant="danger" size="md">Erro</Badge>);
    expect(screen.getByText('Erro')).toBeInTheDocument();

    rerender(<Badge variant="warning">Aviso</Badge>);
    expect(screen.getByText('Aviso')).toBeInTheDocument();

    rerender(<Badge variant="info">Informação</Badge>);
    expect(screen.getByText('Informação')).toBeInTheDocument();

    rerender(<Badge variant="neutral">Padrão</Badge>);
    expect(screen.getByText('Padrão')).toBeInTheDocument();
  });
});
