import React, { useMemo } from 'react';
import { Position, CalculatedDividend, FixedIncomePosition, TreasuryPosition } from '../types';
import { formatMoney } from '../helpers';
import { SectionTitle, AnalysisCard, KPIScorecard, ProgressBar, AlertBadge, StatPill } from './sharedComponents';

interface TaxEfficiencySectionProps {
  positions: Position[];
  dividends: CalculatedDividend[];
  fiPositions: FixedIncomePosition[];
  treasuryPositions: TreasuryPosition[];
  kpiCurrency: string;
}

export default function TaxEfficiencySection({
  positions,
  dividends,
  fiPositions,
  treasuryPositions,
  kpiCurrency,
}: TaxEfficiencySectionProps) {
  const taxMetrics = useMemo(() => {
    // 1. Patrimônio Isento vs Tributável
    let isentoEquity = 0;
    let tributavelEquity = 0;

    positions.forEach(p => {
      const val = p.current_value || 0;
      if (['STOCK_BR', 'FII', 'FIAGRO', 'ETF_BR'].includes(p.type)) {
        isentoEquity += val;
      } else {
        tributavelEquity += val;
      }
    });

    let isentoFI = 0;
    let tributavelFI = 0;

    fiPositions.forEach(p => {
      const type = p.asset?.type || (p as any).type || '';
      if (['LCI', 'LCA'].includes(type)) {
        isentoFI += p.net_value;
      } else {
        tributavelFI += p.net_value;
      }
    });

    const tributavelTD = treasuryPositions.reduce((s, p) => s + p.net_value, 0);

    const totalIsento = isentoEquity + isentoFI;
    const totalTributavel = tributavelEquity + tributavelFI + tributavelTD;
    const totalPortfolio = totalIsento + totalTributavel;

    const isentoPct = totalPortfolio > 1e-6 ? (totalIsento / totalPortfolio) * 100 : 0;
    const tributavelPct = totalPortfolio > 1e-6 ? (totalTributavel / totalPortfolio) * 100 : 0;

    // 2. Proventos Isentos vs Tributáveis (JCP / Renda Fixa)
    let proventosIsentos = 0;
    let proventosJCP = 0;
    let proventosRF = 0;

    dividends.forEach(d => {
      if (d.is_accrued) {
        proventosRF += d.net_amount;
      } else if (d.type && d.type.toLowerCase().includes('jcp')) {
        proventosJCP += d.net_amount;
      } else {
        proventosIsentos += d.net_amount;
      }
    });

    const totalProventos = proventosIsentos + proventosJCP + proventosRF;
    const proventosIsentosPct = totalProventos > 1e-6 ? (proventosIsentos / totalProventos) * 100 : 0;

    // Estimativa de imposto retido na fonte em JCP (JCP tem 15% retido; líquido é 85% do bruto)
    // Imposto estimado = líquido / 0.85 * 0.15
    const irRetidoJCP = proventosJCP > 1e-6 ? (proventosJCP / 0.85) * 0.15 : 0;

    // 3. IR e Taxas Retidos em Renda Fixa & Tesouro Direto
    const irRetidoFI = fiPositions.reduce((s, p) => s + (p.ir_amount || 0), 0);
    const iofRetidoFI = fiPositions.reduce((s, p) => s + (p.iof_amount || 0), 0);

    const irRetidoTD = treasuryPositions.reduce((s, p) => s + (p.ir_tax || 0), 0);
    const iofRetidoTD = treasuryPositions.reduce((s, p) => s + (p.iof_tax || 0), 0);
    const taxaB3TD = treasuryPositions.reduce((s, p) => s + (p.b3_fee || 0), 0);

    const totalIRRetidoRF = irRetidoFI + irRetidoTD;
    const totalIOFRetido = iofRetidoFI + iofRetidoTD;
    const totalImpostosEDividas = totalIRRetidoRF + totalIOFRetido + taxaB3TD + irRetidoJCP;

    return {
      totalPortfolio,
      totalIsento,
      totalTributavel,
      isentoPct: Number(isentoPct.toFixed(1)),
      tributavelPct: Number(tributavelPct.toFixed(1)),
      isentoEquity,
      isentoFI,
      proventosIsentos,
      proventosJCP,
      proventosRF,
      totalProventos,
      proventosIsentosPct: Number(proventosIsentosPct.toFixed(1)),
      irRetidoJCP,
      totalIRRetidoRF,
      totalIOFRetido,
      taxaB3TD,
      totalImpostosEDividas,
    };
  }, [positions, dividends, fiPositions, treasuryPositions]);

  const currency = kpiCurrency || 'BRL';

  return (
    <AnalysisCard id="section-tax-efficiency">
      <SectionTitle
        emoji="💸"
        title="Eficiência Tributária"
        subtitle="Análise de isenções fiscais, retenções de IR e otimização tributária da carteira"
      />

      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
        {/* KPI 1: Isenção Patrimonial */}
        <KPIScorecard
          icon="🛡️"
          label="Patrimônio Isento de IR"
          value={`${taxMetrics.isentoPct.toFixed(1)}%`}
          subtitle={`${formatMoney(taxMetrics.totalIsento, currency)} livre de imposto de renda`}
          description={`Um percentual alto de patrimônio em ativos isentos (Ações B3, FIIs, FIAGROs, LCI e LCA) otimiza a rentabilidade líquida acumulada no longo prazo.`}
          color={taxMetrics.isentoPct >= 50 ? '#4ade80' : taxMetrics.isentoPct >= 25 ? '#fbbf24' : '#60a5fa'}
          alertLevel={taxMetrics.isentoPct >= 50 ? 'safe' : taxMetrics.isentoPct >= 25 ? 'moderate' : 'safe'}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <ProgressBar
              value={taxMetrics.totalIsento}
              max={taxMetrics.totalPortfolio}
              color="#4ade80"
              label="Patrimônio Isento (FIIs, LCIs, Ações)"
              sublabel={`${taxMetrics.isentoPct.toFixed(1)}%`}
            />
            <ProgressBar
              value={taxMetrics.totalTributavel}
              max={taxMetrics.totalPortfolio}
              color="#f87171"
              label="Patrimônio Sujeito a IR (CDBs, Tesouro, BDRs)"
              sublabel={`${taxMetrics.tributavelPct.toFixed(1)}%`}
            />
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
              <span>Renda Variável Isenta (B3 / FII):</span>
              <strong style={{ color: 'var(--text-primary)' }}>{formatMoney(taxMetrics.isentoEquity, currency)}</strong>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
              <span>Renda Fixa Isenta (LCI / LCA):</span>
              <strong style={{ color: 'var(--text-primary)' }}>{formatMoney(taxMetrics.isentoFI, currency)}</strong>
            </div>
          </div>
        </KPIScorecard>

        {/* KPI 2: Isenção de Proventos */}
        <KPIScorecard
          icon="✨"
          label="Proventos Isentos"
          value={`${taxMetrics.proventosIsentosPct.toFixed(1)}%`}
          subtitle={`${formatMoney(taxMetrics.proventosIsentos, currency)} recebidos sem desconto de IR`}
          description={`Dividendos de ações nacionais e rendimentos de FIIs são 100% isentos de IR na pessoa física. Juros sobre Capital Próprio (JCP) sofrem 15% de retenção na fonte.`}
          color={taxMetrics.proventosIsentosPct >= 70 ? '#4ade80' : '#fbbf24'}
          alertLevel={taxMetrics.proventosIsentosPct >= 70 ? 'safe' : 'moderate'}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
            <StatPill
              label="Dividendos & Rendimentos (Isentos)"
              value={formatMoney(taxMetrics.proventosIsentos, currency)}
              color="#4ade80"
            />
            <StatPill
              label="JCP Recebido (Líquido de IR 15%)"
              value={formatMoney(taxMetrics.proventosJCP, currency)}
              color="#fbbf24"
            />
            {taxMetrics.irRetidoJCP > 0 && (
              <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', textAlign: 'right', marginTop: '0.2rem' }}>
                IR estimado retido na fonte (JCP): <strong style={{ color: '#f87171' }}>{formatMoney(taxMetrics.irRetidoJCP, currency)}</strong>
              </div>
            )}
          </div>
        </KPIScorecard>

        {/* KPI 3: Impostos Retidos na Renda Fixa */}
        <KPIScorecard
          icon="🏛️"
          label="Impostos Retidos (RF/TD)"
          value={formatMoney(taxMetrics.totalIRRetidoRF, currency)}
          subtitle="Imposto de Renda provisionado/retido na Renda Fixa e Tesouro"
          description={`Valores de IR calculados pela tabela regressiva (22.5% a 15%) sobre o rendimento bruto de CDBs, Debêntures e Tesouro Direto.`}
          color={taxMetrics.totalIRRetidoRF > 0 ? '#f87171' : '#4ade80'}
          alertLevel={taxMetrics.totalIRRetidoRF > 1000 ? 'moderate' : 'safe'}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
            <StatPill
              label="IR Retido em Renda Fixa & Tesouro"
              value={formatMoney(taxMetrics.totalIRRetidoRF, currency)}
              color="#f87171"
            />
            {taxMetrics.taxaB3TD > 0 && (
              <StatPill
                label="Taxa de Custódia B3 (Tesouro Direto)"
                value={formatMoney(taxMetrics.taxaB3TD, currency)}
                color="#c084fc"
              />
            )}
            {taxMetrics.totalIOFRetido > 0 && (
              <StatPill
                label="IOF Retido (Resgates < 30 dias)"
                value={formatMoney(taxMetrics.totalIOFRetido, currency)}
                color="#fbbf24"
              />
            )}
          </div>
        </KPIScorecard>
      </div>

      {taxMetrics.isentoFI > 0 && (
        <AlertBadge
          type="success"
          message={`💡 Excelente escolha de alocação: Você possui ${formatMoney(taxMetrics.isentoFI, currency)} em LCIs/LCAs imunes a imposto de renda, otimizando a rentabilidade líquida do caixa.`}
        />
      )}
    </AnalysisCard>
  );
}
