CREATE TABLE fixed_income_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolio(id) ON DELETE CASCADE,
    institution VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    debt_type VARCHAR(50) NOT NULL,
    indexer VARCHAR(50) NOT NULL,
    rate NUMERIC(10, 4) NOT NULL,
    maturity_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE fixed_income_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES fixed_income_assets(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    amount NUMERIC(15, 4) NOT NULL,
    date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE index_rates (
    indexer VARCHAR(50) NOT NULL,
    date DATE NOT NULL,
    rate NUMERIC(15, 8) NOT NULL,
    PRIMARY KEY (indexer, date)
);

CREATE INDEX idx_fi_assets_portfolio_id ON fixed_income_assets(portfolio_id);
CREATE INDEX idx_fi_transactions_asset_id ON fixed_income_transactions(asset_id);
