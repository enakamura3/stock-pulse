DROP TABLE IF EXISTS treasury_depletions CASCADE;
DROP TABLE IF EXISTS treasury_prices CASCADE;
DROP TABLE IF EXISTS treasury_transactions CASCADE;
DROP TABLE IF EXISTS treasury_assets CASCADE;
DROP TABLE IF EXISTS anbima_holidays CASCADE;

-- Clean up base assets entries added under subclasses
DELETE FROM asset WHERE asset_type = 'TREASURY';
