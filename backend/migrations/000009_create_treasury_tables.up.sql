-- 1. Create ANBIMA Holidays Table
CREATE TABLE anbima_holidays (
    holiday_date DATE PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE anbima_holidays IS 'National financial holidays published by ANBIMA used for DU/252 interest rate calculations';
COMMENT ON COLUMN anbima_holidays.holiday_date IS 'The holiday date (UTC midnight, format YYYY-MM-DD)';
COMMENT ON COLUMN anbima_holidays.description IS 'Description/name of the holiday';

-- 2. Create Treasury Assets Table (Inherits from asset table via Class Table Inheritance)
CREATE TABLE treasury_assets (
    id UUID PRIMARY KEY REFERENCES asset(id) ON DELETE CASCADE,
    treasury_type VARCHAR(50) NOT NULL,
    maturity_date DATE NOT NULL,
    has_coupons BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_treasury_type CHECK (treasury_type IN ('SELIC', 'PREFIXADO', 'IPCA+'))
);

COMMENT ON TABLE treasury_assets IS 'Public treasury assets extending the global asset table';

-- 3. Create Treasury Transactions Table (Lot-by-lot tracking)
CREATE TABLE treasury_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolio(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES treasury_assets(id) ON DELETE RESTRICT,
    type VARCHAR(50) NOT NULL,
    quantity NUMERIC(18, 8) NOT NULL,
    unit_price NUMERIC(15, 6) NOT NULL,
    contracted_rate NUMERIC(10, 4) NOT NULL, -- e.g., 6.15 for 6.15% or 12.45 for 12.45%
    remaining_quantity NUMERIC(18, 8) NOT NULL DEFAULT 0.00000000, -- tracked for FIFO depletion (subscriptions only)
    transaction_date DATE NOT NULL,
    
    -- Realized accounting metrics populated on redemption
    gross_amount NUMERIC(15, 2),
    iof_tax NUMERIC(15, 2),
    ir_tax NUMERIC(15, 2),
    b3_fee NUMERIC(15, 2),
    net_amount NUMERIC(15, 2),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_treasury_tx_type CHECK (type IN ('SUBSCRIPTION', 'REDEMPTION')),
    CONSTRAINT chk_quantity_positive CHECK (quantity > 0),
    CONSTRAINT chk_unit_price_positive CHECK (unit_price > 0),
    CONSTRAINT chk_remaining_qty_non_negative CHECK (remaining_quantity >= 0),
    CONSTRAINT chk_contracted_rate_non_negative CHECK (contracted_rate >= 0)
);

COMMENT ON TABLE treasury_transactions IS 'Lot-by-lot treasury transactions linked to portfolios';

-- 4. Create Treasury Daily Prices Table (For daily MtM and curve logs)
CREATE TABLE treasury_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES treasury_assets(id) ON DELETE CASCADE,
    price_date DATE NOT NULL,
    yield_rate NUMERIC(10, 4) NOT NULL,            -- Offered rate/yield on that day
    selling_price NUMERIC(15, 6) NOT NULL,         -- Preço Venda from Tesouro API (for subscription/MtM reference)
    redemption_price NUMERIC(15, 6) NOT NULL,      -- Preço Resgate from Tesouro API (for redemptions)
    theoretical_price NUMERIC(15, 6),              -- Theoretical curve price
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (asset_id, price_date)
);

COMMENT ON TABLE treasury_prices IS 'Historical daily prices for treasury bonds including MtM and theoretical curve price';

-- 5. Create Treasury Depletions Junction Table (FIFO depletion links)
CREATE TABLE treasury_depletions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_transaction_id UUID NOT NULL REFERENCES treasury_transactions(id) ON DELETE CASCADE,
    redemption_transaction_id UUID NOT NULL REFERENCES treasury_transactions(id) ON DELETE CASCADE,
    quantity NUMERIC(18, 8) NOT NULL,              -- quantity depleted from subscription lot
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_depleted_qty_positive CHECK (quantity > 0)
);

COMMENT ON TABLE treasury_depletions IS 'FIFO depletion links mapping redemptions to subscription lots';

-- 6. Indexes for queries & FIFO sorting
CREATE INDEX idx_treasury_transactions_portfolio_id ON treasury_transactions(portfolio_id);
CREATE INDEX idx_treasury_transactions_asset_id_date ON treasury_transactions(asset_id, transaction_date);
CREATE INDEX idx_treasury_prices_asset_date ON treasury_prices(asset_id, price_date);
CREATE INDEX idx_treasury_depletions_sub_id ON treasury_depletions(subscription_transaction_id);
CREATE INDEX idx_treasury_depletions_red_id ON treasury_depletions(redemption_transaction_id);
