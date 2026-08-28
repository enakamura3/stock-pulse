import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { jwtVerify } from 'jose';

export async function middleware(request: NextRequest) {
  // Resgata o cookie de acesso
  const token = request.cookies.get('access_token')?.value;
  const { pathname } = request.nextUrl;

  // Definição de rotas públicas de autenticação e rotas privadas protegidas
  const isProtectedRoute = pathname.startsWith('/dashboard') || pathname.startsWith('/portfolio');
  const isAuthRoute = pathname === '/login' || pathname === '/register' || pathname === '/';

  let isValid = false;

  if (token) {
    try {
      const secret = new TextEncoder().encode(process.env.JWT_SECRET || 'stock-pulse-dev-secret-key-super-secure');
      await jwtVerify(token, secret);
      isValid = true;
    } catch (err) {
      isValid = false;
    }
  }

  // Se o usuário tentar acessar rota protegida sem token válido, redireciona para login
  if (isProtectedRoute && !isValid) {
    const loginUrl = new URL('/login', request.url);
    // Guarda a página de destino original para posterior redirecionamento se desejado
    loginUrl.searchParams.set('from', pathname);
    
    // Opcional: deletar o cookie inválido na resposta
    const response = NextResponse.redirect(loginUrl);
    response.cookies.delete('access_token');
    return response;
  }

  // Se o usuário já estiver autenticado e tentar acessar login/registro/home, redireciona para dashboard
  if (isAuthRoute && isValid) {
    return NextResponse.redirect(new URL('/dashboard/portfolio', request.url));
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
