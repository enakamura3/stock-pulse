import React from 'react';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  icon?: React.ReactNode;
  children: React.ReactNode;
}

export function Button({
  variant = 'primary',
  size = 'md',
  icon,
  children,
  className = '',
  style,
  ...props
}: ButtonProps) {
  const getVariantStyles = (): React.CSSProperties => {
    switch (variant) {
      case 'primary':
        return {
          background: 'var(--accent-gradient)',
          color: 'var(--accent-foreground)',
          border: 'none',
          boxShadow: '0 4px 14px rgba(var(--accent-rgb), 0.25)',
        };
      case 'secondary':
        return {
          background: 'var(--panel-bg)',
          color: 'var(--text-primary)',
          border: '1px solid var(--panel-border)',
        };
      case 'ghost':
        return {
          background: 'transparent',
          color: 'var(--text-secondary)',
          border: '1px solid transparent',
        };
      case 'danger':
        return {
          background: 'var(--color-danger-bg)',
          color: 'var(--color-danger)',
          border: '1px solid rgba(var(--danger-rgb), 0.3)',
        };
    }
  };

  const getSizeStyles = (): React.CSSProperties => {
    switch (size) {
      case 'sm':
        return { padding: '0.3rem 0.65rem', fontSize: '0.75rem', borderRadius: '4px' };
      case 'md':
        return { padding: '0.45rem 1rem', fontSize: '0.82rem', borderRadius: '4px' };
      case 'lg':
        return { padding: '0.65rem 1.4rem', fontSize: '0.9rem', borderRadius: '5px' };
    }
  };

  const baseStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '6px',
    fontFamily: 'var(--font-mono)',
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'all var(--transition-fast)',
    outline: 'none',
    whiteSpace: 'nowrap',
    letterSpacing: '-0.01em',
    ...getSizeStyles(),
    ...getVariantStyles(),
    ...style,
  };

  return (
    <button className={`ui-button ${className}`} style={baseStyle} {...props}>
      {icon && <span className="button-icon flex-row items-center">{icon}</span>}
      <span>{children}</span>
    </button>
  );
}
