import React from 'react';
import { MONTHS, TX_TYPES, SELECT_STYLE, OPTION_STYLE } from './types';

interface TransactionFilterBarProps {
  filterTxTicker: string;
  setFilterTxTicker: (t: string) => void;
  filterType: string;
  setFilterType: (t: string) => void;
  filterYear: string;
  setFilterYear: (y: string) => void;
  filterMonth: string;
  setFilterMonth: (m: string) => void;
  groupByDate: boolean;
  setGroupByDate: (g: boolean) => void;
  availableYears: string[];
  availableTickers: string[];
  totalFilteredCount: number;
  onLaunchOperation: () => void;
}

export default function TransactionFilterBar({
  filterTxTicker,
  setFilterTxTicker,
  filterType,
  setFilterType,
  filterYear,
  setFilterYear,
  filterMonth,
  setFilterMonth,
  groupByDate,
  setGroupByDate,
  availableYears,
  availableTickers,
  totalFilteredCount,
  onLaunchOperation,
}: TransactionFilterBarProps) {
  return (
    <div className="flex-row justify-between items-center flex-wrap gap-md">
      <div>
        <h3 className="card-title">📜 Histórico de Operações</h3>
        <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
          {totalFilteredCount} registro{totalFilteredCount !== 1 ? 's' : ''} encontrado{totalFilteredCount !== 1 ? 's' : ''}
        </span>
      </div>

      <div className="flex-row gap-sm flex-wrap items-center">
        {/* Agrupamento */}
        <div
          style={{
            display: 'flex',
            background: '#1E293B',
            borderRadius: '6px',
            padding: '2px',
            border: '1px solid var(--panel-border)',
          }}
        >
          <button
            onClick={() => setGroupByDate(true)}
            style={{
              padding: '0.25rem 0.5rem',
              fontSize: '0.75rem',
              borderRadius: '4px',
              border: 'none',
              background: groupByDate ? 'var(--accent-gradient)' : 'transparent',
              color: groupByDate ? '#000' : 'var(--text-secondary)',
              cursor: 'pointer',
              fontWeight: groupByDate ? 700 : 500,
            }}
          >
            📅 Por Data
          </button>
          <button
            onClick={() => setGroupByDate(false)}
            style={{
              padding: '0.25rem 0.5rem',
              fontSize: '0.75rem',
              borderRadius: '4px',
              border: 'none',
              background: !groupByDate ? 'var(--accent-gradient)' : 'transparent',
              color: !groupByDate ? '#000' : 'var(--text-secondary)',
              cursor: 'pointer',
              fontWeight: !groupByDate ? 700 : 500,
            }}
          >
            📄 Lista Simples
          </button>
        </div>

        {/* Filtro Ticker */}
        <select
          value={filterTxTicker}
          onChange={(e) => setFilterTxTicker(e.target.value)}
          style={SELECT_STYLE}
        >
          <option value="" style={OPTION_STYLE}>Ativo: Todos</option>
          {availableTickers.map((t) => (
            <option key={t} value={t} style={OPTION_STYLE}>{t}</option>
          ))}
        </select>

        {/* Filtro Tipo */}
        <select
          value={filterType}
          onChange={(e) => setFilterType(e.target.value)}
          style={SELECT_STYLE}
        >
          {TX_TYPES.map((t) => (
            <option key={t.value} value={t.value} style={OPTION_STYLE}>{t.label}</option>
          ))}
        </select>

        {/* Filtro Ano */}
        <select
          value={filterYear}
          onChange={(e) => setFilterYear(e.target.value)}
          style={SELECT_STYLE}
        >
          <option value="Todos" style={OPTION_STYLE}>Ano: Todos</option>
          {availableYears.map((y) => (
            <option key={y} value={y} style={OPTION_STYLE}>{y}</option>
          ))}
        </select>

        {/* Filtro Mês */}
        <select
          value={filterMonth}
          onChange={(e) => setFilterMonth(e.target.value)}
          style={SELECT_STYLE}
        >
          <option value="Todos" style={OPTION_STYLE}>Mês: Todos</option>
          {MONTHS.map((m) => (
            <option key={m.value} value={m.value} style={OPTION_STYLE}>{m.label}</option>
          ))}
        </select>

        {/* Botão Nova Operação */}
        <button
          onClick={onLaunchOperation}
          className="primary-button"
          style={{ padding: '0.35rem 0.85rem', fontSize: '0.8rem' }}
        >
          + Nova Operação
        </button>
      </div>
    </div>
  );
}
