-- ==============================================================================
-- Script de Correção e Padronização de Categorias de Ativos (asset.asset_type)
-- Executar diretamente no PostgreSQL do container:
-- docker compose exec -T db psql -U root -d stockpulse < scripts/fix_asset_types.sql
-- ==============================================================================

BEGIN;

-- 1. Garante que todos os tipos estejam em maiúsculo e sem espaços em branco
UPDATE asset
SET asset_type = UPPER(TRIM(asset_type)),
    updated_at = NOW()
WHERE asset_type != UPPER(TRIM(asset_type));

-- 2. Converte criptoativos identificados por hífen ou moeda CRYPTO
UPDATE asset
SET asset_type = 'CRYPTO',
    updated_at = NOW()
WHERE (ticker LIKE '%-%' OR currency = 'CRYPTO' OR asset_type = 'CRYPTOCURRENCY')
  AND asset_type != 'CRYPTO';

-- 3. Converte aliases diretos de mercado internacional
UPDATE asset
SET asset_type = 'STOCK_US',
    updated_at = NOW()
WHERE asset_type = 'EQUITY_US';

-- 4. Converte aliases diretos de B3
UPDATE asset
SET asset_type = 'STOCK_BR',
    updated_at = NOW()
WHERE asset_type = 'EQUITY_BR';

-- 5. Converte BDRs (terminados em 34.SA, 35.SA, 39.SA)
UPDATE asset
SET asset_type = 'BDR',
    updated_at = NOW()
WHERE (ticker LIKE '%34.SA' OR ticker LIKE '%35.SA' OR ticker LIKE '%39.SA')
  AND asset_type NOT IN ('BDR');

-- 6. Converte fundos e ETFs negociados na B3 (final 11.SA)
-- 6.1 FIAGRO
UPDATE asset
SET asset_type = 'FIAGRO',
    updated_at = NOW()
WHERE ticker LIKE '%11.SA'
  AND (LOWER(name) LIKE '%fiagro%' OR LOWER(name) LIKE '%agro%')
  AND asset_type NOT IN ('FIAGRO');

-- 6.2 ETF_BR
UPDATE asset
SET asset_type = 'ETF_BR',
    updated_at = NOW()
WHERE ticker LIKE '%11.SA'
  AND (
    LOWER(name) LIKE '%etf%' OR
    LOWER(name) LIKE '%ishares%' OR
    LOWER(name) LIKE '%índice%' OR
    LOWER(name) LIKE '%indice%' OR
    LOWER(name) LIKE '%hashdex%' OR
    LOWER(name) LIKE '%trend%' OR
    LOWER(name) LIKE '%investo%'
  )
  AND asset_type NOT IN ('ETF_BR', 'FIAGRO');

-- 6.3 FII (Fundos Imobiliários)
UPDATE asset
SET asset_type = 'FII',
    updated_at = NOW()
WHERE (ticker LIKE '%11.SA' OR asset_type = 'REAL_ESTATE')
  AND (
    LOWER(name) LIKE '%fii%' OR
    LOWER(name) LIKE '%fundo imob%' OR
    LOWER(name) LIKE '%fdo imob%' OR
    LOWER(name) LIKE '%imobiliário%' OR
    LOWER(name) LIKE '%imobiliario%' OR
    LOWER(name) LIKE '%renda imob%' OR
    LOWER(name) LIKE '%tijolo%' OR
    LOWER(name) LIKE '%lajes%' OR
    LOWER(name) LIKE '%recebíveis%' OR
    LOWER(name) LIKE '%recebiveis%' OR
    asset_type = 'REAL_ESTATE'
  )
  AND asset_type NOT IN ('FII', 'ETF_BR', 'FIAGRO');

-- 7. Converte registros genéricos remanescentes ('EQUITY' ou 'STOCK')
-- 7.1 Ativos brasileiros (.SA ou BRL)
UPDATE asset
SET asset_type = 'STOCK_BR',
    updated_at = NOW()
WHERE asset_type IN ('EQUITY', 'STOCK')
  AND (ticker LIKE '%.SA' OR currency = 'BRL');

-- 7.2 Ativos internacionais (EUA / Global)
UPDATE asset
SET asset_type = CASE
    WHEN LOWER(name) LIKE '%etf%' OR LOWER(name) LIKE '%trust%' OR LOWER(name) LIKE '%fund%' THEN 'ETF_US'
    ELSE 'STOCK_US'
  END,
  updated_at = NOW()
WHERE asset_type IN ('EQUITY', 'STOCK')
  AND ticker NOT LIKE '%.SA'
  AND currency != 'BRL';

-- 8. Verificação final: exibe os tipos resultantes no banco
SELECT asset_type, COUNT(*) AS total_ativos
FROM asset
GROUP BY asset_type
ORDER BY total_ativos DESC;

COMMIT;
