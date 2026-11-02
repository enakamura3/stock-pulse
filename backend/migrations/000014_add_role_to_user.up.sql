ALTER TABLE "user" ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'USER';

-- Promove o primeiro usuário cadastrado para ADMIN para garantir acesso administrativo ao proprietário da instalação
UPDATE "user"
SET role = 'ADMIN'
WHERE id IN (
    SELECT id FROM "user" ORDER BY created_at ASC LIMIT 1
);
