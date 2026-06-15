ALTER TABLE asset_event DROP CONSTRAINT IF EXISTS asset_event_asset_id_ex_date_type_key;
ALTER TABLE asset_event ADD CONSTRAINT asset_event_unique_key UNIQUE (asset_id, ex_date, type, gross_amount);
