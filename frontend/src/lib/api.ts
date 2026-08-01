const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

/**
 * apiFetch é um wrapper sobre o fetch() nativo.
 * Ele automaticamente injeta credentials para cookies e intercepta erros HTTP (incluindo 401).
 * Em caso de 401, ele pausa a requisição, tenta renovar o token (/auth/refresh) 
 * de forma thread-safe (sem concorrência de refresh) e re-tenta a requisição original.
 */
export async function apiFetch(path: string, options: RequestInit = {}): Promise<Response> {
  const url = path.startsWith('http') ? path : `${API_URL}${path}`;

  const config: RequestInit = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include', // Essencial para HttpOnly cookies (access_token, refresh_token)
  };

  let res = await fetch(url, config);

  // Interceptação global de 401 Unauthorized
  if (res.status === 401) {
    const refreshSuccess = await attemptRefresh();
    if (refreshSuccess) {
      // Re-tenta a requisição original se o refresh deu certo
      res = await fetch(url, config);
    } else {
      // Se refresh falhou, derruba a sessão localmente e redireciona (opcional)
      throw new Error('Sessão expirada');
    }
  }

  return res;
}

/**
 * attemptRefresh lida com a concorrência de múltiplos requests recebendo 401 ao mesmo tempo.
 * Ele garante que apenas UMA requisição para `/auth/refresh` seja feita, 
 * e as outras aguardam a resolução.
 */
async function attemptRefresh(): Promise<boolean> {
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  isRefreshing = true;
  refreshPromise = new Promise(async (resolve) => {
    try {
      const res = await fetch(`${API_URL}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
      });
      resolve(res.ok);
    } catch (e) {
      resolve(false);
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  });

  return refreshPromise;
}
