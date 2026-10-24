'use client';

import React, { useState, useEffect } from 'react';
import { useAuth } from '@/context/AuthContext';
import AppSidebar from '@/components/AppSidebar';
import { apiFetch } from '@/lib/api';
import { SettingsIcon } from '@/components/ui/icons';
import IdeStatusBar from '@/components/ui/IdeStatusBar';
import { ASSET_TYPE_OPTIONS } from '@/components/portfolio/helpers';

interface TelegramStatus {
  linked: boolean;
  chat_id: number;
  bot_username: string;
}

interface AdminAssetInfo {
  id: string;
  ticker: string;
  name: string;
  asset_type: string;
  currency: string;
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

  // Estados dos Workers
  interface WorkerInfo {
    name: string;
    description: string;
    last_run: string | null;
    next_run: string | null;
    status: string;
    interval: string;
  }
  const [workers, setWorkers] = useState<WorkerInfo[]>([]);
  const [isLoadingWorkers, setIsLoadingWorkers] = useState(true);
  const [workersError, setWorkersError] = useState<string | null>(null);
  const [triggeringWorker, setTriggeringWorker] = useState<string | null>(null);
  const [workerSuccess, setWorkerSuccess] = useState<string | null>(null);
  const [adminKey, setAdminKey] = useState<string>('');

  // Estados do Catálogo de Ativos (Admin)
  const [adminAssets, setAdminAssets] = useState<AdminAssetInfo[]>([]);
  const [isLoadingAssets, setIsLoadingAssets] = useState(true);
  const [assetsError, setAssetsError] = useState<string | null>(null);
  const [assetsSuccess, setAssetsSuccess] = useState<string | null>(null);
  const [savingAssetId, setSavingAssetId] = useState<string | null>(null);
  const [assetSearchFilter, setAssetSearchFilter] = useState('');

  // Estados de exclusão de conta
  const [deletePassword, setDeletePassword] = useState('');

  const getAdminHeaders = (keyToUse?: string) => {
    const key = keyToUse !== undefined ? keyToUse : adminKey;
    const headers: Record<string, string> = {};
    if (key.trim()) {
      headers['X-Admin-Key'] = key.trim();
    }
    return headers;
  };

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
      const res = await apiFetch(`/telegram/status`);
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

