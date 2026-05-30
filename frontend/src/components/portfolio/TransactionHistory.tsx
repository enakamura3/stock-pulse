import React from 'react';
import { Transaction } from './types';
import { formatMoney, formatQuantity } from './helpers';

interface TransactionHistoryProps {
  transactions: Transaction[];
  filterTxTicker: string;
  setFilterTxTicker: (t: string) => void;
  handleEditTransaction: (tx: Transaction) => void;
  handleDeleteTransaction: (id: string) => void;
}

export default function TransactionHistory({
  transactions, filterTxTicker, setFilterTxTicker, handleEditTransaction, handleDeleteTransaction
}: TransactionHistoryProps) {
  const filteredTransactions = transactions.filter(tx => filterTxTicker === '' || tx.ticker === filterTxTicker);
  const tickers = Array.from(new Set(transactions.map(tx => tx.ticker))).sort();

  return (
    <div className="card flex-col gap-md" style={{ flex: '1 1 350px', minHeight: '380px' }}>
      <div className="flex-row justify-between items-center mb-lg">
        <h3 className="card-title">📜 Últimas Operações</h3>
        {transactions.length > 0 && (
          <select
            value={filterTxTicker}
            onChange={(e) => setFilterTxTicker(e.target.value)}
            style={{ padding: '0.3rem 0.6rem', borderRadius: '6px', border: '1px solid var(--panel-border)', background: '#1E293B', color: '#FFFFFF', fontSize: '0.8rem', outline: 'none', cursor: 'pointer' }}
          >
            <option value="">Todos os Ativos</option>
            {tickers.map(ticker => (
              <option key={ticker!} value={ticker}>{ticker}</option>
            ))}
          </select>
        )}
      </div>

      <div className="flex-col gap-sm" style={{ overflowY: 'auto', flex: 1, maxHeight: '550px' }}>
        {filteredTransactions.length > 0 ? (
          filteredTransactions.map((tx) => {
            const isBuy = tx.type === 'BUY' || tx.type === 'BONUS';
            const isSplit = tx.type === 'SPLIT' || tx.type === 'REVERSE_SPLIT';
            const isReverse = tx.type === 'REVERSE_SPLIT';
            
            return (
              <div key={tx.id} className="flex-row justify-between items-center p-sm" style={{ padding: '0.75rem', background: 'rgba(255,255,255,0.015)', border: '1px solid var(--panel-border)', borderRadius: '8px', fontSize: '0.8rem' }}>
                <div className="flex-row items-center gap-sm flex-wrap" style={{ flex: 1 }}>
                  <span className={`badge ${isBuy ? 'badge-success' : tx.type === 'SPLIT' ? 'badge-neutral' : tx.type === 'REVERSE_SPLIT' ? 'badge-neutral' : 'badge-danger'}`} style={{ color: isBuy ? '#00e676' : tx.type === 'SPLIT' ? '#00f2fe' : tx.type === 'REVERSE_SPLIT' ? '#e040fb' : '#ff3d00', background: isBuy ? 'rgba(0, 230, 118, 0.08)' : tx.type === 'SPLIT' ? 'rgba(0, 242, 254, 0.08)' : tx.type === 'REVERSE_SPLIT' ? 'rgba(156, 39, 176, 0.08)' : 'rgba(255, 61, 0, 0.08)' }}>
                    {isBuy ? (tx.type === 'BONUS' ? 'BÔNUS' : 'COMPRA') : tx.type === 'SPLIT' ? 'SPLIT' : tx.type === 'REVERSE_SPLIT' ? 'AGRUPAMENTO' : 'VENDA'}
                  </span>
                  
                  <span className="font-bold text-primary">{tx.ticker}</span>
                  
                  <span className="text-secondary text-xs" style={{ borderRight: '1px solid var(--panel-border)', paddingRight: '0.5rem' }}>
                    {new Date(tx.executed_at).toISOString().split('T')[0].replace(/-/g, '/')}
                  </span>
                  
                  <div className="flex-row items-center gap-sm text-xs">
                    {!isSplit ? (
                      <>
                        <span className="text-secondary">{formatQuantity(tx.quantity)} un. x {formatMoney(tx.unit_price, tx.currency || 'BRL')}</span>
                        <span className="text-secondary">=</span>
                        <span className="font-bold">{formatMoney(tx.quantity * tx.unit_price, tx.currency || 'BRL')}</span>
                      </>
                    ) : (
                      <span className="font-bold">Fator: {isReverse ? `1 para ${formatQuantity(tx.quantity)}` : `${formatQuantity(tx.quantity)} para 1`}</span>
                    )}
                  </div>
                </div>

                <div className="flex-row items-center gap-sm" style={{ paddingLeft: '0.5rem' }}>
                  <button onClick={() => handleEditTransaction(tx)} className="btn-secondary" style={{ padding: '0.4rem', display: 'flex', alignItems: 'center', justifyContent: 'center' }} title="Editar Transação">✏️</button>
                  <button onClick={() => handleDeleteTransaction(tx.id)} className="btn-danger" style={{ padding: '0.4rem', display: 'flex', alignItems: 'center', justifyContent: 'center' }} title="Excluir Transação">🗑️</button>
                </div>
              </div>
            );
          })
        ) : (
          <div className="flex-col items-center justify-center text-secondary" style={{ height: '200px', border: '1px dashed var(--panel-border)', borderRadius: '8px' }}>
            <span className="text-xl mb-sm">📝</span>
            <p className="text-xs text-center m-0">Sem transações registradas nesta carteira.</p>
          </div>
        )}
      </div>
    </div>
  );
}
