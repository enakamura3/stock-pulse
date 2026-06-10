# 🔐 Autenticação e Segurança (Auth)

O módulo de Autenticação do **stock-pulse** foi projetado para oferecer segurança de nível bancário contra as principais vulnerabilidades da web (como XSS e CSRF), mantendo uma experiência de login fluida com Refresh Tokens contínuos.

## Arquitetura de Criptografia

Diferente de sistemas legados que utilizam `bcrypt` ou `SHA-256`, o Stock Pulse utiliza **Argon2id**, o vencedor do *Password Hashing Competition*, parametrizado com as recomendações estritas da OWASP (64MB de memória, paralelismo = 4). Isso garante extrema resistência estrutural contra ataques de força bruta realizados por GPUs modernas.

## Fluxo de Sessão (JWT & HttpOnly)

O uso de `localStorage` para salvar tokens JWT em aplicações React/Next.js abre enormes brechas para roubos via Cross-Site Scripting (XSS). Para blindar a plataforma, **a aplicação frontend nunca tem acesso ao token JWT no JavaScript**.

A emissão e o consumo de tokens são geridos de forma estrita pelo backend através de Cookies `HttpOnly`.

```mermaid
sequenceDiagram
    participant User as Navegador (Next.js)
    participant API as Go Backend (Auth)
    participant PG as PostgreSQL (Users)
    participant Redis as Redis Cache

    Note over User,API: Fluxo de Login Seguro
    User->>API: POST /api/auth/login {email, password}
    API->>PG: Busca Hash Argon2id pelo E-mail
    PG-->>API: Retorna Hash
    API->>API: Computa e Compara Argon2id
    
    alt Senha Válida
        API->>API: Gera AccessToken (JWT assinado via HMAC-SHA256, TTL 2h)
        API->>API: Gera RefreshToken (String Criptográfica Opaque)
        API->>Redis: Salva chave 'refresh_token:{token}' (TTL 7d)
        API-->>User: Retorna 200 OK + Set-Cookie (HttpOnly, Secure, SameSite=Lax)
    else Senha Inválida
        API-->>User: Retorna 401 Unauthorized
    end
```

### O Sistema de Dual-Token (Access e Refresh)

1. **Access Token (JWT):** Emitido com um "Tempo de Vida" (TTL) extremamente curto (2 horas). Se um atacante interceptar a requisição (mesmo com SSL), o token perde a validade muito rápido. O payload contém apenas a identificação primária (`user_id`).
2. **Refresh Token (Opaque):** Um token gerado aleatoriamente (`crypto/rand`) com 32 bytes de entropia e salvo no **Redis** associado ao ID do usuário, durando 7 dias.

### Renovação de Sessão (Silent Refresh)

O Next.js intercepta automaticamente qualquer erro `401 Unauthorized` retornado pela API e dispara, em background, a tentativa de renovação:

```mermaid
sequenceDiagram
    participant User as Navegador (Axios Interceptor)
    participant API as Go Backend (Auth)
    participant Redis as Redis Cache

    Note over User,API: Interceptor HTTP captura 401
    User->>API: POST /api/auth/refresh (Envia Cookie RefreshToken)
    API->>Redis: Busca chave 'refresh_token:{token}'
    alt Token Existe no Redis
        Redis-->>API: Retorna 'user_id'
        API->>API: Invalida token antigo no Redis (One-Time Use)
        API->>API: Gera novos AccessToken e RefreshToken
        API->>Redis: Salva novo RefreshToken
        API-->>User: Retorna 200 OK + Novos Cookies (HttpOnly)
        Note over User: Re-executa a requisição HTTP original!
    else Token Inexistente ou Expirado
        API-->>User: Retorna 401 Unauthorized
        Note over User: Redireciona para /login e expurga estado local
    end
```

Esse ciclo de **One-Time Use Refresh Tokens** (Rotação de Token) atua como um sistema de detecção de invasão. Se o usuário original e o hacker tentarem usar o mesmo Refresh Token para renovar a sessão, um deles falhará e a sessão inteira será instantaneamente revogada.
