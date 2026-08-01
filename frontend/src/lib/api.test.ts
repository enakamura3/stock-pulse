import { apiFetch } from './api';

// Mocking o fetch global
global.fetch = jest.fn();

describe('apiFetch Wrapper', () => {
  beforeEach(() => {
    (global.fetch as jest.Mock).mockClear();
  });

  it('deve injetar credentials e cabeçalhos corretamente', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
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

  it('deve retornar a resposta original mesmo quando houver erro 400', async () => {
    const mockResponse = {
      ok: false,
      status: 400,
      json: async () => ({ error: 'Parâmetro inválido' }),
    };
    (global.fetch as jest.Mock).mockResolvedValueOnce(mockResponse);

    const res = await apiFetch('/test');
    expect(res.status).toBe(400);
    expect(res.ok).toBe(false);
  });

  it('deve interceptar 401, tentar /auth/refresh e re-tentar a requisição original com sucesso', async () => {
    // 1º chamada falha (401)
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 401,
    });
    // 2º chamada é o POST /auth/refresh que brilha (200 OK)
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      status: 200,
    });
    // 3º chamada é a requisição original repetida, agora com sucesso (200 OK)
    const successResponse = {
      ok: true,
      status: 200,
      json: async () => ({ data: 'sucesso' }),
    };
    (global.fetch as jest.Mock).mockResolvedValueOnce(successResponse);

    const result = await apiFetch('/protected-resource');

    expect(result).toBe(successResponse);
    expect(global.fetch).toHaveBeenCalledTimes(3);
    
    // Verifica se a segunda chamada foi para renovar o token
    expect((global.fetch as jest.Mock).mock.calls[1][0]).toContain('/auth/refresh');
    expect((global.fetch as jest.Mock).mock.calls[1][1].method).toBe('POST');
  });

  it('deve falhar a requisição se o refresh (401) também falhar', async () => {
    // 1º chamada original (401)
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 401,
    });
    // 2º chamada /auth/refresh falha (401/403)
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 401,
    });

    await expect(apiFetch('/protected')).rejects.toThrow('Sessão expirada');
    expect(global.fetch).toHaveBeenCalledTimes(2);
  });
});