  const loadWorkers = async (keyToUse?: string) => {
    setIsLoadingWorkers(true);
    setWorkersError(null);
    try {
      const headers = getAdminHeaders(keyToUse);
      const res = await apiFetch(`/workers`, { headers });
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data)) {
          data.sort((a: WorkerInfo, b: WorkerInfo) => a.name.localeCompare(b.name));
          setWorkers(data);
        } else {
          setWorkersError('Formato de resposta de workers inválido.');
        }
      } else {
        const data = await res.json().catch(() => ({}));
        setWorkersError(data.error || data.message || 'Não foi possível carregar a lista de workers.');
      }
    } catch (err) {
      console.error(err);
      setWorkersError('Erro de conexão ao carregar workers.');
    } finally {
      setIsLoadingWorkers(false);
    }
  };

  const loadAdminAssets = async (keyToUse?: string) => {
    setIsLoadingAssets(true);
    setAssetsError(null);
    try {
      const headers = getAdminHeaders(keyToUse);
      const res = await apiFetch(`/admin/assets`, { headers });
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data)) {
          setAdminAssets(data);
        } else {
          setAdminAssets([]);
        }
      } else {
        const data = await res.json().catch(() => ({}));
        setAssetsError(data.error || data.message || 'Não foi possível carregar o catálogo de ativos.');
      }
    } catch (err) {
      console.error(err);
      setAssetsError('Erro de conexão ao carregar catálogo de ativos.');
    } finally {
      setIsLoadingAssets(false);
    }
  };

  const handleUpdateAssetType = async (assetId: string, ticker: string, newType: string) => {
    setSavingAssetId(assetId);
    setAssetsError(null);
    setAssetsSuccess(null);
    try {
      const headers = {
        ...getAdminHeaders(),
        'Content-Type': 'application/json',
      };
      const res = await apiFetch(`/admin/assets/${assetId}`, {
        method: 'PATCH',
        headers,
        body: JSON.stringify({ asset_type: newType }),
      });
      if (res.ok) {
        setAdminAssets((prev) =>
          prev.map((a) => (a.id === assetId ? { ...a, asset_type: newType } : a))
        );
        setAssetsSuccess(`Categoria de "${ticker}" atualizada para ${newType}!`);
      } else {
        const data = await res.json().catch(() => ({}));
        setAssetsError(data.error || data.message || `Não foi possível atualizar o ativo "${ticker}".`);
      }
    } catch (err) {
      console.error(err);
      setAssetsError(`Erro de rede ao atualizar o ativo "${ticker}".`);
    } finally {
      setSavingAssetId(null);
    }
  };

  const handleTriggerWorker = async (name: string) => {
    setTriggeringWorker(name);
    setWorkerSuccess(null);
    setWorkersError(null);
    try {
      const headers = getAdminHeaders();
      const res = await apiFetch(`/workers/${name}/trigger`, {
        method: 'POST',
        headers,
      });
      if (res.ok) {
        setWorkerSuccess(`Worker "${name}" disparado com sucesso!`);
        setTimeout(() => loadWorkers(), 1500);
      } else {
        const data = await res.json().catch(() => ({}));
        setWorkersError(data.error || data.message || `Não foi possível disparar o worker "${name}".`);
      }
    } catch (err) {
      console.error(err);
      setWorkersError(`Erro de rede ao disparar o worker "${name}".`);
    } finally {
      setTriggeringWorker(null);
    }
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return 'Nunca';
    try {
      const date = new Date(dateStr);
      if (isNaN(date.getTime())) return 'Nunca';
      return date.toLocaleString('pt-BR', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }).replace(', ', ' ');
    } catch (err) {
      return 'Nunca';
    }
  };

  useEffect(() => {
    loadTelegramStatus();
    loadWorkers();
    loadAdminAssets();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (authLoading) {
    return (
      <main className="container" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '80vh' }}>
        <div style={{ textAlign: 'center' }}>
          <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)', width: 40, height: 40 }}></span>
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
      const res = await apiFetch(`/user/profile`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: profileName, email: profileEmail }),
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
      const res = await apiFetch(`/user/password`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
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
      const res = await apiFetch(`/telegram/link`, {
        method: 'POST',
      });
      if (res.ok) {
        const data = await res.json();
        const botUsername = data.bot_username || 'StockPulseBot';
        window.open(`https://t.me/${botUsername}?start=${data.token}`, '_blank');
        
        // Polling rápido para atualizar o status do telegram
        let attempts = 0;
        const interval = setInterval(async () => {
          attempts++;
          const checkRes = await apiFetch(`/telegram/status`, { });
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
      const res = await apiFetch(`/telegram/link`, {
        method: 'DELETE',
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
    if (!user) return;
    if (!deletePassword.trim()) {
      alert('Por favor, informe sua senha atual para confirmar a exclusão da conta.');
      return;
    }

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
      const res = await apiFetch('/user', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ current_password: deletePassword }),
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
    <div className="app-layout">
      <AppSidebar userName={user.name} onLogout={logout} />

      <main className="app-main-content">
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem', maxWidth: '1200px', width: '100%' }}>
          
          {/* Terminal Telemetry Header */}
          <div className="flex-row justify-between items-center" style={{ borderBottom: '1px solid var(--panel-border)', paddingBottom: '0.85rem' }}>
            <div>
              <div className="flex-row items-center gap-xs font-mono text-muted text-xs mb-xs">
                <span className="text-accent">TERMINAL</span>
                <span>/</span>
                <span>SETTINGS</span>
                <span>/</span>
                <span className="text-primary font-semibold">PREFERENCES</span>
              </div>
              <h1 className="font-mono text-xl font-bold" style={{ margin: 0, letterSpacing: '-0.02em', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <SettingsIcon size={20} className="text-accent" />
                Configurações do Sistema
              </h1>
            </div>
            <div className="flex-row items-center gap-sm">
              <span className="badge badge-neutral font-mono text-xs hide-mobile">
                [ USER_ID: {user.id ? user.id.slice(0, 8) : 'ACTIVE'} ]
              </span>
            </div>
          </div>

          {/* Seção 1: Dados Cadastrais */}
          <section className="card flex-col gap-0" style={{ width: '100%', padding: 0, overflow: 'hidden' }}>
            <div className="terminal-bar">
              <div className="flex-row items-center gap-xs font-mono">
                <span className="text-accent" style={{ fontSize: '0.8rem', fontWeight: 'bold' }}>▸</span>
                <span style={{ color: 'var(--text-secondary)' }}>settings/profile.identity</span>
              </div>
              <span className="badge badge-neutral font-mono" style={{ fontSize: '0.65rem' }}>
                [ IDENTITY ]
              </span>
            </div>
            <div style={{ padding: '1.25rem' }}>
              <div className="flex-row justify-between items-center mb-md">
                <h2 className="card-title font-mono" style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.95rem', margin: 0 }}>
                  <span className="text-muted" style={{ opacity: 0.5 }}>{'//'}</span>
                  <span>👤 Perfil do Usuário</span>
                </h2>
              </div>
              <form onSubmit={handleUpdateProfile} style={{ display: 'flex', flexDirection: 'column', gap: '1.2rem' }}>
                {profileError && <div className="alert-error">{profileError}</div>}
                {profileSuccess && <div style={{ color: 'var(--color-success)', fontSize: '0.9rem', padding: '0.5rem 0' }}>{profileSuccess}</div>}

                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label font-mono">Nome Completo</label>
                  <input 
                    type="text" 
                    className="form-input font-mono" 
                    value={profileName} 
                    onChange={(e) => setProfileName(e.target.value)} 
                    required 
                  />
                </div>

                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label font-mono">Endereço de E-mail</label>
                  <input 
                    type="email" 
                    className="form-input font-mono" 
                    value={profileEmail} 
                    onChange={(e) => setProfileEmail(e.target.value)} 
                    required 
                  />
                </div>

                <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '0.5rem' }}>
                  <button type="submit" className="primary-button font-mono" disabled={isSavingProfile}>
                    {isSavingProfile ? 'Salvando...' : 'Salvar Alterações'}
                  </button>
                </div>
              </form>
            </div>
          </section>

          {/* Seção 2: Alterar Senha */}
          <section className="card flex-col gap-0" style={{ width: '100%', padding: 0, overflow: 'hidden' }}>
            <div className="terminal-bar">
              <div className="flex-row items-center gap-xs font-mono">
                <span className="text-accent" style={{ fontSize: '0.8rem', fontWeight: 'bold' }}>▸</span>
                <span style={{ color: 'var(--text-secondary)' }}>settings/security.auth</span>
              </div>
              <span className="badge badge-neutral font-mono" style={{ fontSize: '0.65rem' }}>
                [ CREDENTIALS ]
              </span>
            </div>
            <div style={{ padding: '1.25rem' }}>
              <div className="flex-row justify-between items-center mb-md">
                <h2 className="card-title font-mono" style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.95rem', margin: 0 }}>
                  <span className="text-muted" style={{ opacity: 0.5 }}>{'//'}</span>
                  <span>🔒 Segurança da Conta</span>
                </h2>
              </div>
              <form onSubmit={handleUpdatePassword} style={{ display: 'flex', flexDirection: 'column', gap: '1.2rem' }}>
                {passwordError && <div className="alert-error">{passwordError}</div>}
                {passwordSuccess && <div style={{ color: 'var(--color-success)', fontSize: '0.9rem', padding: '0.5rem 0' }}>{passwordSuccess}</div>}

                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label font-mono">Senha Atual</label>
                  <input 
                    type="password" 
                    className="form-input font-mono" 
                    placeholder="••••••••" 
                    value={currentPassword} 
                    onChange={(e) => setCurrentPassword(e.target.value)} 
                    required 
                  />
                </div>

                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label font-mono">Nova Senha</label>
                  <input 
                    type="password" 
                    className="form-input font-mono" 
                    placeholder="Mínimo 6 caracteres" 
                    value={newPassword} 
                    onChange={(e) => setNewPassword(e.target.value)} 
                    required 
                  />
                </div>

                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label font-mono">Confirmar Nova Senha</label>
                  <input 
                    type="password" 
                    className="form-input font-mono" 
                    placeholder="Repita a nova senha" 
                    value={confirmPassword} 
                    onChange={(e) => setConfirmPassword(e.target.value)} 
                    required 
                  />
                </div>

                <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '0.5rem' }}>
                  <button type="submit" className="primary-button font-mono" disabled={isSavingPassword}>
                    {isSavingPassword ? 'Alterando...' : 'Alterar Senha'}
                  </button>
                </div>
              </form>
            </div>
          </section>

          {/* Seção 3: Integrações */}
          <section className="card flex-col gap-0" style={{ width: '100%', padding: 0, overflow: 'hidden' }}>
            <div className="terminal-bar">
              <div className="flex-row items-center gap-xs font-mono">
                <span className="text-accent" style={{ fontSize: '0.8rem', fontWeight: 'bold' }}>▸</span>
                <span style={{ color: 'var(--text-secondary)' }}>settings/telegram.bot</span>
              </div>
              <span className="badge badge-neutral font-mono" style={{ fontSize: '0.65rem' }}>
                [ {telegramStatus?.linked ? 'LINKED' : 'UNLINKED'} ]
              </span>
            </div>
            <div style={{ padding: '1.25rem' }}>
              <div className="flex-row justify-between items-center mb-md">
                <h2 className="card-title font-mono" style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.95rem', margin: 0 }}>
                  <span className="text-muted" style={{ opacity: 0.5 }}>{'//'}</span>
                  <span>📱 Canal de Alertas (Telegram)</span>
                </h2>
              </div>
              {isLoadingTelegram ? (
                <div style={{ padding: '1rem', textAlign: 'center' }}>
                  <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)' }}></span>
                  <span style={{ marginLeft: '10px', color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '0.85rem' }}>Carregando integrações...</span>
                </div>
              ) : telegramError ? (
                <div className="alert-error">{telegramError}</div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                  <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', margin: 0, textAlign: 'left', lineHeight: 1.5 }}>
                    Vincule sua conta com o bot oficial do <strong>Stock Pulse</strong> no Telegram para receber alertas automáticos de preços e variações em tempo real.
                  </p>

                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '1rem', background: 'rgba(255, 255, 255, 0.02)', borderRadius: '2px', border: '1px solid var(--panel-border)', marginTop: '0.5rem', flexWrap: 'wrap', gap: '1rem' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
                      <span style={{ fontSize: '2rem' }}>📱</span>
                      <div style={{ textAlign: 'left' }}>
                        <span style={{ display: 'block', fontWeight: 700, fontSize: '0.95rem', fontFamily: 'var(--font-mono)' }}>Stock Pulse Telegram Bot</span>
                        <span style={{ color: telegramStatus?.linked ? 'var(--color-success)' : 'var(--text-secondary)', fontSize: '0.8rem', fontWeight: 600, fontFamily: 'var(--font-mono)' }}>
                          {telegramStatus?.linked ? `Vinculado com sucesso (Chat ID: ${telegramStatus.chat_id})` : 'Não Vinculado'}
                        </span>
                      </div>
                    </div>

                    <div>
                      {telegramStatus?.linked ? (
                        <button className="btn-danger font-mono" onClick={handleUnlinkTelegram}>
                          Desvincular
                        </button>
                      ) : (
                        <button className="primary-button font-mono" onClick={handleLinkTelegram} style={{ padding: '0.5rem 1.25rem', fontSize: '0.85rem' }}>
                          Vincular Conta
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </div>
          </section>

          {/* Seção 4: Controle de Workers */}
          <section className="card flex-col gap-0" style={{ width: '100%', padding: 0, overflow: 'hidden' }}>
            <div className="terminal-bar">
              <div className="flex-row items-center gap-xs font-mono">
                <span className="text-accent" style={{ fontSize: '0.8rem', fontWeight: 'bold' }}>▸</span>
                <span style={{ color: 'var(--text-secondary)' }}>settings/workers.daemons</span>
              </div>
              <span className="badge badge-neutral font-mono" style={{ fontSize: '0.65rem' }}>
                [ {workers.length} WORKERS ]
              </span>
            </div>
            <div style={{ padding: '1.25rem' }}>
              <div className="card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 0 1rem 0', margin: 0, borderBottom: '1px solid var(--panel-border)', marginBottom: '1.25rem' }}>
                <h2 className="card-title font-mono" style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.95rem', margin: 0 }}>
                  <span className="text-muted" style={{ opacity: 0.5 }}>{'//'}</span>
                  <span>⚙️ Controle de Workers</span>
                </h2>
                <button 
                  className="btn-secondary font-mono" 
                  onClick={() => loadWorkers()} 
                  disabled={isLoadingWorkers} 
                  style={{ fontSize: '0.82rem', display: 'flex', alignItems: 'center', gap: '5px' }}
                >
                  {isLoadingWorkers ? 'Atualizando...' : '🔄 Atualizar'}
                </button>
              </div>
              
              {/* Chave de Administração para Workers (Segura: Em Memória Apenas) */}
              <div style={{ marginBottom: '1.25rem', display: 'flex', flexDirection: 'column', gap: '6px' }}>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <label htmlFor="adminKeyInput" style={{ fontSize: '0.82rem', fontWeight: 600, color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)' }}>
                    Chave de Administração (X-Admin-Key):
                  </label>
                  <div style={{ display: 'flex', gap: '8px', flex: '1 1 280px' }}>
                    <input
                      id="adminKeyInput"
                      type="password"
                      placeholder="Insira a ADMIN_API_KEY para autenticar"
                      value={adminKey}
                      onChange={(e) => setAdminKey(e.target.value)}
                      className="form-input font-mono"
                      style={{ fontSize: '0.82rem', padding: '0.45rem 0.75rem', flex: 1 }}
                    />
                    <button
                      type="button"
                      className="btn-secondary font-mono"
                      onClick={() => {
                        loadWorkers(adminKey);
                        loadAdminAssets(adminKey);
                      }}
                      style={{ fontSize: '0.82rem', whiteSpace: 'nowrap' }}
                    >
                      Conectar
                    </button>
                  </div>
                </div>
                <span style={{ fontSize: '0.72rem', color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
                  🔒 Chave segura em memória (sessão ativa) — nunca persistida no navegador ou no bundle.
                </span>
              </div>

              {workersError && <div className="alert-error" style={{ marginBottom: '1rem' }}>{workersError}</div>}
              {workerSuccess && <div style={{ color: 'var(--color-success)', fontSize: '0.9rem', padding: '0.5rem 0', fontWeight: 600, marginBottom: '1rem', fontFamily: 'var(--font-mono)' }}>{workerSuccess}</div>}

              {isLoadingWorkers && workers.length === 0 ? (
                <div style={{ padding: '2rem', textAlign: 'center' }}>
                  <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)' }}></span>
                  <span style={{ marginLeft: '10px', color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '0.85rem' }}>Carregando workers...</span>
                </div>
              ) : (
                <div className="table-container" style={{ WebkitOverflowScrolling: 'touch', border: '1px solid var(--panel-border)', borderRadius: '2px' }}>
                  <table className="data-table" style={{ tableLayout: 'auto' }}>
                    <thead>
                      <tr>
                        <th style={{ width: '32px', textAlign: 'center', padding: '0.75rem 0.4rem' }}>#</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Worker</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Descrição</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Intervalo</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Execuções</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Status</th>
                        <th style={{ padding: '0.75rem 0.5rem', textAlign: 'right' }}>Ação</th>
                      </tr>
                    </thead>
                    <tbody>
                      {workers.map((worker, idx) => (
                        <tr key={worker.name}>
                          <td className="table-line-num">{String(idx + 1).padStart(2, '0')}</td>
                          <td style={{ padding: '0.75rem 0.5rem', fontWeight: 700, color: 'var(--text-primary)', fontFamily: 'var(--font-mono)' }}>{worker.name}</td>
                          <td style={{ padding: '0.75rem 0.5rem', color: 'var(--text-secondary)', fontSize: '0.8rem', maxWidth: '220px', whiteSpace: 'normal', wordBreak: 'break-word' }}>{worker.description}</td>
                          <td style={{ padding: '0.75rem 0.5rem', fontSize: '0.85rem' }}>
                            <span className="badge badge-neutral font-mono" style={{ fontSize: '0.75rem', padding: '0.2rem 0.5rem' }}>
                              {worker.interval}
                            </span>
                          </td>
                          <td style={{ padding: '0.75rem 0.5rem' }}>
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '2px', fontSize: '0.8rem', fontFamily: 'var(--font-mono)' }}>
                              <span style={{ color: 'var(--text-primary)', whiteSpace: 'nowrap' }} title="Última Execução">
                                <span style={{ color: 'var(--text-secondary)', marginRight: '4px', fontSize: '0.75rem' }}>Últ:</span>
                                {formatDate(worker.last_run)}
                              </span>
                              <span style={{ color: 'var(--text-primary)', whiteSpace: 'nowrap' }} title="Próxima Execução">
                                <span style={{ color: 'var(--text-secondary)', marginRight: '4px', fontSize: '0.75rem' }}>Próx:</span>
                                {formatDate(worker.next_run)}
                              </span>
                            </div>
                          </td>
                          <td style={{ padding: '0.75rem 0.5rem' }}>
                            {worker.status === 'running' ? (
                              <span className="badge badge-success font-mono" style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                                <span className="pulse-dot"></span>
                                Executando
                              </span>
                            ) : (
                              <span className="badge badge-neutral font-mono">Inativo</span>
                            )}
                          </td>
                          <td style={{ padding: '0.75rem 0.5rem', textAlign: 'right' }}>
                            <button
                              className="primary-button font-mono"
                              onClick={() => handleTriggerWorker(worker.name)}
                              disabled={triggeringWorker === worker.name || worker.status === 'running'}
                              style={{ padding: '0.35rem 0.85rem', fontSize: '0.78rem' }}
                            >
                              {triggeringWorker === worker.name ? 'Disparando...' : 'Executar'}
                            </button>
                          </td>
                        </tr>
                      ))}
                      {workers.length === 0 && (
                        <tr>
                          <td colSpan={7} style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '2rem', fontFamily: 'var(--font-mono)' }}>
                            Nenhum worker registrado no sistema.
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </section>

          {/* Seção 5: Catálogo Global de Ativos (Admin) */}
          <section className="card flex-col gap-0" style={{ width: '100%', padding: 0, overflow: 'hidden' }}>
            <div className="terminal-bar">
              <div className="flex-row items-center gap-xs font-mono">
                <span className="text-accent" style={{ fontSize: '0.8rem', fontWeight: 'bold' }}>▸</span>
                <span style={{ color: 'var(--text-secondary)' }}>settings/assets.catalog</span>
              </div>
              <span className="badge badge-neutral font-mono" style={{ fontSize: '0.65rem' }}>
                [ {adminAssets.length} ATIVOS ]
              </span>
            </div>
            <div style={{ padding: '1.25rem' }}>
              <div className="card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 0 1rem 0', margin: 0, borderBottom: '1px solid var(--panel-border)', marginBottom: '1.25rem', flexWrap: 'wrap', gap: '0.75rem' }}>
                <h2 className="card-title font-mono" style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.95rem', margin: 0 }}>
                  <span className="text-muted" style={{ opacity: 0.5 }}>{'//'}</span>
                  <span>🗂️ Catálogo de Ativos (Admin)</span>
                </h2>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <input
                    type="text"
                    placeholder="Filtrar por ticker ou nome..."
                    value={assetSearchFilter}
                    onChange={(e) => setAssetSearchFilter(e.target.value)}
                    className="form-input font-mono"
                    style={{ fontSize: '0.8rem', padding: '0.4rem 0.75rem', width: '220px' }}
                  />
                  <button
                    type="button"
                    className="btn-secondary font-mono"
                    onClick={() => loadAdminAssets()}
                    disabled={isLoadingAssets}
                    style={{ fontSize: '0.82rem', display: 'flex', alignItems: 'center', gap: '5px' }}
                  >
                    {isLoadingAssets ? 'Carregando...' : '🔄 Recarregar Catálogo'}
                  </button>
                </div>
              </div>

              {assetsError && <div className="alert-error" style={{ marginBottom: '1rem' }}>{assetsError}</div>}
              {assetsSuccess && <div style={{ color: 'var(--color-success)', fontSize: '0.9rem', padding: '0.5rem 0', fontWeight: 600, marginBottom: '1rem', fontFamily: 'var(--font-mono)' }}>{assetsSuccess}</div>}

              {isLoadingAssets && adminAssets.length === 0 ? (
                <div style={{ padding: '2rem', textAlign: 'center' }}>
                  <span className="loading-spinner" style={{ borderTopColor: 'var(--accent-color)' }}></span>
                  <span style={{ marginLeft: '10px', color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '0.85rem' }}>Carregando catálogo de ativos...</span>
                </div>
              ) : (
                <div className="table-container" style={{ WebkitOverflowScrolling: 'touch', border: '1px solid var(--panel-border)', borderRadius: '2px', maxHeight: '420px', overflowY: 'auto' }}>
                  <table className="data-table" style={{ tableLayout: 'auto' }}>
                    <thead>
                      <tr>
                        <th style={{ width: '32px', textAlign: 'center', padding: '0.75rem 0.4rem' }}>#</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Ticker</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Nome</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Moeda</th>
                        <th style={{ padding: '0.75rem 0.5rem' }}>Categoria Oficial</th>
                      </tr>
                    </thead>
                    <tbody>
                      {adminAssets
                        .filter((a) => {
                          const q = assetSearchFilter.trim().toLowerCase();
                          return !q || a.ticker.toLowerCase().includes(q) || a.name.toLowerCase().includes(q);
                        })
                        .map((asset, idx) => (
                          <tr key={asset.id}>
                            <td className="table-line-num">{String(idx + 1).padStart(2, '0')}</td>
                            <td style={{ padding: '0.75rem 0.5rem', fontWeight: 700, color: 'var(--text-primary)', fontFamily: 'var(--font-mono)' }}>{asset.ticker}</td>
                            <td style={{ padding: '0.75rem 0.5rem', color: 'var(--text-secondary)', fontSize: '0.8rem' }}>{asset.name}</td>
                            <td style={{ padding: '0.75rem 0.5rem' }}>
                              <span className="badge badge-neutral font-mono" style={{ fontSize: '0.75rem', padding: '0.2rem 0.5rem' }}>
                                {asset.currency}
                              </span>
                            </td>
                            <td style={{ padding: '0.75rem 0.5rem' }}>
                              <select
                                aria-label={`Categoria ${asset.ticker}`}
                                className="form-input font-mono"
                                value={asset.asset_type}
                                disabled={savingAssetId === asset.id}
                                onChange={(e) => handleUpdateAssetType(asset.id, asset.ticker, e.target.value)}
                                style={{ fontSize: '0.8rem', padding: '0.35rem 0.6rem', minWidth: '180px' }}
                              >
                                {ASSET_TYPE_OPTIONS.map((opt) => (
                                  <option key={opt.value} value={opt.value} style={{ background: '#1c1f24' }}>
                                    {opt.label}
                                  </option>
                                ))}
                              </select>
                            </td>
                          </tr>
                        ))}
                      {adminAssets.length === 0 && (
                        <tr>
                          <td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-secondary)', padding: '2rem', fontFamily: 'var(--font-mono)' }}>
                            Nenhum ativo encontrado no catálogo.
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </section>

          {/* Seção 6: Zona de Perigo */}
          <section className="card flex-col gap-0" style={{ width: '100%', padding: 0, overflow: 'hidden', borderColor: 'rgba(var(--danger-rgb), 0.3)', background: 'var(--color-danger-bg)' }}>
            <div className="terminal-bar" style={{ background: 'rgba(var(--danger-rgb), 0.08)', borderBottomColor: 'rgba(var(--danger-rgb), 0.2)' }}>
              <div className="flex-row items-center gap-xs font-mono">
                <span className="terminal-dot red"></span>
                <span style={{ color: 'var(--color-danger)' }}>settings/danger_zone.destruct</span>
              </div>
              <span className="badge badge-danger font-mono" style={{ fontSize: '0.65rem' }}>
                [ HIGH_SEVERITY ]
              </span>
            </div>
            <div style={{ padding: '1.25rem' }}>
              <div className="card-header" style={{ borderColor: 'rgba(var(--danger-rgb), 0.2)', padding: '0 0 0.75rem 0', margin: 0, marginBottom: '1rem' }}>
                <h2 className="card-title font-mono" style={{ color: 'var(--color-danger)', fontSize: '0.95rem', margin: 0 }}>⚠️ Zona de Perigo</h2>
              </div>
              <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'space-between', alignItems: 'center', gap: '1rem' }}>
                <div style={{ textAlign: 'left', flex: 1, minWidth: '250px' }}>
                  <span style={{ display: 'block', fontWeight: 700, fontSize: '0.92rem', fontFamily: 'var(--font-mono)', color: 'var(--text-primary)' }}>Excluir Conta Permanentemente</span>
                  <span style={{ color: 'var(--text-secondary)', fontSize: '0.82rem' }}>
                    A exclusão é irreversível e remove todos os seus dados da plataforma.
                  </span>
                </div>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <input
                    type="password"
                    aria-label="Senha atual para excluir conta"
                    placeholder="Confirme sua senha atual"
                    value={deletePassword}
                    onChange={(e) => setDeletePassword(e.target.value)}
                    className="form-input font-mono"
                    style={{ fontSize: '0.82rem', padding: '0.5rem 0.75rem', width: '210px' }}
                  />
                  <button className="btn-danger font-mono" onClick={handleDeleteAccount} style={{ padding: '0.6rem 1.5rem' }}>
                    Excluir Conta
                  </button>
                </div>
              </div>
            </div>
          </section>

        </div>
      </main>

      <IdeStatusBar
        wsConnected={true}
        currency="BRL"
        version="v2.4-term"
      />
    </div>
  );
}

