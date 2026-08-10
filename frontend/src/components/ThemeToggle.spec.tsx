import { render, screen } from '@testing-library/react';
import React from 'react';
import ThemeToggle from './ThemeToggle';
import { ThemeProvider } from './ThemeProvider';
import userEvent from '@testing-library/user-event';

describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('renders dark theme button text initially and switches to light mode on click', async () => {
    const user = userEvent.setup();

    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    );

    const button = screen.getByRole('button', { name: /Alternar Tema Escuro\/Claro/i });
    expect(button).toHaveTextContent('Claro');

    await user.click(button);

    expect(button).toHaveTextContent('Escuro');
    expect(localStorage.getItem('stockpulse_theme')).toBe('light');
  });
});
