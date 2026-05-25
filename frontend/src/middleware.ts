import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  // Resgata o cookie de acesso
  const token = request.cookies.get('access_token')?.value;
  const { pathname } = request.nextUrl;

  // Definição de rotas públicas de autenticação e rotas privadas protegidas
  const isProtectedRoute = pathname.startsWith('/dashboard') || pathname.startsWith('/portfolio');
  const isAuthRoute = pathname === '/login' || pathname === '/register' || pathname === '/';

  // Se o usuário tentar acessar rota protegida sem token, redireciona para login
  if (isProtectedRoute && !token) {
    const loginUrl = new URL('/login', request.url);
    // Guarda a página de destino original para posterior redirecionamento se desejado
    loginUrl.searchParams.set('from', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Se o usuário já estiver autenticado e tentar acessar login/registro/home, redireciona para dashboard
  if (isAuthRoute && token) {
    return NextResponse.redirect(new URL('/dashboard', request.url));
  }

  return NextResponse.next();
}

// Configura o matcher do Next.js para rodar o middleware apenas nas rotas mapeadas, otimizando performance
export const config = {
  matcher: [
    '/',
    '/login',
    '/register',
    '/dashboard/:path*',
    '/portfolio/:path*',
  ],
};
