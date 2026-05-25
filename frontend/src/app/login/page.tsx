'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/context/AuthContext';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { login } = useAuth();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      await login(email, password);
    } catch (err: any) {
      setError(err.message || 'E-mail ou senha incorretos.');
      setIsSubmitting(false);
    }
  };

  return (
    <main className="container">
      <div className="glass-panel auth-card">
        <h2>stock-pulse</h2>
        <p className="subtitle">Entre para acompanhar seus investimentos em tempo real</p>

        {error && <div className="alert-error">{error}</div>}

        <form onSubmit={handleSubmit}>
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
              placeholder="Digite sua senha"
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
                Verificando...
              </>
            ) : (
              'Entrar no Dashboard'
            )}
          </button>
        </form>

        <div className="auth-footer">
          Não tem uma conta?{' '}
          <Link className="auth-link" href="/register">
            Cadastre-se grátis
          </Link>
        </div>
      </div>
    </main>
  );
}
