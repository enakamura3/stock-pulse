# Cobertura de Testes - StockPulse

Este documento registra o estado atual e a arquitetura da cobertura de testes unitários da aplicação StockPulse, separados entre Backend e Frontend.

## 🖥️ Backend (Golang) - Média Global: 86.8%

O backend da aplicação utiliza a biblioteca padrão de testes do Go em conjunto com `pgxmock` para simular as interações com o banco de dados PostgreSQL.

| Módulo / Pacote | Cobertura | Status | Observação |
| :--- | :---: | :---: | :--- |
| `internal/docs` (Swagger UI) | **100%** | 🟢 | Coberto 100% com `httptest`. |
| `internal/mail` (SMTP) | **100%** | 🟢 | 100% de cobertura com injeção de dependência e mocks. |
| `internal/market` (Yahoo Fin) | **100%** | 🟢 | Respostas HTTP cacheadas e handlers 100% cobertos. |
| `internal/middleware` | **100%** | 🟢 | Autenticação e métricas (Prometheus) com 100%. |
| `internal/portfolio` | **~96%** | 🟢 | Repositório em 100%. Falta ~4% em workers e serviços auxiliares. |
| `internal/watchlist` | **~93%** | 🟢 | Handlers 100%. Falta ~7% de erros raros de DB (`pgxmock`). |
| `internal/database` | **88.9%** | 🟡 | Resta simular falha grave no Pool de conexão PostgreSQL. |
| `internal/auth` | **~85%** | 🟡 | Falta cobrir falhas raras de geração de Hash e Refresh Tokens falsos. |
| `internal/websocket` | **~83%** | 🟡 | Restam cenários de _timeout_ e queda de rede (_WritePump_). |
| `internal/alert` | **~80%** | 🟡 | Regras de checagem do _worker_ necessitam simulações de falha de _trigger_. |
| `cmd/api` (Ponto de Entrada) | **0%** | 🔴 | O `main.go` intencionalmente excluído dos testes para evitar travamentos (_blockings_). |

---

## 🎨 Frontend (Next.js) - Média Global Aproximada: ~15% - 20%

A suíte do frontend (`Vitest` e `React Testing Library`) iniciou sua jornada com testes de fumaça (_Smoke Tests_) e validações base. Devido ao tamanho de alguns componentes monolíticos de UI, testes _End-to-End_ (E2E) com Playwright são recomendados para complementar as validações de interação.

| Módulo / Arquivo | Cobertura | Status | Observação |
| :--- | :---: | :---: | :--- |
| `src/app/page.tsx` (Home) | **100%** | 🟢 | Totalmente testada (textos e botões localizados e validados). |
| `src/context/AuthContext.tsx` | **~96.8%** | 🟢 | Lógica de estado global, refresh de sessão e chamadas na API 100% testada. |
| `src/middleware.ts` | **Alta** | 🟢 | Redirecionamento de rotas protegidas validado com sucesso. |
| `src/app/layout.tsx` | **Alta** | 🟢 | Injeção do `AuthProvider` e renderização HTML validada. |
| `src/app/login/page.tsx` | **Baixa** | 🔴 | Dificuldade com asserções assíncronas (_Race condition_ entre render e API). |
| `src/app/register/page.tsx` | **Baixa** | 🔴 | O mesmo problema da página de login; fluxo de usuário extenso. |
| `src/app/dashboard/**/*.tsx` | **Baixa** | 🔴 | Testes _smoke_ cobrem renderização, porém há mais de 1000 linhas de web sockets, abas e modals ausentes no teste. |
| `src/components/PortfolioChart`| **Baixa** | 🔴 | Manipulação intensiva do Canvas por `lightweight-charts` exige abstração forte para testes. |

### Próximos Passos e Metas
- Implementar testes E2E via **Playwright** para garantir que fluxos pesados no Frontend (Dashboard, Cadastro, Alertas) sejam testados renderizando a árvore DOM num navegador _headless_.
- Abstrair o `main.go` no Backend para possibilitar validações de setup do _Graceful Shutdown_.
