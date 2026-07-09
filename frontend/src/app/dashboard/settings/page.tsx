'use client';

import React, { useState, useEffect } from 'react';
import { useAuth } from '@/context/AuthContext';
import Link from 'next/link';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

interface TelegramStatus {
  linked: boolean;
  chat_id: number;
  bot_username: string;
}

export default function SettingsPage() {
  const { user, logout, fetchMe, isLoading: authLoading } = useAuth();

  // Estados do perfil
  const [profileName, setProfileName] = useState('');
  const [profileEmail, setProfileEmail] = useState('');
  const [isSavingProfile, setIsSavingProfile] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileSuccess, setProfileSuccess] = useState<string | null>(null);

  // Estados da senha
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isSavingPassword, setIsSavingPassword] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null);

  // Estados do Telegram
  const [telegramStatus, setTelegramStatus] = useState<TelegramStatus | null>(null);
  const [isLoadingTelegram, setIsLoadingTelegram] = useState(true);
  const [telegramError, setTelegramError] = useState<string | null>(null);

  // Inicialização dos campos do perfil
  useEffect(() => {
    if (user) {
      setProfileName(user.name);
      setProfileEmail(user.email);
    }
  }, [user]);

  // Carregar status do Telegram
  const loadTelegramStatus = async () => {
    setIsLoadingTelegram(true);
    try {
      const res = await fetch(`${API_URL}/telegram/status`, {
        credentials: 'include',
      });
      if (res.ok) {
        const data = await res.json();
        setTelegramStatus(data);
      } else {
        setTelegramError('Não foi possível carregar as informações do Telegram.');
      }
    } catch (err) {
      console.error(err);
      setTelegramError('Erro de conexão ao carregar Telegram.');
    } finally {
      setIsLoadingTelegram(false);
    }
  };

  useEffect(() => {
    loadTelegramStatus();
  }, []);

  if (authLoading) {
    return (
      <main className="container" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '80vh' }}>
        <div style={{ textAlign: 'center' }}>
          <span className="loading-spinner" style={{ borderTopColor: '#00f2fe', width: 40, height: 40 }}></span>
          <p style={{ marginTop: '1.5rem', color: 'var(--text-secondary)' }}>Carregando...</p>
        </div>
      </main>
    );
  }

  if (!user) return null;

  // Atualizar dados do perfil
  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setProfileError(null);
    setProfileSuccess(null);
    setIsSavingProfile(true);

    try {
      const res = await fetch(`${API_URL}/user/profile`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: profileName, email: profileEmail }),
        credentials: 'include',
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Erro ao atualizar perfil.');
      }

      await fetchMe(); // Atualiza dados globais do usuário
      setProfileSuccess('Perfil atualizado com sucesso!');
    } catch (err: any) {
      setProfileError(err.message || 'Erro ao conectar no servidor.');
    } finally {
      setIsSavingProfile(false);
    }
  };

  // Alterar senha
  const handleUpdatePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError(null);
    setPasswordSuccess(null);

    if (newPassword !== confirmPassword) {
      setPasswordError('A nova senha e a confirmação não coincidem.');
      return;
    }

    setIsSavingPassword(true);

    try {
      const res = await fetch(`${API_URL}/user/password`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
        credentials: 'include',
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Erro ao atualizar senha.');
      }

      setPasswordSuccess('Senha alterada com sucesso!');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: any) {
      setPasswordError(err.message || 'Erro ao conectar no servidor.');
    } finally {
      setIsSavingPassword(false);
    }
  };

  // Vincular Telegram
  const handleLinkTelegram = async () => {
    try {
      const res = await fetch(`${API_URL}/telegram/link`, {
        method: 'POST',
        credentials: 'include',
      });
      if (res.ok) {
        const data = await res.json();
        const botUsername = data.bot_username || 'StockPulseBot';
        window.open(`https://t.me/${botUsername}?start=${data.token}`, '_blank');
        
        // Polling rápido para atualizar o status do telegram
        let attempts = 0;
        const interval = setInterval(async () => {
          attempts++;
          const checkRes = await fetch(`${API_URL}/telegram/status`, { credentials: 'include' });
          if (checkRes.ok) {
            const checkData = await checkRes.json();
            if (checkData.linked) {
              setTelegramStatus(checkData);
              clearInterval(interval);
            }
          }
          if (attempts > 30) clearInterval(interval); // cancela após 30 segundos
        }, 1000);
      } else {
        alert('Erro ao gerar link de ativação.');
      }
    } catch (err) {
      console.error(err);
      alert('Erro ao comunicar com o servidor.');
    }
  };

  // Desvincular Telegram
  const handleUnlinkTelegram = async () => {
    if (!confirm('Deseja realmente desvincular o Telegram desta conta? Você deixará de receber alertas.')) {
      return;
    }

    try {
      const res = await fetch(`${API_URL}/telegram/link`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        loadTelegramStatus();
      } else {
        alert('Erro ao desvincular o Telegram.');
      }
    } catch (err) {
      console.error(err);
      alert('Erro ao comunicar com o servidor.');
    }
  };

  // Excluir Conta
  const handleDeleteAccount = async () => {
    const doubleConfirm = prompt(
      '⚠️ ATENÇÃO: Esta ação é irreversível e excluirá permanentemente todos os seus dados, incluindo carteiras, transações e alertas. Para confirmar, digite seu e-mail abaixo:'
    );

    if (doubleConfirm !== user.email) {
      if (doubleConfirm !== null) {
        alert('Confirmação inválida. O e-mail digitado não coincide.');
      }
      return;
    }

    try {
      const res = await fetch(`${API_URL}/user`, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (res.ok) {
        alert('Sua conta foi excluída com sucesso. Lamentamos ver você partir.');
        logout();
      } else {
        const data = await res.json();
        alert(data.error || 'Erro ao excluir a conta.');
      }
    } catch (err) {
      console.error(err);
      alert('Erro de rede ao processar exclusão.');
    }
  };

  return (
    <main className="container" style={{ maxWidth: 900 }}>
      {/* Header */}
      <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', borderBottom: '1px solid var(--panel-border)', paddingBottom: '1.25rem', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '2.3rem', background: 'var(--accent-gradient)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent', margin: 0, fontWeight: 800 }}>stock-pulse</h1>
          
          <div style={{ display: 'flex', gap: '1.5rem', marginTop: '0.8rem' }}>
            <Link href="/dashboard/portfolio" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              💼 Minha Carteira
            </Link>
            <Link href="/dashboard" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              📊 Monitoramento
            </Link>
            <Link href="/dashboard/alerts" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '5px' }}>
              🔔 Meus Alertas
            </Link>
            <Link href="/dashboard/settings" style={{ color: 'var(--accent-color)', textDecoration: 'none', fontSize: '0.85rem', fontWeight: 700, borderBottom: '2px solid var(--accent-color)', paddingBottom: '3px', display: 'flex', alignItems: 'center', gap: '5px' }}>
              ⚙️ Configurações
            </Link>
          </div>
        </div>
        
        <div style={{ display: 'flex', alignItems: 'center', gap: '1.25rem' }}>
          <div style={{ textAlign: 'right', fontSize: '0.8rem' }}>
            <span style={{ display: 'block', fontWeight: 600 }}>{user.name}</span>
            <span style={{ color: 'var(--text-secondary)', fontSize: '0.7rem' }}>Sessão Segura</span>
          </div>
          <button className="btn-secondary" onClick={logout} style={{ padding: '0.5rem 1.25rem', fontSize: '0.85rem' }}>
            Sair
          </button>
        </div>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
        
        {/* Seção 1: Dados Cadastrais */}
        <section className="card">
          <div className="card-header">
            <h2 className="card-title">👤 Perfil do Usuário</h2>
          </div>
          <form onSubmit={handleUpdateProfile} style={{ display: 'flex', flexDirection: 'column', gap: '1.2rem' }}>
            {profileError && <div className="alert-error">{profileError}</div>}
            {profileSuccess && <div style={{ color: '#00e676', fontSize: '0.9rem', padding: '0.5rem 0' }}>{profileSuccess}</div>}

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Nome Completo</label>
              <input 
                type="text" 
                className="form-input" 
                value={profileName} 
                onChange={(e) => setProfileName(e.target.value)} 
                required 
              />
            </div>

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Endereço de E-mail</label>
              <input 
                type="email" 
                className="form-input" 
                value={profileEmail} 
                onChange={(e) => setProfileEmail(e.target.value)} 
                required 
              />
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '0.5rem' }}>
              <button type="submit" className="primary-button" disabled={isSavingProfile}>
                {isSavingProfile ? 'Salvando...' : 'Salvar Alterações'}
              </button>
            </div>
          </form>
        </section>

        {/* Seção 2: Alterar Senha */}
        <section className="card">
          <div className="card-header">
            <h2 className="card-title">🔒 Segurança da Conta</h2>
          </div>
          <form onSubmit={handleUpdatePassword} style={{ display: 'flex', flexDirection: 'column', gap: '1.2rem' }}>
            {passwordError && <div className="alert-error">{passwordError}</div>}
            {passwordSuccess && <div style={{ color: '#00e676', fontSize: '0.9rem', padding: '0.5rem 0' }}>{passwordSuccess}</div>}

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Senha Atual</label>
              <input 
                type="password" 
                className="form-input" 
                placeholder="••••••••" 
                value={currentPassword} 
                onChange={(e) => setCurrentPassword(e.target.value)} 
                required 
              />
            </div>

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Nova Senha</label>
              <input 
                type="password" 
                className="form-input" 
                placeholder="Mínimo 6 caracteres" 
                value={newPassword} 
                onChange={(e) => setNewPassword(e.target.value)} 
                required 
              />
            </div>

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Confirmar Nova Senha</label>
              <input 
                type="password" 
                className="form-input" 
                placeholder="Repita a nova senha" 
                value={confirmPassword} 
                onChange={(e) => setConfirmPassword(e.target.value)} 
                required 
              />
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '0.5rem' }}>
              <button type="submit" className="primary-button" disabled={isSavingPassword}>
                {isSavingPassword ? 'Atualizando...' : 'Alterar Senha'}
              </button>
            </div>
          </form>
        </section>

        {/* Seção 3: Integrações */}
        <section className="card">
          <div className="card-header">
            <h2 className="card-title">📱 Canal de Alertas (Telegram)</h2>
          </div>
          {isLoadingTelegram ? (
            <div style={{ padding: '1rem', textAlign: 'center' }}>
              <span className="loading-spinner" style={{ borderTopColor: '#00f2fe' }}></span>
              <span style={{ marginLeft: '10px', color: 'var(--text-secondary)' }}>Carregando integrações...</span>
            </div>
          ) : telegramError ? (
            <div className="alert-error">{telegramError}</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', margin: 0, textAlign: 'left' }}>
                Vincule sua conta com o bot oficial do **Stock Pulse** no Telegram para receber alertas automáticos de preços e variações em tempo real.
              </p>

              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '1rem', background: 'rgba(255,255,255,0.02)', borderRadius: '10px', border: '1px solid var(--panel-border)', marginTop: '0.5rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
                  <span style={{ fontSize: '2rem' }}>📱</span>
                  <div style={{ textAlign: 'left' }}>
                    <span style={{ display: 'block', fontWeight: 700, fontSize: '0.95rem' }}>Stock Pulse Telegram Bot</span>
                    <span style={{ color: telegramStatus?.linked ? '#00e676' : 'var(--text-secondary)', fontSize: '0.8rem', fontWeight: 600 }}>
                      {telegramStatus?.linked ? `Vinculado com sucesso (Chat ID: ${telegramStatus.chat_id})` : 'Não Vinculado'}
                    </span>
                  </div>
                </div>

                <div>
                  {telegramStatus?.linked ? (
                    <button className="btn-danger" onClick={handleUnlinkTelegram}>
                      Desvincular
                    </button>
                  ) : (
                    <button className="primary-button" onClick={handleLinkTelegram} style={{ padding: '0.5rem 1.25rem', fontSize: '0.85rem' }}>
                      Vincular Conta
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}
        </section>

        {/* Seção 4: Zona de Perigo */}
        <section className="card" style={{ borderColor: 'rgba(255, 74, 90, 0.3)', background: 'rgba(255, 74, 90, 0.02)' }}>
          <div className="card-header" style={{ borderColor: 'rgba(255, 74, 90, 0.2)' }}>
            <h2 className="card-title" style={{ color: '#ff4a5a' }}>⚠️ Zona de Perigo</h2>
          </div>
          <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'space-between', alignItems: 'center', gap: '1rem' }}>
            <div style={{ textAlign: 'left', flex: 1, minWidth: '250px' }}>
              <span style={{ display: 'block', fontWeight: 700, fontSize: '0.95rem' }}>Excluir Conta Permanentemente</span>
              <span style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                A exclusão é irreversível e remove todos os seus dados da plataforma.
              </span>
            </div>
            <button className="btn-danger" onClick={handleDeleteAccount} style={{ padding: '0.6rem 1.5rem' }}>
              Excluir Conta
            </button>
          </div>
        </section>

      </div>
    </main>
  );
}
