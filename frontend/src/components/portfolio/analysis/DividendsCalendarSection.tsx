import React, { useState, useMemo } from 'react';
import { CalculatedDividend } from '../types';
import { formatMoney } from '../helpers';
import { SectionTitle, AnalysisCard, StatPill, AssetRiskDetailRow } from './sharedComponents';

interface DividendsCalendarSectionProps {
  dividends: CalculatedDividend[];
  kpiCurrency: string;
}

const WEEKDAYS = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'];
const MONTH_NAMES = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro'
];

export default function DividendsCalendarSection({
  dividends,
  kpiCurrency,
}: DividendsCalendarSectionProps) {
  const [currentDate, setCurrentDate] = useState(() => new Date());
  const [selectedDay, setSelectedDay] = useState<number | null>(null);

  const year = currentDate.getFullYear();
  const month = currentDate.getMonth();

  const handlePrevMonth = () => {
    setCurrentDate(prev => new Date(prev.getFullYear(), prev.getMonth() - 1, 1));
    setSelectedDay(null);
  };

  const handleNextMonth = () => {
    setCurrentDate(prev => new Date(prev.getFullYear(), prev.getMonth() + 1, 1));
    setSelectedDay(null);
  };

  const handleTodayMonth = () => {
    setCurrentDate(new Date());
    setSelectedDay(null);
  };

  // Helper para extrair data válida (payment_date ou cum_date)
  const getDividendEventDate = (d: CalculatedDividend): Date | null => {
    let dateStr = d.payment_date;
    if (!dateStr || dateStr.startsWith('0001')) {
      dateStr = d.cum_date;
    }
    if (!dateStr || dateStr.startsWith('0001')) {
      return null;
    }
    const [y, m, dayStr] = dateStr.split('T')[0].split('-').map(Number);
    if (!y || !m || !dayStr) return null;
    return new Date(y, m - 1, dayStr);
  };

  // Helper para verificar se a data é no passado (Recebido)
  const isPaidEvent = (d: CalculatedDividend): boolean => {
    const eventDate = getDividendEventDate(d);
    if (!eventDate) return false;
    const today = new Date();
    today.setHours(23, 59, 59, 999);
    return eventDate <= today;
  };

  // Agrupamento diário de proventos no mês visível
  const monthlyDividends = useMemo(() => {
    const dailyMap: Record<number, CalculatedDividend[]> = {};
    let monthTotalReceived = 0;
    let monthTotalUpcoming = 0;
    let eventCount = 0;

    dividends.forEach(d => {
      const eventDate = getDividendEventDate(d);
      if (!eventDate) return;

      if (eventDate.getFullYear() === year && eventDate.getMonth() === month) {
        const day = eventDate.getDate();
        if (!dailyMap[day]) {
          dailyMap[day] = [];
        }
        dailyMap[day].push(d);
        eventCount++;

        if (isPaidEvent(d)) {
          monthTotalReceived += d.net_amount;
        } else {
          monthTotalUpcoming += d.net_amount;
        }
      }
    });

    return {
      dailyMap,
      monthTotalReceived,
      monthTotalUpcoming,
      monthTotalSum: monthTotalReceived + monthTotalUpcoming,
      eventCount,
    };
  }, [dividends, year, month]);

  // Construção dos dias do calendário (grid 7 colunas)
  const calendarCells = useMemo(() => {
    const firstDayOfWeek = new Date(year, month, 1).getDay();
    const daysInMonth = new Date(year, month + 1, 0).getDate();

    const paddingCells: null[] = Array(firstDayOfWeek).fill(null);
    const dayCells: number[] = Array.from({ length: daysInMonth }, (_, i) => i + 1);

    return [...paddingCells, ...dayCells];
  }, [year, month]);

  const selectedDayEvents = selectedDay && monthlyDividends.dailyMap[selectedDay]
    ? monthlyDividends.dailyMap[selectedDay]
    : [];

  const currency = kpiCurrency || 'BRL';

  return (
    <AnalysisCard id="section-dividends-calendar">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.75rem', marginBottom: '1rem' }}>
        <SectionTitle
          emoji="📅"
          title="Calendário de Proventos Diário"
          subtitle="Visão detalhada dia a dia de dividendos, JCP e rendimentos recebidos ou a receber"
        />

        {/* Botões de Navegação do Mês */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', background: 'rgba(255,255,255,0.03)', padding: '0.25rem 0.5rem', borderRadius: '10px', border: '1px solid rgba(255,255,255,0.08)' }}>
          <button
            onClick={handlePrevMonth}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--text-primary)',
              cursor: 'pointer',
              padding: '0.2rem 0.5rem',
              fontSize: '0.85rem',
              fontWeight: 700,
            }}
            title="Mês anterior"
          >
            ◀
          </button>
          <span style={{ fontSize: '0.82rem', fontWeight: 700, color: 'var(--text-primary)', minWidth: '110px', textAlign: 'center' }}>
            {MONTH_NAMES[month]} {year}
          </span>
          <button
            onClick={handleNextMonth}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--text-primary)',
              cursor: 'pointer',
              padding: '0.2rem 0.5rem',
              fontSize: '0.85rem',
              fontWeight: 700,
            }}
            title="Próximo mês"
          >
            ▶
          </button>
          <button
            onClick={handleTodayMonth}
            style={{
              background: 'rgba(0,242,254,0.1)',
              border: '1px solid rgba(0,242,254,0.3)',
              borderRadius: '6px',
              color: '#00f2fe',
              cursor: 'pointer',
              padding: '0.2rem 0.5rem',
              fontSize: '0.72rem',
              fontWeight: 600,
              marginLeft: '0.25rem',
            }}
          >
            Hoje
          </button>
        </div>
      </div>

      {/* KPI StatPills do Mês Visível */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '0.75rem', marginBottom: '1.25rem' }}>
        <StatPill
          label="Total do Mês"
          value={formatMoney(monthlyDividends.monthTotalSum, currency)}
          color="#00f2fe"
        />
        <StatPill
          label="Já Recebido"
          value={formatMoney(monthlyDividends.monthTotalReceived, currency)}
          color="#4ade80"
        />
        <StatPill
          label="A Receber"
          value={formatMoney(monthlyDividends.monthTotalUpcoming, currency)}
          color="#fbbf24"
        />
      </div>

      {/* Grid do Calendário (7 colunas) */}
      <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '12px', padding: '1rem', marginBottom: '1rem' }}>
        {/* Cabeçalho dos dias da semana */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: '6px', textAlign: 'center', marginBottom: '0.5rem' }}>
          {WEEKDAYS.map(wd => (
            <span key={wd} style={{ fontSize: '0.7rem', fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)' }}>
              {wd}
            </span>
          ))}
        </div>

        {/* Células dos dias */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: '6px' }}>
          {calendarCells.map((dayNum, idx) => {
            if (dayNum === null) {
              return <div key={`pad-${idx}`} style={{ minHeight: '52px' }} />;
            }

            const events = monthlyDividends.dailyMap[dayNum] || [];
            const hasEvents = events.length > 0;
            const isSelected = selectedDay === dayNum;
            const isToday =
              dayNum === new Date().getDate() &&
              month === new Date().getMonth() &&
              year === new Date().getFullYear();

            const hasPaid = events.some(isPaidEvent);
            const hasUpcoming = events.some(e => !isPaidEvent(e));
            const daySum = events.reduce((s, e) => s + e.net_amount, 0);

            const badgeColor = hasPaid && hasUpcoming ? '#fbbf24' : hasPaid ? '#4ade80' : '#fbbf24';

            return (
              <div
                key={`day-${dayNum}`}
                onClick={() => hasEvents && setSelectedDay(isSelected ? null : dayNum)}
                style={{
                  minHeight: '56px',
                  background: isSelected
                    ? 'rgba(0,242,254,0.15)'
                    : isToday
                      ? 'rgba(255,255,255,0.06)'
                      : hasEvents
                        ? 'rgba(255,255,255,0.03)'
                        : 'rgba(255,255,255,0.01)',
                  border: isSelected
                    ? '1px solid #00f2fe'
                    : isToday
                      ? '1px solid rgba(255,255,255,0.2)'
                      : hasEvents
                        ? `1px solid ${badgeColor}30`
                        : '1px solid rgba(255,255,255,0.03)',
                  borderRadius: '8px',
                  padding: '0.35rem 0.4rem',
                  display: 'flex',
                  flexDirection: 'column',
                  justifyContent: 'space-between',
                  cursor: hasEvents ? 'pointer' : 'default',
                  transition: 'all 0.2s ease',
                  position: 'relative',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span style={{
                    fontSize: '0.78rem',
                    fontWeight: isToday || isSelected ? 800 : 600,
                    color: isToday ? '#00f2fe' : 'var(--text-primary)',
                  }}>
                    {dayNum}
                  </span>
                  {hasEvents && (
                    <span style={{
                      width: '6px',
                      height: '6px',
                      borderRadius: '50%',
                      background: badgeColor,
                      boxShadow: `0 0 6px ${badgeColor}`,
                    }} />
                  )}
                </div>

                {hasEvents && (
                  <div style={{ textAlign: 'right', marginTop: '0.2rem' }}>
                    <div style={{ fontSize: '0.65rem', fontWeight: 700, color: badgeColor, fontVariantNumeric: 'tabular-nums' }}>
                      {daySum >= 1000 ? `${(daySum / 1000).toFixed(1).replace('.0', '')}k` : `R$${Math.round(daySum)}`}
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Painel de Detalhes do Dia Selecionado */}
      {selectedDayEvents.length > 0 && (
        <div style={{ padding: '1rem', background: 'rgba(0,242,254,0.04)', borderRadius: '12px', border: '1px solid rgba(0,242,254,0.15)', marginTop: '0.75rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
            <span style={{ fontSize: '0.8rem', fontWeight: 700, color: '#00f2fe', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
              <span>📍</span> Proventos do dia {selectedDay} de {MONTH_NAMES[month]}
            </span>
            <button
              onClick={() => setSelectedDay(null)}
              style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: '0.75rem' }}
            >
              ✖ Fechar
            </button>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
            {selectedDayEvents.map((item, i) => {
              const paid = isPaidEvent(item);
              const color = paid ? '#4ade80' : '#fbbf24';
              const statusText = paid ? 'Recebido' : 'A receber';

              return (
                <AssetRiskDetailRow
                  key={i}
                  ticker={item.ticker}
                  subText={`${item.type || 'Provento'} · ${statusText}`}
                  valueText={formatMoney(item.net_amount, currency)}
                  valueColor={color}
                />
              );
            })}
          </div>
        </div>
      )}

      {monthlyDividends.eventCount === 0 && (
        <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textAlign: 'center', padding: '0.5rem 0' }}>
          Nenhum provento registrado ou previsto para o mês de {MONTH_NAMES[month]} de {year}.
        </div>
      )}
    </AnalysisCard>
  );
}
