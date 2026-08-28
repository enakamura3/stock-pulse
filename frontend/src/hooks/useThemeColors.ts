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
    accent: '#818cf8',
    success: '#34d399',
    danger: '#fb7185',
    warning: '#fbbf24',
    info: '#60a5fa',
    textPrimary: '#fafafa',
    textSecondary: '#a1a1aa',
    bgColor: '#09090b',
    panelBg: 'rgba(24, 24, 27, 0.65)',
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
      accent: getVar('--accent-color', '#818cf8'),
      success: getVar('--color-success', '#34d399'),
      danger: getVar('--color-danger', '#fb7185'),
      warning: getVar('--color-warning', '#fbbf24'),
      info: getVar('--color-info', '#60a5fa'),
      textPrimary: getVar('--text-primary', '#fafafa'),
      textSecondary: getVar('--text-secondary', '#a1a1aa'),
      bgColor: getVar('--bg-color', '#09090b'),
      panelBg: getVar('--panel-bg', 'rgba(24, 24, 27, 0.65)'),
    });
  }, [theme]);

  return colors;
}
