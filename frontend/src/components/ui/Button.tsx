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
          color: '#000',
          border: 'none',
          boxShadow: '0 4px 14px rgba(0, 242, 254, 0.25)',
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
          border: '1px solid rgba(255, 82, 82, 0.3)',
        };
    }
  };

  const getSizeStyles = (): React.CSSProperties => {
    switch (size) {
      case 'sm':
        return { padding: '0.35rem 0.75rem', fontSize: '0.8rem', borderRadius: '6px' };
      case 'md':
        return { padding: '0.5rem 1.25rem', fontSize: '0.9rem', borderRadius: '8px' };
      case 'lg':
        return { padding: '0.75rem 1.75rem', fontSize: '1rem', borderRadius: '10px' };
    }
  };

  const baseStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '8px',
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'all var(--transition-fast)',
    outline: 'none',
    whiteSpace: 'nowrap',
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
