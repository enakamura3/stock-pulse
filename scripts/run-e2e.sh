#!/bin/bash
set -e

echo "🔧 Criando o banco de testes (stockpulse_test) dentro do container do Postgres já existente..."
source .env 2>/dev/null || true
PG_USER=${POSTGRES_USER:-root}

# Cria a base de dados ignorando o erro se ela já existir
docker exec stock-pulse-db psql -U $PG_USER -d postgres -c "CREATE DATABASE stockpulse_test;" || echo "Banco de testes já existe, prosseguindo..."

echo "🚀 Trocando a conexão do Backend e rodando as migrações no banco de testes..."
docker compose --env-file .env.e2e up -d

echo "⏳ Aguardando os serviços reiniciarem e recompilarem (pode levar alguns segundos)..."
until curl -s http://localhost:8080/healthz > /dev/null; do
  echo "Aguardando backend na porta 8080..."
  sleep 2
done

until curl -s http://localhost:3000 > /dev/null; do
  echo "Aguardando frontend na porta 3000..."
  sleep 2
done

echo "✅ Ambiente pronto!"

echo "🧪 Rodando testes E2E com Playwright em um container isolado..."
docker run --rm \
  --network stock-pulse_stock-pulse-net \
  -v $(pwd)/frontend:/app \
  -w /app \
  -e NEXT_PUBLIC_API_URL=http://backend:8080/api/v1 \
  -e PLAYWRIGHT_TEST_BASE_URL=http://frontend:3000 \
  mcr.microsoft.com/playwright:v1.44.0-jammy \
  bash -c "npm install && npx playwright test"

echo "🧹 Limpando ambiente de testes (revertendo o backend para a base de Dev)..."
docker compose up -d

echo "🗑️ Apagando o banco de testes..."
docker exec stock-pulse-db psql -U $PG_USER -d postgres -c "DROP DATABASE IF EXISTS stockpulse_test;" || true

echo "✅ Testes E2E finalizados com sucesso! A aplicação já está apontando para o seu banco de desenvolvimento novamente."
