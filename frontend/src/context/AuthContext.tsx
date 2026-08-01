'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/api';

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
  fetchMe: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // Busca perfil do usuário logado na inicialização para restaurar sessão ativa
  const fetchMe = async (): Promise<boolean> => {
    try {
      const data = await apiFetch<User>('/auth/me');
      setUser(data);
      return true;
    } catch (error) {
      console.error('Erro ao buscar perfil:', error);
      setUser(null);
      return false;
    }
  };

  // handleRefresh foi migrado para o lib/api.ts (attemptRefresh)

  const initRef = React.useRef(false);

  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;

    const initAuth = async () => {
      await fetchMe();
      setIsLoading(false);
    };
    initAuth();
  }, []);

  const login = async (email: string, password: string) => {
    setIsLoading(true);
    try {
      await apiFetch('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });
      await fetchMe();
      router.push('/dashboard');
    } catch (error: any) {
      setIsLoading(false);
      throw error;
    }
  };

  const register = async (name: string, email: string, password: string) => {
    setIsLoading(true);
    try {
      await apiFetch('/auth/register', {
        method: 'POST',
        body: JSON.stringify({ name, email, password }),
      });

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
      await apiFetch('/auth/logout', { method: 'POST' });
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
        fetchMe,
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
