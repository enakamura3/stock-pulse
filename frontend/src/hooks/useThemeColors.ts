'use client';

import { useEffect, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';

export interface ThemeColors {
  accent: string;
  success: string;
  danger: string;
  warning: string;
  info: string;
  textPrimary: string;
  textSecondary: string;
  bgColor: string;
  panelBg: string;
}

export function useThemeColors(): ThemeColors {
  const { theme } = useTheme();
  const [colors, setColors] = useState<ThemeColors>({
    accent: '#ee6018',
    success: '#10b981',
    danger: '#f43f5e',
    warning: '#f59e0b',
    info: '#38bdf8',
    textPrimary: '#f3f4f6',
    textSecondary: '#9ca3af',
    bgColor: '#08090a',
    panelBg: '#111318',
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const root = document.documentElement;
    const style = getComputedStyle(root);

    const getVar = (name: string, fallback: string) => {
      const val = style.getPropertyValue(name).trim();
      return val || fallback;
    };

    setColors({
      accent: getVar('--accent-color', '#ee6018'),
      success: getVar('--color-success', '#10b981'),
      danger: getVar('--color-danger', '#f43f5e'),
      warning: getVar('--color-warning', '#f59e0b'),
      info: getVar('--color-info', '#38bdf8'),
      textPrimary: getVar('--text-primary', '#f3f4f6'),
      textSecondary: getVar('--text-secondary', '#9ca3af'),
      bgColor: getVar('--bg-color', '#08090a'),
      panelBg: getVar('--panel-bg', '#111318'),
    });
  }, [theme]);

  return colors;
}
