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
          padding: '1rem 1.25rem',
          background: 'rgba(255, 255, 255, 0.015)',
          border: '1px solid var(--panel-border)',
          borderRadius: '10px',
          transition: 'all 0.15s ease',
        }}
      >
        {/* Coluna 1: Ativo, Badge e Data */}
        <div className="flex-row items-center gap-md" style={{ flex: '1.2 1 220px' }}>
          <div
            style={{
              width: 42,
              height: 42,
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

          <div className="flex-col" style={{ gap: '0.25rem' }}>
            <div className="flex-row items-center gap-sm">
              <span style={{ fontWeight: 700, fontSize: '0.95rem', color: '#fff' }}>
                {tx.asset_name}
              </span>
              <span
                style={{
                  padding: '0.2rem 0.6rem',
                  borderRadius: '4px',
                  fontSize: '0.68rem',
                  fontWeight: 700,
                  background: badge.bg,
                  color: badge.color,
                  letterSpacing: '0.3px',
                }}
              >
                {badge.text}
              </span>
            </div>
            <span style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
              {formatDateStr(tx.date)}
            </span>
          </div>
        </div>

        {/* Colunas Intermediárias e Valores (Distribuídas no Meio e Direita) */}
        <div
          className="flex-row items-center flex-wrap"
          style={{ flex: '2 1 400px', justifyContent: 'space-between', gap: '1.25rem' }}
        >
          {isRF ? (
            <>
              {/* Módulo / Tipo */}
              <div className="flex-col items-start" style={{ minWidth: 110 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Módulo</span>
                <span style={{ fontSize: '0.88rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                  {tx.asset_type || 'Renda Fixa'}
                </span>
              </div>

              {/* Saldo Investido Resultante */}
              <div className="flex-col items-start" style={{ minWidth: 130 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Posição após</span>
                <span style={{ fontSize: '0.88rem', fontWeight: 600, color: '#00f2fe' }}>
                  {formatMoney(tx.resulting_invested ?? 0, tx.currency || 'BRL')}
                </span>
              </div>

              {/* Valor da Operação */}
              <div className="flex-col items-end" style={{ minWidth: 130 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Valor da Operação</span>
                <span style={{ fontSize: '0.92rem', fontWeight: 700, color: badge.color }}>
                  {formatMoney(tx.total_value ?? 0, tx.currency || 'BRL')}
                </span>
                {hasFx && (
                  <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: '0.68rem', fontWeight: 'normal', marginTop: '0.1rem' }}>
                    (Câmbio: {tx.exchange_rate!.toFixed(4)})
                  </span>
                )}
              </div>
            </>
          ) : isSplit ? (
            <>
              {/* Proporção */}
              <div className="flex-col items-start" style={{ minWidth: 110 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Proporção</span>
                <span style={{ fontSize: '0.88rem', fontWeight: 700, color: badge.color }}>
                  {tx.type === 'REVERSE_SPLIT'
                    ? `${formatQuantity(tx.quantity ?? 0)} para 1`
                    : `1 para ${formatQuantity(tx.quantity ?? 0)}`}
                </span>
              </div>

              {/* Saldo de Cotas após */}
              <div className="flex-col items-start" style={{ minWidth: 120 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Posição após</span>
                <span style={{ fontSize: '0.88rem', fontWeight: 600, color: 'var(--accent-color)' }}>
                  {formatQuantity(tx.resulting_quantity ?? 0)} un.
                </span>
              </div>

              {/* Espaçador neutro */}
              <div style={{ minWidth: 135 }} />
            </>
          ) : (
            <>
              {/* Quantidade e Preço Unitário (Operados) */}
              <div className="flex-col items-start" style={{ minWidth: 120 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Qtd. & Preço</span>
                <span style={{ fontSize: '0.88rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                  {formatQuantity(tx.quantity || 0)} un. a {formatMoney(tx.unit_price || 0, tx.currency || 'BRL')}
                </span>
                {hasFx && (
                  <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: '0.68rem', fontWeight: 'normal', marginTop: '0.1rem' }}>
                    {`(Câmbio: ${tx.exchange_rate!.toFixed(4)})`}
                  </span>
                )}
              </div>

              {/* Saldo Resultante de Papéis / Cotas após a compra */}
              <div className="flex-col items-start" style={{ minWidth: 110 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Posição após</span>
                <span style={{ fontSize: '0.88rem', fontWeight: 700, color: 'var(--accent-color)' }}>
                  {formatQuantity(tx.resulting_quantity ?? 0)} un.
                </span>
              </div>

              {/* Valor Total & Taxas */}
              <div className="flex-col items-end" style={{ minWidth: 135 }}>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>Valor Total</span>
                <span style={{ fontSize: '0.92rem', fontWeight: 700, color: badge.color }}>
                  {formatMoney(totalValConverted, kpiCurrency || tx.currency || 'BRL')}
                </span>
                {Boolean(tx.fee) && tx.fee! > 0 && (
                  <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: '0.68rem', fontWeight: 'normal', marginTop: '0.1rem' }}>
                    {`(Taxas: ${formatMoney(tx.fee!, tx.currency || 'BRL')})`}
                  </span>
                )}
              </div>
            </>
          )}

          {/* Botões de Ação */}
          <div className="flex-row gap-sm items-center" style={{ gap: '0.5rem', marginLeft: '0.5rem' }}>
            <button
              onClick={() => handleEditTransaction(tx)}
              className="btn-secondary"
              style={{ padding: '0.35rem 0.65rem', fontSize: '0.78rem' }}
              title="Editar operação"
            >
              ✏️
            </button>
            <button
              onClick={() => handleDeleteTransaction(tx.id)}
              className="btn-secondary"
              style={{ padding: '0.35rem 0.65rem', fontSize: '0.78rem', color: '#ff3d00' }}
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
      <div className="flex-col" style={{ gap: '1.75rem' }}>
        {grouped.map((group) => (
          <div key={group.date} className="flex-col">
            {/* Cabeçalho do Grupo de Data */}
            <div
              style={{
                fontSize: '0.8rem',
                fontWeight: 700,
                color: 'var(--accent-color)',
                padding: '0.25rem 0.4rem',
                marginBottom: '0.6rem',
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                display: 'flex',
                alignItems: 'center',
                gap: '0.4rem',
                borderBottom: '1px solid rgba(255, 255, 255, 0.05)',
                paddingBottom: '0.4rem',
              }}
            >
              <span>📅</span>
              <span>{group.label}</span>
            </div>

            {/* Lista de Transações da Data */}
            <div className="flex-col gap-sm">
              {group.txs.map((tx) => renderItem(tx))}
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="flex-col gap-sm">
      {pagedTxs.map((tx) => renderItem(tx))}
    </div>
  );
}
