'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/context/AuthContext';

export default function RegisterPage() {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { register } = useAuth();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (password.length < 6) {
      setError('A senha deve conter pelo menos 6 caracteres.');
      return;
    }

    setIsSubmitting(true);

    try {
      await register(name, email, password);
    } catch (err: any) {
      setError(err.message || 'Falha ao realizar cadastro. Tente outro e-mail.');
      setIsSubmitting(false);
    }
  };

  return (
    <main className="container">
      <div className="glass-panel auth-card">
        <h2>Criar Conta</h2>
        <p className="subtitle">Cadastre-se para gerenciar seus ativos e metas</p>

        {error && <div className="alert-error">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label" htmlFor="name">Nome Completo</label>
            <input
              className="form-input"
              type="text"
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Digite seu nome"
              required
              disabled={isSubmitting}
            />
          </div>

          <div className="form-group">
            <label className="form-label" htmlFor="email">E-mail</label>
            <input
              className="form-input"
              type="email"
              id="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="seu@email.com"
              required
              disabled={isSubmitting}
            />
          </div>

          <div className="form-group">
            <label className="form-label" htmlFor="password">Senha</label>
            <input
              className="form-input"
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Mínimo de 6 caracteres"
              required
              disabled={isSubmitting}
            />
          </div>

          <button
            className="primary-button w-full"
            type="submit"
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <span className="loading-spinner"></span>
                Cadastrando...
              </>
            ) : (
              'Criar Minha Conta'
            )}
          </button>
        </form>

        <div className="auth-footer">
          Já possui cadastro?{' '}
          <Link className="auth-link" href="/login">
            Faça login
          </Link>
        </div>
      </div>
    </main>
  );
}
