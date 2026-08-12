import React, { useState, useEffect } from 'react';
import dynamic from 'next/dynamic';

const PortfolioChart = dynamic(() => import('@/components/PortfolioChart'), { ssr: false });
import { FixedIncomePosition, PerformancePoint } from './types';
import { formatMoney, formatPercentage } from './helpers';
import { apiFetch } from '@/lib/api';

interface FixedIncomeTabProps {
  portfolioId: string;
  onLaunchOperation: () => void;
}

export default function FixedIncomeTab({ portfolioId, onLaunchOperation }: FixedIncomeTabProps) {
  const [positions, setPositions] = useState<FixedIncomePosition[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [performanceData, setPerformanceData] = useState<PerformancePoint[]>([]);
  const [isLoadingPerformance, setIsLoadingPerformance] = useState(false);
  const [period, setPeriod] = useState<string>('ALL');

  // Estados para o modal de resgate
  const [redeemTarget, setRedeemTarget] = useState<FixedIncomePosition | null>(null);
  const [redeemAmount, setRedeemAmount] = useState<number | ''>('');
  const [redeemDate, setRedeemDate] = useState<string>(new Date().toISOString().split('T')[0]);
  const [isSubmittingRedeem, setIsSubmittingRedeem] = useState(false);

  type SortKey = 'institution' | 'rate' | 'start_date' | 'maturity_date' | 'total_invested' | 'gross_value' | 'net_value' | 'net_return_percent';
  type SortDir = 'asc' | 'desc';

  const [sortKey, setSortKey] = useState<SortKey>('institution');
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const sortedPositions = [...positions].sort((a, b) => {
    let aVal: number | string = 0;
    let bVal: number | string = 0;
    switch (sortKey) {
      case 'institution': aVal = a.asset?.institution || ''; bVal = b.asset?.institution || ''; break;
      case 'rate': aVal = a.asset?.rate ?? 0; bVal = b.asset?.rate ?? 0; break;
      case 'start_date': aVal = a.start_date || ''; bVal = b.start_date || ''; break;
      case 'maturity_date': aVal = a.asset?.maturity_date || ''; bVal = b.asset?.maturity_date || ''; break;
      case 'total_invested': aVal = a.total_invested ?? 0; bVal = b.total_invested ?? 0; break;
      case 'gross_value': aVal = a.gross_value ?? 0; bVal = b.gross_value ?? 0; break;
      case 'net_value': aVal = a.net_value ?? 0; bVal = b.net_value ?? 0; break;
      case 'net_return_percent': aVal = a.net_return_percent ?? 0; bVal = b.net_return_percent ?? 0; break;
    }
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortDir === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    }
    return sortDir === 'asc' ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
  });

  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return <span style={{ opacity: 0.3, marginLeft: '4px' }}>⇅</span>;
    return <span style={{ marginLeft: '4px' }}>{sortDir === 'asc' ? '↑' : '↓'}</span>;
  };

  const handleBulkImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    
    const formData = new FormData();
    formData.append("file", file);
    
    try {
      const res = await apiFetch(`/portfolios/${portfolioId}/fixed-income/bulk`, {
        method: "POST",
        body: formData
      });
      
      if (res.ok) {
        const data = await res.json();
        if (data.errors?.length > 0) {
          alert(`Importados com sucesso: ${data.success}\nFalhas:\n- ${data.errors.join("\n- ")}`);
        } else {
          alert(`Importação concluída com sucesso! ${data.success} registros importados.`);
        }
        window.location.reload(); // Recarrega para atualizar a carteira inteira
      } else {
        alert("Erro ao enviar arquivo.");
      }
    } catch (err) {
      alert("Erro de conexão.");
    }
    
    e.target.value = '';
  };

  useEffect(() => {
    if (!portfolioId) return;

    const fetchPositions = async () => {
      setIsLoading(true);
      try {
        const res = await apiFetch(`/portfolios/${portfolioId}/fixed-income/positions`);
        if (res.ok) {
          const data = await res.json();
          setPositions(data || []);
        } else {
          console.error("Failed to fetch fixed income positions");
        }
      } catch (err) {
        console.error("Error fetching fixed income positions:", err);
      } finally {
        setIsLoading(false);
      }
    };

    fetchPositions();
  }, [portfolioId]);

  useEffect(() => {
    if (!portfolioId) return;
    const fetchPerformance = async () => {
      setIsLoadingPerformance(true);
      try {
        const res = await apiFetch(`/portfolios/${portfolioId}/fixed-income/performance?period=${period}`);
        if (res.ok) {
          const data = await res.json();
          setPerformanceData(data || []);
        }
      } catch (err) {
        console.error("Error fetching fixed income performance:", err);
      } finally {
        setIsLoadingPerformance(false);
      }
    };
    fetchPerformance();
  }, [portfolioId, period]);

  const confirmRedeem = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!redeemTarget) return;
    if (!redeemAmount || Number(redeemAmount) <= 0) {
      alert("Informe um valor válido para o resgate.");
      return;
    }

    // Validação básica: não pode resgatar mais que o valor líquido atual (simplificação)
    // Na prática, o backend que dita a regra, mas ajuda no frontend
    if (Number(redeemAmount) > redeemTarget.net_value) {
      if (!confirm(`O valor solicitado (R$ ${Number(redeemAmount).toFixed(2)}) é maior que o saldo líquido atual (R$ ${redeemTarget.net_value.toFixed(2)}). Deseja prosseguir mesmo assim?`)) {
        return;
      }
    }

    setIsSubmittingRedeem(true);
    try {
      const res = await apiFetch(`/portfolios/${portfolioId}/fixed-income/assets/${redeemTarget.asset.id}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: 'REDEMPTION',
          amount: Number(redeemAmount),
          date: new Date(redeemDate).toISOString()
        })
      });

      if (res.ok) {
        alert('Resgate realizado com sucesso!');
        window.location.reload();
      } else {
        const data = await res.json();
        alert(`Erro ao resgatar: ${data.error || 'Erro desconhecido'}`);
      }
    } catch (e) {
      alert("Erro de conexão");
    } finally {
      setIsSubmittingRedeem(false);
      setRedeemTarget(null);
    }
  }

  if (isLoading) {
    return (
      <div className="flex-row items-center justify-center w-full" style={{ height: '300px' }}>
        <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 30, height: 30 }}></span>
      </div>
    );
  }

  return (
    <div className="flex-col gap-md" style={{ width: '100%' }}>
      
      <div className="card flex-col mb-lg" style={{ padding: '1.75rem 2rem', border: '1px solid var(--panel-border)' }}>
        <div className="flex-row justify-between items-center mb-md flex-wrap gap-md">
          <div>
            <h4 className="m-0" style={{ fontSize: '1.1rem' }}>📈 Evolução da Renda Fixa</h4>
            <p className="text-xs text-secondary mt-xs">Curva de juros compostos acumulada</p>
          </div>
          <div className="flex-row gap-sm" style={{ background: 'rgba(255,255,255,0.02)', padding: '0.2rem', borderRadius: '6px', border: '1px solid var(--panel-border)' }}>
            {['1M', '3M', '6M', '1Y', 'ALL'].map((p) => (
              <button key={p} onClick={() => setPeriod(p)} style={{ padding: '0.25rem 0.65rem', fontSize: '0.7rem', borderRadius: '4px', border: 'none', background: period === p ? 'var(--accent-gradient)' : 'transparent', color: period === p ? '#000' : 'var(--text-secondary)', cursor: 'pointer', fontWeight: 700 }}>
                {p}
              </button>
            ))}
          </div>
        </div>

        {isLoadingPerformance ? (
          <div className="flex-row items-center justify-center w-full" style={{ height: '300px' }}>
            <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 30, height: 30 }}></span>
          </div>
        ) : performanceData.length > 0 ? (
          <PortfolioChart data={performanceData} />
        ) : (
          <div className="flex-col items-center justify-center w-full text-secondary" style={{ height: '300px', border: '1px dashed var(--panel-border)', borderRadius: '12px' }}>
            <span className="text-2xl mb-sm">🏛️</span>
            <p className="text-sm m-0">Nenhum dado histórico de Renda Fixa no período selecionado.</p>
          </div>
        )}
      </div>

      <div className="card flex-col gap-md" style={{ flex: '2 1 600px', minHeight: '380px' }}>
        <div className="flex-row justify-between items-center mb-lg">
          <h3 className="card-title">🏛️ Posições de Renda Fixa</h3>
          <div className="flex-row gap-sm">
            <label className="btn-secondary" style={{ padding: '0.45rem 1rem', fontSize: '0.8rem', cursor: 'pointer' }}>
              📥 Importar CSV
              <input 
                type="file" accept=".csv" style={{ display: 'none' }}
                onClick={(e) => { (e.target as HTMLInputElement).value = ''; }}
                onChange={handleBulkImport} 
              />
            </label>
            <button className="primary-button" onClick={onLaunchOperation} style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}>
              + Nova Aplicação
            </button>
          </div>
        </div>
        <div className="table-container flex-col" style={{ flex: 1 }}>
        {positions.length > 0 ? (
          <table className="data-table" style={{ width: '100%' }}>
            <thead>
              <tr>
                <th style={{ cursor: 'pointer' }} onClick={() => handleSort('institution')}>Instituição / Produto {sortIcon('institution')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('rate')}>Taxa {sortIcon('rate')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('start_date')}>Aplicação {sortIcon('start_date')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('maturity_date')}>Vencimento {sortIcon('maturity_date')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('total_invested')}>Valor Aplicado {sortIcon('total_invested')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('gross_value')}>Valor Bruto {sortIcon('gross_value')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('net_value')}>Valor Líquido {sortIcon('net_value')}</th>
                <th className="text-right" style={{ cursor: 'pointer' }} onClick={() => handleSort('net_return_percent')}>Rent. (%) {sortIcon('net_return_percent')}</th>
                <th className="text-center">Ações</th>
              </tr>
            </thead>
            <tbody>
              {sortedPositions.map(pos => {
                const isMatured = pos.is_matured;
                const isZeroDate = pos.asset.maturity_date && pos.asset.maturity_date.startsWith('0001');
                const isNearMaturity = !isZeroDate && pos.days_to_maturity <= 30 && !isMatured;
                
                let rowStyle = {};
                let statusLabel = null;
                
                if (isMatured) {
                  rowStyle = { backgroundColor: 'var(--color-danger-bg)' };
                  statusLabel = <span className="text-xs ml-sm font-bold" style={{color: 'var(--color-danger)'}}>(Vencido)</span>;
                } else if (isNearMaturity) {
                  rowStyle = { backgroundColor: 'var(--color-warning-bg)' };
                  statusLabel = <span className="text-xs ml-sm font-bold" style={{color: 'var(--color-warning)'}}>(Vence em {pos.days_to_maturity}d)</span>;
                }

                let rateStr = '';
                if (pos.asset.debt_type === 'POS') {
                  rateStr = `${pos.asset.rate.toFixed(2)}% ${pos.asset.indexer}`;
                } else if (pos.asset.debt_type === 'HIBRIDO') {
                  rateStr = `${pos.asset.indexer} + ${pos.asset.rate.toFixed(2)}%`;
                } else {
                  rateStr = `${pos.asset.rate.toFixed(2)}% a.a.`;
                }

                return (
                  <tr key={pos.asset.id} style={rowStyle}>
                    <td>
                      <div className="flex-col">
                        <span className="font-bold text-accent">{pos.asset.institution}</span>
                        <div className="text-xs text-secondary flex-row items-center">
                          {pos.asset.type}
                          {statusLabel}
                        </div>
                      </div>
                    </td>
                    <td className="text-right"><span className="font-semibold text-primary" style={{ fontFamily: 'monospace' }}>{rateStr}</span></td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>
                      {pos.start_date ? new Date(pos.start_date).toLocaleDateString('pt-BR', {timeZone: 'UTC'}) : '--'}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>
                      {pos.asset.maturity_date && !pos.asset.maturity_date.startsWith('0001') ? new Date(pos.asset.maturity_date).toLocaleDateString('pt-BR', {timeZone: 'UTC'}) : '--'}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{formatMoney(pos.total_invested, 'BRL')}</td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{formatMoney(pos.gross_value, 'BRL')}</td>
                    <td className="text-right font-bold" style={{ fontFamily: 'monospace', color: 'var(--color-success)' }}>{formatMoney(pos.net_value, 'BRL')}</td>
                    <td className="text-right font-bold" style={{ color: pos.net_return_percent >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
                      {formatPercentage(pos.net_return_percent)}
                    </td>
                    <td className="text-center">
                      <button 
                        onClick={() => {
                          setRedeemTarget(pos);
                          setRedeemAmount(pos.net_value);
                          setRedeemDate(new Date().toISOString().split('T')[0]);
                        }} 
                        style={{ padding: '0.3rem 0.6rem', borderRadius: '4px', background: 'var(--color-danger-bg)', color: 'var(--color-danger)', border: '1px solid rgba(var(--danger-rgb), 0.3)', cursor: 'pointer', fontSize: '0.75rem', fontWeight: 'bold' }}
                      >
                        RESGATAR
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <div className="flex-col items-center justify-center text-secondary" style={{ height: '240px' }}>
            <span className="text-2xl mb-sm">🏛️</span>
            <p className="text-sm">Nenhuma aplicação de Renda Fixa encontrada.</p>
          </div>
        )}
        </div>
      </div>

      {redeemTarget && (
        <div className="modal-overlay" style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, backdropFilter: 'blur(4px)' }}>
          <div className="modal-content card" style={{ width: '100%', maxWidth: '400px', padding: '1.5rem' }}>
            <h3 className="card-title mb-md">💵 Resgatar Aplicação</h3>
            <p className="text-sm text-secondary mb-md">
              Você está resgatando recursos de <strong>{redeemTarget.asset.institution} ({redeemTarget.asset.type})</strong>.
              O saldo líquido atual é de <span className="font-bold text-success">{formatMoney(redeemTarget.net_value, 'BRL')}</span>.
            </p>

            <form onSubmit={confirmRedeem} className="flex-col gap-sm">
              <div className="form-group flex-col gap-xs">
                <label className="text-sm font-semibold">Valor do Resgate (R$)</label>
                <input 
                  type="number" step="0.01" min="0.01" max={redeemTarget.net_value + 1000000} // Permite um limite folgado
                  className="form-input" 
                  value={redeemAmount} 
                  onChange={e => setRedeemAmount(Number(e.target.value))} 
                  required 
                  disabled={isSubmittingRedeem}
                />
              </div>

              <div className="form-group flex-col gap-xs mt-sm">
                <label className="text-sm font-semibold">Data do Resgate</label>
                <input 
                  type="date" 
                  className="form-input" 
                  value={redeemDate} 
                  onChange={e => setRedeemDate(e.target.value)} 
                  required 
                  disabled={isSubmittingRedeem}
                />
              </div>

              <div className="flex-row justify-end gap-sm mt-lg">
                <button type="button" className="btn-secondary font-bold" onClick={() => setRedeemTarget(null)} disabled={isSubmittingRedeem} style={{ padding: '0.5rem 1rem' }}>
                  Cancelar
                </button>
                <button type="submit" className="primary-button font-bold" disabled={isSubmittingRedeem} style={{ padding: '0.5rem 1rem' }}>
                  {isSubmittingRedeem ? 'Processando...' : 'Confirmar Resgate'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
