import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch } from './api';

// Mocking o fetch global
global.fetch = vi.fn();

describe('apiFetch Wrapper', () => {
  beforeEach(() => {
    (global.fetch as any).mockClear();
  });

  it('deve injetar credentials e Content-Type: application/json corretamente', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
    });

    await apiFetch('/test-endpoint');

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/test-endpoint'),
      expect.objectContaining({
        credentials: 'include',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
        }),
      })
    );
  });

  it('deve NÃO injetar Content-Type quando o body for FormData', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
    });

    const formData = new FormData();
    formData.append('file', new Blob(['test']), 'test.csv');

    await apiFetch('/upload', { method: 'POST', body: formData });

    const calledHeaders = (global.fetch as any).mock.calls[0][1].headers;
    expect(calledHeaders['Content-Type']).toBeUndefined();
    expect((global.fetch as any).mock.calls[0][1].credentials).toBe('include');
  });

  it('deve retornar a Response original para que o caller possa checar res.ok', async () => {
    const mockResponse = {
      ok: false,
      status: 400,
      json: async () => ({ error: 'Parâmetro inválido' }),
    };
    (global.fetch as any).mockResolvedValueOnce(mockResponse);

    const res = await apiFetch('/test');
    expect(res.status).toBe(400);
    expect(res.ok).toBe(false);
  });

  it('deve interceptar 401, tentar /auth/refresh e re-tentar a requisição original', async () => {
    // 1ª chamada: endpoint original falha com 401
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 401,
    });
    // 2ª chamada: POST /auth/refresh retorna 200 OK
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
    });
    // 3ª chamada: endpoint original retentado com sucesso
    const successResponse = {
      ok: true,
      status: 200,
      json: async () => ({ data: 'sucesso' }),
    };
    (global.fetch as any).mockResolvedValueOnce(successResponse);

    const result = await apiFetch('/protected-resource');

    expect(result).toBe(successResponse);
    expect(global.fetch).toHaveBeenCalledTimes(3);
    
    // Verifica se a segunda chamada foi para renovar o token
    expect((global.fetch as any).mock.calls[1][0]).toContain('/auth/refresh');
    expect((global.fetch as any).mock.calls[1][1].method).toBe('POST');
  });

  it('deve lançar erro se o refresh também falhar (sessão expirada)', async () => {
    // 1ª chamada: 401
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 401,
    });
    // 2ª chamada: refresh falha
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 401,
    });

    await expect(apiFetch('/protected')).rejects.toThrow('Sessão expirada');
    expect(global.fetch).toHaveBeenCalledTimes(2);
  });

  it('deve construir URL absoluta a partir de path relativo', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
    });

    await apiFetch('/portfolios/123');

    expect((global.fetch as any).mock.calls[0][0]).toContain('/portfolios/123');
    expect((global.fetch as any).mock.calls[0][0]).toMatch(/^http/);
  });

  it('deve preservar URL absoluta quando path já começa com http', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      status: 200,
    });

    await apiFetch('https://custom-api.com/endpoint');

    expect((global.fetch as any).mock.calls[0][0]).toBe('https://custom-api.com/endpoint');
  });
});
