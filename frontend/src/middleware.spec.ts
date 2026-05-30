import { middleware } from './middleware';
import { NextRequest, NextResponse } from 'next/server';
import { vi } from 'vitest';

vi.mock('next/server', () => {
  return {
    NextResponse: {
      redirect: vi.fn((url) => ({ status: 307, url: url.toString(), cookies: { delete: vi.fn() } })),
      next: vi.fn(() => ({ status: 200 })),
    },
  };
});

vi.mock('jose', () => ({
  jwtVerify: vi.fn().mockImplementation(async (token) => {
    if (token === 'invalid_token') throw new Error('Invalid token');
    return { payload: { user_id: '123' } };
  }),
}));

describe('Middleware', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  const createRequest = (path: string, token: string | undefined): NextRequest => {
    return {
      nextUrl: { pathname: path },
      url: `http://localhost${path}`,
      cookies: {
        get: () => (token ? { value: token } : undefined),
      },
    } as unknown as NextRequest;
  };

  it('redirects to login when accessing protected route without token', async () => {
    const req = createRequest('/dashboard', undefined);
    const res = await middleware(req);
    expect(NextResponse.redirect).toHaveBeenCalled();
    expect((res as any).url).toContain('/login');
  });

  it('redirects to dashboard when accessing auth route with token', async () => {
    const req = createRequest('/login', 'token123');
    const res = await middleware(req);
    expect(NextResponse.redirect).toHaveBeenCalled();
    expect((res as any).url).toContain('/dashboard');
  });

  it('allows access to protected route with token', async () => {
    const req = createRequest('/dashboard', 'token123');
    await middleware(req);
    expect(NextResponse.next).toHaveBeenCalled();
  });

  it('allows access to public route without token', async () => {
    const req = createRequest('/about', undefined);
    await middleware(req);
    expect(NextResponse.next).toHaveBeenCalled();
  });
});
