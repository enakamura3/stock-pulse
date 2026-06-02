import React, { useState } from 'react';
import { Transaction } from './types';
import { formatMoney, formatQuantity } from './helpers';

interface TransactionHistoryProps {
  transactions: Transaction[];
  filterTxTicker: string;
  setFilterTxTicker: (t: string) => void;
  handleEditTransaction: (tx: Transaction) => void;
  handleDeleteTransaction: (id: string) => void;
  onLaunchOperation?: () => void;
}

export default function TransactionHistory({
  transactions, filterTxTicker, setFilterTxTicker, handleEditTransaction, handleDeleteTransaction, onLaunchOperation
}: TransactionHistoryProps) {
  const [filterTxYear, setFilterTxYear] = useState<string>('Todos');
  const [filterTxMonth, setFilterTxMonth] = useState<string>('Todos');

  const cumulativeQuantities: Record<string, number> = {};
  const cumulativeInvested: Record<string, number> = {};
  const transactionsWithBalance = [...transactions].reverse().map(tx => {
    let currentBalance = cumulativeQuantities[tx.ticker] || 0;
    let currentInvested = cumulativeInvested[tx.ticker] || 0;
    
    if (tx.type === 'BUY') {
      currentBalance += tx.quantity;
      currentInvested += tx.quantity * tx.unit_price;
    } else if (tx.type === 'BONUS') {
      currentBalance += tx.quantity;
    } else if (tx.type === 'SELL') {
      const prevBalance = currentBalance;
      currentBalance -= tx.quantity;
      if (prevBalance > 0) {
        const avgCost = currentInvested / prevBalance;
        currentInvested = currentBalance > 0 ? currentBalance * avgCost : 0;
      }
    } else if (tx.type === 'SPLIT') {
      currentBalance = currentBalance * tx.quantity;
    } else if (tx.type === 'REVERSE_SPLIT') {
      currentBalance = currentBalance / tx.quantity;
    }
    
    cumulativeQuantities[tx.ticker] = currentBalance;
    cumulativeInvested[tx.ticker] = currentInvested;
    return { ...tx, resulting_quantity: currentBalance, resulting_invested: currentInvested };
  }).reverse();

  const filteredTransactions = transactionsWithBalance.filter(tx => {
    if (filterTxTicker !== '' && tx.ticker !== filterTxTicker) return false;
    
    const year = tx.executed_at ? tx.executed_at.substring(0, 4) : '';
    const month = tx.executed_at ? tx.executed_at.substring(5, 7) : '';
    
    if (filterTxYear !== 'Todos' && year !== filterTxYear) return false;
    if (filterTxMonth !== 'Todos' && month !== filterTxMonth) return false;
    
    return true;
  });

  const tickers = Array.from(new Set(transactions.map(tx => tx.ticker))).sort();
  const availableYears = Array.from(new Set(transactions.map(tx => tx.executed_at ? tx.executed_at.substring(0, 4) : ''))).filter(y => y !== '').sort((a, b) => b.localeCompare(a));

  return (
    <div className="card flex-col gap-md" style={{ flex: '1 1 350px', minHeight: '800px' }}>
      <div className="flex-row justify-between items-center mb-lg flex-wrap gap-md">
        <h3 className="card-title" style={{ margin: 0 }}>📜 Histórico de Operações</h3>
        <div className="flex-row gap-sm flex-wrap items-center">
          {transactions.length > 0 && (
            <>
              <select
              value={filterTxYear}
              onChange={(e) => setFilterTxYear(e.target.value)}
              style={{ padding: '0.3rem 0.6rem', borderRadius: '6px', border: '1px solid var(--panel-border)', background: '#1E293B', color: '#FFFFFF', fontSize: '0.8rem', outline: 'none', cursor: 'pointer' }}
            >
              <option value="Todos" style={{ background: '#1c1f24' }}>Ano: Todos</option>
              {availableYears.map(year => (
                <option key={year} value={year} style={{ background: '#1c1f24' }}>{year}</option>
              ))}
            </select>
            
            <select
              value={filterTxMonth}
              onChange={(e) => setFilterTxMonth(e.target.value)}
              style={{ padding: '0.3rem 0.6rem', borderRadius: '6px', border: '1px solid var(--panel-border)', background: '#1E293B', color: '#FFFFFF', fontSize: '0.8rem', outline: 'none', cursor: 'pointer' }}
            >
              <option value="Todos" style={{ background: '#1c1f24' }}>Mês: Todos</option>
              <option value="01" style={{ background: '#1c1f24' }}>Janeiro</option>
              <option value="02" style={{ background: '#1c1f24' }}>Fevereiro</option>
              <option value="03" style={{ background: '#1c1f24' }}>Março</option>
              <option value="04" style={{ background: '#1c1f24' }}>Abril</option>
              <option value="05" style={{ background: '#1c1f24' }}>Maio</option>
              <option value="06" style={{ background: '#1c1f24' }}>Junho</option>
              <option value="07" style={{ background: '#1c1f24' }}>Julho</option>
              <option value="08" style={{ background: '#1c1f24' }}>Agosto</option>
              <option value="09" style={{ background: '#1c1f24' }}>Setembro</option>
              <option value="10" style={{ background: '#1c1f24' }}>Outubro</option>
              <option value="11" style={{ background: '#1c1f24' }}>Novembro</option>
              <option value="12" style={{ background: '#1c1f24' }}>Dezembro</option>
            </select>

            <select
              value={filterTxTicker}
              onChange={(e) => setFilterTxTicker(e.target.value)}
              style={{ padding: '0.3rem 0.6rem', borderRadius: '6px', border: '1px solid var(--panel-border)', background: '#1E293B', color: '#FFFFFF', fontSize: '0.8rem', outline: 'none', cursor: 'pointer' }}
            >
              <option value="" style={{ background: '#1c1f24' }}>Todos os Ativos</option>
              {tickers.map(ticker => (
                <option key={ticker!} value={ticker} style={{ background: '#1c1f24' }}>{ticker}</option>
              ))}
            </select>
            </>
          )}
          {onLaunchOperation && (
            <button className="primary-button" onClick={onLaunchOperation} style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}>
              + Lançar Operação
            </button>
          )}
        </div>
      </div>

      <div className="flex-col gap-sm" style={{ overflowY: 'auto', flex: 1, maxHeight: '800px' }}>
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
                    {tx.executed_at ? new Date(tx.executed_at).toISOString().split('T')[0].replace(/-/g, '/') : 'N/A'}
                  </span>
                  
                  <div className="flex-row items-center gap-sm text-xs">
                    {!isSplit ? (
                      <>
                        <span className="text-secondary">{formatQuantity(tx.quantity)} un. x {formatMoney(tx.unit_price, tx.currency || 'BRL')}</span>
                        <span className="text-secondary">=</span>
                        <span className="font-bold">{formatMoney(tx.quantity * tx.unit_price, tx.currency || 'BRL')}</span>
                      </>
                    ) : (
                      <span className="font-bold">Fator: {isReverse ? `${formatQuantity(tx.quantity)} para 1` : `1 para ${formatQuantity(tx.quantity)}`}</span>
                    )}

                    <span className="text-secondary" style={{ marginLeft: '1rem', borderLeft: '1px solid var(--panel-border)', paddingLeft: '1rem', fontSize: '0.75rem' }}>
                      Saldo após: <span className="font-bold text-primary" style={{ color: '#00f2fe' }}>{formatQuantity(tx.resulting_quantity)} un.</span>
                      <span className="text-secondary" style={{ marginLeft: '0.5rem', marginRight: '0.5rem' }}>|</span>
                      Investido: <span className="font-bold text-primary" style={{ color: '#00e676' }}>{formatMoney(tx.resulting_invested, tx.currency || 'BRL')}</span>
                    </span>
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
