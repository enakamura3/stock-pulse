import React, { useMemo } from 'react';
import { Position, FixedIncomePosition, PerformancePoint, TreasuryPosition } from '../types';
import { formatMoney } from '../helpers';
import { isFII, isRendaVariavel } from './constants';
import { SectionTitle, AnalysisCard, KPIScorecard, AssetRiskDetailRow } from './sharedComponents';

interface RiskConcentrationSectionProps {
  positions: Position[];
  fiPositions: FixedIncomePosition[];
  treasuryPositions: TreasuryPosition[];
  performanceData: PerformancePoint[];
  kpiCurrency: string;
}

export default function RiskConcentrationSection({
  positions,
  fiPositions,
  treasuryPositions,
  performanceData,
  kpiCurrency,
}: RiskConcentrationSectionProps) {
  const totalPortfolioValue = useMemo(() => {
    const eq = positions.reduce((s, p) => s + (p.current_value || 0), 0);
    const fi = fiPositions.reduce((s, p) => s + p.net_value, 0);
    const td = treasuryPositions.reduce((s, p) => s + p.net_value, 0);
    return eq + fi + td;
  }, [positions, fiPositions, treasuryPositions]);

  const riskMetrics = useMemo(() => {
    if (!performanceData || performanceData.length < 10) {
      return { sharpe: null, beta: null, maxDrawdown: null };
    }

    const dailyReturns: number[] = [];
    for (let i = 1; i < performanceData.length; i++) {
      const prev = performanceData[i - 1].value;
      const curr = performanceData[i].value;
      if (prev > 1e-6) {
        dailyReturns.push((curr - prev) / prev);
      }
    }

    if (dailyReturns.length < 5) {
      return { sharpe: null, beta: null, maxDrawdown: null };
    }

    const meanReturn = dailyReturns.reduce((s, r) => s + r, 0) / dailyReturns.length;
    const variance = dailyReturns.reduce((s, r) => s + Math.pow(r - meanReturn, 2), 0) / (dailyReturns.length - 1);
    const stdDev = Math.sqrt(variance);

    const riskFreeDaily = 0.0005;
    const annualizedExcess = (meanReturn - riskFreeDaily) * 252;
    const annualizedVol = stdDev * Math.sqrt(252);
    const sharpe = annualizedVol > 1e-8 ? annualizedExcess / annualizedVol : 0;

    const marketVol = 0.012;
    const beta = stdDev / marketVol;

    let peak = performanceData[0].value;
    let maxDD = 0;
    for (const p of performanceData) {
      if (p.value > peak) peak = p.value;
      const dd = peak > 1e-6 ? (peak - p.value) / peak : 0;
      if (dd > maxDD) maxDD = dd;
    }

    return {
      sharpe: Number(sharpe.toFixed(2)),
      beta: Number(beta.toFixed(2)),
      maxDrawdown: Number((maxDD * 100).toFixed(1)),
    };
  }, [performanceData]);

  const assetProfitLoss = useMemo(() => {
    const list: { ticker: string; name: string; profitLoss: number; returnPercent: number; weight: number }[] = [];
    
    positions.forEach(p => {
      const pl = p.profit_loss !== undefined ? p.profit_loss : ((p.current_value || 0) - p.total_cost);
      const retPct = p.return_percent !== undefined ? p.return_percent : (p.total_cost > 0 ? (pl / p.total_cost) * 100 : 0);
      const weight = totalPortfolioValue > 1e-6 ? ((p.current_value || 0) / totalPortfolioValue) * 100 : 0;
      list.push({
        ticker: p.ticker,
        name: p.name,
        profitLoss: pl,
        returnPercent: retPct,
        weight,
      });
    });

    fiPositions.forEach(p => {
      const invested = p.total_invested !== undefined ? p.total_invested : ((p as any).invested_amount || 0);
      const pl = p.net_value - invested;
      const retPct = p.net_return_percent !== undefined ? p.net_return_percent : (invested > 0 ? (pl / invested) * 100 : 0);
      const weight = totalPortfolioValue > 1e-6 ? (p.net_value / totalPortfolioValue) * 100 : 0;
      
      const type = p.asset?.type || (p as any).type || 'Renda Fixa';
      const indexer = p.asset?.indexer || (p as any).index_type || (p as any).indexer || 'CDI';
      const institution = p.asset?.institution || (p as any).institution || '';
      const indexerLabel = indexer === 'PREFIXADO' ? 'Pré' : indexer;
      
      list.push({
        ticker: `${type} ${indexerLabel}`,
        name: institution,
        profitLoss: pl,
        returnPercent: retPct,
        weight,
      });
    });

    treasuryPositions.forEach(p => {
      const pl = p.net_value - p.total_invested;
      const retPct = p.total_invested > 0 ? (pl / p.total_invested) * 100 : 0;
      const weight = totalPortfolioValue > 1e-6 ? (p.net_value / totalPortfolioValue) * 100 : 0;
      
      list.push({
        ticker: `${p.ticker} (Tesouro)`,
        name: 'Tesouro Direto',
        profitLoss: pl,
        returnPercent: retPct,
        weight,
      });
    });

    return list;
  }, [positions, fiPositions, treasuryPositions, totalPortfolioValue]);

  const volatilityExposure = useMemo(() => {
    let highVolVal = 0;
    let medVolVal = 0;
    let lowVolVal = 0;

    positions.forEach(p => {
      const val = p.current_value || 0;
      if (isFII(p.type)) {
        medVolVal += val;
      } else if (isRendaVariavel(p.type)) {
        highVolVal += val;
      } else {
        lowVolVal += val;
      }
    });

    const fiVal = fiPositions.reduce((s, p) => s + p.net_value, 0);
    const tdVal = treasuryPositions.reduce((s, p) => s + p.net_value, 0);
    lowVolVal += fiVal + tdVal;

    const total = highVolVal + medVolVal + lowVolVal;
    if (total < 1e-6) {
      return { highPct: 0, medPct: 0, lowPct: 0, topAggressive: [] };
    }

    const highVolAssets = positions
      .filter(p => isRendaVariavel(p.type) && !isFII(p.type))
      .map(p => ({
        ticker: p.ticker,
        weight: totalPortfolioValue > 1e-6 ? ((p.current_value || 0) / totalPortfolioValue) * 100 : 0,
      }))
      .sort((a, b) => b.weight - a.weight)
      .slice(0, 3);

    return {
      highPct: (highVolVal / total) * 100,
      medPct: (medVolVal / total) * 100,
      lowPct: (lowVolVal / total) * 100,
      topAggressive: highVolAssets,
    };
  }, [positions, fiPositions, treasuryPositions, totalPortfolioValue]);

  const concentrationMetrics = useMemo(() => {
    const list: { ticker: string; weight: number }[] = [];

    positions.forEach(p => {
      const w = totalPortfolioValue > 1e-6 ? ((p.current_value || 0) / totalPortfolioValue) * 100 : 0;
      list.push({ ticker: p.ticker, weight: w });
    });

    fiPositions.forEach(p => {
      const w = totalPortfolioValue > 1e-6 ? (p.net_value / totalPortfolioValue) * 100 : 0;
      const type = p.asset?.type || (p as any).type || 'Renda Fixa';
      const indexer = p.asset?.indexer || (p as any).index_type || (p as any).indexer || 'CDI';
      const institution = p.asset?.institution || (p as any).institution || '';
      const indexerLabel = indexer === 'PREFIXADO' ? 'Pré' : indexer;
      
      list.push({ ticker: `${type} ${indexerLabel} (${institution || 'Renda Fixa'})`, weight: w });
    });

    treasuryPositions.forEach(p => {
      const w = totalPortfolioValue > 1e-6 ? (p.net_value / totalPortfolioValue) * 100 : 0;
      list.push({ ticker: `${p.ticker} (Tesouro)`, weight: w });
    });

    const sorted = [...list].sort((a, b) => b.weight - a.weight);
    const top3 = sorted.slice(0, 3);
    const top3Sum = top3.reduce((s, x) => s + x.weight, 0);

    return {
      top3,
      top3Sum,
    };
  }, [positions, fiPositions, treasuryPositions, totalPortfolioValue]);

  const topGainers = useMemo(() => {
    return [...assetProfitLoss]
      .filter(item => item.profitLoss > 0)
      .sort((a, b) => b.profitLoss - a.profitLoss)
      .slice(0, 3);
  }, [assetProfitLoss]);

  const topLosers = useMemo(() => {
    return [...assetProfitLoss]
      .filter(item => item.profitLoss < 0)
      .sort((a, b) => a.profitLoss - b.profitLoss)
      .slice(0, 3);
  }, [assetProfitLoss]);

  const performanceAttribution = useMemo(() => {
    return assetProfitLoss
      .map(item => ({
        ...item,
        contribution: (item.returnPercent * item.weight) / 100,
      }))
      .sort((a, b) => b.contribution - a.contribution);
  }, [assetProfitLoss]);

  const topContributors = useMemo(() => {
    return performanceAttribution.filter(item => item.contribution > 0).slice(0, 5);
  }, [performanceAttribution]);

  const topDetractors = useMemo(() => {
    return [...performanceAttribution]
      .filter(item => item.contribution < 0)
      .sort((a, b) => a.contribution - b.contribution)
      .slice(0, 5);
  }, [performanceAttribution]);

  const totalContribution = useMemo(() => {
    return performanceAttribution.reduce((sum, item) => sum + item.contribution, 0);
  }, [performanceAttribution]);

  return (
    <AnalysisCard id="section-risk">
      <SectionTitle
        emoji="🌡️"
        title="Termômetro de Risco"
        subtitle="Indicadores-chave de risco, eficiência e atribuição de performance da carteira"
      />

      {riskMetrics.sharpe !== null ? (
        <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
          {/* Sharpe Ratio */}
          <KPIScorecard
            icon="⚡"
            label="Índice de Sharpe"
            value={riskMetrics.sharpe!.toFixed(2)}
            subtitle={
              riskMetrics.sharpe! >= 1
                ? 'Excelente eficiência ajustada ao risco'
                : riskMetrics.sharpe! >= 0.5
                  ? 'Eficiência moderada — espaço para otimizar'
                  : 'Eficiência abaixo do ideal — risco não compensado'
            }
            description={
              riskMetrics.sharpe! >= 1
                ? `O retorno gerado supera a volatilidade dos ativos. Cada 1% de risco assumido entregou mais de 1% de retorno excedente, demonstrando excelente equilíbrio.`
                : riskMetrics.sharpe! >= 0.5
                  ? `O retorno compensa a volatilidade de forma moderada. É possível otimizar a carteira reduzindo ativos de alta oscilação ou melhorando a diversificação.`
                  : `A volatilidade da carteira é alta demais para o retorno que ela entrega. Indica que a carteira corre risco pouco eficiente para o ganho obtido.`
            }
            color={riskMetrics.sharpe! >= 1 ? '#4ade80' : riskMetrics.sharpe! >= 0.5 ? '#fbbf24' : '#f87171'}
            alertLevel={riskMetrics.sharpe! >= 1 ? 'safe' : riskMetrics.sharpe! >= 0.5 ? 'moderate' : 'danger'}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              {topGainers.length > 0 && (
                <div>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#4ade80', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                    <span>📈</span> Ativos mais lucrativos
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {topGainers.map(item => (
                      <AssetRiskDetailRow
                        key={item.ticker}
                        ticker={item.ticker}
                        subText={`Peso: ${item.weight.toFixed(1)}%`}
                        valueText={`+${formatMoney(item.profitLoss, kpiCurrency || 'BRL')} (${item.returnPercent.toFixed(1)}%)`}
                        valueColor="#4ade80"
                      />
                    ))}
                  </div>
                </div>
              )}

              {topLosers.length > 0 && (
                <div style={{ marginTop: '0.25rem' }}>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#f87171', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                    <span>📉</span> Ativos detratores
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {topLosers.map(item => (
                      <AssetRiskDetailRow
                        key={item.ticker}
                        ticker={item.ticker}
                        subText={`Peso: ${item.weight.toFixed(1)}%`}
                        valueText={`${formatMoney(item.profitLoss, kpiCurrency || 'BRL')} (${item.returnPercent.toFixed(1)}%)`}
                        valueColor="#f87171"
                      />
                    ))}
                  </div>
                </div>
              )}

              {topGainers.length === 0 && topLosers.length === 0 && (
                <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', textAlign: 'center' }}>
                  Nenhum ativo com ganhos ou perdas significativos.
                </div>
              )}
            </div>
          </KPIScorecard>

          {/* Atribuição de Performance */}
          <KPIScorecard
            icon="🎯"
            label="Atribuição de Performance"
            value={`${totalContribution >= 0 ? '+' : ''}${totalContribution.toFixed(2)}pp`}
            subtitle="Contribuição ponderada de cada ativo para o retorno da carteira"
            description={
              totalContribution >= 0
                ? `A carteira acumula resultado positivo com contribuição líquida de +${totalContribution.toFixed(2)}pp decorrente da alocação ponderada dos seus ativos.`
                : `A carteira acumula resultado negativo de ${totalContribution.toFixed(2)}pp influenciada pela alocação em ativos detratores.`
            }
            color={totalContribution >= 0 ? '#4ade80' : '#f87171'}
            alertLevel={totalContribution >= 0 ? 'safe' : 'danger'}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              {topContributors.length > 0 && (
                <div>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#4ade80', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                    <span>📈</span> Maiores Contribuidores (pp)
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {topContributors.map(item => {
                      const maxContrib = topContributors[0]?.contribution || 1;
                      const barPct = Math.min(100, Math.abs(item.contribution / maxContrib) * 100);
                      return (
                        <AssetRiskDetailRow
                          key={item.ticker}
                          ticker={item.ticker}
                          subText={`Peso: ${item.weight.toFixed(1)}% · Retorno: ${item.returnPercent.toFixed(1)}%`}
                          valueText={`+${item.contribution.toFixed(2)}pp`}
                          valueColor="#4ade80"
                          barPct={barPct}
                          barColor="#4ade80"
                        />
                      );
                    })}
                  </div>
                </div>
              )}

              {topDetractors.length > 0 && (
                <div style={{ marginTop: '0.25rem' }}>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#f87171', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                    <span>📉</span> Maiores Detratores (pp)
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {topDetractors.map(item => {
                      const maxDetract = Math.abs(topDetractors[0]?.contribution || 1);
                      const barPct = Math.min(100, Math.abs(item.contribution / maxDetract) * 100);
                      return (
                        <AssetRiskDetailRow
                          key={item.ticker}
                          ticker={item.ticker}
                          subText={`Peso: ${item.weight.toFixed(1)}% · Retorno: ${item.returnPercent.toFixed(1)}%`}
                          valueText={`${item.contribution.toFixed(2)}pp`}
                          valueColor="#f87171"
                          barPct={barPct}
                          barColor="#f87171"
                        />
                      );
                    })}
                  </div>
                </div>
              )}

              {topContributors.length === 0 && topDetractors.length === 0 && (
                <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', textAlign: 'center' }}>
                  Sem dados suficientes para calcular a contribuição individual dos ativos.
                </div>
              )}
            </div>
          </KPIScorecard>

          {/* Beta */}
          <KPIScorecard
            icon="📊"
            label="Beta"
            value={riskMetrics.beta!.toFixed(2)}
            subtitle={
              riskMetrics.beta! <= 0.8
                ? 'Carteira defensiva — menos volátil que o mercado'
                : riskMetrics.beta! <= 1.2
                  ? 'Volatilidade próxima ao mercado'
                  : 'Carteira agressiva — mais volátil que o mercado'
            }
            description={
              riskMetrics.beta! <= 0.8
                ? `A carteira tende a oscilar cerca de ${riskMetrics.beta!.toFixed(2)}x a variação do Ibovespa, oferecendo um perfil defensivo que amortece as quedas do mercado.`
                : riskMetrics.beta! <= 1.2
                  ? `A carteira varia de forma muito semelhante ao Ibovespa (cerca de ${riskMetrics.beta!.toFixed(2)}x), replicando o comportamento médio do mercado.`
                  : `A carteira tende a variar ${riskMetrics.beta!.toFixed(2)}x mais que o Ibovespa, o que amplifica os ganhos em altas, mas aumenta o risco de perdas em quedas.`
            }
            color={riskMetrics.beta! <= 0.8 ? '#4ade80' : riskMetrics.beta! <= 1.2 ? '#fbbf24' : '#f87171'}
            alertLevel={riskMetrics.beta! <= 0.8 ? 'safe' : riskMetrics.beta! <= 1.2 ? 'moderate' : 'danger'}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              <div>
                <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '0.4rem' }}>
                  Perfil de Oscilação da Carteira
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                  <AssetRiskDetailRow
                    ticker="Alta Oscilação"
                    subText="Ações, ETFs, BDRs, Cripto"
                    valueText={`${volatilityExposure.highPct.toFixed(1)}%`}
                    barPct={volatilityExposure.highPct}
                    barColor="linear-gradient(90deg, #f87171, #ef4444)"
                  />
                  <AssetRiskDetailRow
                    ticker="Média Oscilação"
                    subText="FIIs, FIAGROs"
                    valueText={`${volatilityExposure.medPct.toFixed(1)}%`}
                    barPct={volatilityExposure.medPct}
                    barColor="linear-gradient(90deg, #fbbf24, #f59e0b)"
                  />
                  <AssetRiskDetailRow
                    ticker="Baixa Oscilação"
                    subText="Renda Fixa & Caixa"
                    valueText={`${volatilityExposure.lowPct.toFixed(1)}%`}
                    barPct={volatilityExposure.lowPct}
                    barColor="linear-gradient(90deg, #4ade80, #10b981)"
                  />
                </div>
              </div>

              {volatilityExposure.topAggressive.length > 0 && (
                <div style={{ marginTop: '0.25rem' }}>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: '#fbbf24', marginBottom: '0.4rem', display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                    <span>⚡</span> Ativos mais Voláteis (por peso)
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {volatilityExposure.topAggressive.map(item => (
                      <div key={item.ticker} style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.72rem', color: 'var(--text-secondary)' }}>
                        <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</span>
                        <span>Peso na Carteira: <strong style={{ color: 'var(--text-primary)' }}>{item.weight.toFixed(1)}%</strong></span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </KPIScorecard>

          {/* Max Drawdown */}
          <KPIScorecard
            icon="📉"
            label="Drawdown Máximo"
            value={`-${riskMetrics.maxDrawdown!.toFixed(1)}%`}
            subtitle={
              riskMetrics.maxDrawdown! <= 10
                ? 'Drawdown contido — volatilidade controlada'
                : riskMetrics.maxDrawdown! <= 25
                  ? 'Drawdown moderado — considere proteger posições'
                  : 'Drawdown severo — revise a alocação de risco'
            }
            description={
              riskMetrics.maxDrawdown! <= 10
                ? `A maior queda da carteira a partir do seu pico recente foi de -${riskMetrics.maxDrawdown!.toFixed(1)}%. Este comportamento indica excelente controle de perdas temporárias.`
                : riskMetrics.maxDrawdown! <= 25
                  ? `A carteira sofreu uma queda máxima de -${riskMetrics.maxDrawdown!.toFixed(1)}% em relação ao seu pico recente. Sugere volatilidade intermediária a ser monitorada.`
                  : `A carteira sofreu uma queda severa de -${riskMetrics.maxDrawdown!.toFixed(1)}% a partir do seu pico recente, indicando alta sensibilidade a cenários de forte estresse.`
            }
            color={riskMetrics.maxDrawdown! <= 10 ? '#4ade80' : riskMetrics.maxDrawdown! <= 25 ? '#fbbf24' : '#f87171'}
            alertLevel={riskMetrics.maxDrawdown! <= 10 ? 'safe' : riskMetrics.maxDrawdown! <= 25 ? 'moderate' : 'danger'}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              <div>
                <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '0.4rem' }}>
                  Concentração nos 3 Maiores Ativos
                </div>
                <AssetRiskDetailRow
                  ticker="Concentração Top 3"
                  subText={
                    concentrationMetrics.top3Sum > 50
                      ? 'Concentração elevada (risco alto)'
                      : concentrationMetrics.top3Sum > 30
                        ? 'Concentração moderada'
                        : 'Bem diversificada'
                  }
                  valueText={`${concentrationMetrics.top3Sum.toFixed(1)}%`}
                  barPct={concentrationMetrics.top3Sum}
                  barColor={concentrationMetrics.top3Sum > 50 ? '#f87171' : concentrationMetrics.top3Sum > 30 ? '#fbbf24' : '#4ade80'}
                />
              </div>

              {concentrationMetrics.top3.length > 0 && (
                <div style={{ marginTop: '0.25rem' }}>
                  <div style={{ fontSize: '0.7rem', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '0.4rem' }}>
                    Maiores Alocações Individuais
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                    {concentrationMetrics.top3.map((item, idx) => (
                      <AssetRiskDetailRow
                        key={item.ticker}
                        ticker={`${idx + 1}. ${item.ticker}`}
                        subText="Alocação"
                        valueText={`${item.weight.toFixed(1)}%`}
                        barPct={item.weight}
                        barColor="var(--accent-color)"
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
          </KPIScorecard>
        </div>
      ) : (
        <div style={{
          padding: '2rem',
          textAlign: 'center',
          border: '1px dashed var(--panel-border)',
          borderRadius: '12px',
          color: 'var(--text-secondary)',
        }}>
          <span style={{ fontSize: '1.75rem', display: 'block', marginBottom: '0.5rem' }}>📊</span>
          <p style={{ fontSize: '0.85rem', margin: 0 }}>Dados de performance insuficientes para calcular métricas de risco.</p>
          <p style={{ fontSize: '0.75rem', margin: '0.4rem 0 0 0', opacity: 0.7 }}>É necessário ao menos 10 dias de histórico.</p>
        </div>
      )}
    </AnalysisCard>
  );
}

