CREATE TABLE IF NOT EXISTS portfolio_asset_override (
    portfolio_id UUID NOT NULL REFERENCES portfolio(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES asset(id) ON DELETE CASCADE,
    asset_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (portfolio_id, asset_id)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_asset_override_asset_id ON portfolio_asset_override(asset_id);
