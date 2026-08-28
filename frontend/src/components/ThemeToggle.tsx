'use client';

import React from 'react';
import { useTheme } from './ThemeProvider';

export default function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();

  return (
    <button
      onClick={toggleTheme}
      className="btn-secondary"
      style={{
        padding: '0.4rem 0.75rem',
        fontSize: '0.8rem',
        display: 'inline-flex',
        alignItems: 'center',
        gap: '6px',
        cursor: 'pointer',
        borderRadius: '8px',
        transition: 'all 0.2s ease',
      }}
      title={theme === 'dark' ? 'Alternar para Tema Claro' : 'Alternar para Tema Escuro'}
      aria-label="Alternar Tema Escuro/Claro"
    >
      {theme === 'dark' ? (
        <>
          <span style={{ fontSize: '0.9rem' }}>☀️</span>
          <span>Claro</span>
        </>
      ) : (
        <>
          <span style={{ fontSize: '0.9rem' }}>🌙</span>
          <span>Escuro</span>
        </>
      )}
    </button>
  );
}
