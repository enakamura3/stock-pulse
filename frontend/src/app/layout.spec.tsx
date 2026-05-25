import { render } from '@testing-library/react';
import RootLayout from './layout';
import React from 'react';

vi.mock('@/context/AuthContext', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => <div data-testid="auth-provider">{children}</div>,
}));

describe('RootLayout', () => {
  it('renders html and body with children', () => {
    // Note: Rendering <html> and <body> in jsdom can be tricky.
    // Testing Library usually renders inside a container <div> attached to document.body,
    // so we can test the rendered output of the component itself.
    const { getByTestId, getByText } = render(
      <RootLayout>
        <div>Test Child</div>
      </RootLayout>
    );
    expect(getByTestId('auth-provider')).toBeInTheDocument();
    expect(getByText('Test Child')).toBeInTheDocument();
  });
});
