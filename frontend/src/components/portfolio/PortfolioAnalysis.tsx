'use client';

import React from 'react';
import { Position, CalculatedDividend, FixedIncomePosition, PerformancePoint, TreasuryPosition } from './types';
import StrategicAllocationSection from './analysis/StrategicAllocationSection';
import RiskConcentrationSection from './analysis/RiskConcentrationSection';
import PerformanceBenchmarkSection from './analysis/PerformanceBenchmarkSection';
import PassiveIncomeSection from './analysis/PassiveIncomeSection';
import FundamentalHealthSection from './analysis/FundamentalHealthSection';
import TaxEfficiencySection from './analysis/TaxEfficiencySection';

interface PortfolioAnalysisProps {
  positions: Position[];
  dividends: CalculatedDividend[];
  fiPositions: FixedIncomePosition[];
  treasuryPositions?: TreasuryPosition[];
  performanceData: PerformancePoint[];
  kpiCurrency: string;
}

export default function PortfolioAnalysis({
  positions,
  dividends,
  fiPositions,
  treasuryPositions = [],
  performanceData,
  kpiCurrency,
}: PortfolioAnalysisProps) {
  if (positions.length === 0 && fiPositions.length === 0 && treasuryPositions.length === 0) {
    return (
      <div className="text-center text-secondary" style={{ padding: '3rem' }}>
        <span className="text-2xl" style={{ display: 'block', marginBottom: '0.5rem' }}>📊</span>
        <p>Adicione ativos à carteira para visualizar a análise completa.</p>
      </div>
    );
  }

  return (
    <div className="flex-col gap-xl">
      {/* 🎯 SEÇÃO 1: Alocação Estratégica */}
      <StrategicAllocationSection
        positions={positions}
        fiPositions={fiPositions}
        treasuryPositions={treasuryPositions}
        kpiCurrency={kpiCurrency}
      />

      {/* 📈 SEÇÃO 2: Comparação com Benchmarks */}
      <PerformanceBenchmarkSection
        performanceData={performanceData}
      />

      {/* 🌡️ SEÇÃO 3: Termômetro de Risco */}
      <RiskConcentrationSection
        positions={positions}
        fiPositions={fiPositions}
        treasuryPositions={treasuryPositions}
        performanceData={performanceData}
        kpiCurrency={kpiCurrency}
      />

      {/* 💰 SEÇÃO 4: Geração de Renda (Proventos) */}
      <PassiveIncomeSection
        positions={positions}
        dividends={dividends}
        kpiCurrency={kpiCurrency}
      />

      {/* 🏛️ SEÇÃO 5: Fundamentos, Valuation e Liquidez */}
      <FundamentalHealthSection
        positions={positions}
        fiPositions={fiPositions}
        treasuryPositions={treasuryPositions}
        kpiCurrency={kpiCurrency}
      />

      {/* 💸 SEÇÃO 6: Eficiência Tributária */}
      <TaxEfficiencySection
        positions={positions}
        dividends={dividends}
        fiPositions={fiPositions}
        treasuryPositions={treasuryPositions}
        kpiCurrency={kpiCurrency}
      />
    </div>
  );
}

