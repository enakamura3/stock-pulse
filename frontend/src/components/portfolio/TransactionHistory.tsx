import React, { useState } from 'react';
import { UnifiedTransaction } from './types';
import { formatMoney, formatQuantity } from './helpers';

interface TransactionHistoryProps {
  transactions: UnifiedTransaction[];
  filterTxTicker: string;
  setFilterTxTicker: (t: string) => void;
  handleEditTransaction: (tx: UnifiedTransaction) => void;
  handleDeleteTransaction: (id: string) => void;
  onLaunchOperation?: () => void;
}

export default function TransactionHistory({
  transactions, filterTxTicker, setFilterTxTicker, handleEditTransaction, handleDeleteTransaction, onLaunchOperation
}: TransactionHistoryProps) {
  const [filterTxYear, setFilterTxYear] = useState<string>('Todos');
  const [filterTxMonth, setFilterTxMonth] = useState<string>('Todos');
  const [filterTxModule, setFilterTxModule] = useState<string>('Todos'); // 'Todos', 'RV', 'RF'

  const cumulativeQuantities: Record<string, number> = {};
  const cumulativeInvested: Record<string, number> = {};
  
  // Calculate running balances only for RV transactions (iterating oldest to newest)
  const transactionsWithBalance = [...transactions].reverse().map(tx => {
    if (tx.module === 'RF') {
      return { ...tx };
    }

    // RV specific balance calculation
    let currentBalance = cumulativeQuantities[tx.asset_name] || 0;
    let currentInvested = cumulativeInvested[tx.asset_name] || 0;
    const qty = tx.quantity || 0;
    const price = tx.unit_price || 0;
    
    if (tx.type === 'BUY') {
      currentBalance += qty;
      currentInvested += qty * price;
    } else if (tx.type === 'BONUS') {
      currentBalance += qty;
    } else if (tx.type === 'SELL') {
      const prevBalance = currentBalance;
      currentBalance -= qty;
      if (prevBalance > 0) {
        const avgCost = currentInvested / prevBalance;
        currentInvested = currentBalance > 0 ? currentBalance * avgCost : 0;
      }
    } else if (tx.type === 'SPLIT') {
      currentBalance = currentBalance * qty;
    } else if (tx.type === 'REVERSE_SPLIT') {
      currentBalance = currentBalance / qty;
    }
    
    cumulativeQuantities[tx.asset_name] = currentBalance;
    cumulativeInvested[tx.asset_name] = currentInvested;
    return { ...tx, resulting_quantity: currentBalance, resulting_invested: currentInvested };
  }).reverse(); // Revert back to newest first

  const filteredTransactions = transactionsWithBalance.filter(tx => {
    if (filterTxModule !== 'Todos' && tx.module !== filterTxModule) return false;
    if (filterTxTicker !== '' && tx.asset_name !== filterTxTicker) return false;
    
    const year = tx.date ? tx.date.substring(0, 4) : '';
    const month = tx.date ? tx.date.substring(5, 7) : '';
    
    if (filterTxYear !== 'Todos' && year !== filterTxYear) return false;
    if (filterTxMonth !== 'Todos' && month !== filterTxMonth) return false;
    
    return true;
  });

  const tickers = Array.from(new Set(transactions.map(tx => tx.asset_name))).sort();
  const availableYears = Array.from(new Set(transactions.map(tx => tx.date ? tx.date.substring(0, 4) : ''))).filter(y => y !== '').sort((a, b) => b.localeCompare(a));

  return (
    <div className="card flex-col gap-md" style={{ flex: '1 1 350px', minHeight: '800px' }}>
      <div className="flex-row justify-between items-center mb-lg flex-wrap gap-md">
        <h3 className="card-title" style={{ margin: 0 }}>📜 Histórico de Operações</h3>
        <div className="flex-row gap-sm flex-wrap items-center">
          {transactions.length > 0 && (
            <>
              <select
                value={filterTxModule}
                onChange={(e) => setFilterTxModule(e.target.value)}
                style={{ padding: '0.3rem 0.6rem', borderRadius: '6px', border: '1px solid var(--panel-border)', background: '#1E293B', color: '#FFFFFF', fontSize: '0.8rem', outline: 'none', cursor: 'pointer' }}
              >
                <option value="Todos" style={{ background: '#1c1f24' }}>Todos Módulos</option>
                <option value="RV" style={{ background: '#1c1f24' }}>Renda Variável</option>
                <option value="RF" style={{ background: '#1c1f24' }}>Renda Fixa</option>
              </select>

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
            const isRF = tx.module === 'RF';
            const isBuy = tx.type === 'BUY' || tx.type === 'BONUS' || tx.type === 'SUBSCRIPTION';
            const isSplit = tx.type === 'SPLIT' || tx.type === 'REVERSE_SPLIT';
            const isReverse = tx.type === 'REVERSE_SPLIT';
            
            // Badge styles
            let badgeText = '';
            let badgeColor = '';
            let badgeBg = '';
            
            if (isRF) {
              badgeText = tx.type === 'SUBSCRIPTION' ? 'APLICAÇÃO' : 'RESGATE';
              badgeColor = tx.type === 'SUBSCRIPTION' ? '#2196F3' : '#FF9800'; // Blue for App, Orange for Resgate
              badgeBg = tx.type === 'SUBSCRIPTION' ? 'rgba(33, 150, 243, 0.08)' : 'rgba(255, 152, 0, 0.08)';
            } else {
              if (tx.type === 'BONUS') { badgeText = 'BÔNUS'; badgeColor = '#00e676'; badgeBg = 'rgba(0, 230, 118, 0.08)'; }
              else if (tx.type === 'BUY') { badgeText = 'COMPRA'; badgeColor = '#00e676'; badgeBg = 'rgba(0, 230, 118, 0.08)'; }
              else if (tx.type === 'SELL') { badgeText = 'VENDA'; badgeColor = '#ff3d00'; badgeBg = 'rgba(255, 61, 0, 0.08)'; }
              else if (tx.type === 'SPLIT') { badgeText = 'SPLIT'; badgeColor = '#00f2fe'; badgeBg = 'rgba(0, 242, 254, 0.08)'; }
              else if (tx.type === 'REVERSE_SPLIT') { badgeText = 'AGRUPAMENTO'; badgeColor = '#e040fb'; badgeBg = 'rgba(156, 39, 176, 0.08)'; }
            }
            
            return (
              <div key={tx.id} className="flex-row justify-between items-center p-sm" style={{ padding: '0.75rem', background: 'rgba(255,255,255,0.015)', border: '1px solid var(--panel-border)', borderRadius: '8px', fontSize: '0.8rem' }}>
                <div className="flex-row items-center gap-sm flex-wrap" style={{ flex: 1 }}>
                  
                  <span title={isRF ? "Renda Fixa" : "Renda Variável"} style={{ fontSize: '1.2rem', cursor: 'help' }}>
                    {isRF ? '🏦' : '📈'}
                  </span>

                  <span className="badge" style={{ color: badgeColor, background: badgeBg }}>
                    {badgeText}
                  </span>
                  
                  <span className="font-bold text-primary">{tx.asset_name}</span>
                  
                  <span className="text-secondary text-xs" style={{ borderRight: '1px solid var(--panel-border)', paddingRight: '0.5rem' }}>
                    {tx.date ? new Date(tx.date).toISOString().split('T')[0].replace(/-/g, '/') : 'N/A'}
                  </span>
                  
                  <div className="flex-row items-center gap-sm text-xs">
                    {isRF ? (
                      <>
                        <span className="text-secondary">-- un. x --</span>
                        <span className="text-secondary">=</span>
                        <span className="font-bold">{formatMoney(tx.total_value, tx.currency || 'BRL')}</span>
                      </>
                    ) : !isSplit ? (
                      <>
                        <span className="text-secondary">{formatQuantity(tx.quantity || 0)} un. x {formatMoney(tx.unit_price || 0, tx.currency || 'BRL')}</span>
                        <span className="text-secondary">=</span>
                        <span className="font-bold">{formatMoney(tx.total_value, tx.currency || 'BRL')}</span>
                      </>
                    ) : (
                      <span className="font-bold">Fator: {isReverse ? `${formatQuantity(tx.quantity || 0)} para 1` : `1 para ${formatQuantity(tx.quantity || 0)}`}</span>
                    )}

                    {!isRF && (
                      <span className="text-secondary" style={{ marginLeft: '1rem', borderLeft: '1px solid var(--panel-border)', paddingLeft: '1rem', fontSize: '0.75rem' }}>
                        Saldo após: <span className="font-bold text-primary" style={{ color: '#00f2fe' }}>{formatQuantity(tx.resulting_quantity || 0)} un.</span>
                        <span className="text-secondary" style={{ marginLeft: '0.5rem', marginRight: '0.5rem' }}>|</span>
                        Investido: <span className="font-bold text-primary" style={{ color: '#00e676' }}>{formatMoney(tx.resulting_invested || 0, tx.currency || 'BRL')}</span>
                      </span>
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
