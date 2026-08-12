import React from 'react';
import { renderHook } from '@testing-library/react';
import { useThemeColors } from './useThemeColors';
import { ThemeProvider } from '@/components/ThemeProvider';

describe('useThemeColors', () => {
  it('returns computed or fallback theme colors correctly', () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <ThemeProvider>{children}</ThemeProvider>
    );

    const { result } = renderHook(() => useThemeColors(), { wrapper });

    expect(result.current.accent).toBeTruthy();
    expect(result.current.success).toBeTruthy();
    expect(result.current.danger).toBeTruthy();
    expect(result.current.warning).toBeTruthy();
    expect(result.current.info).toBeTruthy();
    expect(result.current.textPrimary).toBeTruthy();
    expect(result.current.textSecondary).toBeTruthy();
    expect(result.current.bgColor).toBeTruthy();
    expect(result.current.panelBg).toBeTruthy();
  });
});
