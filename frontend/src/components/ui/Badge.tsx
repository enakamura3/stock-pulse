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
          border: '1px solid rgba(0, 230, 118, 0.25)',
        };
      case 'danger':
        return {
          background: 'var(--color-danger-bg)',
          color: 'var(--color-danger)',
          border: '1px solid rgba(255, 82, 82, 0.25)',
        };
      case 'warning':
        return {
          background: 'var(--color-warning-bg)',
          color: 'var(--color-warning)',
          border: '1px solid rgba(255, 193, 7, 0.25)',
        };
      case 'info':
        return {
          background: 'rgba(0, 242, 254, 0.1)',
          color: 'var(--accent-color)',
          border: '1px solid rgba(0, 242, 254, 0.25)',
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
        return { padding: '0.2rem 0.5rem', fontSize: '0.7rem', borderRadius: '4px' };
      case 'md':
      default:
        return { padding: '0.3rem 0.75rem', fontSize: '0.8rem', borderRadius: '6px' };
    }
  };

  const baseStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    fontWeight: 600,
    whiteSpace: 'nowrap',
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
