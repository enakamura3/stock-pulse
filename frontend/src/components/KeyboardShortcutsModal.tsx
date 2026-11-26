'use client';

import React from 'react';

export interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

interface ShortcutItem {
  keys: string;
  description: string;
}

interface ShortcutGroup {
  title: string;
  items: ShortcutItem[];
}

const SHORTCUT_GROUPS: ShortcutGroup[] = [
  {
    title: 'Navegação Global',
    items: [
      { keys: 'Alt + 1', description: 'Minha Carteira' },
      { keys: 'Alt + 2', description: 'Monitoramento' },
      { keys: 'Alt + 3', description: 'Meus Alertas' },
      { keys: 'Alt + 4', description: 'Configurações' },
    ],
  },
  {
    title: 'Sub-módulos da Carteira (em Minha Carteira)',
    items: [
      { keys: '1', description: 'Renda Variável' },
      { keys: '2', description: 'Renda Fixa' },
      { keys: '3', description: 'Tesouro Direto' },
      { keys: '4', description: 'Histórico de Operações' },
      { keys: '5', description: 'Proventos' },
      { keys: '6', description: 'Análise da Carteira' },
      { keys: '7', description: 'Resumo Diário' },
    ],
  },
  {
    title: 'Geral',
    items: [
      { keys: '?', description: 'Abrir / fechar este painel de atalhos' },
      { keys: 'ESC', description: 'Fechar modais e painéis abertos' },
    ],
  },
];

/**
 * Modal de atalhos de teclado globais e contextuais do workbench.
 */
export default function KeyboardShortcutsModal({ isOpen, onClose }: KeyboardShortcutsModalProps) {
  if (!isOpen) return null;

  return (
    <div
      className="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-label="Atalhos de Teclado"
      onClick={onClose}
    >
      <div
        className="modal-content"
        style={{ maxWidth: '520px', maxHeight: '85vh', overflowY: 'auto' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <div className="flex-row items-center gap-xs">
            <span className="font-mono text-accent font-bold">&gt;_</span>
            <h2 className="modal-title m-0">Atalhos de Teclado</h2>
          </div>
          <button
            type="button"
            className="btn-close"
            aria-label="Fechar atalhos de teclado"
            onClick={onClose}
          >
            ✕
          </button>
        </div>

        <div className="flex-col gap-md">
          {SHORTCUT_GROUPS.map((group) => (
            <section key={group.title} className="flex-col gap-xs">
              <h3
                className="font-mono text-muted uppercase m-0"
                style={{ fontSize: '0.68rem', letterSpacing: '0.08em' }}
              >
                {`// ${group.title}`}
              </h3>
              <div className="flex-col" style={{ border: '1px solid var(--panel-border)', borderRadius: '2px' }}>
                {group.items.map((item, idx) => (
                  <div
                    key={item.keys}
                    className="flex-row items-center justify-between"
                    style={{
                      padding: '0.45rem 0.7rem',
                      borderBottom: idx < group.items.length - 1 ? '1px solid var(--panel-border)' : 'none',
                      fontSize: '0.78rem',
                    }}
                  >
                    <span className="text-secondary">{item.description}</span>
                    <kbd
                      className="font-mono text-accent font-bold"
                      style={{
                        background: 'var(--input-bg)',
                        border: '1px solid var(--panel-border-strong)',
                        borderRadius: '2px',
                        padding: '0.12rem 0.45rem',
                        fontSize: '0.72rem',
                      }}
                    >
                      {item.keys}
                    </kbd>
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>

        <div className="flex-row justify-end mt-md pt-sm" style={{ borderTop: '1px solid var(--panel-border)' }}>
          <button
            type="button"
            className="btn-secondary font-mono"
            onClick={onClose}
            style={{ fontSize: '0.76rem', padding: '0.35rem 0.8rem' }}
          >
            Fechar [ESC]
          </button>
        </div>
      </div>
    </div>
  );
}
