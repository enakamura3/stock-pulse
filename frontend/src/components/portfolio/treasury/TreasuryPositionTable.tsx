import React from 'react';
import { TreasuryPosition } from '../types';
import { SortKey, SortDir, fmt, fmtPct, getTreasuryTypeBadgeColor } from './types';

interface TreasuryPositionTableProps {
  positions: TreasuryPosition[];
  isLoadingPositions: boolean;
  isImporting: boolean;
  sortKey: SortKey;
  sortDir: SortDir;
  onSort: (key: SortKey) => void;
  onOpenModal: () => void;
  onImport: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onExport: () => void;
  onRedeem: (pos: TreasuryPosition) => void;
}

export default function TreasuryPositionTable({
  positions,
  isLoadingPositions,
  isImporting,
  sortKey,
  sortDir,
  onSort,
  onOpenModal,
  onImport,
  onExport,
  onRedeem,
}: TreasuryPositionTableProps) {
  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return <span style={{ opacity: 0.3, fontSize: '0.65rem' }}> ↕</span>;
    return <span style={{ color: 'var(--accent-color)', fontSize: '0.65rem' }}> {sortDir === 'asc' ? '↑' : '↓'}</span>;
  };

  const sortedPositions = [...positions].sort((a, b) => {
    let aVal: number | string = 0;
    let bVal: number | string = 0;
    switch (sortKey) {
      case 'ticker': aVal = a.ticker || ''; bVal = b.ticker || ''; break;
      case 'treasury_type': aVal = a.treasury_type || ''; bVal = b.treasury_type || ''; break;
      case 'maturity_date': aVal = a.maturity_date || ''; bVal = b.maturity_date || ''; break;
      case 'total_invested': aVal = a.total_invested ?? 0; bVal = b.total_invested ?? 0; break;
      case 'gross_value': aVal = a.gross_value ?? 0; bVal = b.gross_value ?? 0; break;
      case 'net_value': aVal = a.net_value ?? 0; bVal = b.net_value ?? 0; break;
      case 'net_return':
        aVal = a.total_invested > 0 ? ((a.net_value - a.total_invested) / a.total_invested) : 0;
        bVal = b.total_invested > 0 ? ((b.net_value - b.total_invested) / b.total_invested) : 0;
        break;
      case 'iof_tax': aVal = a.iof_tax ?? 0; bVal = b.iof_tax ?? 0; break;
      case 'ir_tax': aVal = a.ir_tax ?? 0; bVal = b.ir_tax ?? 0; break;
      case 'b3_fee': aVal = a.b3_fee ?? 0; bVal = b.b3_fee ?? 0; break;
      case 'status': aVal = a.is_matured ? 1 : 0; bVal = b.is_matured ? 1 : 0; break;
    }
    if (typeof aVal === 'string') {
      return sortDir === 'asc' ? aVal.localeCompare(bVal as string) : (bVal as string).localeCompare(aVal);
    }
    return sortDir === 'asc' ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
  });

  return (
    <div className="card flex-col gap-md" style={{ flex: '2 1 600px', minHeight: '380px' }}>
      <div className="flex-row justify-between items-center mb-lg">
        <div>
          <h3 className="card-title">📋 Posições Ativas</h3>
          <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
            {positions.length} título{positions.length !== 1 ? 's' : ''}
          </span>
        </div>
        <div className="flex-row gap-sm">
          <label className="btn-secondary" style={{ padding: '0.45rem 1rem', fontSize: '0.8rem', cursor: 'pointer', margin: 0 }}>
            📥 Importar
            <input type="file" accept=".csv" style={{ display: 'none' }} onChange={onImport} />
          </label>
          <button
            onClick={onExport}
            className="btn-secondary"
            style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}
          >
            📤 Exportar
          </button>
          <button
            id="treasury-add-btn"
            onClick={onOpenModal}
            className="primary-button"
            style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}
          >
            + Nova Aplicação
          </button>
        </div>
      </div>

      {(isLoadingPositions || isImporting) ? (
        <div className="flex-row items-center justify-center" style={{ height: '120px' }}>
          <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 28, height: 28 }} />
        </div>
      ) : positions.length === 0 ? (
        <div
          className="flex-col items-center justify-center text-secondary"
          style={{ height: '120px', border: '1px dashed var(--panel-border)', borderRadius: '10px' }}
        >
          <p className="text-sm m-0">Nenhuma posição ativa de Tesouro Direto.</p>
          <button
            onClick={onOpenModal}
            className="btn-secondary mt-md"
            style={{ fontSize: '0.8rem', padding: '0.4rem 1rem' }}
          >
            + Adicionar primeira aplicação
          </button>
        </div>
      ) : (
        <div className="table-container flex-col" style={{ flex: 1 }}>
          <table className="data-table" style={{ width: '100%' }}>
            <thead>
              <tr>
                <th style={{ cursor: 'pointer' }} onClick={() => onSort('ticker')}>Título {sortIcon('ticker')}</th>
                <th style={{ cursor: 'pointer' }} onClick={() => onSort('treasury_type')}>Tipo {sortIcon('treasury_type')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('maturity_date')}>Vencimento {sortIcon('maturity_date')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('total_invested')}>Aplicado {sortIcon('total_invested')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('gross_value')}>Bruto {sortIcon('gross_value')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('net_value')}>Líquido {sortIcon('net_value')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('net_return')}>Retorno Líq. {sortIcon('net_return')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('iof_tax')}>IOF {sortIcon('iof_tax')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('ir_tax')}>IR {sortIcon('ir_tax')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => onSort('b3_fee')}>Taxa B3 {sortIcon('b3_fee')}</th>
                <th className="text-center" style={{ cursor: 'pointer' }} onClick={() => onSort('status')}>Status {sortIcon('status')}</th>
                <th className="text-center">Ações</th>
              </tr>
            </thead>
            <tbody>
              {sortedPositions.map((pos) => {
                const liqReturn = pos.total_invested > 0
                  ? ((pos.net_value - pos.total_invested) / pos.total_invested) * 100
                  : 0;
                const isPositive = liqReturn >= 0;
                return (
                  <tr key={pos.transaction_id}>
                    <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                      {pos.ticker}
                      {pos.has_coupons && (
                        <span
                          title="Paga cupons semestrais"
                          style={{ marginLeft: 6, fontSize: '0.65rem', color: '#ff9800', fontWeight: 400 }}
                        >
                          cupons
                        </span>
                      )}
                    </td>
                    <td>
                      <span style={{
                        padding: '0.2rem 0.55rem',
                        borderRadius: '12px',
                        fontSize: '0.68rem',
                        fontWeight: 700,
                        background: `${getTreasuryTypeBadgeColor(pos.treasury_type)}22`,
                        color: getTreasuryTypeBadgeColor(pos.treasury_type),
                        border: `1px solid ${getTreasuryTypeBadgeColor(pos.treasury_type)}44`,
                      }}>
                        {pos.treasury_type}
                      </span>
                    </td>
                    <td className="text-right" style={{ color: 'var(--text-secondary)' }}>
                      {pos.is_matured
                        ? <span style={{ color: '#f44336', fontWeight: 600 }}>Vencido</span>
                        : `${new Date(pos.maturity_date).toLocaleDateString('pt-BR')} (${pos.days_to_maturity}d)`}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{fmt(pos.total_invested)}</td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{fmt(pos.gross_value)}</td>
                    <td className="text-right font-semibold" style={{ fontFamily: 'monospace', color: pos.net_value >= pos.total_invested ? '#4caf50' : '#f44336' }}>
                      {fmt(pos.net_value)}
                    </td>
                    <td className="text-right font-semibold" style={{ color: isPositive ? '#4caf50' : '#f44336' }}>
                      {fmtPct(liqReturn)}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace', color: '#f44336', fontSize: '0.8rem' }}>
                      {pos.iof_tax > 0 ? `-${fmt(pos.iof_tax)}` : 'R$ 0,00'}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace', color: '#f44336', fontSize: '0.8rem' }}>
                      {pos.ir_tax > 0 ? `-${fmt(pos.ir_tax)}` : 'R$ 0,00'}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace', color: '#ff9800', fontSize: '0.8rem' }}>
                      {pos.b3_fee > 0 ? `-${fmt(pos.b3_fee)}` : 'R$ 0,00'}
                    </td>
                    <td className="text-center">
                      <span style={{
                        padding: '0.2rem 0.5rem',
                        borderRadius: '4px',
                        fontSize: '0.65rem',
                        fontWeight: 700,
                        background: pos.is_matured ? 'rgba(244,67,54,0.15)' : 'rgba(76,175,80,0.15)',
                        color: pos.is_matured ? '#f44336' : '#4caf50',
                      }}>
                        {pos.is_matured ? 'VENCIDO' : 'ATIVO'}
                      </span>
                    </td>
                    <td className="text-center">
                      <button
                        onClick={() => onRedeem(pos)}
                        className="btn-secondary"
                        style={{ padding: '0.25rem 0.6rem', fontSize: '0.72rem', color: '#ff9800', borderColor: 'rgba(255,152,0,0.4)' }}
                        title="Registrar Resgate"
                      >
                        Resgatar
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
