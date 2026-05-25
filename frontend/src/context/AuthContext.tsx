'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

export interface User {
  id: string;
  name: string;
  email: string;
  created_at: string;
  updated_at: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // Busca perfil do usuário logado na inicialização para restaurar sessão ativa
  const fetchMe = async (): Promise<boolean> => {
    try {
      const res = await fetch(`${API_URL}/auth/me`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Essencial para transmitir cookies HttpOnly
      });

      if (res.ok) {
        const data = await res.json();
        setUser(data);
        return true;
      }

      // Se o status for 401, tenta fazer o refresh silencioso
      if (res.status === 401) {
        const refreshSuccess = await handleRefresh();
        if (refreshSuccess) {
          // Se o refresh deu certo, tenta puxar o perfil /me novamente
          const retryRes = await fetch(`${API_URL}/auth/me`, {
            method: 'GET',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
          });
          if (retryRes.ok) {
            const data = await retryRes.json();
            setUser(data);
            return true;
          }
        }
      }

      setUser(null);
      return false;
    } catch (error) {
      console.error('Erro ao buscar perfil:', error);
      setUser(null);
      return false;
    }
  };

  // Realiza a renovação silenciosa do access_token usando o refresh_token
  const handleRefresh = async (): Promise<boolean> => {
    try {
      const res = await fetch(`${API_URL}/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });
      return res.ok;
    } catch (error) {
      console.error('Erro ao renovar sessão:', error);
      return false;
    }
  };

  useEffect(() => {
    const initAuth = async () => {
      await fetchMe();
      setIsLoading(false);
    };
    initAuth();
  }, []);

  const login = async (email: string, password: string) => {
    setIsLoading(true);
    try {
      const res = await fetch(`${API_URL}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
        credentials: 'include',
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || 'Falha ao efetuar login');
      }

      setUser(data);
      window.location.href = '/dashboard';
    } catch (error) {
      setIsLoading(false);
      throw error;
    }
  };

  const register = async (name: string, email: string, password: string) => {
    setIsLoading(true);
    try {
      const res = await fetch(`${API_URL}/auth/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name, email, password }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || 'Falha ao efetuar cadastro');
      }

      // Registro efetuado com sucesso. Vamos automaticamente logar o usuário para melhor UX.
      await login(email, password);
    } catch (error) {
      setIsLoading(false);
      throw error;
    }
  };

  const logout = async () => {
    setIsLoading(true);
    try {
      await fetch(`${API_URL}/auth/logout`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });
    } catch (error) {
      console.error('Erro ao efetuar logout no servidor:', error);
    } finally {
      setUser(null);
      setIsLoading(false);
      window.location.href = '/login';
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth deve ser usado dentro de um AuthProvider');
  }
  return context;
}
