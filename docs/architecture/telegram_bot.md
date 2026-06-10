# 🤖 Integração com Telegram Bot

O módulo do Telegram do **stock-pulse** não é um mero notificador passivo; ele funciona como um terminal bidirecional onde o usuário pode operar suas carteiras nativamente. 

Como o bot interage publicamente na rede do Telegram, qualquer pessoa no mundo poderia conversar com ele. A segurança do sistema repousa em um processo de *Binding* seguro e em um controle de estado baseado em Redis.

## Fluxo de Autenticação Cruzada (Binding)

Para que a conta do Telegram (`chat_id`) do usuário seja vinculada com a conta Web da aplicação (`user_id`), nós construímos um processo de emparelhamento por Token Temporário (Pin Code ou Link Seguro).

```mermaid
sequenceDiagram
    participant User as Usuário Web
    participant App as App Web (Next.js)
    participant Go as Backend (Go)
    participant Redis as Redis Cache
    participant DB as PostgreSQL
    participant Bot as Telegram Bot

    User->>App: Clica em "Vincular Telegram"
    App->>Go: POST /api/telegram/link-token
    Go->>Redis: Gera Token de 6 Caracteres e Guarda 'tg_link:{token} -> user_id' (TTL 5m)
    Go-->>App: Retorna `https://t.me/O_Bot?start={token}`
    App-->>User: Exibe Link ou QR Code
    
    User->>Bot: Clica no link e inicia o chat (/start {token})
    Bot->>Go: Webhook Recebe Payload de Inicialização
    Go->>Redis: Verifica Token Recebido
    alt Token Válido e Ativo
        Redis-->>Go: Retorna 'user_id'
        Go->>DB: Salva `telegram_chat_id` na tabela de Usuário
        Go->>Redis: Apaga Token (One-Time Use)
        Go-->>Bot: "Conta Vinculada com Sucesso!"
    else Token Inválido/Expirado
        Go-->>Bot: "Token Inválido ou Expirado. Tente Novamente."
    end
```

## Gerenciamento de Estado de Sessão Multi-Carteira

Um usuário corporativo ou experiente pode ter múltiplas carteiras (ex: *Principal*, *Dividendos*, *Exterior*, *Cripto*). O Telegram Bot permite trocar de contexto a qualquer momento. Mas como a requisição do Telegram não usa "Cookies", como o bot sabe em qual carteira você está executando um comando (ex: `/resumo` ou `/dividendos`)?

Utilizamos o **Redis** para reter o "State" do Bot:

```mermaid
stateDiagram-v2
    [*] --> ContaNaoVinculada
    
    ContaNaoVinculada --> ContextoPadrao: /start {token}
    
    state ContextoPadrao {
        [*] --> RedisSessionPadrao
        RedisSessionPadrao: tg_session:{chat_id} -> portfolio_id (Carteira A)
    }
    
    ContextoPadrao --> EscolhaCarteira: Comando /carteiras
    
    state EscolhaCarteira {
        [*] --> BotMostraInlineMenu
        BotMostraInlineMenu: Bot envia botões com as Carteiras
        BotMostraInlineMenu --> ClickCarteiraB: Usuário clica na "Carteira B"
        ClickCarteiraB --> UpdateRedis: Altera Redis tg_session:{chat_id} -> portfolio_id (Carteira B)
    }
    
    EscolhaCarteira --> NovoContexto: Redireciona
    
    state NovoContexto {
        [*] --> ExecutandoComandos
        ExecutandoComandos: Qualquer comando injeta o 'portfolio_id' gravado no Redis
    }
```

O Redis permite essa troca veloz e persistente. Cada mensagem que o bot recebe dispara um "Intercepetor (Middleware)" que puxa a sessão do Redis, identifica silenciosamente qual ID de carteira pertence àquele chat, e roteia a lógica bancária exata em milissegundos.
