import React from 'react';
import { Quote } from './types';

interface CreateAlertModalProps {
  activeQuote: Quote;
  alertTargetPrice: string;
  alertCondition: 'ABOVE' | 'BELOW';
  isCreatingAlert: boolean;
  alertErrorMsg: string | null;
  alertSuccessMsg: string | null;
  onTargetPriceChange: (val: string) => void;
  onConditionChange: (val: 'ABOVE' | 'BELOW') => void;
  onSubmit: (e: React.FormEvent) => void;
  onClose: () => void;
}

export default function CreateAlertModal({
  activeQuote,
  alertTargetPrice,
  alertCondition,
  isCreatingAlert,
  alertErrorMsg,
  alertSuccessMsg,
  onTargetPriceChange,
  onConditionChange,
  onSubmit,
  onClose,
}: CreateAlertModalProps) {
  return (
    <div style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      backgroundColor: 'rgba(0,0,0,0.7)',
      backdropFilter: 'blur(8px)',
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      zIndex: 100,
      padding: '1rem'
    }}>
      <div className="glass-panel" style={{
        maxWidth: '450px',
        width: '100%',
        padding: '2.5rem',
        textAlign: 'left',
        boxShadow: '0 20px 50px rgba(0,0,0,0.8)',
        border: '1px solid rgba(255,255,255,0.08)'
      }}>
        <h2 style={{ margin: '0 0 0.5rem 0', fontSize: '1.6rem', fontWeight: 800, color: '#fff' }}>
          🔔 Criar Alerta de Preço
        </h2>
        <p style={{ margin: '0 0 2rem 0', color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
          Defina as regras para o ativo <strong>{activeQuote.symbol}</strong> ({activeQuote.name}).
        </p>

        {alertErrorMsg && (
          <div className="alert-error" style={{ marginBottom: '1.5rem', margin: 0 }}>
            ⚠️ {alertErrorMsg}
          </div>
        )}

        {alertSuccessMsg ? (
          <div style={{
            background: 'rgba(0, 230, 118, 0.08)',
            border: '1px solid #00e676',
            borderRadius: '10px',
            padding: '1.2rem',
            color: '#e2e8f0',
            fontSize: '0.9rem',
            lineHeight: 1.5,
            marginBottom: '1rem'
          }}>
            🎉 {alertSuccessMsg}
          </div>
        ) : (
          <form onSubmit={onSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
            
            {/* Condição */}
            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 600, fontSize: '0.8rem', textTransform: 'uppercase', color: 'var(--accent-color)' }}>
                Condição de Disparo
              </label>
              <select
                className="form-input"
                value={alertCondition}
                onChange={(e) => onConditionChange(e.target.value as 'ABOVE' | 'BELOW')}
                style={{ background: '#111827', width: '100%', padding: '0.6rem 0.9rem' }}
              >
                <option value="ABOVE">Preço sobe acima de (▲)</option>
                <option value="BELOW">Preço cai abaixo de (▼)</option>
              </select>
            </div>

            {/* Preço Alvo */}
            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 600, fontSize: '0.8rem', textTransform: 'uppercase', color: 'var(--accent-color)' }}>
                Preço Alvo ({activeQuote.currency})
              </label>
              <input
                className="form-input"
                type="number"
                step="0.01"
                required
                value={alertTargetPrice}
                onChange={(e) => onTargetPriceChange(e.target.value)}
                placeholder="Ex: 38.50"
                style={{ width: '100%', fontSize: '1.1rem', padding: '0.6rem 0.9rem' }}
              />
            </div>

            {/* Botões */}
            <div style={{ display: 'flex', gap: '1rem', marginTop: '1rem' }}>
              <button
                className="primary-button"
                type="submit"
                disabled={isCreatingAlert}
                style={{ flex: 1, padding: '0.8rem', fontSize: '0.9rem', background: 'linear-gradient(135deg, #00f2fe, #4facfe)', color: '#0b0f19', fontWeight: 700 }}
              >
                {isCreatingAlert ? 'Criando...' : 'Salvar Alerta'}
              </button>
              <button
                className="primary-button"
                type="button"
                onClick={onClose}
                style={{ flex: 1, padding: '0.8rem', fontSize: '0.9rem', background: 'rgba(255,255,255,0.05)', color: '#fff', border: '1px solid var(--panel-border)' }}
              >
                Cancelar
              </button>
            </div>

          </form>
        )}

      </div>
    </div>
  );
}
