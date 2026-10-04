import React from 'react';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'success' | 'danger' | 'warning' | 'neutral' | 'info';
  size?: 'sm' | 'md';
  icon?: React.ReactNode;
  children: React.ReactNode;
}

export function Badge({
  variant = 'neutral',
  size = 'md',
  icon,
  children,
  className = '',
  style,
  ...props
}: BadgeProps) {
  const getVariantStyles = (): React.CSSProperties => {
    switch (variant) {
      case 'success':
        return {
          background: 'var(--color-success-bg)',
          color: 'var(--color-success)',
          border: '1px solid rgba(var(--success-rgb), 0.25)',
        };
      case 'danger':
        return {
          background: 'var(--color-danger-bg)',
          color: 'var(--color-danger)',
          border: '1px solid rgba(var(--danger-rgb), 0.25)',
        };
      case 'warning':
        return {
          background: 'var(--color-warning-bg)',
          color: 'var(--color-warning)',
          border: '1px solid rgba(var(--warning-rgb), 0.25)',
        };
      case 'info':
        return {
          background: 'var(--color-info-bg)',
          color: 'var(--color-info)',
          border: '1px solid rgba(var(--info-rgb), 0.25)',
        };
      case 'neutral':
      default:
        return {
          background: 'var(--panel-bg)',
          color: 'var(--text-secondary)',
          border: '1px solid var(--panel-border)',
        };
    }
  };

  const getSizeStyles = (): React.CSSProperties => {
    switch (size) {
      case 'sm':
        return { padding: '0.15rem 0.45rem', fontSize: '0.68rem', borderRadius: '3px' };
      case 'md':
      default:
        return { padding: '0.22rem 0.6rem', fontSize: '0.74rem', borderRadius: '3px' };
    }
  };

  const baseStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '5px',
    fontFamily: 'var(--font-mono)',
    fontWeight: 600,
    whiteSpace: 'nowrap',
    letterSpacing: '0.02em',
    ...getSizeStyles(),
    ...getVariantStyles(),
    ...style,
  };

  return (
    <span className={`ui-badge ${className}`} style={baseStyle} {...props}>
      {icon && <span className="badge-icon flex-row items-center">{icon}</span>}
      <span>{children}</span>
    </span>
  );
}
