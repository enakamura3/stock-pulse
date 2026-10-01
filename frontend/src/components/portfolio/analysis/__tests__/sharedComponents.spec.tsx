import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import {
  SectionTitle,
  ProgressBar,
  AlertBadge,
  AnalysisCard,
  AssetRiskDetailRow,
  KPIScorecard,
  StatPill,
  ChartTooltipShell,
  DonutCenterLabel,
} from '../sharedComponents';

describe('sharedComponents', () => {
  describe('SectionTitle', () => {
    it('renders emoji, title and subtitle', () => {
      render(<SectionTitle emoji="📊" title="Título da Seção" subtitle="Subtítulo descritivo" />);
      expect(screen.getByText(/📊 Título da Seção/)).toBeInTheDocument();
      expect(screen.getByText('Subtítulo descritivo')).toBeInTheDocument();
    });

    it('renders without subtitle', () => {
      render(<SectionTitle emoji="🚀" title="Sem Subtítulo" />);
      expect(screen.getByText(/🚀 Sem Subtítulo/)).toBeInTheDocument();
    });
  });

  describe('ProgressBar', () => {
    it('renders with label, sublabel, and custom color with positive max', () => {
      const { container } = render(
        <ProgressBar value={50} max={100} color="#00ff00" label="Progresso" sublabel="50%" />
      );
      expect(screen.getByText('Progresso')).toBeInTheDocument();
      expect(screen.getByText('50%')).toBeInTheDocument();
      const fillBar = container.querySelector('div[style*="width: 50%"]');
      expect(fillBar).toBeInTheDocument();
    });

    it('handles max <= 0 and omitted labels', () => {
      const { container } = render(<ProgressBar value={10} max={0} />);
      const fillBar = container.querySelector('div[style*="width: 0%"]');
      expect(fillBar).toBeInTheDocument();
    });
  });

  describe('AlertBadge', () => {
    it('renders warning alert', () => {
      render(<AlertBadge type="warning" message="Aviso importante" />);
      expect(screen.getByText('Aviso importante')).toBeInTheDocument();
    });

    it('renders info alert', () => {
      render(<AlertBadge type="info" message="Informação útil" />);
      expect(screen.getByText('Informação útil')).toBeInTheDocument();
    });

    it('renders success alert', () => {
      render(<AlertBadge type="success" message="Operação bem-sucedida" />);
      expect(screen.getByText('Operação bem-sucedida')).toBeInTheDocument();
    });
  });

  describe('AnalysisCard', () => {
    it('renders children with style and id', () => {
      render(
        <AnalysisCard id="my-card" style={{ marginTop: '10px' }}>
          <span>Card Content</span>
        </AnalysisCard>
      );
      const content = screen.getByText('Card Content');
      expect(content).toBeInTheDocument();
      expect(content.parentElement).toHaveAttribute('id', 'my-card');
    });
  });

  describe('AssetRiskDetailRow', () => {
    it('renders row with barPct and custom barColor and valueColor', () => {
      render(
        <AssetRiskDetailRow
          ticker="PETR4"
          subText="Petrobras"
          valueText="R$ 1.000,00"
          valueColor="#ff0000"
          barPct={45}
          barColor="#ffaa00"
        />
      );
      expect(screen.getByText('PETR4')).toBeInTheDocument();
      expect(screen.getByText('Petrobras')).toBeInTheDocument();
      expect(screen.getByText('R$ 1.000,00')).toBeInTheDocument();
    });

    it('renders row without barPct and default colors', () => {
      render(
        <AssetRiskDetailRow
          ticker="VALE3"
          subText="Vale"
          valueText="R$ 500,00"
        />
      );
      expect(screen.getByText('VALE3')).toBeInTheDocument();
    });
  });

  describe('KPIScorecard', () => {
    it('renders basic KPI card without children and default alert level', () => {
      render(
        <KPIScorecard
          label="Retorno Total"
          value="12.5%"
          color="#00ffff"
          icon="📈"
          subtitle="Últimos 12 meses"
          description="Rendimento acumulado da carteira"
        />
      );
      expect(screen.getByText(/Retorno Total/)).toBeInTheDocument();
      expect(screen.getByText('12.5%')).toBeInTheDocument();
      expect(screen.getByText('Últimos 12 meses')).toBeInTheDocument();
      expect(screen.getByText('Rendimento acumulado da carteira')).toBeInTheDocument();
      expect(screen.queryByText(/Ver detalhes/)).not.toBeInTheDocument();
    });

    it('renders danger and moderate alert levels', () => {
      const { rerender } = render(
        <KPIScorecard
          label="Risco Alto"
          value="80%"
          color="#ff0000"
          icon="⚠️"
          alertLevel="danger"
        />
      );
      expect(screen.getByText('80%')).toBeInTheDocument();

      rerender(
        <KPIScorecard
          label="Risco Médio"
          value="50%"
          color="#ffaa00"
          icon="⚡"
          alertLevel="moderate"
        />
      );
      expect(screen.getByText('50%')).toBeInTheDocument();
    });

    it('handles hover and expand/collapse when children are provided', () => {
      render(
        <KPIScorecard
          label="Concentração"
          value="40%"
          color="#ffff00"
          icon="🎯"
          alertLevel="safe"
        >
          <div data-testid="kpi-children">Detalhes da Concentração</div>
        </KPIScorecard>
      );

      const trigger = screen.getByText('Ver detalhes e ativos');
      expect(trigger).toBeInTheDocument();
      expect(screen.queryByTestId('kpi-children')).not.toBeInTheDocument();

      const card = screen.getByText(/Concentração/).closest('div')!;
      fireEvent.mouseEnter(card);
      fireEvent.mouseLeave(card);

      // Click to expand
      fireEvent.click(trigger);
      expect(screen.getByText('Ocultar detalhes')).toBeInTheDocument();
      const childrenEl = screen.getByTestId('kpi-children');
      expect(childrenEl).toBeInTheDocument();

      // Click inside children stops propagation
      fireEvent.click(childrenEl.parentElement!);
      expect(screen.getByText('Ocultar detalhes')).toBeInTheDocument();

      // Click trigger to collapse
      fireEvent.click(screen.getByText('Ocultar detalhes'));
      expect(screen.queryByTestId('kpi-children')).not.toBeInTheDocument();
    });
  });

  describe('StatPill', () => {
    it('renders label and value with color', () => {
      render(<StatPill label="Sharpe" value="1.85" color="#00ff00" />);
      expect(screen.getByText('Sharpe')).toBeInTheDocument();
      expect(screen.getByText('1.85')).toBeInTheDocument();
    });
  });

  describe('ChartTooltipShell', () => {
    it('returns null when inactive or payload is empty', () => {
      const { container: c1 } = render(<ChartTooltipShell active={false} payload={[{ name: 'A', value: 10 }]} label="L" />);
      expect(c1.firstChild).toBeNull();

      const { container: c2 } = render(<ChartTooltipShell active={true} payload={null} label="L" />);
      expect(c2.firstChild).toBeNull();

      const { container: c3 } = render(<ChartTooltipShell active={true} payload={[]} label="L" />);
      expect(c3.firstChild).toBeNull();
    });

    it('renders payload entries with and without formatter', () => {
      const payload = [
        { name: 'PETR4', value: 100, color: '#ff0000' },
        { name: 'VALE3', value: 200, color: '#00ff00' },
      ];

      const { rerender } = render(
        <ChartTooltipShell
          active={true}
          payload={payload}
          label="Jan 2024"
          formatter={(v: number) => `R$ ${v},00`}
        />
      );

      expect(screen.getByText('Jan 2024')).toBeInTheDocument();
      expect(screen.getByText('PETR4:')).toBeInTheDocument();
      expect(screen.getByText('R$ 100,00')).toBeInTheDocument();

      // Without formatter
      rerender(
        <ChartTooltipShell
          active={true}
          payload={payload}
          label="Fev 2024"
        />
      );
      expect(screen.getByText('Fev 2024')).toBeInTheDocument();
      expect(screen.getByText('100')).toBeInTheDocument();
    });
  });

  describe('DonutCenterLabel', () => {
    it('returns null when viewBox is not provided', () => {
      const { container } = render(
        <svg>
          <DonutCenterLabel title="Total" value="R$ 10.000" />
        </svg>
      );
      expect(container.querySelector('g')).toBeNull();
    });

    it('renders text with cx and cy from viewBox', () => {
      const { container } = render(
        <svg>
          <DonutCenterLabel
            viewBox={{ cx: 150, cy: 120 }}
            title="Patrimônio"
            value="R$ 50.000"
          />
        </svg>
      );
      expect(screen.getByText('Patrimônio')).toBeInTheDocument();
      expect(screen.getByText('R$ 50.000')).toBeInTheDocument();
      const texts = container.querySelectorAll('text');
      expect(texts[0]).toHaveAttribute('x', '150');
      expect(texts[0]).toHaveAttribute('y', '112');
      expect(texts[1]).toHaveAttribute('x', '150');
      expect(texts[1]).toHaveAttribute('y', '134');
    });
  });
});
