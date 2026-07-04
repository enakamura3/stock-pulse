'use client';

import React, { useMemo, useState, useEffect } from 'react';
import { Position, CalculatedDividend, FixedIncomePosition } from './types';
import { getAssetCategory } from './helpers';
import { formatMoney } from './helpers';

interface PortfolioInsightsProps {
  positions: Position[];
  dividends: CalculatedDividend[];
  fiPositions: FixedIncomePosition[];
  kpiCurrency: string;
}

const CONCENTRATION_THRESHOLD = 0.20; // 20%
const MONTHS_FOR_YIELD = 12;
const TOP_N = 5;

// ─── Mini reusable components ───────────────────────────────────────────────

function SectionTitle({ emoji, title, subtitle }: { emoji: string; title: string; subtitle?: string }) {
  return (
    <div className="mb-md">
      <h3 className="font-bold" style={{ fontSize: '1rem', color: 'var(--text-primary)' }}>
        {emoji} {title}
      </h3>
      {subtitle && <p className="text-secondary" style={{ fontSize: '0.78rem', marginTop: '0.2rem' }}>{subtitle}</p>}

      {/* ── 7. Valuation & Margem de Segurança ── */}
      <InsightCard>
        <SectionTitle emoji="⚖️" title="Valuation e Descontos" subtitle="Ativos com maior margem de segurança na carteira" />
        
        {valuationData.grahamItems.length === 0 && valuationData.bazinItems.length === 0 ? (
          <AlertBadge type="info" message="Não há dados suficientes de fundamentos para calcular margem de segurança." />
        ) : (
          <>
             {valuationData.grahamItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>Preço Teto - Graham</p>
                  {valuationData.grahamItems.slice(0, 3).map(item => (
                    <div key={`graham-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.graham!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}

             {valuationData.bazinItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: '0.5rem' }}>Preço Teto - Bazin</p>
                  {valuationData.bazinItems.slice(0, 3).map(item => (
                    <div key={`bazin-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.bazin!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}
          </>
        )}
      </InsightCard>

      {/* ── 8. Sazonalidade de Proventos ── */}
      <InsightCard>
        <SectionTitle emoji="🗓️" title="Sazonalidade de Proventos" subtitle="Mapa de calor do fluxo de caixa (últimos 12 meses)" />
        <div style={{ display: 'flex', alignItems: 'flex-end', height: '120px', gap: '4px', marginTop: '1rem', borderBottom: '1px solid rgba(255,255,255,0.1)', paddingBottom: '0.5rem' }}>
          {dividendSeasonality.map((item, i) => (
             <div key={i} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', gap: '4px' }}>
                <div 
                   title={`${item.month}: ${formatMoney(item.value, 'BRL')}`}
                   style={{ 
                     width: '100%', 
                     height: `${Math.max(item.pct, 2)}%`, 
                     background: item.isCurrent ? '#4ade80' : 'rgba(96,165,250,0.6)', 
                     borderRadius: '4px 4px 0 0',
                     transition: 'height 0.5s ease'
                   }} 
                />
             </div>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem', padding: '0 2px' }}>
           {dividendSeasonality.map((item, i) => (
              <span key={i} style={{ fontSize: '0.6rem', color: item.isCurrent ? '#4ade80' : 'var(--text-secondary)', fontWeight: item.isCurrent ? 700 : 400 }}>{item.month}</span>
           ))}
        </div>
        
        {upcomingDividends.length > 0 && (
           <div style={{ marginTop: '1.25rem', padding: '0.75rem', background: 'rgba(74,222,128,0.05)', borderRadius: '8px', border: '1px solid rgba(74,222,128,0.2)' }}>
              <p style={{ fontSize: '0.75rem', color: '#4ade80', fontWeight: 600, margin: 0, marginBottom: '0.2rem' }}>💰 Proventos a Receber</p>
              <p style={{ fontSize: '1.1rem', color: 'var(--text-primary)', fontWeight: 700, margin: 0, fontVariantNumeric: 'tabular-nums' }}>
                 {formatMoney(upcomingDividends.reduce((s, d) => s + d.net_amount, 0), 'BRL')}
              </p>
           </div>
        )}
      </InsightCard>

      {/* ── 9. Liquidez e Renda Fixa ── */}
      {fiPositions.length > 0 && (
        <InsightCard>
          <SectionTitle emoji="💧" title="Liquidez da Renda Fixa" subtitle="Perfil de vencimento dos seus ativos" />
          
          <div style={{ display: 'flex', height: '12px', borderRadius: '6px', overflow: 'hidden', marginBottom: '1rem' }}>
            {fiLiquidity.map((item, i) => (
              <div key={i} style={{ width: `${(item.value / fiLiquidity.reduce((s, x) => s + x.value, 0)) * 100}%`, background: item.color }} title={`${item.label}: ${formatMoney(item.value, kpiCurrency)}`} />
            ))}
          </div>
          
          <div>
            {fiLiquidity.map((item, i) => {
               const totalLiquidity = fiLiquidity.reduce((s, x) => s + x.value, 0);
               const pct = totalLiquidity > 0 ? (item.value / totalLiquidity) * 100 : 0;
               return (
                 <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                   <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                     <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: item.color }} />
                     <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{item.label}</span>
                   </div>
                   <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{pct.toFixed(1)}%</span>
                 </div>
               );
            })}
          </div>
        </InsightCard>
      )}
    </div>
  );
}

function ProgressBar({
  value, max, color = '#00f2fe', label, sublabel,
}: {
  value: number; max: number; color?: string; label?: string; sublabel?: string;
}) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  return (
    <div style={{ marginBottom: '0.75rem' }}>
      {(label || sublabel) && (
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.3rem' }}>
          <span style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)' }}>{label}</span>
          <span style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>{sublabel}</span>
        </div>
      )}
      <div style={{ background: 'rgba(255,255,255,0.06)', borderRadius: '6px', height: '8px', overflow: 'hidden' }}>
        <div style={{
          height: '100%',
          width: `${pct}%`,
          background: color,
          borderRadius: '6px',
          transition: 'width 0.6s cubic-bezier(0.4,0,0.2,1)',
          boxShadow: `0 0 6px ${color}55`,
        }} />
      </div>

      {/* ── 7. Valuation & Margem de Segurança ── */}
      <InsightCard>
        <SectionTitle emoji="⚖️" title="Valuation e Descontos" subtitle="Ativos com maior margem de segurança na carteira" />
        
        {valuationData.grahamItems.length === 0 && valuationData.bazinItems.length === 0 ? (
          <AlertBadge type="info" message="Não há dados suficientes de fundamentos para calcular margem de segurança." />
        ) : (
          <>
             {valuationData.grahamItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>Preço Teto - Graham</p>
                  {valuationData.grahamItems.slice(0, 3).map(item => (
                    <div key={`graham-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.graham!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}

             {valuationData.bazinItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: '0.5rem' }}>Preço Teto - Bazin</p>
                  {valuationData.bazinItems.slice(0, 3).map(item => (
                    <div key={`bazin-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.bazin!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}
          </>
        )}
      </InsightCard>

      {/* ── 8. Sazonalidade de Proventos ── */}
      <InsightCard>
        <SectionTitle emoji="🗓️" title="Sazonalidade de Proventos" subtitle="Mapa de calor do fluxo de caixa (últimos 12 meses)" />
        <div style={{ display: 'flex', alignItems: 'flex-end', height: '120px', gap: '4px', marginTop: '1rem', borderBottom: '1px solid rgba(255,255,255,0.1)', paddingBottom: '0.5rem' }}>
          {dividendSeasonality.map((item, i) => (
             <div key={i} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', gap: '4px' }}>
                <div 
                   title={`${item.month}: ${formatMoney(item.value, 'BRL')}`}
                   style={{ 
                     width: '100%', 
                     height: `${Math.max(item.pct, 2)}%`, 
                     background: item.isCurrent ? '#4ade80' : 'rgba(96,165,250,0.6)', 
                     borderRadius: '4px 4px 0 0',
                     transition: 'height 0.5s ease'
                   }} 
                />
             </div>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem', padding: '0 2px' }}>
           {dividendSeasonality.map((item, i) => (
              <span key={i} style={{ fontSize: '0.6rem', color: item.isCurrent ? '#4ade80' : 'var(--text-secondary)', fontWeight: item.isCurrent ? 700 : 400 }}>{item.month}</span>
           ))}
        </div>
        
        {upcomingDividends.length > 0 && (
           <div style={{ marginTop: '1.25rem', padding: '0.75rem', background: 'rgba(74,222,128,0.05)', borderRadius: '8px', border: '1px solid rgba(74,222,128,0.2)' }}>
              <p style={{ fontSize: '0.75rem', color: '#4ade80', fontWeight: 600, margin: 0, marginBottom: '0.2rem' }}>💰 Proventos a Receber</p>
              <p style={{ fontSize: '1.1rem', color: 'var(--text-primary)', fontWeight: 700, margin: 0, fontVariantNumeric: 'tabular-nums' }}>
                 {formatMoney(upcomingDividends.reduce((s, d) => s + d.net_amount, 0), 'BRL')}
              </p>
           </div>
        )}
      </InsightCard>

      {/* ── 9. Liquidez e Renda Fixa ── */}
      {fiPositions.length > 0 && (
        <InsightCard>
          <SectionTitle emoji="💧" title="Liquidez da Renda Fixa" subtitle="Perfil de vencimento dos seus ativos" />
          
          <div style={{ display: 'flex', height: '12px', borderRadius: '6px', overflow: 'hidden', marginBottom: '1rem' }}>
            {fiLiquidity.map((item, i) => (
              <div key={i} style={{ width: `${(item.value / fiLiquidity.reduce((s, x) => s + x.value, 0)) * 100}%`, background: item.color }} title={`${item.label}: ${formatMoney(item.value, kpiCurrency)}`} />
            ))}
          </div>
          
          <div>
            {fiLiquidity.map((item, i) => {
               const totalLiquidity = fiLiquidity.reduce((s, x) => s + x.value, 0);
               const pct = totalLiquidity > 0 ? (item.value / totalLiquidity) * 100 : 0;
               return (
                 <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                   <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                     <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: item.color }} />
                     <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{item.label}</span>
                   </div>
                   <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{pct.toFixed(1)}%</span>
                 </div>
               );
            })}
          </div>
        </InsightCard>
      )}
    </div>
  );
}

function InsightCard({ children, style }: { children: React.ReactNode; style?: React.CSSProperties }) {
  return (
    <div style={{
      background: 'linear-gradient(145deg, rgba(255,255,255,0.03) 0%, rgba(255,255,255,0.01) 100%)',
      border: '1px solid rgba(255,255,255,0.06)',
      borderRadius: '16px',
      padding: '1.5rem',
      boxShadow: '0 4px 24px rgba(0,0,0,0.25)',
      ...style,
    }}>
      {children}

      {/* ── 7. Valuation & Margem de Segurança ── */}
      <InsightCard>
        <SectionTitle emoji="⚖️" title="Valuation e Descontos" subtitle="Ativos com maior margem de segurança na carteira" />
        
        {valuationData.grahamItems.length === 0 && valuationData.bazinItems.length === 0 ? (
          <AlertBadge type="info" message="Não há dados suficientes de fundamentos para calcular margem de segurança." />
        ) : (
          <>
             {valuationData.grahamItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>Preço Teto - Graham</p>
                  {valuationData.grahamItems.slice(0, 3).map(item => (
                    <div key={`graham-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.graham!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}

             {valuationData.bazinItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: '0.5rem' }}>Preço Teto - Bazin</p>
                  {valuationData.bazinItems.slice(0, 3).map(item => (
                    <div key={`bazin-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.bazin!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}
          </>
        )}
      </InsightCard>

      {/* ── 8. Sazonalidade de Proventos ── */}
      <InsightCard>
        <SectionTitle emoji="🗓️" title="Sazonalidade de Proventos" subtitle="Mapa de calor do fluxo de caixa (últimos 12 meses)" />
        <div style={{ display: 'flex', alignItems: 'flex-end', height: '120px', gap: '4px', marginTop: '1rem', borderBottom: '1px solid rgba(255,255,255,0.1)', paddingBottom: '0.5rem' }}>
          {dividendSeasonality.map((item, i) => (
             <div key={i} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', gap: '4px' }}>
                <div 
                   title={`${item.month}: ${formatMoney(item.value, 'BRL')}`}
                   style={{ 
                     width: '100%', 
                     height: `${Math.max(item.pct, 2)}%`, 
                     background: item.isCurrent ? '#4ade80' : 'rgba(96,165,250,0.6)', 
                     borderRadius: '4px 4px 0 0',
                     transition: 'height 0.5s ease'
                   }} 
                />
             </div>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem', padding: '0 2px' }}>
           {dividendSeasonality.map((item, i) => (
              <span key={i} style={{ fontSize: '0.6rem', color: item.isCurrent ? '#4ade80' : 'var(--text-secondary)', fontWeight: item.isCurrent ? 700 : 400 }}>{item.month}</span>
           ))}
        </div>
        
        {upcomingDividends.length > 0 && (
           <div style={{ marginTop: '1.25rem', padding: '0.75rem', background: 'rgba(74,222,128,0.05)', borderRadius: '8px', border: '1px solid rgba(74,222,128,0.2)' }}>
              <p style={{ fontSize: '0.75rem', color: '#4ade80', fontWeight: 600, margin: 0, marginBottom: '0.2rem' }}>💰 Proventos a Receber</p>
              <p style={{ fontSize: '1.1rem', color: 'var(--text-primary)', fontWeight: 700, margin: 0, fontVariantNumeric: 'tabular-nums' }}>
                 {formatMoney(upcomingDividends.reduce((s, d) => s + d.net_amount, 0), 'BRL')}
              </p>
           </div>
        )}
      </InsightCard>

      {/* ── 9. Liquidez e Renda Fixa ── */}
      {fiPositions.length > 0 && (
        <InsightCard>
          <SectionTitle emoji="💧" title="Liquidez da Renda Fixa" subtitle="Perfil de vencimento dos seus ativos" />
          
          <div style={{ display: 'flex', height: '12px', borderRadius: '6px', overflow: 'hidden', marginBottom: '1rem' }}>
            {fiLiquidity.map((item, i) => (
              <div key={i} style={{ width: `${(item.value / fiLiquidity.reduce((s, x) => s + x.value, 0)) * 100}%`, background: item.color }} title={`${item.label}: ${formatMoney(item.value, kpiCurrency)}`} />
            ))}
          </div>
          
          <div>
            {fiLiquidity.map((item, i) => {
               const totalLiquidity = fiLiquidity.reduce((s, x) => s + x.value, 0);
               const pct = totalLiquidity > 0 ? (item.value / totalLiquidity) * 100 : 0;
               return (
                 <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                   <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                     <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: item.color }} />
                     <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{item.label}</span>
                   </div>
                   <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{pct.toFixed(1)}%</span>
                 </div>
               );
            })}
          </div>
        </InsightCard>
      )}
    </div>
  );
}

function AlertBadge({ type, message }: { type: 'warning' | 'info' | 'success'; message: string }) {
  const colors = {
    warning: { bg: 'rgba(251,191,36,0.1)', border: 'rgba(251,191,36,0.3)', text: '#fbbf24' },
    info:    { bg: 'rgba(96,165,250,0.1)',  border: 'rgba(96,165,250,0.3)',  text: '#60a5fa' },
    success: { bg: 'rgba(74,222,128,0.1)',  border: 'rgba(74,222,128,0.3)',  text: '#4ade80' },
  };
  const c = colors[type];
  return (
    <div style={{
      background: c.bg, border: `1px solid ${c.border}`, borderRadius: '10px',
      padding: '0.5rem 0.75rem', marginTop: '0.75rem', fontSize: '0.78rem',
      color: c.text, lineHeight: 1.4,
    }}>
      {message}

      {/* ── 7. Valuation & Margem de Segurança ── */}
      <InsightCard>
        <SectionTitle emoji="⚖️" title="Valuation e Descontos" subtitle="Ativos com maior margem de segurança na carteira" />
        
        {valuationData.grahamItems.length === 0 && valuationData.bazinItems.length === 0 ? (
          <AlertBadge type="info" message="Não há dados suficientes de fundamentos para calcular margem de segurança." />
        ) : (
          <>
             {valuationData.grahamItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>Preço Teto - Graham</p>
                  {valuationData.grahamItems.slice(0, 3).map(item => (
                    <div key={`graham-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.graham!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}

             {valuationData.bazinItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: '0.5rem' }}>Preço Teto - Bazin</p>
                  {valuationData.bazinItems.slice(0, 3).map(item => (
                    <div key={`bazin-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.bazin!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}
          </>
        )}
      </InsightCard>

      {/* ── 8. Sazonalidade de Proventos ── */}
      <InsightCard>
        <SectionTitle emoji="🗓️" title="Sazonalidade de Proventos" subtitle="Mapa de calor do fluxo de caixa (últimos 12 meses)" />
        <div style={{ display: 'flex', alignItems: 'flex-end', height: '120px', gap: '4px', marginTop: '1rem', borderBottom: '1px solid rgba(255,255,255,0.1)', paddingBottom: '0.5rem' }}>
          {dividendSeasonality.map((item, i) => (
             <div key={i} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', gap: '4px' }}>
                <div 
                   title={`${item.month}: ${formatMoney(item.value, 'BRL')}`}
                   style={{ 
                     width: '100%', 
                     height: `${Math.max(item.pct, 2)}%`, 
                     background: item.isCurrent ? '#4ade80' : 'rgba(96,165,250,0.6)', 
                     borderRadius: '4px 4px 0 0',
                     transition: 'height 0.5s ease'
                   }} 
                />
             </div>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem', padding: '0 2px' }}>
           {dividendSeasonality.map((item, i) => (
              <span key={i} style={{ fontSize: '0.6rem', color: item.isCurrent ? '#4ade80' : 'var(--text-secondary)', fontWeight: item.isCurrent ? 700 : 400 }}>{item.month}</span>
           ))}
        </div>
        
        {upcomingDividends.length > 0 && (
           <div style={{ marginTop: '1.25rem', padding: '0.75rem', background: 'rgba(74,222,128,0.05)', borderRadius: '8px', border: '1px solid rgba(74,222,128,0.2)' }}>
              <p style={{ fontSize: '0.75rem', color: '#4ade80', fontWeight: 600, margin: 0, marginBottom: '0.2rem' }}>💰 Proventos a Receber</p>
              <p style={{ fontSize: '1.1rem', color: 'var(--text-primary)', fontWeight: 700, margin: 0, fontVariantNumeric: 'tabular-nums' }}>
                 {formatMoney(upcomingDividends.reduce((s, d) => s + d.net_amount, 0), 'BRL')}
              </p>
           </div>
        )}
      </InsightCard>

      {/* ── 9. Liquidez e Renda Fixa ── */}
      {fiPositions.length > 0 && (
        <InsightCard>
          <SectionTitle emoji="💧" title="Liquidez da Renda Fixa" subtitle="Perfil de vencimento dos seus ativos" />
          
          <div style={{ display: 'flex', height: '12px', borderRadius: '6px', overflow: 'hidden', marginBottom: '1rem' }}>
            {fiLiquidity.map((item, i) => (
              <div key={i} style={{ width: `${(item.value / fiLiquidity.reduce((s, x) => s + x.value, 0)) * 100}%`, background: item.color }} title={`${item.label}: ${formatMoney(item.value, kpiCurrency)}`} />
            ))}
          </div>
          
          <div>
            {fiLiquidity.map((item, i) => {
               const totalLiquidity = fiLiquidity.reduce((s, x) => s + x.value, 0);
               const pct = totalLiquidity > 0 ? (item.value / totalLiquidity) * 100 : 0;
               return (
                 <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                   <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                     <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: item.color }} />
                     <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{item.label}</span>
                   </div>
                   <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{pct.toFixed(1)}%</span>
                 </div>
               );
            })}
          </div>
        </InsightCard>
      )}
    </div>
  );
}

// ─── Main Component ──────────────────────────────────────────────────────────

export default function PortfolioInsights({
  positions,
  dividends,
  fiPositions,
  kpiCurrency,
}: PortfolioInsightsProps) {
  const [monthlyGoal, setMonthlyGoal] = useState<number>(0);
  const [goalInput, setGoalInput] = useState<string>('');
  const [editingGoal, setEditingGoal] = useState<boolean>(false);

  // Load goal from localStorage
  useEffect(() => {
    const saved = localStorage.getItem('stockpulse_monthly_goal');
    if (saved) {
      const parsed = parseFloat(saved);
      if (!isNaN(parsed)) {
        setMonthlyGoal(parsed);
        setGoalInput(parsed.toLocaleString('pt-BR', { minimumFractionDigits: 2 }));
      }
    }
  }, []);

  const saveGoal = () => {
    const raw = goalInput.replace(/\./g, '').replace(',', '.');
    const parsed = parseFloat(raw);
    if (!isNaN(parsed) && parsed >= 0) {
      setMonthlyGoal(parsed);
      localStorage.setItem('stockpulse_monthly_goal', String(parsed));
    }
    setEditingGoal(false);
  };

  // ── Computed data ──────────────────────────────────────────────────────────

  const totalCurrentValue = useMemo(() => {
    const eq = positions.reduce((s, p) => s + (p.current_value || 0), 0);
    const fi = fiPositions.reduce((s, p) => s + p.net_value, 0);
    return eq + fi;
  }, [positions, fiPositions]);

  // 1. Concentration per ticker (equity only — FI aggregated as "Renda Fixa")
  const concentration = useMemo(() => {
    if (totalCurrentValue < 1e-6) return [];
    const items = positions.map(p => ({
      ticker: p.ticker,
      value: p.current_value || 0,
      pct: (p.current_value || 0) / totalCurrentValue,
      category: getAssetCategory(p.type),
    }));
    const fiTotal = fiPositions.reduce((s, p) => s + p.net_value, 0);
    if (fiTotal > 0) {
      items.push({ ticker: 'Renda Fixa', value: fiTotal, pct: fiTotal / totalCurrentValue, category: 'Renda Fixa' });
    }
    return items.sort((a, b) => b.pct - a.pct);
  }, [positions, fiPositions, totalCurrentValue]);

  // 2. Allocation by category
  const allocationByCategory = useMemo(() => {
    const map: Record<string, number> = {};
    positions.forEach(p => {
      const cat = getAssetCategory(p.type);
      map[cat] = (map[cat] || 0) + (p.current_value || 0);
    });
    const fiTotal = fiPositions.reduce((s, p) => s + p.net_value, 0);
    if (fiTotal > 0) map['Renda Fixa'] = (map['Renda Fixa'] || 0) + fiTotal;
    return Object.entries(map).sort((a, b) => b[1] - a[1]);
  }, [positions, fiPositions]);

  const categoryColors: Record<string, string> = {
    'Ações':      '#60a5fa',
    'FIIs':       '#c084fc',
    'ETFs':       '#34d399',
    'Renda Fixa': '#fbbf24',
    'BDRs':       '#f87171',
  };

  // 3. Dividend Yield & Sazonalidade
  const { dividendsLast12m, upcomingDividends } = useMemo(() => {
    const now = new Date();
    const twelveMonthsAgo = new Date(now);
    twelveMonthsAgo.setMonth(now.getMonth() - MONTHS_FOR_YIELD);

    const past = dividends.filter(d => {
      if (d.is_accrued) return false;
      const dateStr = (d.payment_date && !d.payment_date.startsWith('0001')) ? d.payment_date : d.ex_date;
      if (!dateStr) return false;
      return new Date(dateStr) >= twelveMonthsAgo;
    });

    const upcoming = dividends.filter(d => d.is_accrued);
    return { dividendsLast12m: past, upcomingDividends: upcoming };
  }, [dividends]);

  // NEW: Sazonalidade de Proventos
  const dividendSeasonality = useMemo(() => {
    const months = Array(12).fill(0);
    const now = new Date();
    const currentMonth = now.getMonth();

    dividendsLast12m.forEach(d => {
       const dateStr = (d.payment_date && !d.payment_date.startsWith('0001')) ? d.payment_date : d.ex_date;
       if (!dateStr) return;
       const date = new Date(dateStr);
       const m = date.getMonth();
       months[m] += d.net_amount;
    });
    
    const monthLabels = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];
    const maxVal = Math.max(...months, 1);
    
    return months.map((val, idx) => ({ month: monthLabels[idx], value: val, pct: (val/maxVal)*100, isCurrent: idx === currentMonth }));
  }, [dividendsLast12m]);

  const totalDivLast12m = dividendsLast12m.reduce((s, d) => s + d.net_amount, 0);
  const totalCost = positions.reduce((s, p) => s + p.total_cost, 0);
  const yieldOnCost = totalCost > 0 ? (totalDivLast12m / totalCost) * 100 : 0;
  const avgMonthly = totalDivLast12m / MONTHS_FOR_YIELD;

  // Yield per ticker (last 12m)
  const yieldPerTicker = useMemo(() => {
    const map: Record<string, number> = {};
    dividendsLast12m.forEach(d => { map[d.ticker] = (map[d.ticker] || 0) + d.net_amount; });
    return Object.entries(map)
      .map(([ticker, total]) => {
        const pos = positions.find(p => p.ticker === ticker);
        const cost = pos ? pos.total_cost : 0;
        return { ticker, total, yoc: cost > 0 ? (total / cost) * 100 : 0 };
      })
      .filter(x => x.yoc > 0)
      .sort((a, b) => b.yoc - a.yoc);
  }, [dividendsLast12m, positions]);

  // 4. Top performers & worst
  const rankedPositions = useMemo(() =>
    [...positions]
      .filter(p => p.return_percent !== undefined)
      .sort((a, b) => (b.return_percent || 0) - (a.return_percent || 0)),
    [positions]
  );
  const topPerformers = rankedPositions.slice(0, TOP_N);
  const worstPerformers = [...rankedPositions]
    .reverse()
    .filter(p => !topPerformers.find(t => t.ticker === p.ticker))
    .slice(0, TOP_N);

  // NEW: Valuation Data (Graham / Bazin)
  const valuationData = useMemo(() => {
    const withGraham = positions.filter(p => p.graham_value && p.graham_value > 0 && p.current_price && p.current_price > 0);
    const withBazin = positions.filter(p => p.bazin_value && p.bazin_value > 0 && p.current_price && p.current_price > 0);
    
    const grahamItems = withGraham.map(p => {
        const discount = ((p.graham_value! - p.current_price!) / p.graham_value!) * 100;
        return { ticker: p.ticker, discount, graham: p.graham_value, current: p.current_price };
    }).sort((a, b) => b.discount - a.discount);

    const bazinItems = withBazin.map(p => {
        const discount = ((p.bazin_value! - p.current_price!) / p.bazin_value!) * 100;
        return { ticker: p.ticker, discount, bazin: p.bazin_value, current: p.current_price };
    }).sort((a, b) => b.discount - a.discount);

    return { grahamItems, bazinItems };
  }, [positions]);

  // NEW: Renda Fixa Liquidez
  const fiLiquidity = useMemo(() => {
    let daily = 0;
    let upTo1Year = 0;
    let upTo3Years = 0;
    let longTerm = 0;

    fiPositions.forEach(p => {
      if (p.days_to_maturity <= 0) {
        daily += p.net_value;
      } else if (p.days_to_maturity <= 365) {
        upTo1Year += p.net_value;
      } else if (p.days_to_maturity <= 1095) {
        upTo3Years += p.net_value;
      } else {
        longTerm += p.net_value;
      }
    });
    return [
      { label: 'Liquidez Diária / Vencido', value: daily, color: '#4ade80' },
      { label: 'Até 1 ano', value: upTo1Year, color: '#60a5fa' },
      { label: '1 a 3 anos', value: upTo3Years, color: '#fbbf24' },
      { label: 'Longo Prazo (> 3 anos)', value: longTerm, color: '#f87171' },
    ].filter(i => i.value > 0);
  }, [fiPositions]);

  // 6. Currency exposure
  const brlValue = useMemo(() => {
    const eq = positions.filter(p => p.currency === 'BRL').reduce((s, p) => s + (p.current_value || 0), 0);
    const fi = fiPositions.reduce((s, p) => s + p.net_value, 0);
    return eq + fi;
  }, [positions, fiPositions]);
  const usdValue = useMemo(() =>
    positions.filter(p => p.currency === 'USD').reduce((s, p) => s + (p.current_value || 0), 0),
    [positions]
  );

  if (positions.length === 0 && fiPositions.length === 0) {
    return (
      <div className="text-center text-secondary p-xl">
        <span className="text-2xl block mb-sm">🧩</span>
        <p>Adicione ativos à carteira para ver os insights.</p>
      </div>
    );
  }

  // Max pct for bar scaling in performers
  const maxAbsReturn = Math.max(...[...topPerformers, ...worstPerformers].map(p => Math.abs(p.return_percent || 0)), 1);

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(min(480px, 100%), 1fr))', gap: '1.25rem' }}>
      {/* ── 1. Concentration ── */}
      <InsightCard>
        <SectionTitle emoji="📊" title="Concentração da Carteira" subtitle="Participação percentual de cada ativo no valor total" />
        {concentration.slice(0, 8).map(item => {
          const isConcentrated = item.pct >= CONCENTRATION_THRESHOLD;
          const color = isConcentrated ? '#fbbf24' : categoryColors[item.category] || '#60a5fa';
          return (
            <ProgressBar
              key={item.ticker}
              value={item.pct * 100}
              max={100}
              color={color}
              label={item.ticker}
              sublabel={`${(item.pct * 100).toFixed(1)}% · ${formatMoney(item.value, kpiCurrency)}`}
            />
          );
        })}
        {concentration.filter(i => i.pct >= CONCENTRATION_THRESHOLD).length > 0 && (
          <AlertBadge
            type="warning"
            message={`⚠️ ${concentration.filter(i => i.pct >= CONCENTRATION_THRESHOLD).map(i => i.ticker).join(', ')} ${concentration.filter(i => i.pct >= CONCENTRATION_THRESHOLD).length > 1 ? 'superam' : 'supera'} 20% da carteira. Considere diversificar.`}
          />
        )}
        {concentration.filter(i => i.pct < CONCENTRATION_THRESHOLD).length === concentration.length && concentration.length > 0 && (
          <AlertBadge type="success" message="✅ Nenhum ativo concentra mais de 20%. Boa diversificação!" />
        )}
      </InsightCard>

      {/* ── 2. Allocation by Category ── */}
      <InsightCard>
        <SectionTitle emoji="🗂️" title="Alocação por Categoria" subtitle="Distribuição do patrimônio entre classes de ativos" />
        {allocationByCategory.map(([cat, val]) => {
          const pct = totalCurrentValue > 0 ? (val / totalCurrentValue) * 100 : 0;
          const color = categoryColors[cat] || '#94a3b8';
          return (
            <ProgressBar
              key={cat}
              value={pct}
              max={100}
              color={color}
              label={cat}
              sublabel={`${pct.toFixed(1)}% · ${formatMoney(val, kpiCurrency)}`}
            />
          );
        })}
        <div style={{ marginTop: '1rem', padding: '0.75rem', background: 'rgba(255,255,255,0.03)', borderRadius: '10px' }}>
          <p style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', margin: 0 }}>
            Patrimônio total: <strong style={{ color: 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(totalCurrentValue, kpiCurrency)}</strong>
          </p>
        </div>
      </InsightCard>

      {/* ── 3. Dividend Yield ── */}
      <InsightCard>
        <SectionTitle emoji="💸" title="Yield da Carteira (12 meses)" subtitle="Renda gerada pelos seus ativos de renda variável" />
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem', marginBottom: '1rem' }}>
          <div style={{ background: 'rgba(0,230,118,0.06)', borderRadius: '12px', padding: '0.75rem', border: '1px solid rgba(0,230,118,0.15)' }}>
            <div style={{ fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.25rem' }}>Total 12m</div>
            <div style={{ fontSize: '1.3rem', fontWeight: 700, color: '#4ade80', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(totalDivLast12m, 'BRL')}</div>
          </div>
          <div style={{ background: 'rgba(0,242,254,0.06)', borderRadius: '12px', padding: '0.75rem', border: '1px solid rgba(0,242,254,0.15)' }}>
            <div style={{ fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.25rem' }}>Média Mensal</div>
            <div style={{ fontSize: '1.3rem', fontWeight: 700, color: '#00f2fe', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(avgMonthly, 'BRL')}</div>
          </div>
        </div>
        {totalCost > 0 && (
          <div style={{ marginBottom: '1rem', padding: '0.6rem 0.75rem', background: 'rgba(255,255,255,0.03)', borderRadius: '10px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>Yield on Cost (YOC)</span>
            <span style={{ fontSize: '1rem', fontWeight: 700, color: yieldOnCost >= 6 ? '#4ade80' : 'var(--text-primary)', fontVariantNumeric: 'tabular-nums' }}>{yieldOnCost.toFixed(2)}% a.a.</span>
          </div>
        )}
        {yieldPerTicker.length > 0 && (
          <>
            <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>Maiores Pagadores (YOC)</p>
            {yieldPerTicker.slice(0, 4).map(item => (
              <ProgressBar
                key={item.ticker}
                value={item.yoc}
                max={Math.max(...yieldPerTicker.map(x => x.yoc), 1)}
                color="#4ade80"
                label={item.ticker}
                sublabel={`${item.yoc.toFixed(2)}% a.a. · ${formatMoney(item.total, 'BRL')}`}
              />
            ))}
          </>
        )}
        {dividendsLast12m.length === 0 && (
          <AlertBadge type="info" message="Nenhum provento recebido nos últimos 12 meses." />
        )}
      </InsightCard>

      {/* ── 4. Top & Worst Performers ── */}
      <InsightCard>
        <SectionTitle emoji="🏆" title="Top Performers vs Piores" subtitle="Rentabilidade acumulada por ativo na sua carteira" />
        {topPerformers.length === 0 ? (
          <AlertBadge type="info" message="Sem dados de rentabilidade disponíveis." />
        ) : (
          <>
            <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: '#4ade80', marginBottom: '0.5rem' }}>Top {Math.min(TOP_N, topPerformers.length)} melhores</p>
            {topPerformers.map(p => (
              <div key={p.ticker} style={{ display: 'flex', alignItems: 'center', marginBottom: '0.5rem', gap: '0.5rem' }}>
                <span style={{ width: '52px', fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-primary)', flexShrink: 0 }}>{p.ticker}</span>
                <div style={{ flex: 1, background: 'rgba(255,255,255,0.05)', borderRadius: '4px', height: '10px', overflow: 'hidden' }}>
                  <div style={{
                    height: '100%',
                    width: `${((p.return_percent || 0) / maxAbsReturn) * 100}%`,
                    background: 'linear-gradient(90deg, #4ade80, #00e676)',
                    borderRadius: '4px',
                    transition: 'width 0.6s ease',
                  }} />
                </div>
                <span style={{ width: '60px', textAlign: 'right', fontSize: '0.8rem', fontWeight: 700, color: '#4ade80', fontVariantNumeric: 'tabular-nums', flexShrink: 0 }}>
                  +{(p.return_percent || 0).toFixed(1)}%
                </span>
              </div>
            ))}

            <div style={{ borderTop: '1px solid rgba(255,255,255,0.06)', margin: '0.75rem 0' }} />

            <p style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: '#f87171', marginBottom: '0.5rem' }}>Top {Math.min(TOP_N, worstPerformers.length)} piores</p>
            {worstPerformers.map(p => (
              <div key={p.ticker} style={{ display: 'flex', alignItems: 'center', marginBottom: '0.5rem', gap: '0.5rem' }}>
                <span style={{ width: '52px', fontSize: '0.8rem', fontWeight: 700, color: 'var(--text-primary)', flexShrink: 0 }}>{p.ticker}</span>
                <div style={{ flex: 1, background: 'rgba(255,255,255,0.05)', borderRadius: '4px', height: '10px', overflow: 'hidden' }}>
                  <div style={{
                    height: '100%',
                    width: `${(Math.abs(p.return_percent || 0) / maxAbsReturn) * 100}%`,
                    background: 'linear-gradient(90deg, #f87171, #ef4444)',
                    borderRadius: '4px',
                    transition: 'width 0.6s ease',
                  }} />
                </div>
                <span style={{ width: '60px', textAlign: 'right', fontSize: '0.8rem', fontWeight: 700, color: '#f87171', fontVariantNumeric: 'tabular-nums', flexShrink: 0 }}>
                  {(p.return_percent || 0).toFixed(1)}%
                </span>
              </div>
            ))}
          </>
        )}
      </InsightCard>

      {/* ── 5. Monthly Income Goal ── */}
      <InsightCard>
        <SectionTitle emoji="🎯" title="Cobertura de Renda Passiva" subtitle="Meta mensal vs média real dos últimos 12 meses" />
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1rem' }}>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.3rem' }}>Meta Mensal</div>
            {editingGoal ? (
              <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                <input
                  type="text"
                  value={goalInput}
                  onChange={e => setGoalInput(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && saveGoal()}
                  placeholder="Ex: 1.000,00"
                  autoFocus
                  style={{
                    background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.15)',
                    borderRadius: '8px', padding: '0.4rem 0.6rem', color: 'var(--text-primary)',
                    fontSize: '0.9rem', width: '130px', outline: 'none', fontVariantNumeric: 'tabular-nums',
                  }}
                />
                <button onClick={saveGoal} style={{ background: 'rgba(74,222,128,0.15)', border: '1px solid rgba(74,222,128,0.3)', borderRadius: '8px', padding: '0.4rem 0.75rem', color: '#4ade80', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 700 }}>Salvar</button>
              </div>
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                <span style={{ fontSize: '1.3rem', fontWeight: 700, color: monthlyGoal > 0 ? 'var(--text-primary)' : 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                  {monthlyGoal > 0 ? formatMoney(monthlyGoal, 'BRL') : 'Não definida'}
                </span>
                <button onClick={() => setEditingGoal(true)} style={{ background: 'transparent', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '6px', padding: '0.2rem 0.5rem', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: '0.72rem' }}>
                  ✏️ {monthlyGoal > 0 ? 'Editar' : 'Definir'}
                </button>
              </div>
            )}
          </div>
          <div style={{ textAlign: 'right' }}>
            <div style={{ fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.3rem' }}>Média Real (12m)</div>
            <div style={{ fontSize: '1.3rem', fontWeight: 700, color: '#4ade80', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(avgMonthly, 'BRL')}</div>
          </div>
        </div>

        {monthlyGoal > 0 && (
          <>
            {(() => {
              const coverage = Math.min((avgMonthly / monthlyGoal) * 100, 100);
              const barColor = coverage >= 100 ? '#4ade80' : coverage >= 60 ? '#fbbf24' : '#f87171';
              return (
                <>
                  <ProgressBar
                    value={avgMonthly}
                    max={monthlyGoal}
                    color={barColor}
                    sublabel={`${coverage.toFixed(1)}% da meta atingida`}
                  />
                  {coverage >= 100 ? (
                    <AlertBadge type="success" message="🎉 Parabéns! Sua renda passiva já cobre 100% da sua meta mensal!" />
                  ) : (
                    <AlertBadge type="info" message={`Faltam ${formatMoney(monthlyGoal - avgMonthly, 'BRL')}/mês para atingir sua meta. Continue investindo!`} />
                  )}
                </>
              );
            })()}
          </>
        )}

        {monthlyGoal === 0 && (
          <AlertBadge type="info" message="Defina uma meta mensal de renda passiva para acompanhar seu progresso." />
        )}
      </InsightCard>

      {/* ── 6. Currency Exposure ── */}
      <InsightCard>
        <SectionTitle emoji="🌍" title="Exposição Cambial" subtitle="Distribuição do patrimônio por moeda" />
        {(brlValue + usdValue) < 1e-6 ? (
          <AlertBadge type="info" message="Sem dados de exposição cambial." />
        ) : (
          <>
            <ProgressBar
              value={brlValue}
              max={brlValue + usdValue}
              color="#4ade80"
              label="🇧🇷 BRL — Real Brasileiro"
              sublabel={`${((brlValue / (brlValue + usdValue)) * 100).toFixed(1)}% · ${formatMoney(brlValue, 'BRL')}`}
            />
            {usdValue > 0 && (
              <ProgressBar
                value={usdValue}
                max={brlValue + usdValue}
                color="#60a5fa"
                label="🇺🇸 USD — Dólar Americano"
                sublabel={`${((usdValue / (brlValue + usdValue)) * 100).toFixed(1)}% · ${formatMoney(usdValue, 'USD')}`}
              />
            )}
            {usdValue === 0 && (
              <AlertBadge type="info" message="Toda a carteira está em BRL. Considere adicionar ativos em USD para diversificação cambial." />
            )}
            {usdValue > 0 && usdValue / (brlValue + usdValue) >= 0.40 && (
              <AlertBadge type="warning" message={`⚠️ Mais de 40% da carteira está em USD. Sua renda em BRL pode ser impactada por oscilações do câmbio.`} />
            )}
          </>
        )}
      </InsightCard>

      {/* ── 7. Valuation & Margem de Segurança ── */}
      <InsightCard>
        <SectionTitle emoji="⚖️" title="Valuation e Descontos" subtitle="Ativos com maior margem de segurança na carteira" />
        
        {valuationData.grahamItems.length === 0 && valuationData.bazinItems.length === 0 ? (
          <AlertBadge type="info" message="Não há dados suficientes de fundamentos para calcular margem de segurança." />
        ) : (
          <>
             {valuationData.grahamItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>Preço Teto - Graham</p>
                  {valuationData.grahamItems.slice(0, 3).map(item => (
                    <div key={`graham-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.graham!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}

             {valuationData.bazinItems.length > 0 && (
                <>
                  <p style={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: '0.5rem' }}>Preço Teto - Bazin</p>
                  {valuationData.bazinItems.slice(0, 3).map(item => (
                    <div key={`bazin-${item.ticker}`} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem', background: 'rgba(255,255,255,0.02)', padding: '0.5rem', borderRadius: '8px' }}>
                       <div style={{ flex: 1 }}>
                         <div style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{item.ticker}</div>
                         <div style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço: {formatMoney(item.current!, kpiCurrency)} · Teto: {formatMoney(item.bazin!, kpiCurrency)}</div>
                       </div>
                       <div style={{ padding: '0.2rem 0.5rem', borderRadius: '6px', background: item.discount > 0 ? 'rgba(74,222,128,0.1)' : 'rgba(248,113,113,0.1)', color: item.discount > 0 ? '#4ade80' : '#f87171', fontSize: '0.75rem', fontWeight: 700 }}>
                          {item.discount > 0 ? '-' : '+'}{Math.abs(item.discount).toFixed(1)}%
                       </div>
                    </div>
                  ))}
                </>
             )}
          </>
        )}
      </InsightCard>

      {/* ── 8. Sazonalidade de Proventos ── */}
      <InsightCard>
        <SectionTitle emoji="🗓️" title="Sazonalidade de Proventos" subtitle="Mapa de calor do fluxo de caixa (últimos 12 meses)" />
        <div style={{ display: 'flex', alignItems: 'flex-end', height: '120px', gap: '4px', marginTop: '1rem', borderBottom: '1px solid rgba(255,255,255,0.1)', paddingBottom: '0.5rem' }}>
          {dividendSeasonality.map((item, i) => (
             <div key={i} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', gap: '4px' }}>
                <div 
                   title={`${item.month}: ${formatMoney(item.value, 'BRL')}`}
                   style={{ 
                     width: '100%', 
                     height: `${Math.max(item.pct, 2)}%`, 
                     background: item.isCurrent ? '#4ade80' : 'rgba(96,165,250,0.6)', 
                     borderRadius: '4px 4px 0 0',
                     transition: 'height 0.5s ease'
                   }} 
                />
             </div>
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem', padding: '0 2px' }}>
           {dividendSeasonality.map((item, i) => (
              <span key={i} style={{ fontSize: '0.6rem', color: item.isCurrent ? '#4ade80' : 'var(--text-secondary)', fontWeight: item.isCurrent ? 700 : 400 }}>{item.month}</span>
           ))}
        </div>
        
        {upcomingDividends.length > 0 && (
           <div style={{ marginTop: '1.25rem', padding: '0.75rem', background: 'rgba(74,222,128,0.05)', borderRadius: '8px', border: '1px solid rgba(74,222,128,0.2)' }}>
              <p style={{ fontSize: '0.75rem', color: '#4ade80', fontWeight: 600, margin: 0, marginBottom: '0.2rem' }}>💰 Proventos a Receber</p>
              <p style={{ fontSize: '1.1rem', color: 'var(--text-primary)', fontWeight: 700, margin: 0, fontVariantNumeric: 'tabular-nums' }}>
                 {formatMoney(upcomingDividends.reduce((s, d) => s + d.net_amount, 0), 'BRL')}
              </p>
           </div>
        )}
      </InsightCard>

      {/* ── 9. Liquidez e Renda Fixa ── */}
      {fiPositions.length > 0 && (
        <InsightCard>
          <SectionTitle emoji="💧" title="Liquidez da Renda Fixa" subtitle="Perfil de vencimento dos seus ativos" />
          
          <div style={{ display: 'flex', height: '12px', borderRadius: '6px', overflow: 'hidden', marginBottom: '1rem' }}>
            {fiLiquidity.map((item, i) => (
              <div key={i} style={{ width: `${(item.value / fiLiquidity.reduce((s, x) => s + x.value, 0)) * 100}%`, background: item.color }} title={`${item.label}: ${formatMoney(item.value, kpiCurrency)}`} />
            ))}
          </div>
          
          <div>
            {fiLiquidity.map((item, i) => {
               const totalLiquidity = fiLiquidity.reduce((s, x) => s + x.value, 0);
               const pct = totalLiquidity > 0 ? (item.value / totalLiquidity) * 100 : 0;
               return (
                 <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                   <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                     <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: item.color }} />
                     <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{item.label}</span>
                   </div>
                   <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>{pct.toFixed(1)}%</span>
                 </div>
               );
            })}
          </div>
        </InsightCard>
      )}
    </div>
  );
}
