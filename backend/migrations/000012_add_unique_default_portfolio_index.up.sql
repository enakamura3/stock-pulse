CREATE UNIQUE INDEX idx_portfolio_single_default 
ON portfolio (user_id) 
WHERE is_default = true;
