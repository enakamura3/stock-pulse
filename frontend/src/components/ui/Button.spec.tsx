import { render, screen } from '@testing-library/react';
import React from 'react';
import { Button } from './Button';
import userEvent from '@testing-library/user-event';
import { PlusIcon } from './icons';

describe('Button Component', () => {
  it('renders correctly with children and default props', () => {
    render(<Button>Clique Aqui</Button>);
    const btn = screen.getByRole('button', { name: /Clique Aqui/i });
    expect(btn).toBeInTheDocument();
  });

  it('renders all variants and sizes', () => {
    const { rerender } = render(<Button variant="primary" size="sm">Primary</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();

    rerender(<Button variant="secondary" size="md">Secondary</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();

    rerender(<Button variant="ghost" size="lg">Ghost</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();

    rerender(<Button variant="danger">Danger</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('handles click events and renders icon', async () => {
    const user = userEvent.setup();
    const onClickMock = vi.fn();

    render(<Button icon={<PlusIcon data-testid="plus-icon" />} onClick={onClickMock}>Adicionar</Button>);

    expect(screen.getByTestId('plus-icon')).toBeInTheDocument();

    await user.click(screen.getByRole('button'));
    expect(onClickMock).toHaveBeenCalledTimes(1);
  });
});
