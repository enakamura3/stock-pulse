import React, { useState, useEffect } from 'react';
import { FixedIncomePosition } from './types';
import { formatMoney, formatPercentage } from './helpers';

interface FixedIncomeTabProps {
  portfolioId: string;
  onLaunchOperation: () => void;
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export default function FixedIncomeTab({ portfolioId, onLaunchOperation }: FixedIncomeTabProps) {
  const [positions, setPositions] = useState<FixedIncomePosition[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Estados para o modal de resgate
  const [redeemTarget, setRedeemTarget] = useState<FixedIncomePosition | null>(null);
  const [redeemAmount, setRedeemAmount] = useState<number | ''>('');
  const [redeemDate, setRedeemDate] = useState<string>(new Date().toISOString().split('T')[0]);
  const [isSubmittingRedeem, setIsSubmittingRedeem] = useState(false);

  useEffect(() => {
    if (!portfolioId) return;

    const fetchPositions = async () => {
      setIsLoading(true);
      try {
        const res = await fetch(`${API_URL}/portfolios/${portfolioId}/fixed-income/positions`, {
          credentials: 'include',
          cache: 'no-store',
        });
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
      const res = await fetch(`${API_URL}/portfolios/${portfolioId}/fixed-income/assets/${redeemTarget.asset.id}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: 'REDEMPTION',
          amount: Number(redeemAmount),
          date: new Date(redeemDate).toISOString()
        }),
        credentials: 'include'
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
    <div className="card flex-col gap-md" style={{ flex: '2 1 600px', minHeight: '380px' }}>
      <div className="flex-row justify-between items-center mb-lg">
        <h3 className="card-title">🏛️ Posições de Renda Fixa</h3>
        <button className="primary-button" onClick={onLaunchOperation} style={{ padding: '0.45rem 1rem', fontSize: '0.8rem' }}>
          + Nova Aplicação
        </button>
      </div>
      
      <div className="table-container flex-col" style={{ flex: 1 }}>
        {positions.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>Instituição / Produto</th>
                <th>Taxa</th>
                <th>Aplicação</th>
                <th>Vencimento</th>
                <th className="text-right">Valor Aplicado</th>
                <th className="text-right">Valor Bruto</th>
                <th className="text-right">Valor Líquido</th>
                <th className="text-right">Rent. (%)</th>
                <th className="text-center">Ações</th>
              </tr>
            </thead>
            <tbody>
              {positions.map(pos => {
                const isMatured = pos.is_matured;
                const isZeroDate = pos.asset.maturity_date && pos.asset.maturity_date.startsWith('0001');
                const isNearMaturity = !isZeroDate && pos.days_to_maturity <= 30 && !isMatured;
                
                let rowStyle = {};
                let statusLabel = null;
                
                if (isMatured) {
                  rowStyle = { backgroundColor: 'rgba(255, 60, 60, 0.05)' };
                  statusLabel = <span className="text-xs ml-sm font-bold" style={{color: '#ff4d4f'}}>(Vencido)</span>;
                } else if (isNearMaturity) {
                  rowStyle = { backgroundColor: 'rgba(250, 173, 20, 0.05)' };
                  statusLabel = <span className="text-xs ml-sm font-bold" style={{color: '#faad14'}}>(Vence em {pos.days_to_maturity}d)</span>;
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
                    <td><span className="font-semibold text-primary">{rateStr}</span></td>
                    <td style={{ fontFamily: 'monospace' }}>
                      {pos.start_date ? new Date(pos.start_date).toLocaleDateString('pt-BR', {timeZone: 'UTC'}) : '--'}
                    </td>
                    <td style={{ fontFamily: 'monospace' }}>
                      {pos.asset.maturity_date && !pos.asset.maturity_date.startsWith('0001') ? new Date(pos.asset.maturity_date).toLocaleDateString('pt-BR', {timeZone: 'UTC'}) : '--'}
                    </td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{formatMoney(pos.total_invested, 'BRL')}</td>
                    <td className="text-right" style={{ fontFamily: 'monospace' }}>{formatMoney(pos.gross_value, 'BRL')}</td>
                    <td className="text-right font-bold" style={{ fontFamily: 'monospace', color: '#00e676' }}>{formatMoney(pos.net_value, 'BRL')}</td>
                    <td className="text-right font-bold" style={{ color: pos.net_return_percent >= 0 ? '#00e676' : '#ff3d00' }}>
                      {formatPercentage(pos.net_return_percent)}
                    </td>
                    <td className="text-center">
                      <button 
                        onClick={() => {
                          setRedeemTarget(pos);
                          setRedeemAmount(pos.net_value);
                          setRedeemDate(new Date().toISOString().split('T')[0]);
                        }} 
                        style={{ padding: '0.3rem 0.6rem', borderRadius: '4px', background: 'rgba(255, 61, 0, 0.1)', color: '#ff3d00', border: '1px solid rgba(255, 61, 0, 0.3)', cursor: 'pointer', fontSize: '0.75rem', fontWeight: 'bold' }}
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
