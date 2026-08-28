import { render } from '@testing-library/react';
import React from 'react';
import {
  WalletIcon,
  ChartIcon,
  BellIcon,
  SettingsIcon,
  LogOutIcon,
  SunIcon,
  MoonIcon,
  MenuIcon,
  XIcon,
  TrendingUpIcon,
  TrendingDownIcon,
  BankIcon,
  CoinsIcon,
  ReceiptIcon,
  MicroscopeIcon,
  CalendarIcon,
  StarIcon,
  TrashIcon,
  DownloadIcon,
  PlusIcon,
} from './icons';

describe('SVG Icons Components', () => {
  it('renders all icons correctly with custom size and color props', () => {
    const icons = [
      WalletIcon, ChartIcon, BellIcon, SettingsIcon, LogOutIcon,
      SunIcon, MoonIcon, MenuIcon, XIcon, TrendingUpIcon,
      TrendingDownIcon, BankIcon, CoinsIcon, ReceiptIcon, MicroscopeIcon,
      CalendarIcon, StarIcon, TrashIcon, DownloadIcon, PlusIcon,
    ];

    icons.forEach((IconComponent) => {
      const { container } = render(<IconComponent size={24} color="#00f2fe" />);
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
      expect(svg).toHaveAttribute('width', '24');
      expect(svg).toHaveAttribute('height', '24');
    });
  });
});
