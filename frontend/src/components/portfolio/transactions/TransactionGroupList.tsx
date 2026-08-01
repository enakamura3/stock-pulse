import React from 'react';
import { UnifiedTransaction } from '../types';
import { formatMoney, formatQuantity } from '../helpers';
import {
  TransactionWithBalance,
  formatDateStr,
  getBadge,
  getTransactionCircleDetails,
} from './types';

interface TransactionGroupListProps {
  grouped: { date: string; label: string; txs: TransactionWithBalance[] }[];
  pagedTxs: TransactionWithBalance[];
  groupByDate: boolean;
  kpiCurrency: string;
  handleEditTransaction: (tx: UnifiedTransaction) => void;
  handleDeleteTransaction: (txId: string) => void;
}

export default function TransactionGroupList({
  grouped,
  pagedTxs,
  groupByDate,
  kpiCurrency,
  handleEditTransaction,
  handleDeleteTransaction,
}: TransactionGroupListProps) {
  const renderItem = (tx: TransactionWithBalance) => {
    const badge = getBadge(tx);
    const circle = getTransactionCircleDetails(tx);
    const isRF = tx.module === 'RF';
    const isSplit = tx.type === 'SPLIT' || tx.type === 'REVERSE_SPLIT';

    const hasFx =
      Boolean(kpiCurrency) &&
      Boolean(tx.currency) &&
      tx.currency !== kpiCurrency &&
      Boolean(tx.exchange_rate) &&
      tx.exchange_rate! > 0;

    const totalValConverted = hasFx
      ? (tx.total_value || 0) * (tx.exchange_rate || 1)
      : tx.total_value || 0;

    return (
      <div
        key={tx.id}
        className="flex-row justify-between items-center flex-wrap gap-md"
        style={{
          padding: '0.85rem 1.1rem',
          background: 'rgba(255, 255, 255, 0.015)',
          border: '1px solid var(--panel-border)',
          borderRadius: '10px',
          transition: 'all 0.15s ease',
        }}
      >
        <div className="flex-row items-center gap-md" style={{ minWidth: 200 }}>
          <div
            style={{
              width: 40,
              height: 40,
              borderRadius: '50%',
              background: circle.gradient,
              border: `1px solid ${circle.borderColor}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '1.2rem',
              flexShrink: 0,
            }}
          >
            {circle.emoji}
          </div>

          <div className="flex-col" style={{ gap: '0.15rem' }}>
            <div className="flex-row items-center gap-sm">
              <span style={{ fontWeight: 700, fontSize: '0.95rem', color: '#fff' }}>
                {tx.asset_name}
              </span>
              <span
                style={{
                  padding: '0.15rem 0.5rem',
                  borderRadius: '4px',
                  fontSize: '0.65rem',
                  fontWeight: 700,
                  background: badge.bg,
                  color: badge.color,
                }}
              >
                {badge.text}
              </span>
            </div>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
              {formatDateStr(tx.date)}
            </span>
          </div>
        </div>

        <div className="flex-row gap-lg items-center flex-wrap" style={{ marginLeft: 'auto' }}>
          {isRF ? (
            <>
              {/* Valor da Operação */}
              <div className="flex-col items-end" style={{ minWidth: 120 }}>
                <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Valor da Operação</span>
                <span style={{ fontSize: '0.9rem', fontWeight: 700, color: badge.color }}>
                  {formatMoney(tx.total_value ?? 0, tx.currency || 'BRL')}
                </span>
                {hasFx && (
                  <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: '0.65rem', fontWeight: 'normal' }}>
                    (Câmbio: {tx.exchange_rate!.toFixed(4)})
                  </span>
                )}
              </div>

              {/* Módulo */}
              <div className="flex-col items-end" style={{ minWidth: 100 }}>
                <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Módulo</span>
                <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                  {tx.asset_type || 'Renda Fixa'}
                </span>
              </div>
            </>
          ) : isSplit ? (
            <>
              {/* Proporção */}
              <div className="flex-col items-end" style={{ minWidth: 90 }}>
                <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Proporção</span>
                <span style={{ fontSize: '0.85rem', fontWeight: 700, color: badge.color }}>
                  {tx.type === 'REVERSE_SPLIT'
                    ? `${formatQuantity(tx.quantity ?? 0)} para 1`
                    : `1 para ${formatQuantity(tx.quantity ?? 0)}`}
                </span>
              </div>

              {/* Saldo de Cotas após */}
              <div className="flex-col items-end" style={{ minWidth: 100 }}>
                <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Saldo de Cotas após</span>
                <span style={{ fontSize: '0.85rem', fontWeight: 600, color: '#00f2fe' }}>
                  {formatQuantity(tx.resulting_quantity ?? 0)} un.
                </span>
              </div>
            </>
          ) : (
            <>
              {/* Quantidade */}
              <div className="flex-col items-end" style={{ minWidth: 90 }}>
                <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Quantidade</span>
                <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                  {formatQuantity(tx.quantity || 0)}
                </span>
              </div>

              {/* Preço Unitário */}
              <div className="flex-col items-end" style={{ minWidth: 100 }}>
                <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Preço Unit.</span>
                <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                  {formatMoney(tx.unit_price || 0, tx.currency || 'BRL')}
                </span>
                {hasFx && (
                  <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: '0.65rem', fontWeight: 'normal' }}>
                    {`(Câmbio: ${tx.exchange_rate!.toFixed(4)})`}
                  </span>
                )}
              </div>

              {/* Valor Total */}
              <div className="flex-col items-end" style={{ minWidth: 120 }}>
                <span style={{ fontSize: '0.7rem', color: 'var(--text-secondary)' }}>Valor Total</span>
                <span style={{ fontSize: '0.9rem', fontWeight: 700, color: badge.color }}>
                  {formatMoney(totalValConverted, kpiCurrency || tx.currency || 'BRL')}
                </span>
              </div>
            </>
          )}

          {/* Ações */}
          <div className="flex-row gap-xs items-center" style={{ marginLeft: '0.5rem' }}>
            <button
              onClick={() => handleEditTransaction(tx)}
              className="btn-secondary"
              style={{ padding: '0.3rem 0.6rem', fontSize: '0.75rem' }}
              title="Editar operação"
            >
              ✏️
            </button>
            <button
              onClick={() => handleDeleteTransaction(tx.id)}
              className="btn-secondary"
              style={{ padding: '0.3rem 0.6rem', fontSize: '0.75rem', color: '#ff3d00' }}
              title="Excluir operação"
            >
              🗑️
            </button>
          </div>
        </div>
      </div>
    );
  };

  if (groupByDate) {
    return (
      <div className="flex-col gap-lg">
        {grouped.map((group) => (
          <div key={group.date} className="flex-col gap-xs">
            <div style={{ fontSize: '0.75rem', fontWeight: 700, color: 'var(--accent-color)', paddingLeft: '0.2rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              {group.label}
            </div>
            <div className="flex-col gap-xs">
              {group.txs.map((tx) => renderItem(tx))}
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="flex-col gap-xs">
      {pagedTxs.map((tx) => renderItem(tx))}
    </div>
  );
}
