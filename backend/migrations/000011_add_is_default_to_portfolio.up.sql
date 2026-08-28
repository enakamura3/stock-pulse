ALTER TABLE portfolio ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT false;

WITH first_portfolios AS (
    SELECT DISTINCT ON (user_id) id
    FROM portfolio
    ORDER BY user_id, created_at ASC
)
UPDATE portfolio
SET is_default = true
WHERE id IN (SELECT id FROM first_portfolios);
